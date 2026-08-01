package ecommerce_maintables

import mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"

type TagVariationMainTable struct {
	VStatusString  string          `json:"-"`
	TagVariationId int             `json:"tagVariationId"`
	VTagId         int             `json:"vTagId"`
	VStampId       int             `json:"vStampId"`
	VGrossWt       float64         `json:"vGrossWt"`
	VLessWeight    float64         `json:"vLessWeight"`
	VNetWt         float64         `json:"vNetWt"`
	VStatus        mysqldb.BitBool `json:"vStatus"`
	VTunch         float64         `json:"vTunch"`
	VWstg          float64         `json:"vWstg"`
	VSellTunch     float64         `json:"vSellTunch"`
	VSellWstg      float64         `json:"vSellWstg"`
	VKarigarCode   string          `json:"vKarigarCode"`
}
