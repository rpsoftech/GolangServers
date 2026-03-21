package interfaces

type SSLConfig struct {
	KeyFilePath  string `json:"key" validate:"required"`
	CertFilePath string `json:"cert" validate:"required"`
	// CAFilePaths  []string `json:"caFilePaths"`
}
