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

const TransactionTableName = "tran1"

var (
	GetTrans1TableCommand = ""
)

func init() {
	GetTrans1TableCommand = fmt.Sprintf("SELECT * FROM %s", TransactionTableName)

}
func removeAndInsertTrans1Table(c *ConfigWithConnection) {
	// _, err := surrealdb.Delete[any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TransactionTableName))
	// if err != nil {
	// 	fmt.Printf("Issue In Deleting Table %s from SurrealDB: %s\n", TransactionTableName, err.Error())
	// }
	_, err := surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", TransactionTableName), nil)
	if err != nil {
		fmt.Printf("Issue In Removing Table %s from SurrealDB: %s\n", TransactionTableName, err.Error())
	}
	_, err = surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(TransactionTableName, mysql_to_surreal_interfaces.Tran1Struct{}, true), nil)
	if err != nil {
		fmt.Printf("Issue In Defining Table %s in SurrealDB: %s\n", TransactionTableName, err.Error())
	}
	fmt.Printf("Removed And Created %s\n", TransactionTableName)
}
func (c *ConfigWithConnection) ReadAndStoreTrans1Table() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetTrans1TableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreTrans1Table For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.Tran1Struct = []*mysql_to_surreal_interfaces.Tran1Struct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.Tran1Struct{}
		err = rows.Scan(
			&row.TRNID,
			&row.TNO,
			&row.ACNO,
			&row.TDATE,
			&row.VTYPEID,
			&row.VONO,
			&row.FINE1,
			&row.FINE2,
			&row.AMOUNT,
			&row.LNARR,
			&row.NARR,
			&row.BILLNO,
			&row.REFNO,
			&row.SERIESID,
			&row.NOSERIESID,
			&row.SAMOUNT,
			&row.TAX,
			&row.DISCOUNT,
			&row.ADJUST,
			&row.DIFF,
			&row.SITEID,
			&row.SITEIDTO,
			&row.ACCOUNT,
			&row.SMACNO,
			&row.Op1,
			&row.Op2,
			&row.Op3,
			&row.Op4,
			&row.Op5,
			&row.Op6,
			&row.NO12,
			&row.RNAMEID,
			&row.GROSSWT1,
			&row.GROSSWT2,
			&row.Cpname,
			&row.Cpadd1,
			&row.Cpadd2,
			&row.Cpadd3,
			&row.CPLOCATION,
			&row.CPCITY,
			&row.CPPHONE,
			&row.CPMOBILE,
			&row.CPDOB,
			&row.CPASARY,
			&row.CPID,
			&row.PTAX,
			&row.SURCHARGE,
			&row.VTYPECODE,
			&row.HISAB,
			&row.LOGINID,
			&row.APPROVED,
			&row.VTCODE,
			&row.DDATE,
			&row.STATUS,
			&row.IDATE,
			&row.KITTYINS,
			&row.BHAVWT,
			&row.BHAVTUNCH,
			&row.BHAVRATE,
			&row.BHAVRTUNCH,
			&row.DLESS,
			&row.SLESS,
			&row.LBRON,
			&row.HISABVONO,
			&row.VTYPE,
			&row.MEMID,
			&row.DTIME,
			&row.GROSSFINE1,
			&row.GROSSFINE2,
			&row.BADLARATE,
			&row.BADLA,
			&row.BADLAON,
			&row.BADLANJ,
			&row.FINAL,
			&row.DELIVER,
			&row.RMODE,
			&row.MAINACNO,
			&row.LOTTID,
			&row.ORDID,
			&row.STAMPID,
			&row.Issgwt,
			&row.Isswt,
			&row.Recgwt,
			&row.Recwt,
			&row.Losswt,
			&row.Lossfine,
			&row.Balwt,
			&row.Balfine,
			&row.EXWT,
			&row.EXFINE,
			&row.Lbramt,
			&row.KHISAB,
			&row.Stnwt,
			&row.STNPC,
			&row.Diawt,
			&row.DIAPC,
			&row.STBEEDS,
			&row.ALWSTG,
			&row.ALFINE,
			&row.DUSTWT,
			&row.TUNCHWT,
			&row.ISSPC,
			&row.RECPC,
			&row.AVGTUNCH,
			&row.LOSSDATE,
			&row.WAXDIAWT,
			&row.WAXSTWT,
			&row.STFINE,
			&row.EXTUNCH,
			&row.MELTUNCH,
			&row.Melwt,
			&row.KADIWT,
			&row.KPCWT,
			&row.KADIPC,
			&row.Konewt,
			&row.BDSCALC,
			&row.STSTNWT,
			&row.STSTNPC,
			&row.Flag,
			&row.RATEFIX,
			&row.Crdays,
			&row.RPAMOUNT,
			&row.QTTNO,
			&row.INVDATE,
			&row.EBILL,
			&row.AUDIT,
			&row.CHQID,
			&row.BBID,
			&row.BILLTYPE,
			&row.DAILY,
			&row.Altvou,
			&row.MHISAB,
			&row.BILLFINE1,
			&row.BILLFINE2,
			&row.BILLBAL,
			&row.ADJFINE1,
			&row.ADJFINE2,
			&row.ADJBAL,
			&row.TGINO,
			&row.TGIGROUPID,
			&row.TGPTUNCH,
			&row.TGPWSTG,
			&row.TGRATE,
			&row.TGMRATE,
			&row.TGSPOLISH,
			&row.TGSTUNCH,
			&row.TGLBR,
			&row.TGSLBR,
			&row.TGSTKINO,
			&row.TGSTAMPID,
			&row.TGPCLESS,
			&row.TGDSLESS,
			&row.TGVALLESS,
			&row.PAMOUNT,
			&row.UPFLG,
			&row.TRDATA,
			&row.MAINRECORD,
			&row.PDC,
			&row.INSFLG,
			&row.JOBID,
			&row.UPHISAB,
			&row.OACNO,
			&row.JOBNO,
			&row.OJOBNO,
			&row.OJOBID,
			&row.DESIGNID,
			&row.Issfine,
			&row.Recfine,
			&row.POLOSSWT,
			&row.POLOSSFINE,
			&row.POSTNFINE,
			&row.BULLID,
			&row.TGPPOLISH,
			&row.MRNAMEID,
			&row.EXPTAX,
			&row.ACNO12ACNO,
			&row.EXRATEOK,
			&row.CDCHARGE,
			&row.STAX,
			&row.DOLLAR,
			&row.METALTYPE,
			&row.CHQTRNID,
			&row.INTHISAB,
			&row.RTUNCH,
			&row.WTUNCH,
			&row.BANKCHG,
			&row.POSTAMT,
			&row.POSTRATE,
			&row.POSTLBR,
			&row.SUMNO,
			&row.SPOLISHWT,
			&row.DUSTFINE,
			&row.ORGACNO,
			&row.INO,
			&row.TRTRNID,
			&row.TGSET,
			&row.ORGCNAME,
			&row.LEDWT,
			&row.OPENING,
			&row.MACNO,
			&row.Tgbatch,
			&row.TUNCHFINE,
			&row.KITTYVONO,
			&row.POSTLFINE,
			&row.MCURRID,
			&row.MCAMOUNT,
			&row.Xrate,
			&row.MCXCNO,
			&row.APPCODE,
			&row.CARDTYPE,
			&row.Billnarr,
			&row.BADLAID,
			&row.MCAMOUNT2,
			&row.MCURRID1,
			&row.MCURRID2,
			&row.ALLOT,
			&row.FINE3,
			&row.FINE4,
			&row.OP7,
			&row.UGAMOUNT,
			&row.LEDGERFLG,
			&row.CNTACNO,
			&row.PRNFLAG,
			&row.LABID,
			&row.DMTNO,
			&row.DMLOGINID,
			&row.DMTIME,
			&row.DEL,
			&row.COMACNO,
			&row.COMAMT,
			&row.MPOINTS,
			&row.COUPONID,
			&row.WEBID,
			&row.MTRANID,
			&row.INTEREST,
			&row.Notecntr,
			&row.DUEDATE,
			&row.EXCISE,
			&row.SCHGAMT,
			&row.INTHISABG,
			&row.INTHISABS,
			&row.CPPAN,
			&row.OTHFINE,
			&row.CASHBANK,
			&row.CGST,
			&row.SGST,
			&row.IGST,
			&row.CPSTATEID,
			&row.CPGSTIN,
			&row.Brid,
			&row.Irn,
			&row.Billinfo,
			&row.Itcstatus,
		)
		row.SurrealId = row.TRNID
		row.SurrealACNO = models.NewRecordID(AccMastTableName, row.ACNO)
		row.SurrealINO = models.NewRecordID(ItemMastTableName, row.INO)
		row.SurrealSITEID = models.NewRecordID(SiteTableName, row.SITEID)
		row.SurrealSITEIDTO = models.NewRecordID(SiteTableName, row.SITEIDTO)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreTrans1Table While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), TransactionTableName, time.Since(startTime))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", TransactionTableName, time.Since(startTime))
	var divided [][]*mysql_to_surreal_interfaces.Tran1Struct
	chunkSize := 50
	for i := 0; i < len(results); i += chunkSize {
		end := min(i+chunkSize, len(results))
		divided = append(divided, results[i:end])
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(divided))
	for k, v := range divided {
		go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	waitGroup.Wait()
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](c.DbConnections.SurrealDbConncetion.Db, models.Table(TransactionTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", TransactionTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", TransactionTableName, time.Since(initalTime))
}
