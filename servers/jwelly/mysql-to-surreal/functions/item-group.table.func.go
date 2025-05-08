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

const ItemGroupTableName = "igroup"

var (
	GetItemGroupTableCommand = ""
)

func init() {
	GetItemGroupTableCommand = fmt.Sprintf("SELECT * FROM %s", ItemGroupTableName)

}

func removeAndInsertItemGroupTable(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", ItemGroupTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(ItemGroupTableName, mysql_to_surreal_interfaces.ItemGroupTableStruct{}, true), nil)
}

func (c *ConfigWithConnection) ReadAndStoreItemGroupTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetItemGroupTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreItemGroupTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.ItemGroupTableStruct = []*mysql_to_surreal_interfaces.ItemGroupTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.ItemGroupTableStruct{}
		err = rows.Scan(
			&row.Igroupid,
			&row.IGROUP,
			&row.PNAME,
			&row.UNDERID,
			&row.ITNAME,
			&row.LTAX,
			&row.CTAX,
			&row.GP,
			&row.OBAL,
			&row.CBAL,
			&row.SACNO,
			&row.PACNO,
			&row.STSACNO,
			&row.STPACNO,
			&row.IACNO,
			&row.RACNO,
			&row.CCODE,
			&row.STVAL,
			&row.DIV,
			&row.INTO,
			&row.UNITID,
			&row.DBHAV,
			&row.GWTNWT,
			&row.STAMPID,
			&row.ITYPE,
			&row.DEL,
			&row.TTYPE,
			&row.ALTTAG,
			&row.SRTOTAL,
			&row.SRDAMT,
			&row.SRSAMT,
			&row.SRLAMT,
			&row.SRMAMT,
			&row.EXTOTAL,
			&row.EXLAMT,
			&row.EXMAMT,
			&row.EXDAMT,
			&row.EXSAMT,
			&row.MRPTYPE,
			&row.DIVIDERATE,
			&row.INTORATE,
			&row.PURLESS,
			&row.DIS,
			&row.SMDET,
			&row.RTAG,
			&row.MINQTY,
			&row.POINT,
			&row.RESDRATE,
			&row.RESSRATE,
			&row.RESMRATE,
			&row.RESLBR,
			&row.Resigrp,
			&row.CPOINT,
			&row.IRLEVEL,
			&row.VRATE,
			&row.IGTUNCH,
			&row.IGTGNUM,
			&row.MCXINTO,
			&row.MCXCOMEX,
			&row.LBRDIS,
			&row.ITRATE,
			&row.UPMRP,
			&row.IGALADD,
			&row.RESMRP,
			&row.RESLESS,
			&row.CONSUME,
			&row.UPSALE,
			&row.RENTACNO,
			&row.PSGST,
			&row.PCGST,
			&row.PIGST,
			&row.HSNCODE,
			&row.JHSNCODE,
			&row.GSTNAME,
			&row.Defino,
		)
		row.SurrealId = row.Igroupid
		if err != nil {
			fmt.Printf("Error in ReadAndStoreItemGroupTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), ItemGroupTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemGroupTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", ItemGroupTableName, time.Since(startTime))

	if len(results) > 50 {
		var divided [][]*mysql_to_surreal_interfaces.ItemGroupTableStruct
		chunkSize := 50
		for i := 0; i < len(results); i += chunkSize {
			end := min(i+chunkSize, len(results))
			divided = append(divided, results[i:end])
		}
		var waitGroup sync.WaitGroup
		waitGroup.Add(len(divided))
		for k, v := range divided {
			go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, ItemGroupTableName, k, v, &waitGroup)
		}
		waitGroup.Wait()
	} else {
		insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, ItemGroupTableName, 1, results, nil)
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemGroupTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", ItemGroupTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", ItemGroupTableName, time.Since(initalTime))
}
