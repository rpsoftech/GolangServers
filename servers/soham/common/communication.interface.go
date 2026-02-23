package soham_common_req_keys

type MessageType string

const (
	STATUS_MESSAGE MessageType = "STATUS_MESSAGE"
)

type WhatsappClientMessage struct {
	Type    MessageType `json:"type"`
	Message any         `json:"message"`
}
