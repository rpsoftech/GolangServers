package bullion_main_server_interfaces

type bullionGeneralUserConfig struct {
	AutoApprove bool `bson:"autoApprove" json:"autoApprove" validate:"boolean"`
	AutoLogin   bool `bson:"autoLogin" json:"autoLogin" validate:"boolean"`
}
type bullionConfigs struct {
	OTPLength                  int    `bson:"otpLength" json:"otpLength" validate:"required"`
	HaveCustomWhatsappAgent    bool   `bson:"haveCustomWhatsappAgent" json:"haveCustomWhatsappAgent" validate:"boolean"`
	DefaultGroupIdForTradeUser string `bson:"defaultGroupIdForTradeUser" json:"defaultGroupIdForTradeUser" validate:"required,uuid"`
}

type limitConfigs struct {
	FreezeTop    int `bson:"freezeTop" json:"freezeTop" validate:"required"`
	FreezeBottom int `bson:"freezeBottom" json:"freezeBottom" validate:"required"`
}

type BullionSiteBasicInfo struct {
	Name      string `bson:"name" json:"name" validate:"required"`
	ShortName string `bson:"shortName" json:"shortName" validate:"required"`
	Domain    string `bson:"domain" json:"domain" validate:"required"`
}
type BullionPublicInfo struct {
	*BaseEntity           `bson:"inline"`
	*BullionSiteBasicInfo `bson:"inline"`
}
type BullionSiteInfoEntity struct {
	*BullionPublicInfo `bson:"inline"`
	LimitConfigs       *limitConfigs             `bson:"limitConfigs" json:"-" validate:"required"`
	BullionConfigs     *bullionConfigs           `bson:"bullionConfigs" json:"-" validate:"required"`
	GeneralUserInfo    *bullionGeneralUserConfig `bson:"generalUserInfo" json:"generalUserInfo" validate:"required"`
}

func (b *BullionSiteInfoEntity) AddGeneralUserInfo(AutoApprove bool, AutoLogin bool) *BullionSiteInfoEntity {
	b.GeneralUserInfo = &bullionGeneralUserConfig{
		AutoApprove: AutoApprove,
		AutoLogin:   AutoLogin,
	}
	return b
}

func CreateNewBullionSiteInfo(name string, shortName string, domain string) *BullionSiteInfoEntity {
	b := BullionSiteInfoEntity{
		BullionPublicInfo: &BullionPublicInfo{
			BaseEntity: &BaseEntity{},
			BullionSiteBasicInfo: &BullionSiteBasicInfo{
				Name:      name,
				ShortName: shortName,
				Domain:    domain,
			},
		},
		BullionConfigs: &bullionConfigs{
			OTPLength:                  5,
			HaveCustomWhatsappAgent:    true,
			DefaultGroupIdForTradeUser: "",
		},
		LimitConfigs: &limitConfigs{
			FreezeTop:    20,
			FreezeBottom: 100,
		},
		GeneralUserInfo: &bullionGeneralUserConfig{
			AutoApprove: false,
			AutoLogin:   true,
		},
	}
	b.createNewId()
	return &b
}
