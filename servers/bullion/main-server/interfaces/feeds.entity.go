package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type (
	FeedsBase struct {
		BullionId string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Title     string `bson:"title" json:"title" validate:"required,min=3"`
		Body      string `bson:"body" json:"body" validate:"required,min=3"`
		IsHtml    bool   `bson:"isHtml" json:"isHtml" validate:"boolean"`
	}
	FeedsEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		*FeedsBase             `bson:"inline"`
	}
	FeedUpdateRequestBody struct {
		FeedId     string `bson:"feedId" json:"feedId" validate:"required,uuid"`
		*FeedsBase `bson:"inline"`
	}
	FeedDeleteRequestBody struct {
		FeedId    string `bson:"feedId" json:"feedId" validate:"required,uuid"`
		BullionId string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
	}
)

func (b *FeedsEntity) CreateNewId() *FeedsEntity {
	if b.BaseEntity == nil {
		b.BaseEntity = &interfaces.BaseEntity{}
	}
	b.CreateNewId()
	return b
}
