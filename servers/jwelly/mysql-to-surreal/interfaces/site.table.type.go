package mysql_to_surreal_interfaces

type SiteTableStruct struct {
	SurrealId int    `json:"id" Index:"S" fieldType:"int"`
	SITEID    int    `json:"SITEID" Index:"U" fieldType:"int"`
	SITE      string `json:"SITE" fieldType:"string | NULL"`
	PATH      string `json:"PATH" fieldType:"string | NULL"`
	PSITE     string `json:"PSITE" fieldType:"string | NULL"`
	PREFIX    string `json:"PREFIX" fieldType:"string | NULL"`
	STATEID   int    `json:"STATEID" fieldType:"int | NULL" defaultValue:"0"`
	SPSTATUS  int    `json:"SPSTATUS" fieldType:"int | NULL" defaultValue:"0"`
	SFEEDING  int    `json:"SFEEDING" fieldType:"int | NULL" defaultValue:"0"`
	SITEACNO  int    `json:"SITEACNO" fieldType:"int | NULL" defaultValue:"0"`
	Stcode    string `json:"stcode" fieldType:"string | NULL"`
	Branchid  int    `json:"branchid" fieldType:"int | NULL" defaultValue:"0"`
	Emailid   string `json:"emailid" fieldType:"string | NULL"`
	Epswd     string `json:"epswd" fieldType:"string | NULL"`
}
