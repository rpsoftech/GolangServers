package ecommerce_maintables

type AccountMasterTable struct {
	Acno     int    `json:"acno"`
	Prefix   string `json:"prefix"`
	Name     string `json:"Name"`
	PName    string `json:"pName"`
	AGroupId int    `json:"aGroupId"`
	City     string `json:"city"`
	Location string `json:"location"`
	Mobile   string `json:"mobile"`
	Phone    string `json:"phone"`
}
