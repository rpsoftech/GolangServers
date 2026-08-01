package ecommerce_maintables

type ItemTagVariationDetails struct {
	ItemTagDetailsId int     `json:"itemTagDetailsId"`
	DItemTagId       int     `json:"dItemTagId"`
	DItemId          int     `json:"dItemId"`
	DSTAMPID         int     `json:"dSTAMPID"`
	DGrossWeight     float64 `json:"dWeight"`
	DExWeight        float64 `json:"dExWeight"`
	DFinalWeight     float64 `json:"dFinalWeight"`
	DRemarks         string  `json:"dRemarks"`
	DPcs             int     `json:"dPcs"`
	DRate            float64 `json:"dRate"`
	DSaleValue       float64 `json:"dSaleValue"`
	DUnitId          int     `json:"dUnitId"`
}
