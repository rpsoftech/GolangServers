package mysql_to_surreal_interfaces

import (
	"time"

	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

type AccMastTableStruct struct {
	SurrealId  int             `json:"id" Index:"U"`
	Acno       int             `json:"acno" fieldType:"int | NULL" Index:"U"`
	PREFIX     string          `json:"PREFIX" fieldType:"string | NULL"`
	CNAME      string          `json:"CNAME" fieldType:"string | NULL" Index:"I"`
	GROUPID    int             `json:"GROUPID" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	PNAME      string          `json:"PNAME" fieldType:"string | NULL"`
	ADD1       string          `json:"ADD1" fieldType:"string | NULL"`
	ADD2       string          `json:"ADD2" fieldType:"string | NULL"`
	ADD3       string          `json:"ADD3" fieldType:"string | NULL"`
	CITY       string          `json:"CITY" fieldType:"string | NULL"`
	LOCATION   string          `json:"LOCATION" fieldType:"string | NULL"`
	PHONE      string          `json:"PHONE" fieldType:"string | NULL"`
	MOBILE     string          `json:"MOBILE" fieldType:"string | NULL"`
	DOB        time.Time       `json:"DOB" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	ASARY      time.Time       `json:"ASARY" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	STNO       string          `json:"STNO" fieldType:"string | NULL"`
	PAN        string          `json:"PAN" fieldType:"string | NULL"`
	PERCENT    float64         `json:"PERCENT" fieldType:"float | NULL" defaultValue:"0.00"`
	CRFINE1    float64         `json:"CRFINE1" fieldType:"float | NULL" defaultValue:"0.00"`
	CRFINE2    float64         `json:"CRFINE2" fieldType:"float | NULL" defaultValue:"0.00"`
	CRAMT      float64         `json:"CRAMT" fieldType:"float | NULL" defaultValue:"0.00"`
	DEL        string          `json:"DEL" fieldType:"string | NULL"`
	ACTIVE     string          `json:"ACTIVE" fieldType:"string | NULL"`
	CRDAYS     int             `json:"CRDAYS" fieldType:"int | NULL" defaultValue:"0"`
	IGROUPID   int             `json:"IGROUPID" fieldType:"int | NULL" defaultValue:"0"`
	EMAIL      string          `json:"EMAIL" fieldType:"string | NULL"`
	PIN        string          `json:"PIN" fieldType:"string | NULL"`
	STOPDATE   time.Time       `json:"STOPDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	Shortname  string          `json:"shortname" fieldType:"string | NULL" Index:"I"`
	KPREFIX    string          `json:"KPREFIX" fieldType:"string | NULL"`
	KMNO       string          `json:"KMNO" fieldType:"string | NULL"`
	KVMNO      int             `json:"KVMNO" fieldType:"int | NULL" defaultValue:"0"`
	SALARY     int             `json:"SALARY" fieldType:"int | NULL" defaultValue:"0"`
	PF         string          `json:"PF" fieldType:"string | NULL"`
	PFACNO     string          `json:"PFACNO" fieldType:"string | NULL"`
	ESI        string          `json:"ESI" fieldType:"string | NULL"`
	ESIACNO    string          `json:"ESIACNO" fieldType:"string | NULL"`
	NO12       int             `json:"NO12" fieldType:"int | NULL" defaultValue:"0"`
	PHOTO      string          `json:"PHOTO" fieldType:"string | NULL"`
	QUARTER1   string          `json:"QUARTER1" fieldType:"string | NULL"`
	QUARTER2   string          `json:"QUARTER2" fieldType:"string | NULL"`
	QUARTER3   string          `json:"QUARTER3" fieldType:"string | NULL"`
	QUARTER4   string          `json:"QUARTER4" fieldType:"string | NULL"`
	TDSBILL    int             `json:"TDSBILL" fieldType:"int | NULL" defaultValue:"0"`
	WTBAL      int             `json:"WTBAL" fieldType:"int | NULL" defaultValue:"0"`
	LBRON      int             `json:"LBRON" fieldType:"int | NULL" defaultValue:"0"`
	COUNTRY    string          `json:"COUNTRY" fieldType:"string | NULL"`
	DEP        float64         `json:"DEP" fieldType:"float | NULL" defaultValue:"0.00"`
	NATURE     string          `json:"NATURE" fieldType:"string | NULL"`
	BANKACNO   string          `json:"BANKACNO" fieldType:"string | NULL"`
	LEAVE      time.Time       `json:"LEAVE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	JOINDATE   time.Time       `json:"JOINDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	DEPT       string          `json:"DEPT" fieldType:"string | NULL"`
	SALE       int             `json:"SALE" fieldType:"int | NULL" defaultValue:"0"`
	PURCHASE   int             `json:"PURCHASE" fieldType:"int | NULL" defaultValue:"0"`
	ORD        int             `json:"ORD" fieldType:"int | NULL" defaultValue:"0"`
	REPAIR     int             `json:"REPAIR" fieldType:"int | NULL" defaultValue:"0"`
	KITTY      int             `json:"KITTY" fieldType:"int | NULL" defaultValue:"0"`
	ADVANCE    int             `json:"ADVANCE" fieldType:"int | NULL" defaultValue:"0"`
	ISSUE      int             `json:"ISSUE" fieldType:"int | NULL" defaultValue:"0"`
	RECEIVE    int             `json:"RECEIVE" fieldType:"int | NULL" defaultValue:"0"`
	TAGING     int             `json:"TAGING" fieldType:"int | NULL" defaultValue:"0"`
	JOBID      int             `json:"JOBID" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	DLESS      int             `json:"DLESS" fieldType:"int | NULL" defaultValue:"0"`
	SLESS      int             `json:"SLESS" fieldType:"int | NULL" defaultValue:"0"`
	TRAN       string          `json:"TRAN" fieldType:"string | NULL"`
	KREFBY     string          `json:"KREFBY" fieldType:"string | NULL"`
	KSMAN      string          `json:"KSMAN" fieldType:"string | NULL"`
	KREMARKS   string          `json:"KREMARKS" fieldType:"string | NULL"`
	KINSAMT    int             `json:"KINSAMT" fieldType:"int | NULL" defaultValue:"0"`
	KMONTHS    int             `json:"KMONTHS" fieldType:"int | NULL" defaultValue:"0"`
	KINSREC    int             `json:"KINSREC" fieldType:"int | NULL" defaultValue:"0"`
	KSDATE     time.Time       `json:"KSDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	KEDATE     time.Time       `json:"KEDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	REFGROUP   string          `json:"REFGROUP" fieldType:"string | NULL"`
	STAMPID    int             `json:"STAMPID" fieldType:"int | NULL" defaultValue:"0"`
	PAYDATE    time.Time       `json:"PAYDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	BANKCHG    float64         `json:"BANKCHG" fieldType:"float | NULL" defaultValue:"0.00"`
	BRPATH     string          `json:"BRPATH" fieldType:"string | NULL"`
	ACNO12     int             `json:"ACNO12" fieldType:"int | NULL" defaultValue:"0"`
	REFBY      string          `json:"REFBY" fieldType:"string | NULL"`
	KPMONTH    int             `json:"KPMONTH" fieldType:"int | NULL" defaultValue:"0"`
	SCODE      string          `json:"SCODE" fieldType:"string | NULL"`
	KREFMNO    string          `json:"KREFMNO" fieldType:"string | NULL"`
	BANKBADLA  float64         `json:"BANKBADLA" fieldType:"float | NULL" defaultValue:"0.00"`
	BANKEX     float64         `json:"BANKEX" fieldType:"float | NULL" defaultValue:"0.00"`
	METAL      string          `json:"METAL" fieldType:"string | NULL"`
	LOGINID    int             `json:"LOGINID" fieldType:"int | NULL" defaultValue:"0"`
	DTIME      time.Time       `json:"DTIME" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	BALLOW     int             `json:"BALLOW" fieldType:"int | NULL" defaultValue:"0"`
	VTDS       int             `json:"VTDS" fieldType:"int | NULL" defaultValue:"0"`
	PVOU       string          `json:"PVOU" fieldType:"string | NULL"`
	INTR       float64         `json:"INTR" fieldType:"float | NULL" defaultValue:"0.00"`
	COMINT     float64         `json:"COMINT" fieldType:"float | NULL" defaultValue:"0.00"`
	COMINTDEF  int             `json:"COMINTDEF" fieldType:"int | NULL" defaultValue:"0"`
	GRTRAN     int             `json:"GRTRAN" fieldType:"int | NULL" defaultValue:"0"`
	MINT       int             `json:"MINT" fieldType:"int | NULL" defaultValue:"0"`
	HOLIDAY    int             `json:"HOLIDAY" fieldType:"int | NULL" defaultValue:"0"`
	DIR        string          `json:"DIR" fieldType:"string | NULL"`
	KHOME      int             `json:"KHOME" fieldType:"int | NULL" defaultValue:"0"`
	ACNARR     string          `json:"ACNARR" fieldType:"string | NULL"`
	DOC        string          `json:"DOC" fieldType:"string | NULL"`
	BKTRC      int             `json:"BKTRC" fieldType:"int | NULL" defaultValue:"0"`
	ORGACNO    int             `json:"ORGACNO" fieldType:"int | NULL" defaultValue:"0"`
	COMPATH    string          `json:"COMPATH" fieldType:"string | NULL"`
	COMPNO     string          `json:"COMPNO" fieldType:"string | NULL"`
	VCOM       float64         `json:"VCOM" fieldType:"float | NULL" defaultValue:"0.00"`
	SITEID     int             `json:"SITEID" fieldType:"int | NULL" defaultValue:"0"`
	MCURRID    int             `json:"MCURRID" fieldType:"int | NULL" defaultValue:"0"`
	MCXCOMEX   int             `json:"MCXCOMEX" fieldType:"int | NULL" defaultValue:"0"`
	RSCOMM     float64         `json:"RSCOMM" fieldType:"float | NULL" defaultValue:"0.00"`
	REMARKS    string          `json:"REMARKS" fieldType:"string | NULL"`
	AGENTACNO  int             `json:"AGENTACNO" fieldType:"int | NULL" defaultValue:"0"`
	Hpname     string          `json:"hpname" fieldType:"string | NULL"`
	ODLIMIT    float64         `json:"ODLIMIT" fieldType:"float | NULL" defaultValue:"0.00"`
	DEP2       float64         `json:"DEP2" fieldType:"float | NULL" defaultValue:"0.00"`
	OCCUPATION string          `json:"OCCUPATION" fieldType:"string | NULL"`
	Sflag      mysqldb.BitBool `json:"sflag" fieldType:"bool | NULL"`
	BANKCHG2   float64         `json:"BANKCHG2" fieldType:"float | NULL" defaultValue:"0.00"`
	KTACNO     int             `json:"KTACNO" fieldType:"int | NULL" defaultValue:"0"`
	SCBONY     string          `json:"SCBONY" fieldType:"string | NULL"`
	BANKNO     string          `json:"BANKNO" fieldType:"string | NULL"`
	SWCODE     string          `json:"SWCODE" fieldType:"string | NULL"`
	NOSTRO     string          `json:"NOSTRO" fieldType:"string | NULL"`
	MINLBR     float64         `json:"MINLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	SALARYB    float64         `json:"SALARYB" fieldType:"float | NULL" defaultValue:"0.00"`
	PRNROW     int             `json:"PRNROW" fieldType:"int | NULL" defaultValue:"0"`
	MAXVTGNO   int             `json:"MAXVTGNO" fieldType:"int | NULL" defaultValue:"0"`
	MAIN       int             `json:"MAIN" fieldType:"int | NULL" defaultValue:"0"`
	ADD4       string          `json:"ADD4" fieldType:"string | NULL"`
	OTHCHG     float64         `json:"OTHCHG" fieldType:"float | NULL" defaultValue:"0.00"`
	GENDER     int             `json:"GENDER" fieldType:"int | NULL" defaultValue:"0"`
	SPOUSE     string          `json:"SPOUSE" fieldType:"string | NULL"`
	SPOUSEDOB  time.Time       `json:"SPOUSEDOB" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	MARRIED    int             `json:"MARRIED" fieldType:"int | NULL" defaultValue:"0"`
	EDUCATION  string          `json:"EDUCATION" fieldType:"string | NULL"`
	KNOWLEDGE  string          `json:"KNOWLEDGE" fieldType:"string | NULL"`
	POTENTIAL  int             `json:"POTENTIAL" fieldType:"int | NULL" defaultValue:"0"`
	OPPOINTS   int             `json:"OPPOINTS" fieldType:"int | NULL" defaultValue:"0"`
	WEBSITE    string          `json:"WEBSITE" fieldType:"string | NULL"`
	CSTNO      string          `json:"CSTNO" fieldType:"string | NULL"`
	RBOOKDAY   int             `json:"RBOOKDAY" fieldType:"int | NULL" defaultValue:"0"`
	KINSWT     float64         `json:"KINSWT" fieldType:"float | NULL" defaultValue:"0.00"`
	BRANCHID   int             `json:"BRANCHID" fieldType:"int | NULL" defaultValue:"0"`
	UIDNO      string          `json:"UIDNO" fieldType:"string | NULL"`
	WRKHRS     float64         `json:"WRKHRS" fieldType:"float | NULL" defaultValue:"0.00"`
	LOTTBAL    int             `json:"LOTTBAL" fieldType:"int | NULL" defaultValue:"0"`
	OTHFINE    float64         `json:"OTHFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	GSTIN      string          `json:"GSTIN" fieldType:"string | NULL"`
	STCODE     string          `json:"STCODE" fieldType:"string | NULL"`
	STATEID    int             `json:"STATEID" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	GST        float64         `json:"GST" fieldType:"float | NULL" defaultValue:"0.00"`
	HSNCODE    string          `json:"HSNCODE" fieldType:"string | NULL"`
	COMPS      int             `json:"COMPS" fieldType:"int | NULL" defaultValue:"0"`
	ECOM       int             `json:"ECOM" fieldType:"int | NULL" defaultValue:"0"`
	EXPTYPE    string          `json:"EXPTYPE" fieldType:"string | NULL"`
	OPLOAN     float64         `json:"OPLOAN" fieldType:"float | NULL" defaultValue:"0.00"`
	PFSALARY   float64         `json:"PFSALARY" fieldType:"float | NULL" defaultValue:"0.00"`
	Opwithdraw float64         `json:"opwithdraw" fieldType:"float | NULL" defaultValue:"0.00"`
	Salins     string          `json:"salins" fieldType:"string | NULL"`
	Cdc        int             `json:"cdc" fieldType:"int | NULL" defaultValue:"0"`
	Netbank    int             `json:"netbank" fieldType:"int | NULL" defaultValue:"0"`
	Userid     string          `json:"userid" fieldType:"string | NULL"`
	Corpid     string          `json:"corpid" fieldType:"string | NULL"`
	Accno      string          `json:"accno" fieldType:"string | NULL"`
	Ifsc       string          `json:"ifsc" fieldType:"string | NULL"`
	Urnid      string          `json:"urnid" fieldType:"string | NULL"`
	Bnkstatus  string          `json:"bnkstatus" fieldType:"string | NULL"`
	Bnkalias   string          `json:"bnkalias" fieldType:"string | NULL"`
	Sysid      string          `json:"sysid" fieldType:"string | NULL"`
	Impcol     string          `json:"impcol" fieldType:"string | NULL"`
	Spemail    string          `json:"spemail" fieldType:"string | NULL"`
	Spmobile   string          `json:"spmobile" fieldType:"string | NULL"`
	Upd        int             `json:"upd" fieldType:"int | NULL" defaultValue:"0"`
	Hisab      int             `json:"hisab" fieldType:"int | NULL" defaultValue:"0"`
	Salarydet  string          `json:"salarydet" fieldType:"string | NULL"`
	Stcatid    int             `json:"stcatid" fieldType:"int | NULL" defaultValue:"0"`
	Tcs        float64         `json:"tcs" fieldType:"float | NULL" defaultValue:"0.00"`
	Hmno       string          `json:"hmno" fieldType:"string | NULL"`
	Wstgon     int             `json:"wstgon" fieldType:"int | NULL" defaultValue:"0"`
	Dtswid     string          `json:"dtswid" fieldType:"string | NULL"`
	Msme       string          `json:"msme" fieldType:"string | NULL"`
}
