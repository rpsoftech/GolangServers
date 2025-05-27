package mysql_to_surreal_functions

import (
	"fmt"
	"sync"
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
	var divided [][]*mysql_to_surreal_interfaces.UnitTableStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	var waitGroup sync.WaitGroup

	for k, v := range divided {
		base := (k * chunkSize)
		waitGroup.Add(len(v))
		for k1, v1 := range v {
			go upsertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, UnitTableName, base+k1, v1, &waitGroup)
		}
		waitGroup.Wait()
		// go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(UnitTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", UnitTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", UnitTableName, time.Since(initalTime))
}
