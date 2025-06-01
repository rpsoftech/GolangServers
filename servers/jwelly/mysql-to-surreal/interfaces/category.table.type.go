package mysql_to_surreal_interfaces

import mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"

type CategoryTableStruct struct {
	SurrealId int             `json:"id" Index:"U"`
	CATID     int             `json:"CATID" Index:"U" fieldType:"int"`
	Category  string          `json:"category" fieldType:"string | NULL" Index:"I"`
	REMARKS   string          `json:"REMARKS" fieldType:"string | NULL" Index:"I"`
	Flag      mysqldb.BitBool `json:"flag" fieldType:"bool | NULL" defaultValue:"false"`
	Mobile    string          `json:"mobile" fieldType:"string | NULL" Index:"I"`
}
