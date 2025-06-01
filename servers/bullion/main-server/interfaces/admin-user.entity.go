package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type AdminUserEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	*UserRolesInterface    `bson:"inline"`
	UserName               string `bson:"userName" json:"userName" validate:"require"`
	Password               string `bson:"password" json:"password" validate:"require"`
	NickName               string `bson:"nickName" json:"nickName" validate:"require"`
	BullionId              string `bson:"bullionId" json:"bullionId" validate:"require,uuid"`
}

func (admin *AdminUserEntity) MatchPassword(password string) bool {
	return admin.Password == password
}

func (admin *AdminUserEntity) CreateNewEntity(UserName string, Password string, NickName string, BullionId string) *AdminUserEntity {
	admin.BaseEntity = &interfaces.BaseEntity{}
	admin.UserName = UserName
	admin.Password = Password
	admin.NickName = NickName
	admin.BullionId = BullionId
	admin.UserRolesInterface = &UserRolesInterface{
		Role: ROLE_ADMIN,
	}
	admin.CreateNewId()
	return admin
}
