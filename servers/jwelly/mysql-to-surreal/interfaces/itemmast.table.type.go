package mysql_to_surreal_interfaces

import (
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type ItemMastTableStruct struct {
	SurrealId       int             `json:"id" Index:"U"`
	Ino             int             `json:"ino" fieldType:"int | NULL" Index:"U"`
	IDESC           string          `json:"IDESC" fieldType:"string | NULL" Index:"I"`
	DIAST           int             `json:"DIAST" fieldType:"int | NULL" defaultValue:"0"`
	SurrealIGROUPID models.RecordID `json:"SurrealIGROUPID"  fieldType:"record<igroup>"`
	IGROUPID        int             `json:"IGROUPID" fieldType:"int | NULL" Index:"I"`
	PNAME           string          `json:"PNAME" fieldType:"string | NULL"`
	SurrealUNITID   models.RecordID `json:"SurrealUNITID"  fieldType:"record<units>"`
	UNITID          int             `json:"UNITID" fieldType:"int | NULL" defaultValue:"0"`
	SM              int             `json:"SM" fieldType:"int | NULL" defaultValue:"0"`
	Tpre            string          `json:"tpre" fieldType:"string | NULL"`
	PC              int             `json:"PC" fieldType:"int | NULL" defaultValue:"0"`
	TREYPC          int             `json:"TREYPC" fieldType:"int | NULL" defaultValue:"0"`
	MAXTGNO         int             `json:"MAXTGNO" fieldType:"int | NULL" defaultValue:"0"`
	STARTTGNO       int             `json:"STARTTGNO" fieldType:"int | NULL" defaultValue:"0"`
	PCSTOCK         int             `json:"PCSTOCK" fieldType:"int | NULL" defaultValue:"0"`
	BLIST           int             `json:"BLIST" fieldType:"int | NULL" defaultValue:"0"`
	ADDWT           int             `json:"ADDWT" fieldType:"int | NULL" defaultValue:"0"`
	TWODET          int             `json:"TWODET" fieldType:"int | NULL" defaultValue:"0"`
	SALE            int             `json:"SALE" fieldType:"int | NULL" defaultValue:"0"`
	PURCHASE        int             `json:"PURCHASE" fieldType:"int | NULL" defaultValue:"0"`
	KARIGAR         int             `json:"KARIGAR" fieldType:"int | NULL" defaultValue:"0"`
	STOCK           int             `json:"STOCK" fieldType:"int | NULL" defaultValue:"0"`
	STVAL           int             `json:"STVAL" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	STUDED          int             `json:"STUDED" fieldType:"int | NULL" defaultValue:"0"`
	SurrealSTAMPID  models.RecordID `json:"SurrealSTAMPID" fieldType:"record<stamp>"`
	STAMPID         int             `json:"STAMPID" fieldType:"int | NULL" defaultValue:"0"`
	GCODEID         int             `json:"GCODEID" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	FINECAL         int             `json:"FINECAL" fieldType:"int | NULL" defaultValue:"0"`
	CSTAMP          int             `json:"CSTAMP" fieldType:"int | NULL" defaultValue:"0"`
	CBEEDS          int             `json:"CBEEDS" fieldType:"int | NULL" defaultValue:"0"`
	CGWT            int             `json:"CGWT" fieldType:"int | NULL" defaultValue:"0"`
	GRADE           string          `json:"GRADE" fieldType:"string | NULL"`
	SurrealCatID    models.RecordID `json:"SurrealCatID" fieldType:"record<category>"`
	CATID           int             `json:"CATID" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	RESTGWT         int             `json:"RESTGWT" fieldType:"int | NULL" defaultValue:"0"`
	RESTTUNCH       int             `json:"RESTTUNCH" fieldType:"int | NULL" defaultValue:"0"`
	RESTWSTG        int             `json:"RESTWSTG" fieldType:"int | NULL" defaultValue:"0"`
	Shortname       string          `json:"shortname" fieldType:"string | NULL"`
	BOXWT           float64         `json:"BOXWT" fieldType:"float | NULL" defaultValue:"0.00"`
	POLISH          float64         `json:"POLISH" fieldType:"float | NULL" defaultValue:"0.00"`
	PROFIT          float64         `json:"PROFIT" fieldType:"float | NULL" defaultValue:"0.00"`
	INTORATE        float64         `json:"INTORATE" fieldType:"float | NULL" defaultValue:"0.00"`
	DESIPRE         string          `json:"DESIPRE" fieldType:"string | NULL"`
	WSWSTG          float64         `json:"WSWSTG" fieldType:"float | NULL" defaultValue:"0.00"`
	WSLBR           float64         `json:"WSLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	RETWSTG         float64         `json:"RETWSTG" fieldType:"float | NULL" defaultValue:"0.00"`
	RETLBR          float64         `json:"RETLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	Lcode           string          `json:"lcode" fieldType:"string | NULL"`
	DEFTUNCH        float64         `json:"DEFTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	STONES          int             `json:"STONES" fieldType:"int | NULL" defaultValue:"0"`
	LINKTG          int             `json:"LINKTG" fieldType:"int | NULL" defaultValue:"0"`
	LINKINO         int             `json:"LINKINO" fieldType:"int | NULL" defaultValue:"0"`
	STMETHOD        int             `json:"STMETHOD" fieldType:"int | NULL" defaultValue:"0"`
	SRNO            int             `json:"SRNO" fieldType:"int | NULL" defaultValue:"0" Index:"I"`
	GCREATE         int             `json:"GCREATE" fieldType:"int | NULL" defaultValue:"0"`
	BPRETYPE        int             `json:"BPRETYPE" fieldType:"int | NULL" defaultValue:"0"`
	ISTUD           int             `json:"ISTUD" fieldType:"int | NULL" defaultValue:"0"`
	BADHEL          int             `json:"BADHEL" fieldType:"int | NULL" defaultValue:"0"`
	DITEMRATE       int             `json:"DITEMRATE" fieldType:"int | NULL" defaultValue:"0"`
	RESTRATE        int             `json:"RESTRATE" fieldType:"int | NULL" defaultValue:"0"`
	STVALTUNCH      float64         `json:"STVALTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	STVALWSTG       float64         `json:"STVALWSTG" fieldType:"float | NULL" defaultValue:"0.00"`
	STVALLBR        float64         `json:"STVALLBR" fieldType:"float | NULL" defaultValue:"0.00"`
	IRGWT           int             `json:"IRGWT" fieldType:"int | NULL" defaultValue:"0"`
	TPCODE          int             `json:"TPCODE" fieldType:"int | NULL" defaultValue:"0"`
	PARTYDST        int             `json:"PARTYDST" fieldType:"int | NULL" defaultValue:"0"`
	RESTZERO        int             `json:"RESTZERO" fieldType:"int | NULL" defaultValue:"0"`
	GCOMM           float64         `json:"GCOMM" fieldType:"float | NULL" defaultValue:"0.00"`
	GCON            string          `json:"GCON" fieldType:"string | NULL"`
	GCDIS           float64         `json:"GCDIS" fieldType:"float | NULL" defaultValue:"0.00"`
	SCOMM           float64         `json:"SCOMM" fieldType:"float | NULL" defaultValue:"0.00"`
	SCON            string          `json:"SCON" fieldType:"string | NULL"`
	SCDIS           float64         `json:"SCDIS" fieldType:"float | NULL" defaultValue:"0.00"`
	VSTK            string          `json:"VSTK" fieldType:"string | NULL"`
	TGDIGIT         int             `json:"TGDIGIT" fieldType:"int | NULL" defaultValue:"0"`
	DESIDIGIT       int             `json:"DESIDIGIT" fieldType:"int | NULL" defaultValue:"0"`
	REM1            string          `json:"REM1" fieldType:"string | NULL"`
	REM2            string          `json:"REM2" fieldType:"string | NULL"`
	LOGINID         int             `json:"LOGINID" fieldType:"int | NULL" defaultValue:"0"`
	DTIME           time.Time       `json:"DTIME" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	LBRON           int             `json:"LBRON" fieldType:"int | NULL" defaultValue:"0"`
	SLBRON          int             `json:"SLBRON" fieldType:"int | NULL" defaultValue:"0"`
	DBNAME          string          `json:"DBNAME" fieldType:"string | NULL"`
	DESIGN          string          `json:"DESIGN" fieldType:"string | NULL"`
	MRP             int             `json:"MRP" fieldType:"int | NULL" defaultValue:"0"`
	RAPARATE        string          `json:"RAPARATE" fieldType:"string | NULL"`
	SALELESS        string          `json:"SALELESS" fieldType:"string | NULL"`
	RESTSTAMP       int             `json:"RESTSTAMP" fieldType:"int | NULL" defaultValue:"0"`
	BOXPC           int             `json:"BOXPC" fieldType:"int | NULL" defaultValue:"0"`
	PCRATE          float64         `json:"PCRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	RESTBATCH       int             `json:"RESTBATCH" fieldType:"int | NULL" defaultValue:"0"`
	Sqrmtr          float64         `json:"sqrmtr" fieldType:"float | NULL" defaultValue:"0.00"`
	PEXWT           float64         `json:"PEXWT" fieldType:"float | NULL" defaultValue:"0.00"`
	RESTOTH         int             `json:"RESTOTH" fieldType:"int | NULL" defaultValue:"0"`
	TAGWT           float64         `json:"TAGWT" fieldType:"float | NULL" defaultValue:"0.00"`
	HPNAME          string          `json:"HPNAME" fieldType:"string | NULL"`
	RESTWL          int             `json:"RESTWL" fieldType:"int | NULL" defaultValue:"0"`
	SHORTCNM        string          `json:"SHORTCNM" fieldType:"string | NULL"`
	RESTPC          int             `json:"RESTPC" fieldType:"int | NULL" defaultValue:"0"`
	SCOWT           float64         `json:"SCOWT" fieldType:"float | NULL" defaultValue:"0.00"`
	SCORS           float64         `json:"SCORS" fieldType:"float | NULL" defaultValue:"0.00"`
	BINTORATE       float64         `json:"BINTORATE" fieldType:"float | NULL" defaultValue:"0.00"`
	POLYWT          float64         `json:"POLYWT" fieldType:"float | NULL" defaultValue:"0.00"`
	WIREWT          float64         `json:"WIREWT" fieldType:"float | NULL" defaultValue:"0.00"`
	Lstamp          string          `json:"lstamp" fieldType:"string | NULL"`
	PRNCMD          string          `json:"PRNCMD" fieldType:"string | NULL"`
	MPOINTS         float64         `json:"MPOINTS" fieldType:"float | NULL" defaultValue:"0.00"`
	PRNSNO          int             `json:"PRNSNO" fieldType:"int | NULL" defaultValue:"0"`
	TGSTKLESS       int             `json:"TGSTKLESS" fieldType:"int | NULL" defaultValue:"0"`
	DTRNSTK         string          `json:"DTRNSTK" fieldType:"string | NULL"`
	LESSDET         int             `json:"LESSDET" fieldType:"int | NULL" defaultValue:"0"`
	MINO            int             `json:"MINO" fieldType:"int | NULL" defaultValue:"0"`
	MSIZEID         int             `json:"MSIZEID" fieldType:"int | NULL" defaultValue:"0"`
	POINTUPON       string          `json:"POINTUPON" fieldType:"string | NULL"`
	TGLBRADD        float64         `json:"TGLBRADD" fieldType:"float | NULL" defaultValue:"0.00"`
	ITHSNCODE       string          `json:"ITHSNCODE" fieldType:"string | NULL"`
	SRATEUNIT       int             `json:"SRATEUNIT" fieldType:"int | NULL" defaultValue:"0"`
	SVSTK           int             `json:"SVSTK" fieldType:"int | NULL" defaultValue:"0"`
	Minstk          float64         `json:"minstk" fieldType:"float | NULL" defaultValue:"0.00"`
	Insamt          float64         `json:"insamt" fieldType:"float | NULL" defaultValue:"0.00"`
	Webdis          string          `json:"webdis" fieldType:"string | NULL"`
	Webidesc        string          `json:"webidesc" fieldType:"string | NULL"`
	Webcat          string          `json:"webcat" fieldType:"string | NULL"`
	Webconf         string          `json:"webconf" fieldType:"string | NULL"`
	Barslist        int             `json:"barslist" fieldType:"int | NULL" defaultValue:"0"`
	Chgitm          int             `json:"chgitm" fieldType:"int | NULL" defaultValue:"0"`
	Itmconf         string          `json:"itmconf" fieldType:"string | NULL"`
	Snobcode        string          `json:"snobcode" fieldType:"string | NULL"`
	Minmg           string          `json:"minmg" fieldType:"string | NULL"`
}
