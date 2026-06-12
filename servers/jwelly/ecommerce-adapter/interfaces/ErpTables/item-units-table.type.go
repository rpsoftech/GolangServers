package ecommerce_erptables

type ItemUnitErpTable struct {
	ItemUnitId  int    `json:"UNITID"`
	ItemUnit    string `json:"UNIT"`
	ItemDel     string `json:"DEL"`
	ItemUQC     string `json:"UQC"`
	ItemDecimal int    `json:"DECIMAL"`
}
