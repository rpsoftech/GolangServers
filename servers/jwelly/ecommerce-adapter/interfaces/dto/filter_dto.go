package ecommerce_dto

type FilterOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FilterOptionsDTO struct {
	Categories []FilterOption `json:"categories"`
	Purities   []FilterOption `json:"purities"`
}
