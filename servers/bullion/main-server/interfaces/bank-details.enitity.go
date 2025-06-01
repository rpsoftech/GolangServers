package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type (
	BankDetailsBase struct {
		BullionId     string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		AccountName   string `bson:"accountName" json:"accountName" validate:"required,min=3"`
		BankName      string `bson:"bankName" json:"bankName" validate:"required,min=3"`
		AccountNumber string `bson:"accountNumber" json:"accountNumber" validate:"required,min=3"`
		IFSC          string `bson:"ifsc" json:"ifsc" validate:"required,min=3"`
		Sequence      int    `bson:"sequence" json:"sequence" validate:"required,min=1"`
		BranchName    string `bson:"branchName" json:"branchName" validate:"required,min=3"`
	}

	BankDetailsEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		*BankDetailsBase       `bson:"inline"`
	}
	UpdateBankDetailsRequestBody struct {
		Id string `json:"id"`
		*BankDetailsBase
	}
)

func CreateNewBankDetails(base *BankDetailsBase) *BankDetailsEntity {
	entity := &BankDetailsEntity{
		BaseEntity:      &interfaces.BaseEntity{},
		BankDetailsBase: base,
	}
	entity.CreateNewId()
	return entity
}
