package mysql_to_surreal_functions

import (
	"fmt"
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
func RemoveAndInsertCategory(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", CategoryTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(CategoryTableName, mysql_to_surreal_interfaces.CategoryTableStruct{}, true), nil)

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
	startTime = time.Now()
	var divided [][]*mysql_to_surreal_interfaces.CategoryTableStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	for k, v := range divided {
		_, err := surrealdb.Insert[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(CategoryTableName), v)
		if err != nil {
			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, CategoryTableName, "TLDR;")
		}
		fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(v), CategoryTableName, time.Since(startTime))
		startTime = time.Now()
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(CategoryTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", CategoryTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", CategoryTableName, time.Since(initalTime))
}
