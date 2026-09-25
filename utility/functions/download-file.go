package utility_functions

import (
	"context"
	"os"
)

// DownloadFile saves a caller-supplied http(s) URL to filepath. It uses the
// same guards as FetchUserURL: public addresses only, size capped. An existing
// file at filepath is replaced, not appended to.
func DownloadFile(filepath string, url string) error {
	data, err := FetchUserURL(context.Background(), url)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}
