package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type (
	TradeUserGroupBase struct {
		BullionId string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Name      string `bson:"name" json:"name" validate:"required"`
		IsActive  bool   `bson:"isActive" json:"isActive" validate:"boolean"`
		CanTrade  bool   `bson:"canTrade" json:"canTrade" validate:"boolean"`
		CanLogin  bool   `bson:"canLogin" json:"canLogin" validate:"boolean"`
	}
	TradeUserGroupEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		*TradeUserGroupBase    `bson:"inline"`
		Gold                   *GroupPremiumBase `bson:"gold" json:"gold" validate:"required"`
		Silver                 *GroupPremiumBase `bson:"silver" json:"silver" validate:"required"`
	}

	TradeUserGroupMapEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		*TradeUserGroupMapBase `bson:"inline"`
	}

	TradeUserGroupMapBase struct {
		BullionId         string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		GroupId           string `bson:"groupId" json:"groupId" validate:"required,uuid"`
		ProductId         string `bson:"productId" json:"productId" validate:"required,uuid"`
		IsActive          bool   `bson:"isActive" json:"isActive" validate:"boolean"`
		CanTrade          bool   `bson:"canTrade" json:"canTrade" validate:"boolean"`
		*GroupPremiumBase `bson:"groupPremiumBase" json:"groupPremiumBase" validate:"required"`
		*GroupVolumeBase  `bson:"groupVolumeBase" json:"groupVolumeBase" validate:"required"`
	}

	GroupPremiumBase struct {
		Buy  float64 `bson:"buy" json:"buy"`
		Sell float64 `bson:"sell" json:"sell"`
	}

	GroupVolumeBase struct {
		OneClick int `bson:"oneClick" json:"oneClick"`
		Step     int `bson:"step" json:"step"`
		Total    int `bson:"total" json:"total"`
	}
)

func (r *TradeUserGroupEntity) CreateNew() *TradeUserGroupEntity {
	r.BaseEntity = &interfaces.BaseEntity{}
	r.CreateNewId()
	return r
}

func (r *TradeUserGroupMapEntity) CreateNew() *TradeUserGroupMapEntity {
	r.BaseEntity = &interfaces.BaseEntity{}
	r.CreateNewId()
	return r
}

func (r *TradeUserGroupMapEntity) ValidateVolume(weight int) bool {
	if weight < r.GroupVolumeBase.OneClick {
		return false
	} else if weight > r.GroupVolumeBase.Total {
		return false
	} else if weight == r.GroupVolumeBase.OneClick || weight == r.GroupVolumeBase.Total {
		return true
	}
	return (weight-r.GroupVolumeBase.OneClick)%r.GroupVolumeBase.Step == 0
}

func (r *TradeUserGroupMapBase) UpdateDetails(base *TradeUserGroupMapBase) *TradeUserGroupMapBase {
	r.IsActive = base.IsActive
	r.CanTrade = base.CanTrade
	r.GroupPremiumBase = base.GroupPremiumBase
	r.GroupVolumeBase = base.GroupVolumeBase
	return r
}
