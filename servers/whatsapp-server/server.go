package main

import (
	"errors"
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	whatsapp_middleware "github.com/rpsoftech/golang-servers/functions/whatsapp/middleware"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_server_apis "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/apis"
)

// requestLogFormat lists every field the request log writes. It has no
// header, query or body tags, so the X-Api-Token header never reaches the log.
const requestLogFormat = "[${time}] ${ip} ${status} - ${latency} ${method} ${path} ${error}\n"

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
			logger.TagPath: writeRedactedPath,
			logger.TagURL:  writeRedactedPath,
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
