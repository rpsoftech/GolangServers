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

const ItemTransTableName = "itran1"

var (
	GetItemTransTableCommand = ""
)

func init() {
	GetItemTransTableCommand = fmt.Sprintf("SELECT * FROM %s", ItemTransTableName)

}

func removeAndInsertItemTransTable(c *ConfigWithConnection) {
	_, err := surrealdb.Delete[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemTransTableName))
	if err != nil {
		_, err := surrealdb.Delete[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemTransTableName))
		fmt.Printf("Issue In Deleting Table %s from SurrealDB: %s\n", ItemTransTableName, err.Error())
	}
	_, err = surrealdb.Query[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", ItemTransTableName), nil)
	if err != nil {
		fmt.Printf("Issue In Removing Table %s from SurrealDB: %s\n", ItemTransTableName, err.Error())
	}
	_, err = surrealdb.Query[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(ItemTransTableName, mysql_to_surreal_interfaces.ITransStruct{}, true), nil)
	if err != nil {
		fmt.Printf("Issue In Defining Table %s in SurrealDB: %s\n", ItemTransTableName, err.Error())
	}
	fmt.Printf("Removed And Created %s\n", ItemTransTableName)
}
func (c *ConfigWithConnection) ReadAndStoreItemTransTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetItemTransTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreItemTransTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.ITransStruct = []*mysql_to_surreal_interfaces.ITransStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.ITransStruct{}
		err = rows.Scan(
			&row.ITRNID,
			&row.TRNID,
			&row.VONO,
			&row.TNO,
			&row.ITSTDID,
			&row.SNTRNID,
			&row.TDATE,
			&row.INO,
			&row.REMARKS,
			&row.GWT,
			&row.WT,
			&row.LESSWT,
			&row.PC,
			&row.Rate,
			&row.TUNCH,
			&row.WSTG,
			&row.MAMT,
			&row.TYPE,
			&row.STOCK,
			&row.UNITID,
			&row.STAMPID,
			&row.SITEID,
			&row.TVALUE,
			&row.STUDED,
			&row.DAMT,
			&row.SAMT,
			&row.LAMT,
			&row.BAMT,
			&row.OTHERS,
			&row.MRP,
			&row.LBR,
			&row.SLBR,
			&row.SLBR2,
			&row.SLBR3,
			&row.LCODE,
			&row.PTAX,
			&row.Design,
			&row.COLOUR,
			&row.CLARITY,
			&row.Shape,
			&row.Size,
			&row.FINE1,
			&row.FINE2,
			&row.GROSSWT1,
			&row.GROSSWT2,
			&row.TGNO,
			&row.Vtgno,
			&row.TPRE,
			&row.TSNO,
			&row.PDIS,
			&row.COSTAMT,
			&row.DIAWT,
			&row.STNWT,
			&row.DIVIDE,
			&row.PREWSTG,
			&row.PREWSTGWT,
			&row.POLISH,
			&row.POLISHWT,
			&row.ORGRATE,
			&row.ORGGWT,
			&row.ORGTOTAL,
			&row.DLESS,
			&row.SLESS,
			&row.TOTAL,
			&row.KACNO,
			&row.Karigar,
			&row.NO12,
			&row.BATCH,
			&row.Goldwt,
			&row.Silwt,
			&row.PLATWT,
			&row.Othwt,
			&row.Metalwt,
			&row.JOBNO,
			&row.JOBSTART,
			&row.RETWT,
			&row.RETPC,
			&row.BROKWT,
			&row.BROKPC,
			&row.ADJWT,
			&row.ADJVALUE,
			&row.BALWT,
			&row.BALPC,
			&row.SPRICE,
			&row.MRATE,
			&row.DTIME,
			&row.SETNO,
			&row.Pktwt,
			&row.Pktless,
			&row.GWT1,
			&row.GWT2,
			&row.PART,
			&row.PURTUNCH,
			&row.BEEDS,
			&row.SBEEDS,
			&row.RATEUNIT,
			&row.BEEDSRATE,
			&row.BANFINE,
			&row.SAMPLE,
			&row.METTYPE,
			&row.HISAB,
			&row.SVALUE,
			&row.TUNCHWT,
			&row.ACTUNCH,
			&row.RPTUNCH,
			&row.TRVONO,
			&row.TRID,
			&row.TUNCHREM,
			&row.TCHTNO,
			&row.PHOTOPATH,
			&row.DIAPC,
			&row.STNPC,
			&row.APPROVAL,
			&row.Certno,
			&row.SLBR2PC1,
			&row.SLBR2PC2,
			&row.LINKTGNO,
			&row.LINKTSNO,
			&row.STDFINE1,
			&row.STDFINE2,
			&row.QUALITY,
			&row.Hm,
			&row.LINKITRNID,
			&row.PEXWT,
			&row.EXWT,
			&row.Stunch,
			&row.SWSTG,
			&row.SWT,
			&row.PROFIT,
			&row.LESS,
			&row.SPOLISH,
			&row.SPOLISHWT,
			&row.TINT,
			&row.Sothers,
			&row.ALLOT,
			&row.TGDID,
			&row.PLESS,
			&row.GROSSFINE1,
			&row.GROSSFINE2,
			&row.IWT,
			&row.IPC,
			&row.CALCWT,
			&row.CALCPC,
			&row.RITRNID,
			&row.BBID,
			&row.LINKBBID,
			&row.PRETWT,
			&row.LOTTID,
			&row.SIZEID,
			&row.OWNER,
			&row.OWNACNO,
			&row.TRTNO,
			&row.LABID,
			&row.DELDATE,
			&row.SALEMRP,
			&row.JOBID,
			&row.Stkfine1,
			&row.Stkfine2,
			&row.OTGDID,
			&row.KTINO,
			&row.KTTGNO,
			&row.KTVTGNO,
			&row.KTSLBR2,
			&row.KTSOTHERS,
			&row.KTTSNO,
			&row.KTITRNID,
			&row.Sttunch,
			&row.BILLTYPE,
			&row.STDLBR,
			&row.COSTRATE,
			&row.OJOBNO,
			&row.OJOBID,
			&row.OCANCEL,
			&row.OCREMARKS,
			&row.OCDATE,
			&row.OCLOGINID,
			&row.OCTIME,
			&row.DESIGNID,
			&row.COSTVALUE,
			&row.COSTMAMT,
			&row.COSTDAMT,
			&row.COSTSAMT,
			&row.COSTLAMT,
			&row.COSTWSTG,
			&row.DIS,
			&row.SRTYPE,
			&row.COSTTOTAL,
			&row.GOLDAMT,
			&row.SILAMT,
			&row.TAGFINE1,
			&row.TAGFINE2,
			&row.PURCHASE,
			&row.Pcrate,
			&row.SETNOBAL,
			&row.JOBGWT,
			&row.Orgwt,
			&row.Othrem,
			&row.REVISED,
			&row.Saleless,
			&row.PPROFIT,
			&row.LBRADD,
			&row.DAILY,
			&row.DIAWT1,
			&row.DIAWT2,
			&row.STNWT1,
			&row.STNWT2,
			&row.LAKHWT,
			&row.SBANFINE,
			&row.UPFLG,
			&row.WID,
			&row.OACNO,
			&row.OAMT,
			&row.POAMT,
			&row.KHISAB,
			&row.MRP1,
			&row.MRP2,
			&row.COSTING1,
			&row.COSTING2,
			&row.WITNO,
			&row.LBRON,
			&row.DOLLAR,
			&row.DOLXRATE,
			&row.KISNO,
			&row.Diaremark,
			&row.LAMTIDET,
			&row.Distype,
			&row.IHISAB,
			&row.SMACNO,
			&row.Psgst,
			&row.Pcgst,
			&row.PIGST,
			&row.SGST,
			&row.CGST,
			&row.IGST,
			&row.TGCODE,
			&row.Lnkjobid,
			&row.KTBATCH,
			&row.Ttvalue,
			&row.Ksitrnid,
			&row.Ktsvalue,
			&row.Xtfld,
		)
		// row.SurrealId = fmt.Sprintf("%d", row.TSNO)
		row.SurrealId = row.ITRNID
		row.SurrealSTAMPID = models.NewRecordID(StampTableName, row.STAMPID)
		row.SurrealUNITID = models.NewRecordID(UnitTableName, row.UNITID)
		row.SurrealSITEID = models.NewRecordID(SiteTableName, row.SITEID)
		// row.SurrealTRSITEID = models.NewRecordID(SiteTableName, row.TRSITEID)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreItemTransTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), ItemTransTableName, time.Since(startTime))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", ItemTransTableName, time.Since(startTime))
	var divided [][]*mysql_to_surreal_interfaces.ITransStruct
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
			go upsertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, ItemTransTableName, base+k1, v1, &waitGroup)
		}
		waitGroup.Wait()
		// go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	waitGroup.Wait()
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(ItemTransTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", ItemTransTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", ItemTransTableName, time.Since(initalTime))
}
