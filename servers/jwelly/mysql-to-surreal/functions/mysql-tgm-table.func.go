package mysql_to_surreal_functions

import (
	"fmt"
	"time"

	mysql_to_surreal_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/interfaces"
	localSurrealdb "github.com/rpsoftech/golang-servers/utility/surrealdb"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const TgmTableName = "tgm1"

var (
	GetTgm1TableCommand = ""
)

func init() {
	GetTgm1TableCommand = fmt.Sprintf("SELECT * FROM %s", TgmTableName)

}
func (c *ConfigWithConnection) ReadAndStoreTgm1Table() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetTgm1TableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreTgMaster For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.TGM1Struct = []*mysql_to_surreal_interfaces.TGM1Struct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.TGM1Struct{}
		err = rows.Scan(
			&row.TSNO,
			&row.TGNO,
			&row.VTGNO,
			&row.INO,
			&row.TPRE,
			&row.REMARKS,
			&row.TDATE,
			&row.ITRNID,
			&row.GWT,
			&row.LESSWT,
			&row.WT,
			&row.DIAWT,
			&row.STNWT,
			&row.GOLDWT,
			&row.SILWT,
			&row.PLATWT,
			&row.OTHWT,
			&row.LBR,
			&row.LBR2,
			&row.LBR3,
			&row.SLBR,
			&row.Slbr2,
			&row.SLBR3,
			&row.STATUS,
			&row.TUNCH,
			&row.WSTG,
			&row.STUNCH,
			&row.SWSTG,
			&row.BEEDS,
			&row.SBEEDS,
			&row.SOTHERS,
			&row.Othrem,
			&row.Design,
			&row.DESINO,
			&row.KACNO,
			&row.Karigar,
			&row.MRATE,
			&row.COSTRATE,
			&row.GWT1,
			&row.GWT2,
			&row.SLBR2PC1,
			&row.SLBR2PC2,
			&row.STAMPID,
			&row.Flag,
			&row.PHOTOPATH,
			&row.Size,
			&row.QUALITY,
			&row.COLOUR,
			&row.CLARITY,
			&row.POLISH,
			&row.POLISHWT,
			&row.SITEID,
			&row.LOGIN,
			&row.SETNO,
			&row.ORDNO,
			&row.DLESS,
			&row.SLESS,
			&row.LINKTGNO,
			&row.LINKTSNO,
			&row.ADDMRP,
			&row.LAMT,
			&row.DAMT,
			&row.SAMT,
			&row.MAMT,
			&row.OTHERS,
			&row.TAGFINE1,
			&row.TAGFINE2,
			&row.COSTDAMT,
			&row.COSTSAMT,
			&row.COSTMAMT,
			&row.MRP,
			&row.Hm,
			&row.Certno,
			&row.SPOLISH,
			&row.SPOLISHWT,
			&row.SWT,
			&row.VONO,
			&row.TNO,
			&row.Type,
			&row.SDLESS,
			&row.SSLESS,
			&row.BILLTYPE,
			&row.UNITID,
			&row.PC,
			&row.SALEMRP,
			&row.JOBID,
			&row.COSTTOTAL,
			&row.COSTWSTG,
			&row.GROSSFINE1,
			&row.GROSSFINE2,
			&row.FINE1,
			&row.FINE2,
			&row.TOTAL,
			&row.Pcrate,
			&row.PPROFIT,
			&row.UPFLAG,
			&row.STKFINE1,
			&row.STKFINE2,
			&row.RATE,
			&row.DAILY,
			&row.TRALT,
			&row.DIAWT1,
			&row.DIAWT2,
			&row.STNWT1,
			&row.STNWT2,
			&row.LAKHWT,
			&row.BANFINE,
			&row.SBANFINE,
			&row.UPSTATUS,
			&row.UPINFO,
			&row.APPROVAL,
			&row.DESIGNID,
			&row.REPWT,
			&row.LASTDATE,
			&row.DIAPC,
			&row.STNPC,
			&row.MRP1,
			&row.MRP2,
			&row.COSTING1,
			&row.COSTING2,
			&row.MFLAG,
			&row.SIZEID,
			&row.GITRNID,
			&row.Diaremark,
			&row.DOLLAR,
			&row.DOLXRATE,
			&row.PDIS,
			&row.OLDSTATUS,
			&row.LCODE,
			&row.SHAPE,
			&row.TVALUE,
			&row.TRITRNID,
			&row.UPTGNO,
			&row.SVALUE,
			&row.RFID,
			&row.BATCH,
			&row.COSTLAMT,
			&row.OJOBID,
			&row.OLINKTGNO,
			&row.SDAMT,
			&row.SSAMT,
			&row.SLAMT,
			&row.SMAMT,
			&row.DIS,
			&row.PKTWT,
			&row.PKTLESS,
			&row.ORGGWT,
			&row.ORGTOTAL,
			&row.OACNO,
			&row.VTOTAL,
			&row.TGMIDESC,
			&row.SDIAWT,
			&row.SSTNWT,
			&row.TRSITEID,
			&row.SAMPLE,
			&row.SRTYPE,
			&row.OAMT,
			&row.POAMT,
			&row.RLID,
			&row.COSTAMT,
			&row.OCDATE,
			&row.TRLBR,
			&row.TRLCODE,
			&row.TRFLAG,
			&row.TGCODE,
			&row.Stagfine1,
			&row.Stagfine2,
			&row.Pless,
			&row.Calmrp,
			&row.Gst,
			&row.Saleless,
			&row.Csmamt,
			&row.Grade,
			&row.Cslamt,
			&row.Srate,
			&row.Deldate,
			&row.Pgst,
			&row.Xtfld,
			&row.Cstdmamt,
		)
		row.SurrealId = fmt.Sprintf("%d", row.TSNO)
		// row.SurrealSTAMPID = fmt.Sprintf("%s:%d", StampTableName, row.STAMPID)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreTgMaster While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), TgmTableName, time.Since(startTime))
	startTime = time.Now()
	// surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TgmTableName))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", TgmTableName, time.Since(startTime))
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", TgmTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(TgmTableName, mysql_to_surreal_interfaces.TGM1Struct{}, true), nil)
	startTime = time.Now()
	var divided [][]*mysql_to_surreal_interfaces.TGM1Struct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	for k, v := range divided {
		_, err := surrealdb.Insert[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TgmTableName), v)
		if err != nil {
			fmt.Printf("Issue In Round %d while inserting %s with a struct: %s\n", k, TgmTableName, "TLDR;")
		}
		fmt.Printf("Roiund %d Inserted %d rows to %s in SurrealDB in Duration of %s\n", k, len(v), TgmTableName, time.Since(startTime))
		startTime = time.Now()
	}
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]interface{}](c.DbConnections.SurrealDbConncetion.Db, models.Table(TgmTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", TgmTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", TgmTableName, time.Since(initalTime))
}
