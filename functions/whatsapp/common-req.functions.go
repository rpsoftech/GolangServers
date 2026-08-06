package whatsapp_functions

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
)

func ExtractKeyFromHeader(c *fiber.Ctx, key string) string {
	reqHeaders := c.GetReqHeaders()
	if tokenString, foundToken := reqHeaders[key]; !foundToken || len(tokenString) != 1 || tokenString[0] == "" {
		return ""
	} else {
		return tokenString[0]
	}
}
func ExtractNumberFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(whatsapp_interfaces.REQ_LOCAL_NUMBER_KEY).(string)
	if !ok {
		return "", &interfaces.RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.INVALID_NUMBER_FROM_TOKEN,
			Message:    "Invalid Number From Token",
			Name:       "INVALID_NUMBER_FROM_TOKEN",
		}
	}
	return id, nil
}

// fetchFileFromURL downloads a file from the given HTTP/HTTPS URL.
func FetchFileFromURL(urls string) ([]byte, string, error) {
	resp, err := http.Get(urls)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download file, HTTP status code: %d", resp.StatusCode)
	}

	bytesData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to extract filename from URL path if not explicitly provided
	fileName := ""
	parsedURL, err := url.Parse(urls)
	if err == nil {
		parts := strings.Split(parsedURL.Path, "/")
		if len(parts) > 0 {
			fileName = parts[len(parts)-1]
		}
	}

	return bytesData, fileName, nil
}
