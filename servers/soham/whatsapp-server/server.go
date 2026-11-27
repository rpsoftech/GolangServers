package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

// bodyLimit matches the websocket read limit of the soham whatsapp-client:
// a request body larger than this could not be forwarded to the client.
const bodyLimit = 64 * 1024 * 1024

// errorHandler is the global Fiber error handler.
//   - *interfaces.RequestError (also when wrapped) is returned as is.
//   - *fiber.Error (404, 413 body too large, bad JSON, ...) keeps its status and message.
//   - Any other error is logged on the server and returned as a generic 500.
func errorHandler(c fiber.Ctx, err error) error {
	var reqErr *interfaces.RequestError
	if errors.As(err, &reqErr) {
		status := reqErr.StatusCode
		if status == 0 {
			status = fiber.StatusInternalServerError
		}
		return c.Status(status).JSON(reqErr)
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code < fiber.StatusInternalServerError {
		return c.Status(fiberErr.Code).JSON(interfaces.RequestError{
			Code:    fiberErr.Code,
			Message: fiberErr.Message,
			Name:    "HTTP_ERROR",
		})
	}
	log.Errorf("unhandled error on %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(interfaces.RequestError{
		Code:    interfaces.ERROR_INTERNAL_SERVER,
		Message: "Some Internal Error",
		Name:    "Global Error Handler Function",
	})
}

// loadTLSConfig reads the SSL config file at sslPath.
// It returns (nil, nil) when the file does not exist, so the server runs without TLS.
// When the file exists, any problem (bad JSON, missing fields, missing or invalid
// certificate or key) is returned as an error. The certificate file may hold the
// full chain (leaf first, then intermediates).
func loadTLSConfig(sslPath string) (*tls.Config, error) {
	dat, err := os.ReadFile(sslPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read SSL config %s: %w", sslPath, err)
	}
	sslConfig := new(interfaces.SSLConfig)
	if err := json.Unmarshal(dat, sslConfig); err != nil {
		return nil, fmt.Errorf("parse SSL config %s: %w", sslPath, err)
	}
	if errs := validator.Validator.Validate(sslConfig); len(errs) > 0 {
		fields := make([]string, 0, len(errs))
		for _, e := range errs {
			fields = append(fields, e.FailedField)
		}
		return nil, fmt.Errorf("invalid SSL config %s: missing or invalid fields %v", sslPath, fields)
	}
	if _, err := os.Stat(sslConfig.CertFilePath); err != nil {
		return nil, fmt.Errorf("SSL cert file %s: %w", sslConfig.CertFilePath, err)
	}
	if _, err := os.Stat(sslConfig.KeyFilePath); err != nil {
		return nil, fmt.Errorf("SSL key file %s: %w", sslConfig.KeyFilePath, err)
	}
	certificate, err := tls.LoadX509KeyPair(sslConfig.CertFilePath, sslConfig.KeyFilePath)
	if err != nil {
		return nil, fmt.Errorf("load SSL certificate %s and key %s: %w", sslConfig.CertFilePath, sslConfig.KeyFilePath, err)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
	}, nil
}
