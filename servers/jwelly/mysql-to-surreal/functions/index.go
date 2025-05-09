package mysql_to_surreal_functions

import (
	"fmt"
	"sync"
	"time"

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

func insertDataToSurrealDb(c *surrealdb.SurrealDBStruct, table string, k int, v any, wg *sync.WaitGroup) {
	startTime := time.Now()
	d, err := godSurrealdb.Insert[any](c.Db, models.Table(table), v)
	if err != nil {
		fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, table, "TLDR;")
	}
	fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(*d), table, time.Since(startTime))
	if wg != nil {
		wg.Done()
	}
}

func (c *ConfigWithConnection) ClearSurrealDbAndInsert() {
	array := []func(*ConfigWithConnection){
		removeAndInsertCategory,
		removeAndInsertItemGroupTable,
		removeAndInsertItemTransTable,
		removeAndInsertSiteTable,
		removeAndInsertItemMastTable,
		removeAndInsertStampTable,
		// removeAndInsertTgMaster,
		removeAndInsertTgm1Table,
		removeAndInsertUnitTable,
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
}
