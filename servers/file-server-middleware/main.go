package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid" // Run: go get github.com/google/uuid
	"github.com/rpsoftech/golang-servers/env"
)

const (
	PUBLIC_DOMAIN        = "PUBLIC_DOMAIN"
	MAIN_SERVER_URL      = "MAIN_SERVER_URL"
	MAIN_SERVER_SUB_PATH = "MAIN_SERVER_SUB_PATH"
	MAIN_SERVER_TOKEN    = "MAIN_SERVER_TOKEN"
	CURRENT_SERVER_TOKEN = "CURRENT_SERVER_TOKEN"
)

type SeverConfig struct {
	// The public URL prefix you will return to the user
	PublicDomain string `json:"publicDomain" validate:"required"`
	// The hidden backend server that actually stores the files
	MainServerURL      string `json:"mainServerUrl" validate:"required"`
	MainServerSubPath  string `json:"mainServerSubPath" validate:"required"`
	MainServerToken    string `json:"mainServerToken" validate:"required"`
	CurrentServerToken string `json:"currentServerToken" validate:"required"`
}

var config SeverConfig

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(env.XApiToken)
	if token == "" || token != config.CurrentServerToken {
		http.Error(w, "Invalid Token", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Error reading stream", http.StatusInternalServerError)
			return
		}

		if part.FileName() != "" {
			ext := filepath.Ext(part.FileName())
			onlyName := uuid.New().String()
			newFileName := onlyName + ext

			// ==========================================
			// 1. SPOOL FILE TO LOCAL DISK (Saves RAM)
			// ==========================================
			tmpFile, err := os.CreateTemp("", "upl-*.tmp")
			if err != nil {
				http.Error(w, "Server disk error", http.StatusInternalServerError)
				return
			}

			// Stream from user to local disk
			if _, err := io.Copy(tmpFile, part); err != nil {
				os.Remove(tmpFile.Name()) // Clean up on fail
				http.Error(w, "Failed to read upload", http.StatusInternalServerError)
				return
			}

			// Get exact file size and rewind the file reader to the beginning
			fileStat, _ := tmpFile.Stat()
			tmpFile.Seek(0, io.SeekStart)

			// ==========================================
			// 2. BUILD THE MULTIPART HEADERS IN MEMORY
			// ==========================================
			headerBuf := &bytes.Buffer{}
			writer := multipart.NewWriter(headerBuf)
			writer.WriteField("path", config.MainServerSubPath)
			// Create the file boundary header (but don't write the file data to it)
			_, err = writer.CreateFormFile(onlyName, newFileName)
			if err != nil {
				os.Remove(tmpFile.Name())
				http.Error(w, "Failed to create form format", http.StatusInternalServerError)
				return
			}

			// Manually construct the footer boundary
			footerStr := fmt.Sprintf("\r\n--%s--\r\n", writer.Boundary())
			footerReader := strings.NewReader(footerStr)

			// Calculate the EXACT final Content-Length
			totalSize := int64(headerBuf.Len()) + fileStat.Size() + int64(len(footerStr))

			// ==========================================
			// 3. COMBINE EVERYTHING INTO A SINGLE STREAM
			// ==========================================
			// This chains the Headers -> File Data (from disk) -> Footer into one stream
			multiReader := io.MultiReader(headerBuf, tmpFile, footerReader)

			// ==========================================
			// 4. FIRE THE REQUEST TO MAIN SERVER
			// ==========================================
			reqUrl := fmt.Sprintf("%s/%s?appendExt=true", config.MainServerURL, onlyName)
			req, err := http.NewRequest("POST", reqUrl, multiReader)
			if err != nil {
				os.Remove(tmpFile.Name())
				http.Error(w, "Failed to create backend request", http.StatusInternalServerError)
				return
			}

			// THE MAGIC BULLET: Tell CloudLinux the exact size so it doesn't use Chunked Encoding
			req.ContentLength = totalSize

			req.Header.Set("Authorization", "Bearer "+config.MainServerToken)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			client := &http.Client{}
			resp, err := client.Do(req)

			// Always clean up the temp file from the disk when the request finishes
			tmpFile.Close()
			os.Remove(tmpFile.Name())

			if err != nil {
				log.Printf("Failed to connect: %v", err)
				http.Error(w, "Main server unreachable", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errorBody, _ := io.ReadAll(resp.Body)
				log.Printf("❌ MAIN SERVER REJECTED! Code: %d, Body: %s", resp.StatusCode, string(errorBody))
				http.Error(w, "Main server rejected the upload", http.StatusBadGateway)
				return
			}

			// Success!
			finalURL := fmt.Sprintf("%s/%s", config.PublicDomain, newFileName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "success", "url": "%s"}`, finalURL)
			return
		}
	}

	http.Error(w, "No file found in request", http.StatusBadRequest)
}

// startTempFileSweeper runs a background goroutine that cleans up orphaned
// upload files in case the server crashes before a defer os.Remove() executes.
func startTempFileSweeper() {
	// Wake up and run the scan every 1 hour
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for range ticker.C {
			tempDir := os.TempDir() // Gets the system temp directory (/tmp)

			// Read all files in the temp directory
			entries, err := os.ReadDir(tempDir)
			if err != nil {
				log.Printf("Sweeper error: could not read temp directory: %v", err)
				continue
			}

			now := time.Now()
			deletedCount := 0

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				name := entry.Name()
				// Only target files created by our uploadHandler
				if strings.HasPrefix(name, "upl-") && strings.HasSuffix(name, ".tmp") {
					info, err := entry.Info()
					if err != nil {
						continue
					}

					// If the file is older than 24 hours, delete it
					if now.Sub(info.ModTime()) > 24*time.Hour {
						fullPath := filepath.Join(tempDir, name)
						if err := os.Remove(fullPath); err == nil {
							deletedCount++
						}
					}
				}
			}

			if deletedCount > 0 {
				log.Printf("Background Sweeper: Cleaned up %d orphaned temp files.", deletedCount)
			}
		}
	}()
}

func main() {
	env.LoadEnv("file-server.env")
	startTempFileSweeper()
	config = SeverConfig{
		PublicDomain:       env.Env.GetEnv(PUBLIC_DOMAIN),
		MainServerURL:      env.Env.GetEnv(MAIN_SERVER_URL),
		MainServerSubPath:  env.Env.GetEnv(MAIN_SERVER_SUB_PATH),
		MainServerToken:    env.Env.GetEnv(MAIN_SERVER_TOKEN),
		CurrentServerToken: env.Env.GetEnv(CURRENT_SERVER_TOKEN),
	}
	env.ValidateEnv(config)
	http.HandleFunc("/upload", uploadHandler)
	port := env.GetServerPort(env.PORT_KEY)
	fmt.Printf("Upload Gateway running on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
