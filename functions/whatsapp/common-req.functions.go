package whatsapp_functions

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
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

// FetchFileFromURL downloads a caller-supplied HTTP/HTTPS URL. Private,
// loopback and metadata addresses are refused and the size is capped; see
// utility_functions.FetchUserURL.
func FetchFileFromURL(ctx context.Context, urls string) ([]byte, string, error) {
	bytesData, err := utility_functions.FetchUserURL(ctx, urls)
	if err != nil {
		return nil, "", err
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
