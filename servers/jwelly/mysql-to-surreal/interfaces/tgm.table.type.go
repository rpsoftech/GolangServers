package mysql_to_surreal_interfaces

import (
	"time"

	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type TGM1Struct struct {
	SurrealId      int             `json:"id" Index:"U"`
	TSNO           int             `json:"tsno" Index:"U"`
	TGNO           string          `json:"TGNO" Index:"I"`
	VTGNO          int             `json:"vtgno" Index:"I"`
	INO            int             `json:"INO" Index:"I"`
	TPRE           string          `json:"TPRE" Index:"I"`
	REMARKS        string          `json:"REMARKS"`
	TDATE          time.Time       `json:"TDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	ITRNID         int             `json:"ITRNID" Index:"I"`
	GWT            float64         `json:"GWT" fieldType:"float"`
	LESSWT         float64         `json:"LESSWT" fieldType:"float"`
	WT             float64         `json:"WT" fieldType:"float"`
	DIAWT          float64         `json:"DIAWT" fieldType:"float"`
	STNWT          float64         `json:"STNWT" fieldType:"float"`
	GOLDWT         float64         `json:"GOLDWT" fieldType:"float"`
	SILWT          float64         `json:"SILWT" fieldType:"float"`
	PLATWT         float64         `json:"PLATWT" fieldType:"float"`
	OTHWT          float64         `json:"OTHWT" fieldType:"float"`
	LBR            float64         `json:"LBR" fieldType:"float"`
	LBR2           float64         `json:"LBR2" fieldType:"float"`
	LBR3           float64         `json:"LBR3" fieldType:"float"`
	SLBR           float64         `json:"SLBR" fieldType:"float"`
	Slbr2          float64         `json:"slbr2" fieldType:"float"`
	SLBR3          float64         `json:"SLBR3" fieldType:"float"`
	STATUS         string          `json:"STATUS" Index:"I"`
	TUNCH          float64         `json:"TUNCH" fieldType:"float"`
	WSTG           float64         `json:"WSTG" fieldType:"float"`
	STUNCH         float64         `json:"STUNCH" fieldType:"float"`
	SWSTG          float64         `json:"SWSTG" fieldType:"float"`
	BEEDS          float64         `json:"BEEDS" fieldType:"float"`
	SBEEDS         float64         `json:"SBEEDS" fieldType:"float"`
	SOTHERS        float64         `json:"SOTHERS" fieldType:"float"`
	Othrem         string          `json:"othrem"`
	Design         string          `json:"design"`
	DESINO         int             `json:"DESINO"`
	KACNO          int             `json:"KACNO"`
	Karigar        string          `json:"karigar"`
	MRATE          float64         `json:"MRATE" fieldType:"float"`
	COSTRATE       float64         `json:"COSTRATE" fieldType:"float"`
	GWT1           float64         `json:"GWT1" fieldType:"float"`
	GWT2           float64         `json:"GWT2" fieldType:"float"`
	SLBR2PC1       float64         `json:"SLBR2PC1" fieldType:"float"`
	SLBR2PC2       float64         `json:"SLBR2PC2" fieldType:"float"`
	SurrealSTAMPID models.RecordID `json:"SurrealSTAMPID" Index:"I" fieldType:"record<stamp>"`
	STAMPID        int             `json:"STAMPID" Index:"I"`
	Flag           mysqldb.BitBool `json:"flag" fieldType:"bool"`
	PHOTOPATH      string          `json:"PHOTOPATH"`
	Size           string          `json:"size"`
	QUALITY        string          `json:"QUALITY"`
	COLOUR         string          `json:"COLOUR"`
	CLARITY        string          `json:"CLARITY"`
	POLISH         float64         `json:"POLISH" fieldType:"float"`
	POLISHWT       float64         `json:"POLISHWT" fieldType:"float"`
	SurrealSITEID  models.RecordID `json:"SurrealSITEID" Index:"I" fieldType:"record<site>"`
	SITEID         int             `json:"SITEID" Index:"I"`
	LOGIN          string          `json:"LOGIN"`
	SETNO          int             `json:"SETNO"`
	ORDNO          string          `json:"ORDNO"`
	DLESS          int             `json:"DLESS"`
	SLESS          int             `json:"SLESS"`
	LINKTGNO       string          `json:"LINKTGNO" Index:"I"`
	LINKTSNO       int             `json:"LINKTSNO" Index:"I"`
	ADDMRP         int             `json:"ADDMRP" fieldType:"int | NULL" defaultValue:"0"`
	LAMT           float64         `json:"LAMT" fieldType:"float"`
	DAMT           float64         `json:"DAMT" fieldType:"float"`
	SAMT           float64         `json:"SAMT" fieldType:"float"`
	MAMT           float64         `json:"MAMT" fieldType:"float"`
	OTHERS         float64         `json:"OTHERS" fieldType:"float"`
	TAGFINE1       float64         `json:"TAGFINE1" fieldType:"float"`
	TAGFINE2       float64         `json:"TAGFINE2" fieldType:"float"`
	COSTDAMT       float64         `json:"COSTDAMT" fieldType:"float"`
	COSTSAMT       float64         `json:"COSTSAMT" fieldType:"float"`
	COSTMAMT       float64         `json:"COSTMAMT" fieldType:"float"`
	MRP            float64         `json:"MRP" fieldType:"float"`
	Hm             string          `json:"hm"`
	Certno         string          `json:"certno"`
	SPOLISH        float64         `json:"SPOLISH" fieldType:"float"`
	SPOLISHWT      float64         `json:"SPOLISHWT" fieldType:"float"`
	SWT            float64         `json:"SWT" fieldType:"float"`
	VONO           int             `json:"VONO"`
	TNO            int             `json:"TNO"`
	Type           string          `json:"type"`
	SDLESS         int             `json:"SDLESS"`
	SSLESS         int             `json:"SSLESS"`
	BILLTYPE       int             `json:"BILLTYPE" Index:"I"`
	SurrealUNITID  models.RecordID `json:"SurrealUNITID" Index:"I" fieldType:"record<units>"`
	UNITID         int             `json:"UNITID"`
	PC             int             `json:"PC"`
	SALEMRP        float64         `json:"SALEMRP" fieldType:"float"`
	JOBID          int             `json:"JOBID" Index:"I"`
	COSTTOTAL      float64         `json:"COSTTOTAL" fieldType:"float"`
	COSTWSTG       float64         `json:"COSTWSTG" fieldType:"float"`
	GROSSFINE1     float64         `json:"GROSSFINE1" fieldType:"float"`
	GROSSFINE2     float64         `json:"GROSSFINE2" fieldType:"float"`
	FINE1          float64         `json:"FINE1" fieldType:"float"`
	FINE2          float64         `json:"FINE2" fieldType:"float"`
	TOTAL          float64         `json:"TOTAL" fieldType:"float"`
	Pcrate         string          `json:"pcrate"`
	PPROFIT        float64         `json:"PPROFIT" fieldType:"float"`
	UPFLAG         int             `json:"UPFLAG"`
	STKFINE1       float64         `json:"STKFINE1" fieldType:"float"`
	STKFINE2       float64         `json:"STKFINE2" fieldType:"float"`
	RATE           float64         `json:"RATE" fieldType:"float"`
	DAILY          int             `json:"DAILY" Index:"I"`
	TRALT          int             `json:"TRALT"`
	DIAWT1         float64         `json:"DIAWT1" fieldType:"float"`
	DIAWT2         float64         `json:"DIAWT2" fieldType:"float"`
	STNWT1         float64         `json:"STNWT1" fieldType:"float"`
	STNWT2         float64         `json:"STNWT2" fieldType:"float"`
	LAKHWT         float64         `json:"LAKHWT" fieldType:"float"`
	BANFINE        float64         `json:"BANFINE" fieldType:"float"`
	SBANFINE       float64         `json:"SBANFINE" fieldType:"float"`
	UPSTATUS       int             `json:"UPSTATUS"`
	UPINFO         int             `json:"UPINFO"`
	APPROVAL       string          `json:"APPROVAL"`
	DESIGNID       int             `json:"DESIGNID" Index:"I"`
	REPWT          float64         `json:"REPWT" fieldType:"float"`
	LASTDATE       time.Time       `json:"LASTDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	DIAPC          int             `json:"DIAPC"`
	STNPC          int             `json:"STNPC"`
	MRP1           float64         `json:"MRP1" fieldType:"float"`
	MRP2           float64         `json:"MRP2" fieldType:"float"`
	COSTING1       float64         `json:"COSTING1" fieldType:"float"`
	COSTING2       float64         `json:"COSTING2" fieldType:"float"`
	MFLAG          int             `json:"MFLAG"`
	SIZEID         int             `json:"SIZEID"`
	GITRNID        int             `json:"GITRNID"`
	Diaremark      string          `json:"diaremark"`
	DOLLAR         float64         `json:"DOLLAR" fieldType:"float"`
	DOLXRATE       float64         `json:"DOLXRATE" fieldType:"float"`
	PDIS           float64         `json:"PDIS" fieldType:"float"`
	OLDSTATUS      string          `json:"OLDSTATUS"`
	LCODE          string          `json:"LCODE"`
	SHAPE          string          `json:"SHAPE"`
	TVALUE         float64         `json:"TVALUE" fieldType:"float"`
	TRITRNID       int             `json:"TRITRNID"`
	UPTGNO         int             `json:"UPTGNO"`
	SVALUE         float64         `json:"SVALUE" fieldType:"float"`
	RFID           string          `json:"RFID"`
	BATCH          string          `json:"BATCH"`
	COSTLAMT       float64         `json:"COSTLAMT" fieldType:"float"`
	OJOBID         int             `json:"OJOBID"`
	OLINKTGNO      string          `json:"OLINKTGNO"`
	SDAMT          int             `json:"SDAMT"`
	SSAMT          int             `json:"SSAMT"`
	SLAMT          int             `json:"SLAMT"`
	SMAMT          int             `json:"SMAMT"`
	DIS            float64         `json:"DIS" fieldType:"float"`
	PKTWT          float64         `json:"PKTWT" fieldType:"float"`
	PKTLESS        float64         `json:"PKTLESS" fieldType:"float"`
	ORGGWT         float64         `json:"ORGGWT" fieldType:"float"`
	ORGTOTAL       float64         `json:"ORGTOTAL" fieldType:"float"`
	OACNO          int             `json:"OACNO"`
	VTOTAL         int             `json:"VTOTAL"`
	TGMIDESC       string          `json:"TGMIDESC"`
	SDIAWT         float64         `json:"SDIAWT" fieldType:"float"`
	SSTNWT         float64         `json:"SSTNWT" fieldType:"float"`
	// SurrealTRSITEID models.RecordID `json:"SurrealTRSITEID" Index:"I" fieldType:"record<site>"`
	TRSITEID  int       `json:"TRSITEID"`
	SAMPLE    string    `json:"SAMPLE"`
	SRTYPE    string    `json:"SRTYPE"`
	OAMT      float64   `json:"OAMT" fieldType:"float"`
	POAMT     float64   `json:"POAMT" fieldType:"float"`
	RLID      int       `json:"RLID"`
	COSTAMT   float64   `json:"COSTAMT" fieldType:"float"`
	OCDATE    time.Time `json:"OCDATE" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	TRLBR     float64   `json:"TRLBR" fieldType:"float"`
	TRLCODE   string    `json:"TRLCODE"`
	TRFLAG    int       `json:"TRFLAG"`
	TGCODE    string    `json:"TGCODE"`
	Stagfine1 float64   `json:"stagfine1" fieldType:"float"`
	Stagfine2 float64   `json:"stagfine2" fieldType:"float"`
	Pless     float64   `json:"pless" fieldType:"float"`
	Calmrp    float64   `json:"calmrp" fieldType:"float"`
	Gst       float64   `json:"gst" fieldType:"float"`
	Saleless  string    `json:"saleless"`
	Csmamt    float64   `json:"csmamt" fieldType:"float"`
	Grade     string    `json:"grade"`
	Cslamt    float64   `json:"cslamt" fieldType:"float"`
	Srate     float64   `json:"srate" fieldType:"float"`
	Deldate   time.Time `json:"deldate" fieldType:"datetime | NULL" defaultValue:"0000-00-00"`
	Pgst      float64   `json:"pgst" fieldType:"float"`
	Xtfld     string    `json:"xtfld"`
	Cstdmamt  float64   `json:"cstdmamt" fieldType:"float"`
}
