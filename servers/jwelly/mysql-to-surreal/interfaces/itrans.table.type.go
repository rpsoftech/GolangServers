package mysql_to_surreal_interfaces

import (
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type ITransStruct struct {
	SurrealId      int             `json:"id" Index:"U" fieldType:"int"`
	ITRNID         int             `json:"ITRNID" fieldType:"int | NULL" Index:"U" defaultValue:"0"`
	TRNID          int             `json:"TRNID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	VONO           int             `json:"VONO" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	TNO            int             `json:"TNO" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	ITSTDID        int             `json:"ITSTDID" fieldType:"int | NULL" defaultValue:"0"`
	SNTRNID        int             `json:"SNTRNID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	TDATE          time.Time       `json:"TDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"  Index:"I"`
	INO            int             `json:"INO" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	REMARKS        string          `json:"REMARKS" fieldType:"string | NULL"`
	GWT            float64         `json:"GWT" fieldType:"float | NULL" defaultValue:"0.0"`
	WT             float64         `json:"WT" fieldType:"float | NULL" defaultValue:"0.0"`
	LESSWT         float64         `json:"LESSWT" fieldType:"float | NULL" defaultValue:"0.0"`
	PC             int             `json:"PC" fieldType:"int | NULL" defaultValue:"0"`
	Rate           float64         `json:"rate" fieldType:"float | NULL" defaultValue:"0.0"`
	TUNCH          float64         `json:"TUNCH" fieldType:"float | NULL" defaultValue:"0.0"`
	WSTG           float64         `json:"WSTG" fieldType:"float | NULL" defaultValue:"0.0"`
	MAMT           float64         `json:"MAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	TYPE           string          `json:"TYPE" fieldType:"string | NULL"  Index:"I"`
	STOCK          string          `json:"STOCK" fieldType:"string | NULL"  Index:"I"`
	SurrealUNITID  models.RecordID `json:"SurrealUNITID"  fieldType:"record<units>"`
	UNITID         int             `json:"UNITID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	SurrealSTAMPID models.RecordID `json:"SurrealSTAMPID" fieldType:"record<stamp>"`
	STAMPID        int             `json:"STAMPID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	SurrealSITEID  models.RecordID `json:"SurrealSITEID" Index:"I" fieldType:"record<site>"`
	SITEID         int             `json:"SITEID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	TVALUE         float64         `json:"TVALUE" fieldType:"float | NULL" defaultValue:"0.0"`
	STUDED         string          `json:"STUDED" fieldType:"string | NULL"  Index:"I"`
	DAMT           float64         `json:"DAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	SAMT           float64         `json:"SAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	LAMT           float64         `json:"LAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	BAMT           float64         `json:"BAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	OTHERS         float64         `json:"OTHERS" fieldType:"float | NULL" defaultValue:"0.0"`
	MRP            float64         `json:"MRP" fieldType:"float | NULL" defaultValue:"0.0"`
	LBR            float64         `json:"LBR" fieldType:"float | NULL" defaultValue:"0.0"`
	SLBR           float64         `json:"SLBR" fieldType:"float | NULL" defaultValue:"0.0"`
	SLBR2          float64         `json:"SLBR2" fieldType:"float | NULL" defaultValue:"0.0"`
	SLBR3          float64         `json:"SLBR3" fieldType:"float | NULL" defaultValue:"0.0"`
	LCODE          string          `json:"LCODE" fieldType:"string | NULL"`
	PTAX           float64         `json:"PTAX" fieldType:"float | NULL" defaultValue:"0.0"`
	Design         string          `json:"design" fieldType:"string | NULL"`
	COLOUR         string          `json:"COLOUR" fieldType:"string | NULL"`
	CLARITY        string          `json:"CLARITY" fieldType:"string | NULL"`
	Shape          string          `json:"shape" fieldType:"string | NULL"`
	Size           string          `json:"size" fieldType:"string | NULL"`
	FINE1          float64         `json:"FINE1" fieldType:"float | NULL" defaultValue:"0.0"`
	FINE2          float64         `json:"FINE2" fieldType:"float | NULL" defaultValue:"0.0"`
	GROSSWT1       float64         `json:"GROSSWT1" fieldType:"float | NULL" defaultValue:"0.0"`
	GROSSWT2       float64         `json:"GROSSWT2" fieldType:"float | NULL" defaultValue:"0.0"`
	TGNO           string          `json:"TGNO" fieldType:"string | NULL"`
	Vtgno          int             `json:"vtgno" fieldType:"int | NULL" defaultValue:"0"`
	TPRE           string          `json:"TPRE" fieldType:"string | NULL"`
	TSNO           int             `json:"TSNO" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	PDIS           float64         `json:"PDIS" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTAMT        float64         `json:"COSTAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	DIAWT          float64         `json:"DIAWT" fieldType:"float | NULL" defaultValue:"0.0"`
	STNWT          float64         `json:"STNWT" fieldType:"float | NULL" defaultValue:"0.0"`
	DIVIDE         float64         `json:"DIVIDE" fieldType:"float | NULL" defaultValue:"0.0"`
	PREWSTG        float64         `json:"PREWSTG" fieldType:"float | NULL" defaultValue:"0.0"`
	PREWSTGWT      float64         `json:"PREWSTGWT" fieldType:"float | NULL" defaultValue:"0.0"`
	POLISH         float64         `json:"POLISH" fieldType:"float | NULL" defaultValue:"0.0"`
	POLISHWT       float64         `json:"POLISHWT" fieldType:"float | NULL" defaultValue:"0.0"`
	ORGRATE        float64         `json:"ORGRATE" fieldType:"float | NULL" defaultValue:"0.0"`
	ORGGWT         float64         `json:"ORGGWT" fieldType:"float | NULL" defaultValue:"0.0"`
	ORGTOTAL       float64         `json:"ORGTOTAL" fieldType:"float | NULL" defaultValue:"0.0"`
	DLESS          int             `json:"DLESS" fieldType:"int | NULL" defaultValue:"0"`
	SLESS          int             `json:"SLESS" fieldType:"int | NULL" defaultValue:"0"`
	TOTAL          float64         `json:"TOTAL" fieldType:"float | NULL" defaultValue:"0.0"`
	KACNO          int             `json:"KACNO" fieldType:"int | NULL" defaultValue:"0"`
	Karigar        string          `json:"karigar" fieldType:"string | NULL"`
	NO12           int             `json:"NO12" fieldType:"int | NULL" defaultValue:"0"`
	BATCH          string          `json:"BATCH" fieldType:"string | NULL"`
	Goldwt         float64         `json:"goldwt" fieldType:"float | NULL" defaultValue:"0.0"`
	Silwt          float64         `json:"silwt" fieldType:"float | NULL" defaultValue:"0.0"`
	PLATWT         float64         `json:"PLATWT" fieldType:"float | NULL" defaultValue:"0.0"`
	Othwt          float64         `json:"othwt" fieldType:"float | NULL" defaultValue:"0.0"`
	Metalwt        float64         `json:"metalwt" fieldType:"float | NULL" defaultValue:"0.0"`
	JOBNO          string          `json:"JOBNO" fieldType:"string | NULL"`
	JOBSTART       int             `json:"JOBSTART" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	RETWT          float64         `json:"RETWT" fieldType:"float | NULL" defaultValue:"0.0"`
	RETPC          int             `json:"RETPC" fieldType:"int | NULL" defaultValue:"0"`
	BROKWT         float64         `json:"BROKWT" fieldType:"float | NULL" defaultValue:"0.0"`
	BROKPC         int             `json:"BROKPC" fieldType:"int | NULL" defaultValue:"0"`
	ADJWT          float64         `json:"ADJWT" fieldType:"float | NULL" defaultValue:"0.0"`
	ADJVALUE       float64         `json:"ADJVALUE" fieldType:"float | NULL" defaultValue:"0.0"`
	BALWT          float64         `json:"BALWT" fieldType:"float | NULL" defaultValue:"0.0"`
	BALPC          int             `json:"BALPC" fieldType:"int | NULL" defaultValue:"0"`
	SPRICE         float64         `json:"SPRICE" fieldType:"float | NULL" defaultValue:"0.0"`
	MRATE          float64         `json:"MRATE" fieldType:"float | NULL" defaultValue:"0.0"`
	DTIME          time.Time       `json:"DTIME" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	SETNO          int             `json:"SETNO" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	Pktwt          float64         `json:"pktwt" fieldType:"float | NULL" defaultValue:"0.0"`
	Pktless        float64         `json:"pktless" fieldType:"float | NULL" defaultValue:"0.0"`
	GWT1           float64         `json:"GWT1" fieldType:"float | NULL" defaultValue:"0.0"`
	GWT2           float64         `json:"GWT2" fieldType:"float | NULL" defaultValue:"0.0"`
	PART           string          `json:"PART" fieldType:"string | NULL"`
	PURTUNCH       float64         `json:"PURTUNCH" fieldType:"float | NULL" defaultValue:"0.0"`
	BEEDS          float64         `json:"BEEDS" fieldType:"float | NULL" defaultValue:"0.0"`
	SBEEDS         float64         `json:"SBEEDS" fieldType:"float | NULL" defaultValue:"0.0"`
	RATEUNIT       string          `json:"RATEUNIT" fieldType:"string | NULL"`
	BEEDSRATE      float64         `json:"BEEDSRATE" fieldType:"float | NULL" defaultValue:"0.0"`
	BANFINE        float64         `json:"BANFINE" fieldType:"float | NULL" defaultValue:"0.0"`
	SAMPLE         string          `json:"SAMPLE" fieldType:"string | NULL"`
	METTYPE        int             `json:"METTYPE" fieldType:"int | NULL" defaultValue:"0"`
	HISAB          int             `json:"HISAB" fieldType:"int | NULL" defaultValue:"0"`
	SVALUE         int             `json:"SVALUE" fieldType:"int | NULL" defaultValue:"0"`
	TUNCHWT        float64         `json:"TUNCHWT" fieldType:"float | NULL" defaultValue:"0.0"`
	ACTUNCH        float64         `json:"ACTUNCH" fieldType:"float | NULL" defaultValue:"0.0"`
	RPTUNCH        float64         `json:"RPTUNCH" fieldType:"float | NULL" defaultValue:"0.0"`
	TRVONO         int             `json:"TRVONO" fieldType:"int | NULL" defaultValue:"0"`
	TRID           int             `json:"TRID" fieldType:"int | NULL" defaultValue:"0"`
	TUNCHREM       string          `json:"TUNCHREM" fieldType:"string | NULL"`
	TCHTNO         int             `json:"TCHTNO" fieldType:"int | NULL" defaultValue:"0"`
	PHOTOPATH      string          `json:"PHOTOPATH" fieldType:"string | NULL"`
	DIAPC          int             `json:"DIAPC" fieldType:"int | NULL" defaultValue:"0"`
	STNPC          int             `json:"STNPC" fieldType:"int | NULL" defaultValue:"0"`
	APPROVAL       string          `json:"APPROVAL" fieldType:"string | NULL"  Index:"I"`
	Certno         string          `json:"certno" fieldType:"string | NULL"`
	SLBR2PC1       int             `json:"SLBR2PC1" fieldType:"int | NULL" defaultValue:"0"`
	SLBR2PC2       int             `json:"SLBR2PC2" fieldType:"int | NULL" defaultValue:"0"`
	LINKTGNO       string          `json:"LINKTGNO" fieldType:"string | NULL"`
	LINKTSNO       int             `json:"LINKTSNO" fieldType:"int | NULL" defaultValue:"0"`
	STDFINE1       float64         `json:"STDFINE1" fieldType:"float | NULL" defaultValue:"0.0"`
	STDFINE2       float64         `json:"STDFINE2" fieldType:"float | NULL" defaultValue:"0.0"`
	QUALITY        string          `json:"QUALITY" fieldType:"string | NULL"`
	Hm             string          `json:"hm" fieldType:"string | NULL"`
	LINKITRNID     int             `json:"LINKITRNID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	PEXWT          float64         `json:"PEXWT" fieldType:"float | NULL" defaultValue:"0.0"`
	EXWT           float64         `json:"EXWT" fieldType:"float | NULL" defaultValue:"0.0"`
	Stunch         float64         `json:"stunch" fieldType:"float | NULL" defaultValue:"0.0"`
	SWSTG          float64         `json:"SWSTG" fieldType:"float | NULL" defaultValue:"0.0"`
	SWT            float64         `json:"SWT" fieldType:"float | NULL" defaultValue:"0.0"`
	PROFIT         float64         `json:"PROFIT" fieldType:"float | NULL" defaultValue:"0.0"`
	LESS           string          `json:"LESS" fieldType:"string | NULL"`
	SPOLISH        float64         `json:"SPOLISH" fieldType:"float | NULL" defaultValue:"0.0"`
	SPOLISHWT      float64         `json:"SPOLISHWT" fieldType:"float | NULL" defaultValue:"0.0"`
	TINT           float64         `json:"TINT" fieldType:"float | NULL" defaultValue:"0.0"`
	Sothers        float64         `json:"sothers" fieldType:"float | NULL" defaultValue:"0.0"`
	ALLOT          int             `json:"ALLOT" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	TGDID          int             `json:"TGDID" fieldType:"int | NULL" defaultValue:"0"`
	PLESS          float64         `json:"PLESS" fieldType:"float | NULL" defaultValue:"0.0"`
	GROSSFINE1     float64         `json:"GROSSFINE1" fieldType:"float | NULL" defaultValue:"0.0"`
	GROSSFINE2     float64         `json:"GROSSFINE2" fieldType:"float | NULL" defaultValue:"0.0"`
	IWT            float64         `json:"IWT" fieldType:"float | NULL" defaultValue:"0.0"`
	IPC            int             `json:"IPC" fieldType:"int | NULL" defaultValue:"0"`
	CALCWT         float64         `json:"CALCWT" fieldType:"float | NULL" defaultValue:"0.0"`
	CALCPC         int             `json:"CALCPC" fieldType:"int | NULL" defaultValue:"0"`
	RITRNID        int             `json:"RITRNID" fieldType:"int | NULL" defaultValue:"0"`
	BBID           int             `json:"BBID" fieldType:"int | NULL" defaultValue:"0"`
	LINKBBID       int             `json:"LINKBBID" fieldType:"int | NULL" defaultValue:"0"`
	PRETWT         float64         `json:"PRETWT" fieldType:"float | NULL" defaultValue:"0.0"`
	LOTTID         int             `json:"LOTTID" fieldType:"int | NULL" defaultValue:"0"`
	SIZEID         int             `json:"SIZEID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	OWNER          string          `json:"OWNER" fieldType:"string | NULL"`
	OWNACNO        int             `json:"OWNACNO" fieldType:"int | NULL" defaultValue:"0"`
	TRTNO          int             `json:"TRTNO" fieldType:"int | NULL" defaultValue:"0"`
	LABID          int             `json:"LABID" fieldType:"int | NULL" defaultValue:"0"`
	DELDATE        time.Time       `json:"DELDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	SALEMRP        float64         `json:"SALEMRP" fieldType:"float | NULL" defaultValue:"0.0"`
	JOBID          int             `json:"JOBID" fieldType:"int | NULL"  Index:"I" defaultValue:"0"`
	Stkfine1       float64         `json:"stkfine1" fieldType:"float | NULL" defaultValue:"0.0"`
	Stkfine2       float64         `json:"stkfine2" fieldType:"float | NULL" defaultValue:"0.0"`
	OTGDID         int             `json:"OTGDID" fieldType:"int | NULL" defaultValue:"0"`
	KTINO          int             `json:"KTINO" fieldType:"int | NULL" defaultValue:"0"`
	KTTGNO         string          `json:"KTTGNO" fieldType:"string | NULL"`
	KTVTGNO        int             `json:"KTVTGNO" fieldType:"int | NULL" defaultValue:"0"`
	KTSLBR2        float64         `json:"KTSLBR2" fieldType:"float | NULL" defaultValue:"0.0"`
	KTSOTHERS      float64         `json:"KTSOTHERS" fieldType:"float | NULL" defaultValue:"0.0"`
	KTTSNO         int             `json:"KTTSNO" fieldType:"int | NULL" defaultValue:"0"`
	KTITRNID       int             `json:"KTITRNID" fieldType:"int | NULL" defaultValue:"0"`
	Sttunch        float64         `json:"sttunch" fieldType:"float | NULL" defaultValue:"0.0"`
	BILLTYPE       int             `json:"BILLTYPE" fieldType:"int | NULL" defaultValue:"0"`
	STDLBR         float64         `json:"STDLBR" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTRATE       float64         `json:"COSTRATE" fieldType:"float | NULL" defaultValue:"0.0"`
	OJOBNO         string          `json:"OJOBNO" fieldType:"string | NULL"`
	OJOBID         int             `json:"OJOBID" fieldType:"int | NULL" defaultValue:"0"`
	OCANCEL        int             `json:"OCANCEL" fieldType:"int | NULL" defaultValue:"0"`
	OCREMARKS      string          `json:"OCREMARKS" fieldType:"string | NULL"`
	OCDATE         time.Time       `json:"OCDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	OCLOGINID      int             `json:"OCLOGINID" fieldType:"int | NULL" defaultValue:"0"`
	OCTIME         time.Time       `json:"OCTIME" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	DESIGNID       int             `json:"DESIGNID" fieldType:"int | NULL" defaultValue:"0"`
	COSTVALUE      float64         `json:"COSTVALUE" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTMAMT       float64         `json:"COSTMAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTDAMT       float64         `json:"COSTDAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTSAMT       float64         `json:"COSTSAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTLAMT       float64         `json:"COSTLAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	COSTWSTG       float64         `json:"COSTWSTG" fieldType:"float | NULL" defaultValue:"0.0"`
	DIS            float64         `json:"DIS" fieldType:"float | NULL" defaultValue:"0.0"`
	SRTYPE         string          `json:"SRTYPE" fieldType:"string | NULL" Index:"I"`
	COSTTOTAL      float64         `json:"COSTTOTAL" fieldType:"float | NULL" defaultValue:"0.0"`
	GOLDAMT        float64         `json:"GOLDAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	SILAMT         float64         `json:"SILAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	TAGFINE1       float64         `json:"TAGFINE1" fieldType:"float | NULL" defaultValue:"0.0"`
	TAGFINE2       float64         `json:"TAGFINE2" fieldType:"float | NULL" defaultValue:"0.0"`
	PURCHASE       string          `json:"PURCHASE" fieldType:"string | NULL"`
	Pcrate         string          `json:"pcrate" fieldType:"string | NULL"`
	SETNOBAL       string          `json:"SETNOBAL" fieldType:"string | NULL"`
	JOBGWT         float64         `json:"JOBGWT" fieldType:"float | NULL" defaultValue:"0.0"`
	Orgwt          float64         `json:"orgwt" fieldType:"float | NULL" defaultValue:"0.0"`
	Othrem         string          `json:"othrem" fieldType:"string | NULL"`
	REVISED        int             `json:"REVISED" fieldType:"int | NULL" defaultValue:"0"`
	Saleless       string          `json:"saleless" fieldType:"string | NULL"`
	PPROFIT        float64         `json:"PPROFIT" fieldType:"float | NULL" defaultValue:"0.0"`
	LBRADD         float64         `json:"LBRADD" fieldType:"float | NULL" defaultValue:"0.0"`
	DAILY          int             `json:"DAILY" fieldType:"int | NULL" defaultValue:"0"`
	DIAWT1         float64         `json:"DIAWT1" fieldType:"float | NULL" defaultValue:"0.0"`
	DIAWT2         float64         `json:"DIAWT2" fieldType:"float | NULL" defaultValue:"0.0"`
	STNWT1         float64         `json:"STNWT1" fieldType:"float | NULL" defaultValue:"0.0"`
	STNWT2         float64         `json:"STNWT2" fieldType:"float | NULL" defaultValue:"0.0"`
	LAKHWT         float64         `json:"LAKHWT" fieldType:"float | NULL" defaultValue:"0.0"`
	SBANFINE       float64         `json:"SBANFINE" fieldType:"float | NULL" defaultValue:"0.0"`
	UPFLG          int             `json:"UPFLG" fieldType:"int | NULL" defaultValue:"0"`
	WID            int             `json:"WID" fieldType:"int | NULL" defaultValue:"0"`
	OACNO          int             `json:"OACNO" fieldType:"int | NULL" defaultValue:"0"`
	OAMT           int             `json:"OAMT" fieldType:"int | NULL" defaultValue:"0"`
	POAMT          float64         `json:"POAMT" fieldType:"float | NULL" defaultValue:"0.0"`
	KHISAB         int             `json:"KHISAB" fieldType:"int | NULL" defaultValue:"0"`
	MRP1           int             `json:"MRP1" fieldType:"int | NULL" defaultValue:"0"`
	MRP2           int             `json:"MRP2" fieldType:"int | NULL" defaultValue:"0"`
	COSTING1       int             `json:"COSTING1" fieldType:"int | NULL" defaultValue:"0"`
	COSTING2       int             `json:"COSTING2" fieldType:"int | NULL" defaultValue:"0"`
	WITNO          int             `json:"WITNO" fieldType:"int | NULL" defaultValue:"0"`
	LBRON          int             `json:"LBRON" fieldType:"int | NULL" defaultValue:"0"`
	DOLLAR         float64         `json:"DOLLAR" fieldType:"float | NULL" defaultValue:"0.0"`
	DOLXRATE       float64         `json:"DOLXRATE" fieldType:"float | NULL" defaultValue:"0.0"`
	KISNO          int             `json:"KISNO" fieldType:"int | NULL" defaultValue:"0"`
	Diaremark      string          `json:"diaremark" fieldType:"string | NULL"`
	LAMTIDET       float64         `json:"LAMTIDET" fieldType:"float | NULL" defaultValue:"0.0"`
	Distype        int             `json:"distype" fieldType:"int | NULL" defaultValue:"0"`
	IHISAB         int             `json:"IHISAB" fieldType:"int | NULL" defaultValue:"0"`
	SMACNO         int             `json:"SMACNO" fieldType:"int | NULL" defaultValue:"0"`
	Psgst          float64         `json:"psgst" fieldType:"float | NULL" defaultValue:"0.0"`
	Pcgst          float64         `json:"pcgst" fieldType:"float | NULL" defaultValue:"0.0"`
	PIGST          float64         `json:"PIGST" fieldType:"float | NULL" defaultValue:"0.0"`
	SGST           float64         `json:"SGST" fieldType:"float | NULL" defaultValue:"0.0"`
	CGST           float64         `json:"CGST" fieldType:"float | NULL" defaultValue:"0.0"`
	IGST           float64         `json:"IGST" fieldType:"float | NULL" defaultValue:"0.0"`
	TGCODE         string          `json:"TGCODE" fieldType:"string | NULL"`
	Lnkjobid       float64         `json:"lnkjobid" fieldType:"float | NULL" defaultValue:"0.0"`
	KTBATCH        string          `json:"KTBATCH" fieldType:"string | NULL"`
	Ttvalue        float64         `json:"ttvalue" fieldType:"float | NULL" defaultValue:"0.0"`
	Ksitrnid       int             `json:"ksitrnid" fieldType:"int | NULL" defaultValue:"0"`
	Ktsvalue       int             `json:"ktsvalue" fieldType:"int | NULL" defaultValue:"0"`
	Xtfld          string          `json:"xtfld" fieldType:"string | NULL"`
}
