package mysql_to_surreal_interfaces

import (
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

type TgMasterStruct struct {
	Idesc     string             `json:"idesc"`
	Tgno      mysqldb.NullString `json:"tgno" fieldType:"string"`
	Vtgno     int                `json:"vtgno"`
	Tpre      mysqldb.NullString `json:"tpre" fieldType:"string"`
	Remarks   mysqldb.NullString `json:"remarks" fieldType:"string"`
	Tdate     mysqldb.NullString `json:"tdate" fieldType:"string"`
	Gwt       float64            `json:"gwt" fieldType:"float"`
	Lesswt    float64            `json:"lesswt" fieldType:"float"`
	Wt        float64            `json:"wt" fieldType:"float"`
	Sdiawt    float64            `json:"sdiawt" fieldType:"float"`
	Sstnwt    float64            `json:"sstnwt" fieldType:"float"`
	Goldwt    float64            `json:"goldwt" fieldType:"float"`
	Silwt     float64            `json:"silwt" fieldType:"float"`
	Platwt    float64            `json:"platwt" fieldType:"float"`
	Othwt     float64            `json:"othwt" fieldType:"float"`
	Slbr      float64            `json:"slbr" fieldType:"float"`
	Slbr2     float64            `json:"slbr2" fieldType:"float"`
	Slbr3     float64            `json:"slbr3" fieldType:"float"`
	Status    TgMasterStatus     `json:"status" fieldType:"string"`
	Stunch    float64            `json:"stunch" fieldType:"float"`
	Swstg     float64            `json:"swstg" fieldType:"float"`
	Sbeeds    float64            `json:"sbeeds" fieldType:"float"`
	Sothers   float64            `json:"sothers" fieldType:"float"`
	Othrem    mysqldb.NullString `json:"othrem" fieldType:"string"`
	Design    mysqldb.NullString `json:"design" fieldType:"string"`
	Karigar   mysqldb.NullString `json:"karigar" fieldType:"string"`
	Mrate     float64            `json:"mrate" fieldType:"float"`
	Gwt1      float64            `json:"gwt1" fieldType:"float"`
	Gwt2      float64            `json:"gwt2" fieldType:"float"`
	Stamp     mysqldb.NullString `json:"stamp" fieldType:"string"`
	Size      mysqldb.NullString `json:"size" fieldType:"string"`
	Quality   mysqldb.NullString `json:"quality" fieldType:"string"`
	Color     mysqldb.NullString `json:"color" fieldType:"string"`
	Clarity   mysqldb.NullString `json:"clarity" fieldType:"string"`
	Site      mysqldb.NullString `json:"site" fieldType:"string"`
	Linktgno  mysqldb.NullString `json:"linktgno" fieldType:"string"`
	Diapc     int                `json:"diapc"`
	Stnpc     int                `json:"stnpc"`
	Mrp1      float64            `json:"mrp1" fieldType:"float"`
	Mrp2      float64            `json:"mrp2" fieldType:"float"`
	Sdamt     int                `json:"sdamt"`
	Ssamt     int                `json:"ssamt"`
	Slamt     int                `json:"slamt"`
	Smamt     int                `json:"smamt"`
	Scamt     int                `json:"scamt"`
	Mrp       float64            `json:"mrp" fieldType:"float"`
	Hm        mysqldb.NullString `json:"hm" fieldType:"string"`
	Certno    mysqldb.NullString `json:"certno" fieldType:"string"`
	Spolish   float64            `json:"spolish" fieldType:"float"`
	Spolishwt float64            `json:"spolishwt" fieldType:"float"`
	Pc        int                `json:"pc"`
	Salemrp   float64            `json:"salemrp" fieldType:"float"`
	Diawt1    float64            `json:"diawt1" fieldType:"float"`
	Diawt2    float64            `json:"diawt2" fieldType:"float"`
	Stnwt1    float64            `json:"stnwt1" fieldType:"float"`
	Stnwt2    float64            `json:"stnwt2" fieldType:"float"`
	Lakhwt    float64            `json:"lakhwt" fieldType:"float"`
	Shape     mysqldb.NullString `json:"shape" fieldType:"string"`
	Itname    mysqldb.NullString `json:"itname" fieldType:"string"`
	Category  any                `json:"category" fieldType:"string | NULL"`
	Gcode     any                `json:"gcode" fieldType:"string | NULL"`
	Tsno      int                `json:"tsno"`
	Designid  mysqldb.NullString `json:"designid" fieldType:"string"`
	Pname     mysqldb.NullString `json:"pname" fieldType:"string"`
	Branch    mysqldb.NullString `json:"branch" fieldType:"string"`
	Tgmastid  int                `json:"tgmastid"`
	Diacut    mysqldb.NullString `json:"diacut" fieldType:"string"`
	Diapol    mysqldb.NullString `json:"diapol" fieldType:"string"`
	Diasymm   mysqldb.NullString `json:"diasymm" fieldType:"string"`
	Pdis      float64            `json:"pdis" fieldType:"float"`
	Cindex    mysqldb.NullString `json:"cindex" fieldType:"string"`
	Photopath mysqldb.NullString `json:"photopath" fieldType:"string"`
	Igroup    mysqldb.NullString `json:"igroup" fieldType:"string"`
	Rfid      mysqldb.NullString `json:"rfid" fieldType:"string"`
	Gst       float64            `json:"gst" fieldType:"float"`
	Billamt   float64            `json:"billamt" fieldType:"float"`
	Billtype  int                `json:"billtype"`
	Tgimage   mysqldb.NullString `json:"tgimage" fieldType:"string"`
	Ino       int                `json:"ino"`
	Csmamt    float64            `json:"csmamt" fieldType:"float"`
	Cslamt    float64            `json:"cslamt" fieldType:"float"`
	Itype     int                `json:"itype"`
	Unit      mysqldb.NullString `json:"unit" fieldType:"string"`
	Srate     float64            `json:"srate" fieldType:"float"`
	Pgst      float64            `json:"pgst" fieldType:"float"`
	Siteid    int                `json:"siteid"`
}
