package soham_common_req_keys

import "time"

// MaxWebsocketMessageSize caps one websocket message between the server and
// the client. The client sets it as its read limit, and the server rejects a
// request whose message would be larger before it writes it. Base64 media is
// the largest payload.
const MaxWebsocketMessageSize = 64 << 20 // 64 MiB

// Base send timeouts per recipient, and the overall cap.
const (
	TextSendTimeout     = 10 * time.Second
	MediaSendTimeout    = 30 * time.Second
	WebMediaSendTimeout = 45 * time.Second
	MaxSendTimeout      = 5 * time.Minute
)

// ERROR_MESSAGE_TOO_LARGE is returned when a request does not fit in one
// websocket message.
const ERROR_MESSAGE_TOO_LARGE = 210

// BaseSendTimeout is the send timeout for one recipient of messageType. It is
// 0 for a type that sends nothing.
func BaseSendTimeout(messageType MessageType) time.Duration {
	switch messageType {
	case SEND_TEXT_MESSAGE:
		return TextSendTimeout
	case SEND_BASE64_IMAGE, SEND_FILE_PATH_MEDIA:
		return MediaSendTimeout
	case SEND_WEB_MEDIA:
		return WebMediaSendTimeout
	}
	return 0
}

// SendTimeout is how long the client may take to send messageType to
// recipients: the base timeout per recipient, capped at MaxSendTimeout.
// Fewer than one recipient counts as one, and an unknown type gets the cap.
// The server waits a little longer than this for the client's reply.
func SendTimeout(messageType MessageType, recipients int) time.Duration {
	base := BaseSendTimeout(messageType)
	if recipients < 1 {
		recipients = 1
	}
	if base <= 0 {
		return MaxSendTimeout
	}
	if int64(recipients) > int64(MaxSendTimeout/base) {
		return MaxSendTimeout
	}
	return base * time.Duration(recipients)
}
