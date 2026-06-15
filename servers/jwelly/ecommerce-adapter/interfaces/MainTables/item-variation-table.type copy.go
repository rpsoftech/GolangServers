package ecommerce_maintables

type TagVariationMainTable struct {
	TagVariationId int     `json:"tagVariationId"`
	VTagId         int     `json:"vTagId"`
	VStampId       int     `json:"vStampId"`
	VGrossWt       float64 `json:"vGrossWt"`
	VLessWeight    float64 `json:"vLessWeight"`
	VNetWt         float64 `json:"vNetWt"`
}