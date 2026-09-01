package mysql_backup_interfaces

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"

	"github.com/rpsoftech/golang-servers/validator"
)

type SFileServerConfig struct {
	URL   string `json:"url" validate:"required"`
	TOKEN string `json:"token" validate:"required"`
}

type SFileServerType1 struct {
	*SFileServerConfig
	FolderPath string `json:"folderPath" validate:"required"`
}

type IFileServerConfigInterface interface {
	Validate() (bool, error)
	Upload(fileReader io.Reader, fileName string, cb *ConfigWithConnection) error
}

func (s *SFileServerConfig) Validate() (bool, error) {
	if errs := validator.Validator.Validate(s); len(errs) > 0 {
		// panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
		return false, fmt.Errorf("CONFIG_ERROR %#v", errs)
	}
	return true, nil
}
func (s *SFileServerType1) Validate() (bool, error) {
	if errs := validator.Validator.Validate(s); len(errs) > 0 {
		// panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
		return false, fmt.Errorf("CONFIG_ERROR %#v", errs)
	}
	return true, nil
}
func (s *SFileServerType1) Upload(fileReader io.Reader, fileName string, cb *ConfigWithConnection) error {
	// Stream the multipart form data directly to HTTP to avoid loading the file into RAM
	bodyReader, bodyWriter := io.Pipe()
	writer := multipart.NewWriter(bodyWriter)

	go func() {
		defer bodyWriter.Close()
		defer writer.Close()

		_ = writer.WriteField("path", s.FolderPath)
		part, err := writer.CreateFormFile("file", fileName) // field name is standard 'file'
		if err != nil {
			log.Printf("form file error: %v", err)
			return
		}

		if _, err := io.Copy(part, fileReader); err != nil {
			log.Printf("io.Copy error: %v", err)
		}
	}()

	u, err := url.Parse(s.URL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, fileName)

	req, err := http.NewRequest("POST", u.String(), bodyReader)
	if err != nil {
		return err
	}
	// 1. Mirror the anti-bot headers so your file server doesn't block the download
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Add("Authorization", "Bearer "+s.TOKEN)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("server returned %d: %s", res.StatusCode, string(body))
	}

	return nil
}
