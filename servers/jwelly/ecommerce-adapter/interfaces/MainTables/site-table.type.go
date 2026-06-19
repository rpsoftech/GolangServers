package ecommerce_maintables

type SiteTable struct {
	SiteId      int    `json:"siteId"`
	SiteName    string `json:"siteName"`
	SiteAddress string `json:"siteAddress"`
	SPrefix     string `json:"sPrefix"`
}
