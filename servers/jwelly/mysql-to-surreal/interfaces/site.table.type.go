package mysql_to_surreal_interfaces

type SiteTableStruct struct {
	SurrealId int    `json:"id" Index:"S" fieldType:"int"`
	SITEID    int    `json:"SITEID" Index:"U" fieldType:"int"`
	SITE      string `json:"SITE" fieldType:"string | NULL"`
}
