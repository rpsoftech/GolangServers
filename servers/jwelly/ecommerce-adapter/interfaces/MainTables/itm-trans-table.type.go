package ecommerce_maintables

import "time"

type ItemTransactionTable struct {
	ItransId int       `json:"itransId"`
	TrnId    int       `json:"trnId"`
	VONO     int       `json:"vono"`
	TNO      int       `json:"tno"`
	SNTranId int       `json:"snTranId"`
	TDATE    time.Time `json:"tdate"`
	INO      int       `json:"ino"`
	Remarks  string    `json:"remarks"`
	GWT      float64   `json:"gwt"`
	WT       float64   `json:"wt"`
	LESSWT   float64   `json:"lesswt"`
	PC       int       `json:"pc"`
	Rate     float64   `json:"rate"`
	TUNCH    float64   `json:"tunch"`
	WSTG     float64   `json:"wstg"`
	MAMT     float64   `json:"mamt"`
	TYPE     string    `json:"type"`
	Stock    string    `json:"stock"`
	UnitId   string    `json:"unitId"`
	StampId  string    `json:"stampId"`
	SiteId   string    `json:"siteId"`
	TValue   float64   `json:"tValue"`
	DAmt     float64   `json:"damt"`
	SAmt     float64   `json:"samt"`
	LAmt     float64   `json:"lamt"`
	BAmt     float64   `json:"bamt"`
	Others   float64   `json:"others"`
	LBR      float64   `json:"lbr"`
	Size     string    `json:"size"`
	FINE1    float64   `json:"fine1"`
	TGNO     string    `json:"tgno"`
	VTGNO    int       `json:"vtgno"`
	TPRE     string    `json:"tpre"`
	TSNO     int       `json:"tsno"`
	ORGRate  float64   `json:"orgRate"`
	ORGGwt   float64   `json:"orgGwt"`
	ORGTotal float64   `json:"orgTotal"`
	Total    float64   `json:"total"`
	KACNO    int       `json:"kacno"`
	Karigar  string    `json:"karigar"`
	Metalwt  float64   `json:"metalwt"`
}
