package mysql_to_surreal_functions

import (
	"fmt"
	"time"

	mysql_to_surreal_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/interfaces"
	localSurrealdb "github.com/rpsoftech/golang-servers/utility/surrealdb"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const ItemMastTableName = "itemmast"

var (
	GetItemMastTableCommand = ""
)

func init() {
	GetItemMastTableCommand = fmt.Sprintf("SELECT * FROM %s", ItemMastTableName)

}
func (c *ConfigWithConnection) ReadAndStoreItemMast() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetItemMastTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreItemMast For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.ItemMastTableStruct = []*mysql_to_surreal_interfaces.ItemMastTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.ItemMastTableStruct{}
		err = rows.Scan(
			&row.Ino,
			&row.IDESC,
			&row.DIAST,
			&row.IGROUPID,
			&row.PNAME,
			&row.UNITID,
			&row.SM,
			&row.Tpre,
			&row.PC,
			&row.TREYPC,
			&row.MAXTGNO,
			&row.STARTTGNO,
			&row.PCSTOCK,
			&row.BLIST,
			&row.ADDWT,
			&row.TWODET,
			&row.SALE,
			&row.PURCHASE,
			&row.KARIGAR,
			&row.STOCK,
			&row.STVAL,
			&row.STUDED,
			&row.STAMPID,
			&row.GCODEID,
			&row.FINECAL,
			&row.CSTAMP,
			&row.CBEEDS,
			&row.CGWT,
			&row.GRADE,
			&row.CATID,
			&row.RESTGWT,
			&row.RESTTUNCH,
			&row.RESTWSTG,
			&row.Shortname,
			&row.BOXWT,
			&row.POLISH,
			&row.PROFIT,
			&row.INTORATE,
			&row.DESIPRE,
			&row.WSWSTG,
			&row.WSLBR,
			&row.RETWSTG,
			&row.RETLBR,
			&row.Lcode,
			&row.DEFTUNCH,
			&row.STONES,
			&row.LINKTG,
			&row.LINKINO,
			&row.STMETHOD,
			&row.SRNO,
			&row.GCREATE,
			&row.BPRETYPE,
			&row.ISTUD,
			&row.BADHEL,
			&row.DITEMRATE,
			&row.RESTRATE,
			&row.STVALTUNCH,
			&row.STVALWSTG,
			&row.STVALLBR,
			&row.IRGWT,
			&row.TPCODE,
			&row.PARTYDST,
			&row.RESTZERO,
			&row.GCOMM,
			&row.GCON,
			&row.GCDIS,
			&row.SCOMM,
			&row.SCON,
			&row.SCDIS,
			&row.VSTK,
			&row.TGDIGIT,
			&row.DESIDIGIT,
			&row.REM1,
			&row.REM2,
			&row.LOGINID,
			&row.DTIME,
			&row.LBRON,
			&row.SLBRON,
			&row.DBNAME,
			&row.DESIGN,
			&row.MRP,
			&row.RAPARATE,
			&row.SALELESS,
			&row.RESTSTAMP,
			&row.BOXPC,
			&row.PCRATE,
			&row.RESTBATCH,
			&row.Sqrmtr,
			&row.PEXWT,
			&row.RESTOTH,
			&row.TAGWT,
			&row.HPNAME,
			&row.RESTWL,
			&row.SHORTCNM,
			&row.RESTPC,
			&row.SCOWT,
			&row.SCORS,
			&row.BINTORATE,
			&row.POLYWT,
			&row.WIREWT,
			&row.Lstamp,
			&row.PRNCMD,
			&row.MPOINTS,
			&row.PRNSNO,
			&row.TGSTKLESS,
			&row.DTRNSTK,
			&row.LESSDET,
			&row.MINO,
			&row.MSIZEID,
			&row.POINTUPON,
			&row.TGLBRADD,
			&row.ITHSNCODE,
			&row.SRATEUNIT,
			&row.SVSTK,
			&row.Minstk,
			&row.Insamt,
			&row.Webdis,
			&row.Webidesc,
			&row.Webcat,
			&row.Webconf,
			&row.Barslist,
			&row.Chgitm,
			&row.Itmconf,
			&row.Snobcode,
			&row.Minmg,
		)
		row.SurrealId = row.Ino
		row.SurrealSTAMPID = models.NewRecordID(StampTableName, row.STAMPID)
		row.SurrealUNITID = models.NewRecordID(UnitTableName, row.UNITID)
		row.SurrealIGROUPID = models.NewRecordID(ItemGroupTableName, row.IGROUPID)
		row.SurrealCatID = models.NewRecordID(CategoryTableName, row.CATID)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreItemMast While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), ItemMastTableName, time.Since(startTime))
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemMastTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", ItemMastTableName, time.Since(startTime))
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", ItemMastTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(ItemMastTableName, mysql_to_surreal_interfaces.ItemMastTableStruct{}, true), nil)
	startTime = time.Now()
	var divided [][]*mysql_to_surreal_interfaces.ItemMastTableStruct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	for k, v := range divided {
		_, err := surrealdb.Insert[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemMastTableName), v)
		if err != nil {
			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, ItemMastTableName, "TLDR;")
		}
		fmt.Printf("Round %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(v), ItemMastTableName, time.Since(startTime))
		startTime = time.Now()
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemMastTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", ItemMastTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", ItemMastTableName, time.Since(initalTime))
}
