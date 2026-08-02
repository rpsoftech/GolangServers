package ecommerce_dto

type ProductSearchRequest struct {
	Page   int    `query:"page" json:"page"`
	Limit  int    `query:"limit" json:"limit"`
	Search string `query:"search" json:"search"`

	// Arrays in URL queries require special attention
	CategoryIDs   []string `query:"category_ids" json:"category_ids"`
	CollectionIDs []string `query:"collection_ids" json:"collection_ids"`
	ProductTypes  []string `query:"product_types" json:"product_types"`
	MetalColors   []string `query:"metal_colors" json:"metal_colors"`
	Purities      []string `query:"purities" json:"purities"`
	StoneTypes    []string `query:"stone_types" json:"stone_types"`

	WeightMin *float64 `query:"weight_min" json:"weight_min"`
	WeightMax *float64 `query:"weight_max" json:"weight_max"`
	SortBy    string   `query:"sort_by" json:"sort_by"`
}
type VariantDTO struct {
	VariantID   int     `json:"variant_id"`
	GrossWeight float64 `json:"gross_weight"`
	NetWeight   float64 `json:"net_weight"`
	VSellTunch  float64 `json:"vSellTunch"`
	VSellWstg   float64 `json:"vSellWstg"`
	IsActive    bool    `json:"isActive"`
}

type ProductDTO struct {
	ProductID      string        `json:"product_id"`
	SKU            string        `json:"sku"`
	ProductName    string        `json:"product_name"`
	CategoryID     string        `json:"category_id"`
	CollectionName string        `json:"collection_name"`
	ProductType    string        `json:"product_type"`
	IsActive       bool          `json:"isActive"`
	Variants       []*VariantDTO `json:"variants"`
}
