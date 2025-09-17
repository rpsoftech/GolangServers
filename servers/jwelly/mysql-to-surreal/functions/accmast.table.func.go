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

const AccMastTableName = "accmast"

var (
	GetAccMastTableCommand = ""
)

func removeAndInsertAccMastTable(c *ConfigWithConnection) {
	// surrealdb.Query(c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", AccMastTableName), nil)
	_, err := surrealdb.Delete[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(AccMastTableName))
	if err != nil {
		_, err := surrealdb.Delete[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(AccMastTableName))
		fmt.Printf("Issue In Deleting Table %s from SurrealDB: %s\n", AccMastTableName, err.Error())
	}
	_, err = surrealdb.Query[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", AccMastTableName), nil)
	if err != nil {
		fmt.Printf("Issue In Removing Table %s from SurrealDB: %s\n", AccMastTableName, err.Error())
	}
	_, err = surrealdb.Query[any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, localSurrealdb.GenerateDefineQueryWithIndexAndByStruct(AccMastTableName, mysql_to_surreal_interfaces.AccMastTableStruct{}, true), nil)
	if err != nil {
		fmt.Printf("Issue In Defining Table %s in SurrealDB: %s\n", AccMastTableName, err.Error())
	}
	fmt.Printf("Removed And Created %s\n", AccMastTableName)
}

func init() {
	GetAccMastTableCommand = fmt.Sprintf("SELECT * FROM %s", AccMastTableName)

}
func (c *ConfigWithConnection) ReadAndStoreAccMastTable() {
	rows, err := c.DbConnections.MysqlDbConncetion.Db.Query(GetAccMastTableCommand)
	initalTime := time.Now()
	startTime := initalTime
	if err != nil {
		fmt.Printf("Error in ReadAndStoreAccMastTable For %s", c.ServerConfig.Name)
		fmt.Println(err.Error())
		return
	}
	var results []*mysql_to_surreal_interfaces.AccMastTableStruct = []*mysql_to_surreal_interfaces.AccMastTableStruct{}
	for rows.Next() {
		row := &mysql_to_surreal_interfaces.AccMastTableStruct{}
		err = rows.Scan(
			&row.Acno,
			&row.PREFIX,
			&row.CNAME,
			&row.GROUPID,
			&row.PNAME,
			&row.ADD1,
			&row.ADD2,
			&row.ADD3,
			&row.CITY,
			&row.LOCATION,
			&row.PHONE,
			&row.MOBILE,
			&row.DOB,
			&row.ASARY,
			&row.STNO,
			&row.PAN,
			&row.PERCENT,
			&row.CRFINE1,
			&row.CRFINE2,
			&row.CRAMT,
			&row.DEL,
			&row.ACTIVE,
			&row.CRDAYS,
			&row.IGROUPID,
			&row.EMAIL,
			&row.PIN,
			&row.STOPDATE,
			&row.Shortname,
			&row.KPREFIX,
			&row.KMNO,
			&row.KVMNO,
			&row.SALARY,
			&row.PF,
			&row.PFACNO,
			&row.ESI,
			&row.ESIACNO,
			&row.NO12,
			&row.PHOTO,
			&row.QUARTER1,
			&row.QUARTER2,
			&row.QUARTER3,
			&row.QUARTER4,
			&row.TDSBILL,
			&row.WTBAL,
			&row.LBRON,
			&row.COUNTRY,
			&row.DEP,
			&row.NATURE,
			&row.BANKACNO,
			&row.LEAVE,
			&row.JOINDATE,
			&row.DEPT,
			&row.SALE,
			&row.PURCHASE,
			&row.ORD,
			&row.REPAIR,
			&row.KITTY,
			&row.ADVANCE,
			&row.ISSUE,
			&row.RECEIVE,
			&row.TAGING,
			&row.JOBID,
			&row.DLESS,
			&row.SLESS,
			&row.TRAN,
			&row.KREFBY,
			&row.KSMAN,
			&row.KREMARKS,
			&row.KINSAMT,
			&row.KMONTHS,
			&row.KINSREC,
			&row.KSDATE,
			&row.KEDATE,
			&row.REFGROUP,
			&row.STAMPID,
			&row.PAYDATE,
			&row.BANKCHG,
			&row.BRPATH,
			&row.ACNO12,
			&row.REFBY,
			&row.KPMONTH,
			&row.SCODE,
			&row.KREFMNO,
			&row.BANKBADLA,
			&row.BANKEX,
			&row.METAL,
			&row.LOGINID,
			&row.DTIME,
			&row.BALLOW,
			&row.VTDS,
			&row.PVOU,
			&row.INTR,
			&row.COMINT,
			&row.COMINTDEF,
			&row.GRTRAN,
			&row.MINT,
			&row.HOLIDAY,
			&row.DIR,
			&row.KHOME,
			&row.ACNARR,
			&row.DOC,
			&row.BKTRC,
			&row.ORGACNO,
			&row.COMPATH,
			&row.COMPNO,
			&row.VCOM,
			&row.SITEID,
			&row.MCURRID,
			&row.MCXCOMEX,
			&row.RSCOMM,
			&row.REMARKS,
			&row.AGENTACNO,
			&row.Hpname,
			&row.ODLIMIT,
			&row.DEP2,
			&row.OCCUPATION,
			&row.Sflag,
			&row.BANKCHG2,
			&row.KTACNO,
			&row.SCBONY,
			&row.BANKNO,
			&row.SWCODE,
			&row.NOSTRO,
			&row.MINLBR,
			&row.SALARYB,
			&row.PRNROW,
			&row.MAXVTGNO,
			&row.MAIN,
			&row.ADD4,
			&row.OTHCHG,
			&row.GENDER,
			&row.SPOUSE,
			&row.SPOUSEDOB,
			&row.MARRIED,
			&row.EDUCATION,
			&row.KNOWLEDGE,
			&row.POTENTIAL,
			&row.OPPOINTS,
			&row.WEBSITE,
			&row.CSTNO,
			&row.RBOOKDAY,
			&row.KINSWT,
			&row.BRANCHID,
			&row.UIDNO,
			&row.WRKHRS,
			&row.LOTTBAL,
			&row.OTHFINE,
			&row.GSTIN,
			&row.STCODE,
			&row.STATEID,
			&row.GST,
			&row.HSNCODE,
			&row.COMPS,
			&row.ECOM,
			&row.EXPTYPE,
			&row.OPLOAN,
			&row.PFSALARY,
			&row.Opwithdraw,
			&row.Salins,
			&row.Cdc,
			&row.Netbank,
			&row.Userid,
			&row.Corpid,
			&row.Accno,
			&row.Ifsc,
			&row.Urnid,
			&row.Bnkstatus,
			&row.Bnkalias,
			&row.Sysid,
			&row.Impcol,
			&row.Spemail,
			&row.Spmobile,
			&row.Upd,
			&row.Hisab,
			&row.Salarydet,
			&row.Stcatid,
			&row.Tcs,
			&row.Hmno,
			&row.Wstgon,
			&row.Dtswid,
			&row.Msme,
		)
		// row.SurrealId = fmt.Sprintf("%d", row.TSNO)
		row.SurrealId = row.Acno
		// row.SurrealSTAMPID = models.NewRecordID(StampTableName, row.STAMPID)
		// row.SurrealUNITID = models.NewRecordID(UnitTableName, row.UNITID)
		// row.SurrealSITEID = models.NewRecordID(SiteTableName, row.SITEID)
		// row.SurrealTRSITEID = models.NewRecordID(SiteTableName, row.TRSITEID)
		if err != nil {
			fmt.Printf("Error in ReadAndStoreAccMastTable While Scanning %s", c.ServerConfig.Name)
			fmt.Println(err.Error())
			return
		}
		results = append(results, row)
	}
	fmt.Printf("Fetched Total %d rows from %s in Duration of %s\n", len(results), AccMastTableName, time.Since(startTime))
	// fmt.Printf("Delete All %s from SurrealDB in Duration of %s\n", AccMastTableName, time.Since(startTime))

	var divided [][]*mysql_to_surreal_interfaces.AccMastTableStruct
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
			go upsertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, AccMastTableName, base+k1, v1, &waitGroup)
		}
		waitGroup.Wait()
		// go insertDataToSurrealDb(c.DbConnections.SurrealDbConncetion, TransactionTableName, k, v, &waitGroup)
	}
	waitGroup.Wait()
	startTime = time.Now()
	// surrealdb.Q
	if dddd, err := surrealdb.Select[[]any](localSurrealdb.SurrealCTX, c.DbConnections.SurrealDbConncetion.Db, models.Table(AccMastTableName)); err == nil {
		fmt.Printf("Select All %s from SurrealDB in Duration of %s with total rows %d\n", AccMastTableName, time.Since(startTime), len(*dddd))
	}
	fmt.Printf("%s Operation Completed in Duration of %s\n", AccMastTableName, time.Since(initalTime))
}
