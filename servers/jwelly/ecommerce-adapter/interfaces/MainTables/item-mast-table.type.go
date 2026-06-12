package ecommerce_maintables

type ItemMasterTable struct {
	ItemId        int    `json:"itemId"`
	ItemName      string `json:"itemName"`
	IGroupId      int    `json:"iGroupId"`
	ItemPrintName string `json:"itemPrintName"`
	IUnitId       int    `json:"iUnitId"`
	ItemTagPrefix string `json:"itemTagPrefix"`
}
