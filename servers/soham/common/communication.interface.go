package soham_common_req_keys

type MessageType string

const (
	STATUS_MESSAGE    MessageType = "STATUS_MESSAGE"
	REPSONSE_MESSAGE  MessageType = "RESPONSE_MESSAGE"
	SEND_TEXT_MESSAGE MessageType = "SEND_TEXT_MESSAGE"
)

type WhatsappClientMessage struct {
	ReqId   string      `json:"reqid"`
	Type    MessageType `json:"type"`
	Message any         `json:"message"`
}

type ApiSendRequestBase struct {
	From int   `json:"from" validate:"required"`
	To   []int `json:"to" validate:"required"`
	Wait int   `json:"wait"`
}
type SendTextMessage struct {
	ApiSendRequestBase
	Message string `json:"message" validate:"required"`
}
