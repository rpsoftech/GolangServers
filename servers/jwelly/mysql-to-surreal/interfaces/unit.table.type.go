package mysql_to_surreal_interfaces

type UnitTableStruct struct {
	SurrealId int    `json:"id" Index:"U" fieldType:"int"`
	UNITID    int    `json:"UNITID" Index:"U" fieldType:"int"`
	UNIT      string `json:"UNIT" fieldType:"string | NULL" Index:"I"`
	DEL       string `json:"DEL" fieldType:"string | NULL"`
	UQC       string `json:"UQC" fieldType:"string | NULL"`
	DECIMAL   int    `json:"DECIMAL" fieldType:"int | NULL" defaultValue:"0"`
}
