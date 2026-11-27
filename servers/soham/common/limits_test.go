package soham_common_req_keys

import (
	"testing"
	"time"
)

func TestSendTimeout(t *testing.T) {
	tests := []struct {
		messageType MessageType
		recipients  int
		want        time.Duration
	}{
		{SEND_TEXT_MESSAGE, 0, 10 * time.Second},
		{SEND_TEXT_MESSAGE, -5, 10 * time.Second},
		{SEND_TEXT_MESSAGE, 1, 10 * time.Second},
		{SEND_TEXT_MESSAGE, 3, 30 * time.Second},
		{SEND_BASE64_IMAGE, 4, 2 * time.Minute},
		{SEND_FILE_PATH_MEDIA, 2, time.Minute},
		{SEND_WEB_MEDIA, 2, 90 * time.Second},
		{SEND_TEXT_MESSAGE, 30, MaxSendTimeout},
		{SEND_TEXT_MESSAGE, 31, MaxSendTimeout},
		{SEND_BASE64_IMAGE, 11, MaxSendTimeout},
		{SEND_WEB_MEDIA, 1 << 62, MaxSendTimeout},
		{STATUS_MESSAGE, 1, MaxSendTimeout},
		{MessageType("UNKNOWN"), 3, MaxSendTimeout},
	}
	for _, tc := range tests {
		if got := SendTimeout(tc.messageType, tc.recipients); got != tc.want {
			t.Errorf("SendTimeout(%s, %d) = %s, want %s", tc.messageType, tc.recipients, got, tc.want)
		}
	}
}

func TestMaxWebsocketMessageSize(t *testing.T) {
	if MaxWebsocketMessageSize != 64<<20 {
		t.Fatalf("MaxWebsocketMessageSize = %d, want %d", MaxWebsocketMessageSize, 64<<20)
	}
}
