package mysql_to_surreal_functions

import (
	"fmt"
	"time"

	mysql_to_surreal_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/interfaces"
	localSurrealdb "github.com/rpsoftech/golang-servers/utility/surrealdb"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const UnitTableName = "units"

var (
	GetUnitTableCommand = ""
)

func removeAndInsertUnitTable(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", UnitTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(UnitTableName, mysql_to_surreal_interfaces.UnitTableStruct{}, true), nil)
	fmt.Printf("Removed And Created %s\n", UnitTableName)
}

func init() {
	GetUnitTableCommand = fmt.Sprintf("SELECT * FROM %s", UnitTableName)

}
func (c *ConfigWithConnection) ReadAndStoreUnitTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetUnitTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreUnitTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.UnitTableStruct = []*mysql_to_surreal_interfaces.UnitTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.UnitTableStruct{}
		err = rows.Scan(
			&row.UNITID,
			&row.UNIT,
			&row.DEL,
			&row.UQC,
			&row.DECIMAL,
		)
		row.SurrealId = row.UNITID
		if err != nil {
			fmt.Printf("Error in ReadAndStoreUnitTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), UnitTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(UnitTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", UnitTableName, time.Since(startTime))
	startTime = time.Now()
	var divided [][]*mysql_to_surreal_interfaces.UnitTableStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	for k, v := range divided {
		_, err := surrealdb.Insert[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(UnitTableName), v)
		if err != nil {
			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, UnitTableName, "TLDR;")
		}
		fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(v), UnitTableName, time.Since(startTime))
		startTime = time.Now()
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(UnitTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", UnitTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", UnitTableName, time.Since(initalTime))
}
