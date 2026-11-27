package main

import (
	"errors"
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_middleware "github.com/rpsoftech/golang-servers/functions/whatsapp/middleware"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_server_apis "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/apis"
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

func newRequestLogger(stream io.Writer) fiber.Handler {
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

func setupFiberServer(logStream io.Writer) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var mappedError *interfaces.RequestError
			if !errors.As(err, &mappedError) {
				log.Printf("[Fiber Error] %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(interfaces.RequestError{
					Code:    interfaces.ERROR_INTERNAL_SERVER,
					Message: "Some Internal Error",
					Name:    "Global Error Handler Function",
				})
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})

	app.Use(newRequestLogger(logStream))
	whatsapp_server_apis.AddApis(app.Group("/v1", whatsapp_middleware.TokenDecrypter, whatsapp_middleware.AllowOnlyValidTokenMiddleWare))
	app.Get("/scan/:id", whatsapp_server_apis.OpenBrowserWithQr)
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})

	return app
}
