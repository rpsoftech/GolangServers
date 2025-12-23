package messagedump_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// type DumpConfig struct{}

type MessageDumpServerConfig struct {
	*interfaces.BaseEntity `bson:",inline"`
	Name                   string              `bson:"name" json:"name" validate:"required,min=3,max=100"`
	MessageDumpConfigs     []MessageDumpConfig `bson:"messageDumpConfigs" json:"messageDumpConfigs" validate:"required,dive"`
}

type MessageDumpConfig struct {
	// *interfaces.BaseEntity `bson:",inline"`
	SourceChannel  string              `bson:"sourceChannel" json:"sourceChannel" validate:"required,oneof=whatsapp"`
	WhatsappConfig *WhatsappSideConfig `bson:"whatsappConfig" json:"whatsappConfig" validate:"required,dive"`
	TelegramConfig *TelegramSideConfig `bson:"telegramConfig" json:"telegramConfig" validate:"required,dive"`
}

type WhatsappSideConfig struct {
	WhatsappServerToken string   `bson:"whatsappServerToken" json:"whatsappServerToken" validate:"required"`
	WhatsappServerUrl   string   `bson:"whatsappServerUrl" json:"whatsappServerUrl" validate:"required,url"`
	SendNumbers         []string `bson:"sendNumbers" json:"sendNumbers" validate:"required"`
	SendWhatsappId      []string `bson:"sendWhatsappId" json:"sendWhatsappId" validate:"required"`
}
type TelegramSideConfig struct {
	TelegramBotToken string  `bson:"telegramBotToken" json:"telegramBotToken" validate:"required"`
	UserChatId       []int64 `bson:"userChatId" json:"userChatId" validate:"required"`
}
