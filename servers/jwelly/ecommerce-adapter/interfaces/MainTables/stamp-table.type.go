package ecommerce_maintables

type StampMainTable struct {
	StampId    int     `json:"stampId"`
	Stamp      string  `json:"STAMP"`
	Tunch      float64 `json:"tunch"`
	StockTunch float64 `json:"stockTunch"`
	SellTunch  float64 `json:"sellTunch"`
}
