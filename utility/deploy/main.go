package main

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

const (
	FileServerURL = "https://files.rpso.in/upload/"
	KeyValueURL   = "https://keyvalue.rpso.in/public/"
)

var (
	targets = []struct{ OS, Arch string }{
		{"linux", "amd64"},
		{"windows", "amd64"},
	}
	components = map[string]string{
		updater.MysqlBackupCmdProjectName: "./servers/jwelly/mysql-backup-cmd/main.go",
		updater.WhatsappProjectName:       "./servers/whatsapp-server/main.go",
	}
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "keygen" {
		keygen()
		return
	}

	fileServerToken := os.Getenv("FILE_SERVER_TOKEN")
	kvToken := os.Getenv("KV_TOKEN")
	if fileServerToken == "" || kvToken == "" {
		log.Fatal("FATAL: FILE_SERVER_TOKEN and KV_TOKEN environment variables are missing")
	}

	// Fail before building anything if releases cannot be signed with the key
	// the shipped updater trusts.
	signingKey, err := updater.ParseSigningKey(os.Getenv("UPDATE_SIGNING_KEY"))
	if err != nil {
		log.Fatalf("FATAL: UPDATE_SIGNING_KEY: %v (run `go run ./utility/deploy keygen` once)", err)
	}

	deployEnv := os.Getenv("DEPLOY_ENV")
	if deployEnv == "" {
		deployEnv = string(env.APP_ENV_STAGING) // Fail-safe default
		log.Println("⚠️ DEPLOY_ENV not set. Defaulting to 'staging'.")
	}

	if err := os.MkdirAll("build", 0755); err != nil {
		log.Fatalf("Failed to create build directory: %v", err)
	}

	for _, target := range targets {
		for compName, compPath := range components {
			version := getNextVersion(deployEnv, compName, target.OS, target.Arch)
			log.Printf("🚀 Building %s v%d for %s/%s", compName, version, target.OS, target.Arch)

			binName := updater.GetFileKey(deployEnv, compName, target.OS, target.Arch)
			if target.OS == "windows" {
				binName += ".exe"
			}
			binPath := filepath.Join("build", binName)

			// 1. Build
			cmd := exec.Command("go", "build", "-ldflags", fmt.Sprintf("-s -w -X main.version=%d", version), "-o", binPath, compPath)
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.OS, "GOARCH="+target.Arch)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Fatalf("❌ Build failed: %v", err)
			}

			// 2. Compress
			gzPath := binPath + ".gz"
			log.Printf("📦 Compressing %s...", binName)
			if err := compress(binPath, gzPath); err != nil {
				log.Fatalf("❌ Compression failed: %v", err)
			}

			// 3. Hash
			hash, err := hashFile(gzPath)
			if err != nil {
				log.Fatalf("❌ Hashing failed: %v", err)
			}
			log.Printf("🔐 SHA256: %s", hash)

			// 4. Upload
			log.Printf("☁️ Uploading to File Server...")
			targetFolder := "mysql_backup"
			uploadFilename := binName + ".gz"
			if err := uploadFile(gzPath, uploadFilename, targetFolder, fileServerToken); err != nil {
				log.Fatalf("❌ Upload failed: %v", err)
			}

			// 5. Sign and update KV
			kvKey := updater.GetFileKey(deployEnv, compName, target.OS, target.Arch)
			fileURL := fmt.Sprintf("https://files.rpso.in/static/%s/%s", targetFolder, uploadFilename)
			signature := base64.StdEncoding.EncodeToString(ed25519.Sign(signingKey, updater.SignedMessage(kvKey, version, hash)))
			vInfo, _ := json.Marshal(updater.KVResponse{Version: version, URL: fileURL, SHA256: hash, Signature: signature})

			req, _ := http.NewRequest("POST", KeyValueURL+kvKey, bytes.NewReader(vInfo))
			req.Header.Set("Authorization", "Bearer "+kvToken)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json, text/plain, */*")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalf("❌ KV Store update request failed: %v", err)
			}
			resp.Body.Close() // Ensure body is closed to prevent leaks

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("❌ KV Store returned non-200 status: %d", resp.StatusCode)
			}

			log.Printf("✅ Deployed %s successfully!\n\n", binName)
		}
	}
}

// keygen prints a new release signing key pair. Paste the public key into
// updater.ReleasePublicKey and store the private key as UPDATE_SIGNING_KEY on
// the deploy machine only. Never commit the private key.
func keygen() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("keygen failed: %v", err)
	}
	fmt.Println("Public key  (paste into utility/updater/signing.go ReleasePublicKey):")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	fmt.Println()
	fmt.Println("Private key (set as UPDATE_SIGNING_KEY; keep secret, do not commit):")
	fmt.Println(base64.StdEncoding.EncodeToString(priv))
}

// Helpers with Error Handling Added

func getNextVersion(envName, comp, osName, arch string) int {
	req, _ := http.NewRequest("GET", KeyValueURL+updater.GetFileKey(envName, comp, osName, arch), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 1
	}
	defer resp.Body.Close()

	var v map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&v); err == nil {
		if val, ok := v["version"].(float64); ok {
			return int(val) + 1
		}
	}
	return 1
}

func compress(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	w := gzip.NewWriter(out)
	defer w.Close()

	_, err = io.Copy(w, in)
	return err
}

func hashFile(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func uploadFile(path, filename, uploadPathFolder, token string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile(filename, filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	writer.WriteField("path", uploadPathFolder)
	writer.Close()

	req, err := http.NewRequest("POST", FileServerURL+filename, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // CRITICAL: Prevent memory leaks

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
