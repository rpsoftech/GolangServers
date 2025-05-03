package bullion_main_server_interfaces

type (
	MsgTemplateBase struct {
		WhatsappTemplate string `bson:"whatsappTemplate" json:"whatsappTemplate" validate:"required"`
		MSG91Id          string `bson:"msg91Id" json:"msg91Id" validate:"required"`
	}
)
