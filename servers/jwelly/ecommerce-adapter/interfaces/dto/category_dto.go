package ecommerce_dto

type CategoryDTO struct {
	CategoryID       string  `json:"category_id"`
	CategoryName     string  `json:"category_name"`
	ParentCategoryID *string `json:"parent_category_id"` // Pointer to allow null
	ImageURL         string  `json:"image_url"`
	SortOrder        int     `json:"sort_order"`
	IsActive         bool    `json:"isActive"`
}
