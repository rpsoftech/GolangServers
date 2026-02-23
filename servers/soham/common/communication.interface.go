package soham_common_req_keys

type MessageType string

const (
	STATUS_MESSAGE       MessageType = "STATUS_MESSAGE"
	REPSONSE_MESSAGE     MessageType = "RESPONSE_MESSAGE"
	SEND_TEXT_MESSAGE    MessageType = "SEND_TEXT_MESSAGE"
	SEND_BASE64_IMAGE    MessageType = "SEND_BASE64_IMAGE"
	SEND_WEB_MEDIA       MessageType = "SEND_WEB_MEDIA"
	SEND_FILE_PATH_MEDIA MessageType = "SEND_FILE_PATH_MEDIA"
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

type SendBase64Image struct {
	ApiSendRequestBase
	Base64 string `json:"base64" validate:"required"`
	// Ext       string `json:"ext_name" validate:"required,regexp=^.[\\w]*$"`
	Media     string `json:"media_name" validate:"required"`
	ImageDesc string `json:"image_desc"`
}

type SendWebMediaType struct {
	ApiSendRequestBase
	WebMediaLink string `json:"web_media_link" validate:"required,url"`
	MediaName    string `json:"media_name" validate:"required"`
	ImageDesc    string `json:"image_desc"`
}

type SendFilePathMediaType struct {
	ApiSendRequestBase
	LocalMediaPath string `json:"local_media_path" validate:"required"`
	ImageDesc      string `json:"image_desc"`
	//  {
	//   key: 'local_media_path',
	//   required: true,
	//   type: 'string',
	// },
	// {
	//   key: 'image_desc',
	//   required: false,
	//   type: 'string',
	// },
}
