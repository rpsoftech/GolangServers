package dump_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type EndPoint struct {
	*interfaces.BaseEntity `bson:"inline"`
	Name                   string `bson:"name" json:"name" validate:"required,min=3,max=100"`
	Desc                   string `bson:"desc,omitempty" json:"desc,omitempty" validate:"required,min=3,max=100"`
	CustomChanel           string `bson:"customChanel" json:"customChanel" validate:"required,min=3,max=100"`
}

func (endPoint *EndPoint) CreateNewEntity(Name string, Desc string, CustomChanel string) *EndPoint {
	endPoint.BaseEntity = &interfaces.BaseEntity{}
	endPoint.Desc = Desc
	endPoint.Name = Name
	endPoint.CustomChanel = CustomChanel
	endPoint.CreateNewId()
	return endPoint
}

func (endPoint *EndPoint) GetChanel() string {
	if endPoint.CustomChanel == "" {
		endPoint.CustomChanel = endPoint.ID
	}
	return endPoint.CustomChanel
}
