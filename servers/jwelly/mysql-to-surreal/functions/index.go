package mysql_to_surreal_functions

import (
	"fmt"
	"sync"

	mysql_to_surreal_env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/env"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/utility/surrealdb"
	godSurrealdb "github.com/surrealdb/surrealdb.go"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type ConfigWithConnection struct {
	ServerConfig  *mysql_to_surreal_env.ServerConfig
	DbConnections *ConnectionsToDb
}

type ConnectionsToDb struct {
	MysqlDbConncetion   *mysqldb.MysqlDBStruct
	SurrealDbConncetion *surrealdb.SurrealDBStruct
}

//	func insertDataToSurrealDb(c *surrealdb.SurrealDBStruct, table string, k int, v any, wg *sync.WaitGroup) {
//		startTime := time.Now()
//		d, err := godSurrealdb.Insert[any](c.Db, models.Table(table), v)
//		if err != nil {
//			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, table, "TLDR;")
//		}
//		fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(*d), table, time.Since(startTime))
//		if wg != nil {
//			wg.Done()
//		}
//	}
func upsertDataToSurrealDb(c *surrealdb.SurrealDBStruct, table string, k int, v any, wg *sync.WaitGroup) {
	// startTime := time.Now()
	_, err := godSurrealdb.Upsert[any](c.Db, models.Table(table), v)
	if err != nil {
		panic(fmt.Errorf("issue in round %d while inserting %s with a struct: %s", k+1, table, "TLDR;"))
	}
	// fmt.Printf("Record %d Upserted to %s in SurrealDB in Duration of %s\n", k+1, table, time.Since(startTime))
	if wg != nil {
		wg.Done()
	}
}

func (c *ConfigWithConnection) ClearSurrealDbAndInsert() {
	array := []func(*ConfigWithConnection){
		removeAndInsertAccMastTable,
		removeAndInsertCategory,
		removeAndInsertItemGroupTable,
		removeAndInsertItemTransTable,
		removeAndInsertSiteTable,
		removeAndInsertItemMastTable,
		removeAndInsertStampTable,
		removeAndInsertTrans1Table,
		removeAndInsertTgm1Table,
		removeAndInsertUnitTable,
	}
	viewArray := []func(*ConfigWithConnection){
		removeAndInsertTgMaster,
	}
	var wg sync.WaitGroup
	wg.Add(len(array))
	for _, a := range array {
		go func() {
			a(c)
			wg.Done()
		}()
	}
	wg.Wait()
	wg.Add(len(viewArray))
	for _, a := range viewArray {
		go func() {
			a(c)
			wg.Done()
		}()
	}
	wg.Wait()
}
