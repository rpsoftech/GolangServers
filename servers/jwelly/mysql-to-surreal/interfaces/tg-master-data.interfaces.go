package mysql_to_surreal_interfaces

import (
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

type TgMasterStruct struct {
	Idesc     mysqldb.NullString `json:"idesc"`
	Tgno      mysqldb.NullString `json:"tgno"`
	Vtgno     int                `json:"vtgno"`
	Tpre      mysqldb.NullString `json:"tpre"`
	Remarks   mysqldb.NullString `json:"remarks"`
	Tdate     mysqldb.NullString `json:"tdate"`
	Gwt       float64            `json:"gwt"`
	Lesswt    float64            `json:"lesswt"`
	Wt        float64            `json:"wt"`
	Sdiawt    float64            `json:"sdiawt"`
	Sstnwt    float64            `json:"sstnwt"`
	Goldwt    float64            `json:"goldwt"`
	Silwt     float64            `json:"silwt"`
	Platwt    float64            `json:"platwt"`
	Othwt     float64            `json:"othwt"`
	Slbr      float64            `json:"slbr"`
	Slbr2     float64            `json:"slbr2"`
	Slbr3     float64            `json:"slbr3"`
	Status    TgMasterStatus     `json:"status"`
	Stunch    float64            `json:"stunch"`
	Swstg     float64            `json:"swstg"`
	Sbeeds    float64            `json:"sbeeds"`
	Sothers   float64            `json:"sothers"`
	Othrem    mysqldb.NullString `json:"othrem"`
	Design    mysqldb.NullString `json:"design"`
	Karigar   mysqldb.NullString `json:"karigar"`
	Mrate     float64            `json:"mrate"`
	Gwt1      float64            `json:"gwt1"`
	Gwt2      float64            `json:"gwt2"`
	Stamp     mysqldb.NullString `json:"stamp"`
	Size      mysqldb.NullString `json:"size"`
	Quality   mysqldb.NullString `json:"quality"`
	Color     mysqldb.NullString `json:"color"`
	Clarity   mysqldb.NullString `json:"clarity"`
	Site      mysqldb.NullString `json:"site"`
	Linktgno  mysqldb.NullString `json:"linktgno"`
	Diapc     int                `json:"diapc"`
	Stnpc     int                `json:"stnpc"`
	Mrp1      float64            `json:"mrp1"`
	Mrp2      float64            `json:"mrp2"`
	Sdamt     int                `json:"sdamt"`
	Ssamt     int                `json:"ssamt"`
	Slamt     int                `json:"slamt"`
	Smamt     int                `json:"smamt"`
	Scamt     int                `json:"scamt"`
	Mrp       float64            `json:"mrp"`
	Hm        mysqldb.NullString `json:"hm"`
	Certno    mysqldb.NullString `json:"certno"`
	Spolish   float64            `json:"spolish"`
	Spolishwt float64            `json:"spolishwt"`
	Pc        int                `json:"pc"`
	Salemrp   float64            `json:"salemrp"`
	Diawt1    float64            `json:"diawt1"`
	Diawt2    float64            `json:"diawt2"`
	Stnwt1    float64            `json:"stnwt1"`
	Stnwt2    float64            `json:"stnwt2"`
	Lakhwt    float64            `json:"lakhwt"`
	Shape     mysqldb.NullString `json:"shape"`
	Itname    mysqldb.NullString `json:"itname"`
	Category  any                `json:"category"`
	Gcode     any                `json:"gcode"`
	Tsno      int                `json:"tsno"`
	Designid  mysqldb.NullString `json:"designid"`
	Pname     mysqldb.NullString `json:"pname"`
	Branch    mysqldb.NullString `json:"branch"`
	Tgmastid  int                `json:"tgmastid"`
	Diacut    mysqldb.NullString `json:"diacut"`
	Diapol    mysqldb.NullString `json:"diapol"`
	Diasymm   mysqldb.NullString `json:"diasymm"`
	Pdis      float64            `json:"pdis"`
	Cindex    mysqldb.NullString `json:"cindex"`
	Photopath mysqldb.NullString `json:"photopath"`
	Igroup    mysqldb.NullString `json:"igroup"`
	Rfid      mysqldb.NullString `json:"rfid"`
	Gst       float64            `json:"gst"`
	Billamt   float64            `json:"billamt"`
	Billtype  int                `json:"billtype"`
	Tgimage   mysqldb.NullString `json:"tgimage"`
	Ino       int                `json:"ino"`
	Csmamt    float64            `json:"csmamt"`
	Cslamt    float64            `json:"cslamt"`
	Itype     int                `json:"itype"`
	Unit      mysqldb.NullString `json:"unit"`
	Srate     float64            `json:"srate"`
	Pgst      float64            `json:"pgst"`
	Siteid    int                `json:"siteid"`
}

const SurrealCreateTgMasterQuery = `DEFINE FIELD idesc ON TABLE tg_master TYPE string;        
	DEFINE FIELD tgno ON TABLE tg_master TYPE string;        
	DEFINE FIELD vtgno ON TABLE tg_master TYPE int;           
	DEFINE FIELD tpre ON TABLE tg_master TYPE string;        
	DEFINE FIELD remarks ON TABLE tg_master TYPE string;        
	DEFINE FIELD tdate ON TABLE tg_master TYPE string;        
	DEFINE FIELD gwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD lesswt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD wt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD sdiawt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD sstnwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD goldwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD silwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD platwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD othwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD slbr ON TABLE tg_master TYPE float64;       
	DEFINE FIELD slbr2 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD slbr3 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD status ON TABLE tg_master TYPE TgMasterStatus;
	DEFINE FIELD stunch ON TABLE tg_master TYPE float64;       
	DEFINE FIELD swstg ON TABLE tg_master TYPE float64;       
	DEFINE FIELD sbeeds ON TABLE tg_master TYPE float64;       
	DEFINE FIELD sothers ON TABLE tg_master TYPE float64;       
	DEFINE FIELD othrem ON TABLE tg_master TYPE string;        
	DEFINE FIELD design ON TABLE tg_master TYPE string;        
	DEFINE FIELD karigar ON TABLE tg_master TYPE string;        
	DEFINE FIELD mrate ON TABLE tg_master TYPE float64;       
	DEFINE FIELD gwt1 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD gwt2 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD stamp ON TABLE tg_master TYPE string;        
	DEFINE FIELD size ON TABLE tg_master TYPE string;        
	DEFINE FIELD quality ON TABLE tg_master TYPE string;        
	DEFINE FIELD color ON TABLE tg_master TYPE string;        
	DEFINE FIELD clarity ON TABLE tg_master TYPE string;        
	DEFINE FIELD site ON TABLE tg_master TYPE string;        
	DEFINE FIELD linktgno ON TABLE tg_master TYPE string;        
	DEFINE FIELD diapc ON TABLE tg_master TYPE int;           
	DEFINE FIELD stnpc ON TABLE tg_master TYPE int;           
	DEFINE FIELD mrp1 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD mrp2 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD sdamt ON TABLE tg_master TYPE int;           
	DEFINE FIELD ssamt ON TABLE tg_master TYPE int;           
	DEFINE FIELD slamt ON TABLE tg_master TYPE int;           
	DEFINE FIELD smamt ON TABLE tg_master TYPE int;           
	DEFINE FIELD scamt ON TABLE tg_master TYPE int;           
	DEFINE FIELD mrp ON TABLE tg_master TYPE float64;       
	DEFINE FIELD hm ON TABLE tg_master TYPE string;        
	DEFINE FIELD certno ON TABLE tg_master TYPE string;        
	DEFINE FIELD spolish ON TABLE tg_master TYPE float64;       
	DEFINE FIELD spolishwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD pc ON TABLE tg_master TYPE int;           
	DEFINE FIELD salemrp ON TABLE tg_master TYPE float64;       
	DEFINE FIELD diawt1 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD diawt2 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD stnwt1 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD stnwt2 ON TABLE tg_master TYPE float64;       
	DEFINE FIELD lakhwt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD shape ON TABLE tg_master TYPE string;        
	DEFINE FIELD itname ON TABLE tg_master TYPE string;        
	DEFINE FIELD category ON TABLE tg_master TYPE any;           
	DEFINE FIELD gcode ON TABLE tg_master TYPE any;           
	DEFINE FIELD tsno ON TABLE tg_master TYPE int;           
	DEFINE FIELD designid ON TABLE tg_master TYPE string;        
	DEFINE FIELD pname ON TABLE tg_master TYPE string;        
	DEFINE FIELD branch ON TABLE tg_master TYPE string;        
	DEFINE FIELD tgmastid ON TABLE tg_master TYPE int;           
	DEFINE FIELD diacut ON TABLE tg_master TYPE string;        
	DEFINE FIELD diapol ON TABLE tg_master TYPE string;        
	DEFINE FIELD diasymm ON TABLE tg_master TYPE string;        
	DEFINE FIELD pdis ON TABLE tg_master TYPE float64;       
	DEFINE FIELD cindex ON TABLE tg_master TYPE string;        
	DEFINE FIELD photopath ON TABLE tg_master TYPE string;        
	DEFINE FIELD igroup ON TABLE tg_master TYPE string;        
	DEFINE FIELD rfid ON TABLE tg_master TYPE string;        
	DEFINE FIELD gst ON TABLE tg_master TYPE float64;       
	DEFINE FIELD billamt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD billtype ON TABLE tg_master TYPE int;           
	DEFINE FIELD tgimage ON TABLE tg_master TYPE string;        
	DEFINE FIELD ino ON TABLE tg_master TYPE int;           
	DEFINE FIELD csmamt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD cslamt ON TABLE tg_master TYPE float64;       
	DEFINE FIELD itype ON TABLE tg_master TYPE int;           
	DEFINE FIELD unit ON TABLE tg_master TYPE string;        
	DEFINE FIELD srate ON TABLE tg_master TYPE float64;       
	DEFINE FIELD pgst ON TABLE tg_master TYPE float64;       
	DEFINE FIELD siteid ON TABLE tg_master TYPE int;`
