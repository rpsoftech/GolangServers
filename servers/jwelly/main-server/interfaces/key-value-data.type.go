package jwelly_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type KeyValueDataStruct struct {
	*interfaces.BaseEntity `bson:"inline"`
	Key                    string `bson:"key" json:"key" validate:"required"`
	Value                  any    `bson:"value" json:"value" validate:"required"`
}

func (keyValueData *KeyValueDataStruct) CreateNew() *KeyValueDataStruct {
	keyValueData.CreateNewId()
	return keyValueData
}
