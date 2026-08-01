package ecommerce_dto

type ProductSearchRequest struct {
	Page          int      `json:"page"`
	Limit         int      `json:"limit"`
	Search        string   `json:"search"`
	CategoryIDs   []string `json:"category_ids"`
	CollectionIDs []string `json:"collection_ids"`
	ProductTypes  []string `json:"product_types"`
	MetalColors   []string `json:"metal_colors"`
	Purities      []string `json:"purities"`
	StoneTypes    []string `json:"stone_types"`
	WeightMin     *float64 `json:"weight_min"`
	WeightMax     *float64 `json:"weight_max"`
	SortBy        string   `json:"sort_by"` // "latest", "popular"
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
