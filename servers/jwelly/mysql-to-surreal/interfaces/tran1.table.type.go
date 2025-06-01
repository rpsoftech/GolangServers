package mysql_to_surreal_interfaces

import (
	"time"

	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type Tran1Struct struct {
	SurrealId       int             `json:"id" Index:"U"`
	TRNID           int             `json:"TRNID" fieldType:"int" Index:"U"`
	TNO             int             `json:"TNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	SurrealACNO     models.RecordID `json:"SurrealACNO" Index:"I" fieldType:"record<accmast>"`
	ACNO            int             `json:"ACNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	TDATE           time.Time       `json:"TDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000" Index:"I"`
	VTYPEID         int             `json:"VTYPEID" fieldType:"int | NULL" defaultValie:"0"`
	VONO            int             `json:"VONO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	FINE1           float64         `json:"FINE1" fieldType:"float | NULL" defaultValue:"0.00"`
	FINE2           float64         `json:"FINE2" fieldType:"float | NULL" defaultValue:"0.00"`
	AMOUNT          float64         `json:"AMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	LNARR           string          `json:"LNARR" fieldType:"string | NULL"`
	NARR            string          `json:"NARR" fieldType:"string | NULL"`
	BILLNO          int             `json:"BILLNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	REFNO           string          `json:"REFNO" fieldType:"string | NULL" Index:"I"`
	SERIESID        int             `json:"SERIESID" fieldType:"int | NULL" defaultValie:"0"`
	NOSERIESID      int             `json:"NOSERIESID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	SAMOUNT         float64         `json:"SAMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	TAX             float64         `json:"TAX" fieldType:"float | NULL" defaultValue:"0.00"`
	DISCOUNT        float64         `json:"DISCOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	ADJUST          float64         `json:"ADJUST" fieldType:"float | NULL" defaultValue:"0.00"`
	DIFF            float64         `json:"DIFF" fieldType:"float | NULL" defaultValue:"0.00"`
	SurrealSITEID   models.RecordID `json:"SurrealSITEID" Index:"I" fieldType:"record<site>"`
	SITEID          int             `json:"SITEID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	SurrealSITEIDTO models.RecordID `json:"SurrealSITEIDTO" Index:"I" fieldType:"record<site>"`
	SITEIDTO        int             `json:"SITEIDTO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	ACCOUNT         string          `json:"ACCOUNT" fieldType:"string | NULL"`
	SMACNO          int             `json:"SMACNO" fieldType:"int | NULL" defaultValie:"0"`
	Op1             string          `json:"op1" fieldType:"string | NULL"`
	Op2             string          `json:"op2" fieldType:"string | NULL"`
	Op3             string          `json:"op3" fieldType:"string | NULL"`
	Op4             string          `json:"op4" fieldType:"string | NULL"`
	Op5             string          `json:"op5" fieldType:"string | NULL"`
	Op6             string          `json:"op6" fieldType:"string | NULL"`
	NO12            int             `json:"NO12" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	RNAMEID         int             `json:"RNAMEID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	GROSSWT1        float64         `json:"GROSSWT1" fieldType:"float | NULL" defaultValue:"0.00"`
	GROSSWT2        float64         `json:"GROSSWT2" fieldType:"float | NULL" defaultValue:"0.00"`
	Cpname          string          `json:"cpname" fieldType:"string | NULL"`
	Cpadd1          string          `json:"cpadd1" fieldType:"string | NULL"`
	Cpadd2          string          `json:"cpadd2" fieldType:"string | NULL"`
	Cpadd3          string          `json:"cpadd3" fieldType:"string | NULL"`
	CPLOCATION      string          `json:"CPLOCATION" fieldType:"string | NULL"`
	CPCITY          string          `json:"CPCITY" fieldType:"string | NULL"`
	CPPHONE         string          `json:"CPPHONE" fieldType:"string | NULL"`
	CPMOBILE        string          `json:"CPMOBILE" fieldType:"string | NULL"`
	CPDOB           time.Time       `json:"CPDOB" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	CPASARY         time.Time       `json:"CPASARY" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	CPID            int             `json:"CPID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	PTAX            float64         `json:"PTAX" fieldType:"float | NULL" defaultValue:"0.00"`
	SURCHARGE       float64         `json:"SURCHARGE" fieldType:"float | NULL" defaultValue:"0.00"`
	VTYPECODE       int             `json:"VTYPECODE" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	HISAB           int             `json:"HISAB" fieldType:"int | NULL" defaultValie:"0"`
	LOGINID         int             `json:"LOGINID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	APPROVED        int             `json:"APPROVED" fieldType:"int | NULL" defaultValie:"0"`
	VTCODE          int             `json:"VTCODE" fieldType:"int | NULL" defaultValie:"0"`
	DDATE           time.Time       `json:"DDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	STATUS          string          `json:"STATUS" fieldType:"string | NULL"`
	IDATE           time.Time       `json:"IDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	KITTYINS        int             `json:"KITTYINS" fieldType:"int | NULL" defaultValie:"0"`
	BHAVWT          float64         `json:"BHAVWT" fieldType:"float | NULL" defaultValue:"0.00"`
	BHAVTUNCH       float64         `json:"BHAVTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	BHAVRATE        float64         `json:"BHAVRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	BHAVRTUNCH      float64         `json:"BHAVRTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	DLESS           int             `json:"DLESS" fieldType:"int | NULL" defaultValie:"0"`
	SLESS           int             `json:"SLESS" fieldType:"int | NULL" defaultValie:"0"`
	LBRON           int             `json:"LBRON" fieldType:"int | NULL" defaultValie:"0"`
	HISABVONO       int             `json:"HISABVONO" fieldType:"int | NULL" defaultValie:"0"`
	VTYPE           string          `json:"VTYPE" fieldType:"string | NULL" Index:"I"`
	MEMID           string          `json:"MEMID" fieldType:"string | NULL" Index:"I"`
	DTIME           time.Time       `json:"DTIME" fieldType:"datetime | NULL"`
	GROSSFINE1      float64         `json:"GROSSFINE1" fieldType:"float | NULL" defaultValue:"0.00"`
	GROSSFINE2      float64         `json:"GROSSFINE2" fieldType:"float | NULL" defaultValue:"0.00"`
	BADLARATE       float64         `json:"BADLARATE" fieldType:"float | NULL" defaultValue:"0.00"`
	BADLA           float64         `json:"BADLA" fieldType:"float | NULL" defaultValue:"0.00"`
	BADLAON         string          `json:"BADLAON" fieldType:"string | NULL"`
	BADLANJ         int             `json:"BADLANJ" fieldType:"int | NULL" defaultValie:"0"`
	FINAL           int             `json:"FINAL" fieldType:"int | NULL" defaultValie:"0"`
	DELIVER         int             `json:"DELIVER" fieldType:"int | NULL" defaultValie:"0"`
	RMODE           string          `json:"RMODE" fieldType:"string | NULL"`
	MAINACNO        int             `json:"MAINACNO" fieldType:"int | NULL" defaultValie:"0"`
	LOTTID          int             `json:"LOTTID" fieldType:"int | NULL" defaultValie:"0"`
	ORDID           int             `json:"ORDID" fieldType:"int | NULL" defaultValie:"0"`
	STAMPID         int             `json:"STAMPID" fieldType:"int | NULL" defaultValie:"0"`
	Issgwt          float64         `json:"issgwt" fieldType:"float | NULL" defaultValue:"0.00"`
	Isswt           float64         `json:"isswt" fieldType:"float | NULL" defaultValue:"0.00"`
	Recgwt          float64         `json:"recgwt" fieldType:"float | NULL" defaultValue:"0.00"`
	Recwt           float64         `json:"recwt" fieldType:"float | NULL" defaultValue:"0.00"`
	Losswt          float64         `json:"losswt" fieldType:"float | NULL" defaultValue:"0.00"`
	Lossfine        float64         `json:"lossfine" fieldType:"float | NULL" defaultValue:"0.00"`
	Balwt           float64         `json:"balwt" fieldType:"float | NULL" defaultValue:"0.00"`
	Balfine         float64         `json:"balfine" fieldType:"float | NULL" defaultValue:"0.00"`
	EXWT            float64         `json:"EXWT" fieldType:"float | NULL" defaultValue:"0.00"`
	EXFINE          float64         `json:"EXFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	Lbramt          float64         `json:"lbramt" fieldType:"float | NULL" defaultValue:"0.00"`
	KHISAB          int             `json:"KHISAB" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	Stnwt           float64         `json:"stnwt" fieldType:"float | NULL" defaultValue:"0.00"`
	STNPC           int             `json:"STNPC" fieldType:"int | NULL" defaultValie:"0"`
	Diawt           float64         `json:"diawt" fieldType:"float | NULL" defaultValue:"0.00"`
	DIAPC           int             `json:"DIAPC" fieldType:"int | NULL" defaultValie:"0"`
	STBEEDS         float64         `json:"STBEEDS" fieldType:"float | NULL" defaultValue:"0.00"`
	ALWSTG          float64         `json:"ALWSTG" fieldType:"float | NULL" defaultValue:"0.00"`
	ALFINE          float64         `json:"ALFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	DUSTWT          float64         `json:"DUSTWT" fieldType:"float | NULL" defaultValue:"0.00"`
	TUNCHWT         float64         `json:"TUNCHWT" fieldType:"float | NULL" defaultValue:"0.00"`
	ISSPC           int             `json:"ISSPC" fieldType:"int | NULL" defaultValie:"0"`
	RECPC           int             `json:"RECPC" fieldType:"int | NULL" defaultValie:"0"`
	AVGTUNCH        float64         `json:"AVGTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	LOSSDATE        time.Time       `json:"LOSSDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	WAXDIAWT        float64         `json:"WAXDIAWT" fieldType:"float | NULL" defaultValue:"0.00"`
	WAXSTWT         float64         `json:"WAXSTWT" fieldType:"float | NULL" defaultValue:"0.00"`
	STFINE          float64         `json:"STFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	EXTUNCH         float64         `json:"EXTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	MELTUNCH        float64         `json:"MELTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	Melwt           float64         `json:"melwt" fieldType:"float | NULL" defaultValue:"0.00"`
	KADIWT          float64         `json:"KADIWT" fieldType:"float | NULL" defaultValue:"0.00"`
	KPCWT           float64         `json:"KPCWT" fieldType:"float | NULL" defaultValue:"0.00"`
	KADIPC          int             `json:"KADIPC" fieldType:"int | NULL" defaultValie:"0"`
	Konewt          float64         `json:"konewt" fieldType:"float | NULL" defaultValue:"0.00"`
	BDSCALC         int             `json:"BDSCALC" fieldType:"int | NULL" defaultValie:"0"`
	STSTNWT         float64         `json:"STSTNWT" fieldType:"float | NULL" defaultValue:"0.00"`
	STSTNPC         int             `json:"STSTNPC" fieldType:"int | NULL" defaultValie:"0"`
	Flag            mysqldb.BitBool `json:"flag" fieldType:"bool | NULL" defaultValue:"false"`
	RATEFIX         int             `json:"RATEFIX" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	Crdays          float64         `json:"crdays" fieldType:"float | NULL" defaultValue:"0.00"`
	RPAMOUNT        float64         `json:"RPAMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	QTTNO           int             `json:"QTTNO" fieldType:"int | NULL" defaultValie:"0"`
	INVDATE         time.Time       `json:"INVDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	EBILL           int             `json:"EBILL" fieldType:"int | NULL" defaultValie:"0"`
	AUDIT           int             `json:"AUDIT" fieldType:"int | NULL" defaultValie:"0"`
	CHQID           int             `json:"CHQID" fieldType:"int | NULL" defaultValie:"0"`
	BBID            int             `json:"BBID" fieldType:"int | NULL" defaultValie:"0"`
	BILLTYPE        int             `json:"BILLTYPE" fieldType:"int | NULL" defaultValie:"0"`
	DAILY           int             `json:"DAILY" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	Altvou          int             `json:"altvou" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	MHISAB          int             `json:"MHISAB" fieldType:"int | NULL" defaultValie:"0"`
	BILLFINE1       float64         `json:"BILLFINE1" fieldType:"float | NULL" defaultValue:"0.00"`
	BILLFINE2       float64         `json:"BILLFINE2" fieldType:"float | NULL" defaultValue:"0.00"`
	BILLBAL         float64         `json:"BILLBAL" fieldType:"float | NULL" defaultValue:"0.00"`
	ADJFINE1        float64         `json:"ADJFINE1" fieldType:"float | NULL" defaultValue:"0.00"`
	ADJFINE2        float64         `json:"ADJFINE2" fieldType:"float | NULL" defaultValue:"0.00"`
	ADJBAL          float64         `json:"ADJBAL" fieldType:"float | NULL" defaultValue:"0.00"`
	TGINO           int             `json:"TGINO" fieldType:"int | NULL" defaultValie:"0"`
	TGIGROUPID      int             `json:"TGIGROUPID" fieldType:"int | NULL" defaultValie:"0"`
	TGPTUNCH        float64         `json:"TGPTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	TGPWSTG         float64         `json:"TGPWSTG" fieldType:"float | NULL" defaultValue:"0.00"`
	TGRATE          float64         `json:"TGRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	TGMRATE         float64         `json:"TGMRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	TGSPOLISH       float64         `json:"TGSPOLISH" fieldType:"float | NULL" defaultValue:"0.00"`
	TGSTUNCH        float64         `json:"TGSTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	TGLBR           float64         `json:"TGLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	TGSLBR          float64         `json:"TGSLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	TGSTKINO        int             `json:"TGSTKINO" fieldType:"int | NULL" defaultValie:"0"`
	TGSTAMPID       int             `json:"TGSTAMPID" fieldType:"int | NULL" defaultValie:"0"`
	TGPCLESS        int             `json:"TGPCLESS" fieldType:"int | NULL" defaultValie:"0"`
	TGDSLESS        int             `json:"TGDSLESS" fieldType:"int | NULL" defaultValie:"0"`
	TGVALLESS       int             `json:"TGVALLESS" fieldType:"int | NULL" defaultValie:"0"`
	PAMOUNT         float64         `json:"PAMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	UPFLG           int             `json:"UPFLG" fieldType:"int | NULL" defaultValie:"0"`
	TRDATA          int             `json:"TRDATA" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	MAINRECORD      int             `json:"MAINRECORD" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	PDC             int             `json:"PDC" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	INSFLG          int             `json:"INSFLG" fieldType:"int | NULL" defaultValie:"0"`
	JOBID           int             `json:"JOBID" fieldType:"int | NULL" defaultValie:"0"`
	UPHISAB         int             `json:"UPHISAB" fieldType:"int | NULL" defaultValie:"0"`
	OACNO           int             `json:"OACNO" fieldType:"int | NULL" defaultValie:"0"`
	JOBNO           string          `json:"JOBNO" fieldType:"string | NULL"`
	OJOBNO          string          `json:"OJOBNO" fieldType:"string | NULL"`
	OJOBID          int             `json:"OJOBID" fieldType:"int | NULL" defaultValie:"0"`
	DESIGNID        int             `json:"DESIGNID" fieldType:"int | NULL" defaultValie:"0"`
	Issfine         float64         `json:"issfine" fieldType:"float | NULL" defaultValue:"0.00"`
	Recfine         float64         `json:"recfine" fieldType:"float | NULL" defaultValue:"0.00"`
	POLOSSWT        float64         `json:"POLOSSWT" fieldType:"float | NULL" defaultValue:"0.00"`
	POLOSSFINE      float64         `json:"POLOSSFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	POSTNFINE       float64         `json:"POSTNFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	BULLID          int             `json:"BULLID" fieldType:"int | NULL" defaultValie:"0"`
	TGPPOLISH       float64         `json:"TGPPOLISH" fieldType:"float | NULL" defaultValue:"0.00"`
	MRNAMEID        int             `json:"MRNAMEID" fieldType:"int | NULL" defaultValie:"0"`
	EXPTAX          float64         `json:"EXPTAX" fieldType:"float | NULL" defaultValue:"0.00"`
	ACNO12ACNO      int             `json:"ACNO12ACNO" fieldType:"int | NULL" defaultValie:"0"`
	EXRATEOK        int             `json:"EXRATEOK" fieldType:"int | NULL" defaultValie:"0"`
	CDCHARGE        float64         `json:"CDCHARGE" fieldType:"float | NULL" defaultValue:"0.00"`
	STAX            float64         `json:"STAX" fieldType:"float | NULL" defaultValue:"0.00"`
	DOLLAR          float64         `json:"DOLLAR" fieldType:"float | NULL" defaultValue:"0.00"`
	METALTYPE       string          `json:"METALTYPE" fieldType:"string | NULL"`
	CHQTRNID        int             `json:"CHQTRNID" fieldType:"int | NULL" defaultValie:"0"`
	INTHISAB        int             `json:"INTHISAB" fieldType:"int | NULL" defaultValie:"0"`
	RTUNCH          float64         `json:"RTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	WTUNCH          float64         `json:"WTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	BANKCHG         float64         `json:"BANKCHG" fieldType:"float | NULL" defaultValue:"0.00"`
	POSTAMT         float64         `json:"POSTAMT" fieldType:"float | NULL" defaultValue:"0.00"`
	POSTRATE        float64         `json:"POSTRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	POSTLBR         float64         `json:"POSTLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	SUMNO           int             `json:"SUMNO" fieldType:"int | NULL" defaultValie:"0"`
	SPOLISHWT       float64         `json:"SPOLISHWT" fieldType:"float | NULL" defaultValue:"0.00"`
	DUSTFINE        float64         `json:"DUSTFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	ORGACNO         int             `json:"ORGACNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	SurrealINO      models.RecordID `json:"SurrealINO" fieldType:"record<itemmast>" Index:"I"`
	INO             int             `json:"INO" fieldType:"int | NULL" defaultValie:"0"`
	TRTRNID         int             `json:"TRTRNID" fieldType:"int | NULL" defaultValie:"0"`
	TGSET           string          `json:"TGSET" fieldType:"string | NULL"`
	ORGCNAME        string          `json:"ORGCNAME" fieldType:"string | NULL"`
	LEDWT           float64         `json:"LEDWT" fieldType:"float | NULL" defaultValue:"0.00"`
	OPENING         int             `json:"OPENING" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	MACNO           int             `json:"MACNO" fieldType:"int | NULL" defaultValie:"0"`
	Tgbatch         string          `json:"tgbatch" fieldType:"string | NULL"`
	TUNCHFINE       float64         `json:"TUNCHFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	KITTYVONO       int             `json:"KITTYVONO" fieldType:"int | NULL" defaultValie:"0"`
	POSTLFINE       float64         `json:"POSTLFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	MCURRID         int             `json:"MCURRID" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	MCAMOUNT        float64         `json:"MCAMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	Xrate           float64         `json:"xrate" fieldType:"float | NULL" defaultValue:"0.00"`
	MCXCNO          int             `json:"MCXCNO" fieldType:"int | NULL" defaultValie:"0"`
	APPCODE         string          `json:"APPCODE" fieldType:"string | NULL"`
	CARDTYPE        string          `json:"CARDTYPE" fieldType:"string | NULL"`
	Billnarr        string          `json:"billnarr" fieldType:"string | NULL"`
	BADLAID         int             `json:"BADLAID" fieldType:"int | NULL" defaultValie:"0"`
	MCAMOUNT2       float64         `json:"MCAMOUNT2" fieldType:"float | NULL" defaultValue:"0.00"`
	MCURRID1        int             `json:"MCURRID1" fieldType:"int | NULL" defaultValie:"0"`
	MCURRID2        int             `json:"MCURRID2" fieldType:"int | NULL" defaultValie:"0"`
	ALLOT           int             `json:"ALLOT" fieldType:"int | NULL" defaultValie:"0"`
	FINE3           float64         `json:"FINE3" fieldType:"float | NULL" defaultValue:"0.00"`
	FINE4           float64         `json:"FINE4" fieldType:"float | NULL" defaultValue:"0.00"`
	OP7             int             `json:"OP7" fieldType:"int | NULL" defaultValie:"0"`
	UGAMOUNT        float64         `json:"UGAMOUNT" fieldType:"float | NULL" defaultValue:"0.00"`
	LEDGERFLG       int             `json:"LEDGERFLG" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	CNTACNO         int             `json:"CNTACNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	PRNFLAG         int             `json:"PRNFLAG" fieldType:"int | NULL" defaultValie:"0"`
	LABID           int             `json:"LABID" fieldType:"int | NULL" defaultValie:"0"`
	DMTNO           int             `json:"DMTNO" fieldType:"int | NULL" defaultValie:"0"`
	DMLOGINID       int             `json:"DMLOGINID" fieldType:"int | NULL" defaultValie:"0"`
	DMTIME          time.Time       `json:"DMTIME" fieldType:"datetime | NULL"`
	DEL             string          `json:"DEL" fieldType:"string | NULL"`
	COMACNO         int             `json:"COMACNO" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	COMAMT          float64         `json:"COMAMT" fieldType:"float | NULL" defaultValue:"0.00"`
	MPOINTS         int             `json:"MPOINTS" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	COUPONID        int             `json:"COUPONID" fieldType:"int | NULL" defaultValie:"0"`
	WEBID           int             `json:"WEBID" fieldType:"int | NULL" defaultValie:"0"`
	MTRANID         int             `json:"MTRANID" fieldType:"int | NULL" defaultValie:"0"`
	INTEREST        float64         `json:"INTEREST" fieldType:"float | NULL" defaultValue:"0.00"`
	Notecntr        string          `json:"notecntr" fieldType:"string | NULL"`
	DUEDATE         time.Time       `json:"DUEDATE" fieldType:"datetime | NULL" defaultValue:"00-00-000"`
	EXCISE          float64         `json:"EXCISE" fieldType:"float | NULL" defaultValue:"0.00"`
	SCHGAMT         float64         `json:"SCHGAMT" fieldType:"float | NULL" defaultValue:"0.00"`
	INTHISABG       int             `json:"INTHISABG" fieldType:"int | NULL" defaultValie:"0"`
	INTHISABS       int             `json:"INTHISABS" fieldType:"int | NULL" defaultValie:"0"`
	CPPAN           string          `json:"CPPAN" fieldType:"string | NULL"`
	OTHFINE         float64         `json:"OTHFINE" fieldType:"float | NULL" defaultValue:"0.00"`
	CASHBANK        int             `json:"CASHBANK" fieldType:"int | NULL" defaultValie:"0" Index:"I"`
	CGST            float64         `json:"CGST" fieldType:"float | NULL" defaultValue:"0.00"`
	SGST            float64         `json:"SGST" fieldType:"float | NULL" defaultValue:"0.00"`
	IGST            float64         `json:"IGST" fieldType:"float | NULL" defaultValue:"0.00"`
	CPSTATEID       int             `json:"CPSTATEID" fieldType:"int | NULL" defaultValie:"0"`
	CPGSTIN         string          `json:"CPGSTIN" fieldType:"string | NULL"`
	Brid            int             `json:"brid" fieldType:"int | NULL" defaultValie:"0"`
	Irn             string          `json:"irn" fieldType:"string | NULL"`
	Billinfo        string          `json:"billinfo" fieldType:"string | NULL"`
	Itcstatus       string          `json:"itcstatus" fieldType:"string | NULL"`
}
