package ecommerce_maintables

import "time"

type ItemTagMainTable struct {
	ItemTagId      int       `json:"itemTagId"`
	ItemTag        string    `json:"itemTag"`
	ItemVTagId     int       `json:"itemVTagId"`
	TItemId        int       `json:"tItemId"`
	TagCreatedDate time.Time `json:"tagCreatedDate"`
}
