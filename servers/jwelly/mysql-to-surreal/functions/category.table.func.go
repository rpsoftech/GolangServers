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

const CategoryTableName = "category"

var (
	GetCategoryTableCommand = ""
)

func init() {
	GetCategoryTableCommand = fmt.Sprintf("SELECT * FROM %s", CategoryTableName)

}
func removeAndInsertCategory(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", CategoryTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(CategoryTableName, mysql_to_surreal_interfaces.CategoryTableStruct{}, true), nil)
	fmt.Printf("Removed And Created %s\n", CategoryTableName)
}
func (c *ConfigWithConnection) ReadAndStoreCategory() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetCategoryTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreCategory For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.CategoryTableStruct = []*mysql_to_surreal_interfaces.CategoryTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.CategoryTableStruct{}
		err = rows.Scan(
			&row.CATID,
			&row.Category,
			&row.REMARKS,
			&row.Flag,
			&row.Mobile,
		)
		row.SurrealId = row.CATID
		if err != nil {
			fmt.Printf("Error in ReadAndStoreCategory While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), CategoryTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(CategoryTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", CategoryTableName, time.Since(startTime))
	var divided [][]*mysql_to_surreal_interfaces.CategoryTableStruct
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
			go upsertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, CategoryTableName, base+k1, v1, &waitGroup)
		}
		waitGroup.Wait()
		// go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(CategoryTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", CategoryTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", CategoryTableName, time.Since(initalTime))
}
