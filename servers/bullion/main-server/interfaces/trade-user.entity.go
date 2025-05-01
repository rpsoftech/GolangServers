package bullion_main_server_interfaces

import (
	"net/http"

	"github.com/rpsoftech/golang-servers/interfaces"
)

type (
	TradeUserBase struct {
		BullionId   string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Name        string `bson:"name" json:"name" validate:"required,min=2,max=100"`
		Number      string `bson:"number" json:"number" validate:"required,min=12,max=12"`
		Email       string `bson:"email" json:"email" validate:"required"`
		CompanyName string `bson:"companyName" json:"companyName" validate:"required,min=2,max=50"`
		GstNumber   string `bson:"gstNumber" json:"gstNumber" validate:"required,gstNumber"`
		RawPassword string `bson:"rawPassword" json:"password" mapstructure:"password" validate:"required,min=4"`
	}

	TradeUserAdvanced struct {
		GroupId  string `bson:"groupId" json:"groupId" validate:"required,uuid"`
		UserName string `bson:"userName" json:"userName" validate:"required"`
		IsActive bool   `bson:"isActive" json:"isActive" validate:"boolean"`
		UNumber  string `bson:"uNumber" json:"uNumber" validate:"required"`
	}
	UserMarginsDataStruct struct {
		Gold   int `bson:"gold" json:"gold" validate:"min=0"`
		Silver int `bson:"silver" json:"silver" validate:"min=0"`
	}

	TradeUserMargins struct {
		AllotedMargins *UserMarginsDataStruct `bson:"allotedMargins" json:"allotedMargins" validate:"required"`
		UsedMargins    *UserMarginsDataStruct `bson:"usedMargins" json:"availableMargins" validate:"required"`
	}

	TradeUserEntity struct {
		*BaseEntity        `bson:"inline"`
		*TradeUserBase     `bson:"inline"`
		*passwordEntity    `bson:"inline"`
		*TradeUserAdvanced `bson:"inline"`
		*TradeUserMargins  `bson:"margins" json:"margins"`
	}

	ApiTradeUserRegisterResponse struct {
		UserToken   string `json:"userToken"`
		OtpReqToken string `json:"otpReqToken"`
	}
)

func (user *TradeUserEntity) CreateNew() *TradeUserEntity {
	user.createNewId()
	return user
}

func (user *TradeUserEntity) UpdateUser() *TradeUserEntity {
	user.passwordEntity = CreatePasswordEntity(user.RawPassword)
	user.Updated()
	return user
}
func (user *TradeUserEntity) DeletePassword() *TradeUserEntity {
	user.RawPassword = ""
	return user
}

func (user *TradeUserEntity) UpdateMarginAfterOrder(weight int, symbol SourceSymbolEnum) (*TradeUserEntity, error) {
	availableMargin := 0
	if symbol == SOURCE_SYMBOL_GOLD {
		availableMargin = user.AllotedMargins.Gold - user.UsedMargins.Gold
	} else if symbol == SOURCE_SYMBOL_SILVER {
		availableMargin = user.AllotedMargins.Silver - user.UsedMargins.Silver
	}

	if availableMargin < 0 || availableMargin-weight < 0 {
		return nil, &RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INSUFFICIENT_MARGIN,
			Message:    "Insufficient Margin",
			Name:       "INSUFFICIENT_MARGIN",
			Extra:      map[string]interface{}{"margins": user.TradeUserMargins, "weight": weight},
		}
	}

	if symbol == SOURCE_SYMBOL_GOLD {
		user.UsedMargins.Gold = user.UsedMargins.Gold + weight
	} else if symbol == SOURCE_SYMBOL_SILVER {
		user.UsedMargins.Silver = user.UsedMargins.Silver + weight
	}
	return user, nil
}
