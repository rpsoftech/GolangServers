package whatsappdump_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// type DumpConfig struct{}

type MessageDumpConfig struct {
	*interfaces.BaseEntity `bson:",inline"`
	Name                   string              `bson:"name" json:"name" validate:"required,min=3,max=100"`
	SourceChannel          string              `bson:"sourceChannel" json:"sourceChannel" validate:"required,oneof=whatsapp"`
	WhatsappConfig         *WhatsappSideConfig `bson:"whatsappConfig" json:"whatsappConfig" validate:"required,dive"`
}

type WhatsappSideConfig struct {
	WhatsappServerUrl string   `bson:"whatsappServerUrl" json:"whatsappServerUrl" validate:"required,url"`
	SendNumbers       []string `bson:"sendNumbers" json:"sendNumbers" validate:"required"`
	SendWhatsappId    []string `bson:"sendWhatsappId" json:"sendWhatsappId" validate:"required"`
}
