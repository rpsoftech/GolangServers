package ecommerece_functions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ecommerce_maintables "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/MainTables"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

// tsno,TGNO,vtgno,INO,TPRE,REMARKS,TDATE,ITRNID,GWT,LESSWT,WT,DIAWT,STNWT,GOLDWT,SILWT,PLATWT,OTHWT,LBR,LBR2,LBR3,SLBR,slbr2,SLBR3,STATUS,TUNCH,WSTG,STUNCH,SWSTG,BEEDS,SBEEDS,SOTHERS,othrem,design,DESINO,KACNO,karigar,MRATE,COSTRATE,GWT1,GWT2,SLBR2PC1,SLBR2PC2,STAMPID,flag,PHOTOPATH,size,QUALITY,COLOUR,CLARITY,POLISH,POLISHWT,SITEID,LOGIN,SETNO,ORDNO,DLESS,SLESS,LINKTGNO,LINKTSNO,ADDMRP,LAMT,DAMT,SAMT,MAMT,OTHERS,TAGFINE1,TAGFINE2,COSTDAMT,COSTSAMT,COSTMAMT,MRP,hm,certno,SPOLISH,SPOLISHWT,SWT,VONO,TNO,type,SDLESS,SSLESS,BILLTYPE,UNITID,PC,SALEMRP,JOBID,COSTTOTAL,COSTWSTG,GROSSFINE1,GROSSFINE2,FINE1,FINE2,TOTAL,pcrate,PPROFIT,UPFLAG,STKFINE1,STKFINE2,RATE,DAILY,TRALT,DIAWT1,DIAWT2,STNWT1,STNWT2,LAKHWT,BANFINE,SBANFINE,UPSTATUS,UPINFO,APPROVAL,DESIGNID,REPWT,LASTDATE,DIAPC,STNPC,MRP1,MRP2,COSTING1,COSTING2,MFLAG,SIZEID,GITRNID,diaremark,DOLLAR,DOLXRATE,PDIS,OLDSTATUS,LCODE,SHAPE,TVALUE,TRITRNID,UPTGNO,SVALUE,RFID,BATCH,COSTLAMT,OJOBID,OLINKTGNO,SDAMT,SSAMT,SLAMT,SMAMT,DIS,PKTWT,PKTLESS,ORGGWT,ORGTOTAL,OACNO,VTOTAL,TGMIDESC,SDIAWT,SSTNWT,TRSITEID,SAMPLE,SRTYPE,OAMT,POAMT,RLID,COSTAMT,OCDATE,TRLBR,TRLCODE,TRFLAG,TGCODE,stagfine1,stagfine2,pless,calmrp,gst,saleless,csmamt,grade,cslamt,srate,deldate,pgst,xtfld,cstdmamt
// tsno,TGNO,vtgno,INO,TDATE,STAMPID
const SelectItemTagQuery = "select tsno,TGNO,vtgno,INO,TDATE from tgm1 where vtgno != 0 and STATUS != 'ITM' and TPRE !='REP';"
const ServerItemTagOfColumns = 5
const ItemTagUpsertQuery = `INSERT INTO ItemsTag (itemTagId,itemTag,itemVTagId,tItemId,tagCreatedDate) VALUES %s AS new_row ON DUPLICATE KEY UPDATE itemVTagId = new_row.itemVTagId,tItemId = new_row.tItemId,tagCreatedDate = new_row.tagCreatedDate`

func SyncItemTagTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectItemTagQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.ItemTagMainTable = []*ecommerce_maintables.ItemTagMainTable{}
	for rows.Next() {
		row := &ecommerce_maintables.ItemTagMainTable{}
		err = rows.Scan(
			&row.ItemTagId,
			&row.ItemTag,
			&row.ItemVTagId,
			&row.TItemId,
			&row.TagCreatedDate,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Item Tag Main in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerItemTagOfColumns)

	for _, tag := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?)")
		valueArgs = append(valueArgs,
			tag.ItemTagId,
			tag.ItemTag,
			tag.ItemVTagId,
			tag.TItemId,
			tag.TagCreatedDate,
		)
	}
	finalQuery := fmt.Sprintf(ItemTagUpsertQuery, strings.Join(valueStrings, ", "))
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for Item Tag Main successful!  Rows affected metric: %d\n", rowsAffected)

}
