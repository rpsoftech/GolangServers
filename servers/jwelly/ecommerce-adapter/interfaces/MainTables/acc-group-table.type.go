package ecommerce_maintables

type AccountGroupTable struct {
	GroupId   int    `json:"groupId"`
	GroupName string `json:"groupName"`
	UnderId   int    `json:"underId"`
	GType     string `json:"gType"`
}
