package mysql_to_surreal_functions

import (
	"fmt"
	"time"

	mysql_to_surreal_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/interfaces"
	localSurrealdb "github.com/rpsoftech/golang-servers/utility/surrealdb"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const StampTableName = "stamp"

var (
	GetStampTableCommand = ""
)

func init() {
	GetStampTableCommand = fmt.Sprintf("SELECT * FROM %s", StampTableName)

}
func removeAndInsertStampTable(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", StampTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(StampTableName, mysql_to_surreal_interfaces.StampTableStruct{}, true), nil)
	fmt.Printf("Removed And Created %s\n", StampTableName)
}
func (c *ConfigWithConnection) ReadAndStoreStampTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetStampTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreStampTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.StampTableStruct = []*mysql_to_surreal_interfaces.StampTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.StampTableStruct{}
		err = rows.Scan(
			&row.STAMPID,
			&row.STAMP,
			&row.STUNCH,
			&row.GROUP,
			&row.REM1,
			&row.REM2,
			&row.DIAST,
			&row.PRATE,
			&row.SRATE,
			&row.DIVIDE,
			&row.MRATE,
			&row.Colour,
			&row.Clarity,
			&row.Shape,
			&row.SIZE,
			&row.TUNCH,
			&row.INO,
			&row.STKRATE,
			&row.PTUNCH,
			&row.STKTUNCH,
			&row.BHAVTUNCH,
			&row.TUNCHF,
			&row.TUNCHT,
			&row.Pcwt,
			&row.ASSORT,
			&row.PPROFIT,
			&row.VPROFIT,
			&row.Pname,
			&row.SIZEID,
			&row.SRNO,
			&row.DISPBHAV,
			&row.PBHAVTUNCH,
			&row.TSTAMP,
			&row.ADVPER,
			&row.OLDLESS,
			&row.OLDTUNCH,
			&row.HIDE,
			&row.Bhavroff,
			&row.Webstamp,
		)
		row.SurrealId = row.STAMPID
		if err != nil {
			fmt.Printf("Error in ReadAndStoreStampTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), StampTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(StampTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", StampTableName, time.Since(startTime))
	startTime = time.Now()
	var divided [][]*mysql_to_surreal_interfaces.StampTableStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	for k, v := range divided {
		_, err := surrealdb.Insert[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(StampTableName), v)
		if err != nil {
			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, StampTableName, "TLDR;")
		}
		fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(v), StampTableName, time.Since(startTime))
		startTime = time.Now()
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(StampTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", StampTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", StampTableName, time.Since(initalTime))
}
