package ecommerce_maintables

type ItemGroupMainTable struct {
	ItemGroupId   string `json:"itemGroupId"`
	ItemGroup     string `json:"itemGroup"`
	ItemPrintName string `json:"itemPrintName"`
	ItemUnitId    int    `json:"itemUnitId"`
}
