package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	FileServerURL = "https://files.rpso.in/upload/"
	KeyValueURL   = "https://keyvalue.rpso.in/public/"
)

type VersionInfo struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

func Sha256File(path string) (string, error) {

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func UploadFile(path string, filename string, uploadPath string, fileServerURL string, fileServerToken string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile(filename, filepath.Base(path))
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: time.Second * 540,
	}
	io.Copy(part, file)

	writer.WriteField("path", uploadPath)

	err = writer.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}

	req, err := http.NewRequest(
		"POST",
		fileServerURL+filename,
		payload,
	)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+fileServerToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("failed to upload file With Status Code: %s", resp.StatusCode)
		return fmt.Errorf("failed to upload file: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(body))
	fmt.Println("Uploaded:", filename)

	return nil
}

func UpdateKeyValue(key string, data []byte, keyValueURL string, kvToken string) error {
	req, err := http.NewRequest(
		"POST",
		keyValueURL+key,
		bytes.NewBuffer(data),
	)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+kvToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	fmt.Println("KeyValue updated:", key)

	return nil
}

func Atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
