package ecommerce_maintables

type ItemGroupMainTable struct {
	ItemGroupId   int    `json:"itemGroupId"`
	ItemGroup     string `json:"itemGroup"`
	ItemPrintName string `json:"itemPrintName"`
	ItemUnitId    int    `json:"itemUnitId"`
}
