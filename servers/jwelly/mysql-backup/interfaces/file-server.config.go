package mysql_backup_interfaces

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

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
	Upload(*os.File, *ConfigWithConnection, string)
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

func (s *SFileServerType1) Upload(f *os.File, cb *ConfigWithConnection, _ string) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	fileName := filepath.Base(f.Name())
	part1, errFile1 := writer.CreateFormFile(fileName, filepath.Join(cb.BaseDir, fileName))
	if errFile1 != nil {
		fmt.Println(errFile1)
		return
	}
	_, errFile1 = io.Copy(part1, f)
	if errFile1 != nil {
		fmt.Println(errFile1)
		return
	}
	_ = writer.WriteField("path", s.FolderPath)
	err := writer.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	u, err := url.Parse(s.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	u.Path = path.Join(u.Path, fileName)
	client := &http.Client{}
	req, err := http.NewRequest("POST", u.String(), payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+s.TOKEN)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
