package mysql_to_surreal_functions

import (
	"fmt"
	"sync"
	"time"

	mysql_to_surreal_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/interfaces"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const TgMasterTableName = "tg_master"

var (
	GetTgMasterSqlCommand = ""
)

func init() {
	GetTgMasterSqlCommand = fmt.Sprintf("SELECT * FROM %s", TgMasterTableName)

}

// func removeAndInsertTgMaster(c *ConfigWithConnection) {
// 	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", TgMasterTableName), nil)
// 	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(TgMasterTableName, mysql_to_surreal_interfaces.TgMasterStruct{}, true), nil)
// 	fmt.Printf("Removed And Created %s\n", TgMasterTableName)
// }

func (c *ConfigWithConnection) ReadAndStoreTgMaster() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query("SELECT * FROM tg_master")
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreTgMaster For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.TgMasterStruct = []*mysql_to_surreal_interfaces.TgMasterStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.TgMasterStruct{}
		err = rows.Scan(
			&row.Idesc,
			&row.Tgno,
			&row.Vtgno,
			&row.Tpre,
			&row.Remarks,
			&row.Tdate,
			&row.Gwt,
			&row.Lesswt,
			&row.Wt,
			&row.Sdiawt,
			&row.Sstnwt,
			&row.Goldwt,
			&row.Silwt,
			&row.Platwt,
			&row.Othwt,
			&row.Slbr,
			&row.Slbr2,
			&row.Slbr3,
			&row.Status,
			&row.Stunch,
			&row.Swstg,
			&row.Sbeeds,
			&row.Sothers,
			&row.Othrem,
			&row.Design,
			&row.Karigar,
			&row.Mrate,
			&row.Gwt1,
			&row.Gwt2,
			&row.Stamp,
			&row.Size,
			&row.Quality,
			&row.Color,
			&row.Clarity,
			&row.Site,
			&row.Linktgno,
			&row.Diapc,
			&row.Stnpc,
			&row.Mrp1,
			&row.Mrp2,
			&row.Sdamt,
			&row.Ssamt,
			&row.Slamt,
			&row.Smamt,
			&row.Scamt,
			&row.Mrp,
			&row.Hm,
			&row.Certno,
			&row.Spolish,
			&row.Spolishwt,
			&row.Pc,
			&row.Salemrp,
			&row.Diawt1,
			&row.Diawt2,
			&row.Stnwt1,
			&row.Stnwt2,
			&row.Lakhwt,
			&row.Shape,
			&row.Itname,
			&row.Category,
			&row.Gcode,
			&row.Tsno,
			&row.Designid,
			&row.Pname,
			&row.Branch,
			&row.Tgmastid,
			&row.Diacut,
			&row.Diapol,
			&row.Diasymm,
			&row.Pdis,
			&row.Cindex,
			&row.Photopath,
			&row.Igroup,
			&row.Rfid,
			&row.Gst,
			&row.Billamt,
			&row.Billtype,
			&row.Tgimage,
			&row.Ino,
			&row.Csmamt,
			&row.Cslamt,
			&row.Itype,
			&row.Unit,
			&row.Srate,
			&row.Pgst,
			&row.Siteid,
		)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreTgMaster While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), TgMasterTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TgMasterTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", TgMasterTableName, time.Since(startTime))

	var divided [][]*mysql_to_surreal_interfaces.TgMasterStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(divided))
	for k, v := range divided {
		go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TgMasterTableName, k, v, &waitGroup)
	}
	waitGroup.Wait()
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TgMasterTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", TgMasterTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", TgMasterTableName, time.Since(initalTime))
}
