package whatsapp_server_apis

import (
	"context"

	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
)

// whatsappSender is the part of a WhatsApp connection that the send handlers
// use. *whatsapp_core.WhatsappConnection satisfies it. Tests replace
// getConnection to return a stub.
type whatsappSender interface {
	ReturnStatusError() error
	SendTextMessage(ctx context.Context, to []string, msg string) *map[string]bool
	SendMediaFileFromURLs(ctx context.Context, to []string, mediaURL []string, msg string)
	SendMediaFileFromURL(ctx context.Context, to []string, mediaURL string, fileName string, msg string) *map[string]bool
	SendMediaFileBase64(ctx context.Context, to []string, base64Data string, fileName string, msg string) *map[string]bool
	SendMediaFileWithPath(ctx context.Context, to []string, filePath string, fileName string, msg string) *map[string]bool
}

var _ whatsappSender = (*whatsapp_core.WhatsappConnection)(nil)

// getConnection looks up the connection for a token. It never returns a
// non-nil interface that wraps a nil pointer.
var getConnection = func(token string) (whatsappSender, bool) {
	connection, ok := whatsapp_core.ConnectionMap.Get(token)
	if !ok || connection == nil {
		return nil, false
	}
	return connection, true
}
