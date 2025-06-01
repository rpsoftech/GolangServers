package interfaces

import (
	"time"

	"github.com/google/uuid"
)

type BaseEntity struct {
	ID                 string    `bson:"id" json:"id" validate:"required,uuid"`
	CreatedAtExported  time.Time `bson:"-" json:"createdAt,omitempty"`
	ModifiedAtExported time.Time `bson:"-" json:"modifiedAt,omitempty"`
	CreatedAt          time.Time `bson:"createdAt" json:"-" validate:"required"`
	ModifiedAt         time.Time `bson:"modifiedAt" json:"-" validate:"required"`
}

type BaseEntityInterface interface {
	AddTimeStamps() *BaseEntity
	RestoreTimeStamp() *BaseEntity
	CreateNewId() *BaseEntity
}

// type baseEntityAlias BaseEntity
// type baseEntityWithTimes struct {
// 	CreatedAt  time.Time `json:"createdAt"`
// 	ModifiedAt time.Time `json:"modifiedAt"`
// 	*baseEntityAlias
// }

// func (base *BaseEntity) MarshalJSON() ([]byte, error) {
// 	if base.ExportCreatedAt {
// 		return json.Marshal(baseEntityWithTimes{
// 			CreatedAt:       base.CreatedAt,
// 			ModifiedAt:      base.ModifiedAt,
// 			baseEntityAlias: (*baseEntityAlias)(base),
// 		})
// 	}
// 	return json.Marshal(base)
// }

func (b *BaseEntity) AddTimeStamps() *BaseEntity {
	b.CreatedAtExported = b.CreatedAt
	b.ModifiedAtExported = b.ModifiedAt
	return b
}
func (b *BaseEntity) RestoreTimeStamp() *BaseEntity {
	b.CreatedAt = b.CreatedAtExported
	b.ModifiedAt = b.ModifiedAtExported
	return b
}
func (b *BaseEntity) CreateNewId() *BaseEntity {
	b.ID = uuid.New().String()
	b.CreatedAt = time.Now()
	b.ModifiedAt = b.CreatedAt
	return b
}

func (b *BaseEntity) Updated() *BaseEntity {
	b.ModifiedAt = time.Now()
	return b
}
