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

const SiteTableName = "site"

var (
	GetSiteTableCommand = ""
)

func removeAndInsertSiteTable(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", SiteTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(SiteTableName, mysql_to_surreal_interfaces.SiteTableStruct{}, true), nil)
	fmt.Printf("Removed And Created %s\n", SiteTableName)
}

func init() {
	GetSiteTableCommand = fmt.Sprintf("SELECT * FROM %s", SiteTableName)

}
func (c *ConfigWithConnection) ReadAndStoreSiteTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetSiteTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreSiteTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.SiteTableStruct = []*mysql_to_surreal_interfaces.SiteTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.SiteTableStruct{}
		err = rows.Scan(
			&row.SITEID,
			&row.SITE,
			&row.PATH,
			&row.PSITE,
			&row.PREFIX,
			&row.STATEID,
			&row.SPSTATUS,
			&row.SFEEDING,
			&row.SITEACNO,
			&row.Stcode,
			&row.Branchid,
			&row.Emailid,
			&row.Epswd,
		)
		row.SurrealId = row.SITEID
		if err != nil {
			fmt.Printf("Error in ReadAndStoreSiteTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), SiteTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(SiteTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", SiteTableName, time.Since(startTime))
	var divided [][]*mysql_to_surreal_interfaces.SiteTableStruct
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
			go upsertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, SiteTableName, base+k1, v1, &waitGroup)
		}
		waitGroup.Wait()
		// go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(SiteTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", SiteTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", SiteTableName, time.Since(initalTime))
}
