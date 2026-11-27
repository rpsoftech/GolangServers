package whatsapp_server_apis

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_functions "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/functions"
)

const (
	// perRecipientTimeout is the send budget for one recipient (or one
	// recipient and one file, for multi-file sends).
	perRecipientTimeout = 30 * time.Second
	// maxSendTimeout caps the budget for large recipient lists.
	maxSendTimeout = 15 * time.Minute
	// minNumberDigits is the shortest phone number accepted in `to`.
	minNumberDigits = 7
)

// uploadDir is where multipart uploads are stored until they are sent.
// Tests point it at a temporary directory.
var uploadDir = "./tmp"

// sendTimeout scales the per-recipient budget by units and caps it.
func sendTimeout(units int) time.Duration {
	if units < 1 {
		units = 1
	}
	if units > int(maxSendTimeout/perRecipientTimeout) {
		return maxSendTimeout
	}
	return perRecipientTimeout * time.Duration(units)
}

func invalidInput(message string, extra any) *interfaces.RequestError {
	return &interfaces.RequestError{
		StatusCode: http.StatusBadRequest,
		Code:       interfaces.ERROR_INVALID_INPUT,
		Message:    message,
		Name:       "ERROR_INVALID_INPUT",
		Extra:      extra,
	}
}

func invalidBody(err error) *interfaces.RequestError {
	return invalidInput("Invalid Request Body", err.Error())
}

// normalizeRecipients checks every entry of to and returns a copy with
// whitespace and an optional leading "+" removed. The core send functions
// add the "+" themselves.
func normalizeRecipients(to []string) ([]string, error) {
	if len(to) == 0 {
		return nil, invalidInput("Number Not Found", nil)
	}
	out := make([]string, len(to))
	for i, raw := range to {
		number := strings.TrimPrefix(strings.TrimSpace(raw), "+")
		if len(number) < minNumberDigits || !allDigits(number) {
			return nil, invalidInput(fmt.Sprintf("Invalid Number %q", raw), nil)
		}
		out[i] = number
	}
	return out, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseRecipientsField reads the multipart `to` field. It accepts a JSON
// array of strings or numbers, or a comma-separated list.
func parseRecipientsField(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "[") {
		var asStrings []string
		if err := json.Unmarshal([]byte(raw), &asStrings); err == nil {
			return asStrings, nil
		}
		var asNumbers []json.Number
		if err := json.Unmarshal([]byte(raw), &asNumbers); err == nil {
			out := make([]string, len(asNumbers))
			for i, n := range asNumbers {
				out[i] = n.String()
			}
			return out, nil
		}
		return nil, invalidInput("Invalid `to` Field, Expected JSON Array Or Comma Separated List", nil)
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out, nil
}

// parseMsgField reads the multipart `msg` field as plain text. A value that
// is a JSON string literal (the old client format) is decoded.
func parseMsgField(raw string) string {
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		var decoded string
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			return decoded
		}
	}
	return raw
}

// lookupConnection returns the logged-in connection for token, or the
// request error to send back.
func lookupConnection(token string) (whatsappSender, error) {
	connection, ok := getConnection(token)
	if !ok {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_NOT_FOUND,
			Message:    fmt.Sprintf("Number %s Not Found", token),
			Name:       "ERROR_CONNECTION_NOT_FOUND",
		}
	}
	if err := connection.ReturnStatusError(); err != nil {
		return nil, err
	}
	return connection, nil
}

func isHeadless(c fiber.Ctx) bool {
	headless, err := strconv.ParseBool(whatsapp_functions.ExtractKeyFromHeader(c, "Headless"))
	return err == nil && headless
}

// sendFailed is returned when a core send function gives back no result,
// which it does when it could not prepare the media.
func sendFailed(statusCode int, message string) *interfaces.RequestError {
	code := interfaces.ERROR_INTERNAL_SERVER
	name := "ERROR_INTERNAL_SERVER"
	if statusCode == http.StatusBadRequest {
		code = interfaces.ERROR_INVALID_INPUT
		name = "ERROR_INVALID_INPUT"
	}
	return &interfaces.RequestError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Name:       name,
	}
}

// saveUpload writes an uploaded file to a new, unique file in uploadDir and
// returns its path. The caller must remove the file.
func saveUpload(file *multipart.FileHeader) (path string, err error) {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", err
	}
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.CreateTemp(uploadDir, "upload-*"+filepath.Ext(filepath.Base(file.Filename)))
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(dst.Name())
		}
	}()
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	return dst.Name(), nil
}
