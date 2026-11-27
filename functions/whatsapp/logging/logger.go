// Package whatsapp_logging holds the HTTP request logger shared by the
// WhatsApp servers. The logger keeps API tokens out of the log output.
package whatsapp_logging

import (
	"io"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
)

// requestLogFormat lists every field the request log writes. It has no
// header, query or body tags, so the X-Api-Token header never reaches the log.
// It must differ from logger.DefaultFormat: Fiber writes that format with a
// fast path that ignores CustomTags.
const requestLogFormat = "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n"

const redactedSegment = "***"

// redactLogPath hides the token in /scan/<token>. Routing is not case
// sensitive, so the "scan" segment is matched in any case, and every segment
// after it is masked.
func redactLogPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.EqualFold(segment, "scan") && i+1 < len(segments) {
			for j := i + 1; j < len(segments); j++ {
				if segments[j] != "" {
					segments[j] = redactedSegment
				}
			}
			return strings.Join(segments, "/")
		}
	}
	return path
}

func writeRedactedPath(output logger.Buffer, c fiber.Ctx, _ *logger.Data, _ string) (int, error) {
	return output.WriteString(sanitizeLogValue(redactLogPath(c.Path())))
}

// writeRedactedError logs the handler error with the request's token removed.
// Some API errors quote the token, for example "Number <token> Not Found".
func writeRedactedError(output logger.Buffer, c fiber.Ctx, data *logger.Data, _ string) (int, error) {
	if data == nil || data.ChainErr == nil {
		return output.WriteString("-")
	}
	return output.WriteString(sanitizeLogValue(redactSecrets(data.ChainErr.Error(), requestSecrets(c))))
}

// requestSecrets returns the values that must never be logged for c: the
// X-Api-Token header and the /scan/ path segments.
func requestSecrets(c fiber.Ctx) []string {
	var secrets []string
	if token := c.Get(env.RequestTokenHeaderKey); token != "" {
		secrets = append(secrets, token)
	}
	path := c.Path()
	redacted := redactLogPath(path)
	if redacted != path {
		pathSegments := strings.Split(path, "/")
		for i, segment := range strings.Split(redacted, "/") {
			if segment == redactedSegment && pathSegments[i] != "" {
				secrets = append(secrets, pathSegments[i])
			}
		}
	}
	return secrets
}

func redactSecrets(value string, secrets []string) string {
	for _, secret := range secrets {
		value = strings.ReplaceAll(value, secret, redactedSegment)
	}
	return value
}

// sanitizeLogValue replaces control characters so a request path cannot
// start a fake log line.
func sanitizeLogValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '?'
		}
		return r
	}, value)
}

// NewRedactingLogger returns a request logger that writes to stdout and never
// logs the API token: /scan/<token> path segments are masked, the
// X-Api-Token header is not logged, and both are removed from handler errors.
func NewRedactingLogger() fiber.Handler {
	return NewRedactingLoggerTo(os.Stdout)
}

// NewRedactingLoggerTo is NewRedactingLogger with a custom output stream.
// A nil stream writes to stdout.
func NewRedactingLoggerTo(stream io.Writer) fiber.Handler {
	if stream == nil {
		stream = os.Stdout
	}
	return logger.New(logger.Config{
		Format: requestLogFormat,
		Stream: stream,
		CustomTags: map[string]logger.LogFunc{
			// Override both tags so a later format change cannot log the
			// raw path or the query string.
			logger.TagPath:  writeRedactedPath,
			logger.TagURL:   writeRedactedPath,
			logger.TagError: writeRedactedError,
		},
	})
}
