package mysql_to_surreal_interfaces

type ItemGroupTableStruct struct {
	SurrealId  string  `json:"id" Index:"U" fieldType:"string"`
	Igroupid   int     `json:"igroupid" fieldType:"int" Index:"U"`
	IGROUP     string  `json:"IGROUP" fieldType:"string | NULL" Index:"I"`
	PNAME      string  `json:"PNAME" fieldType:"string | NULL"`
	UNDERID    int     `json:"UNDERID" fieldType:"int | NULL"`
	ITNAME     string  `json:"ITNAME" fieldType:"string | NULL"`
	LTAX       float64 `json:"LTAX" fieldType:"float | NULL"`
	CTAX       float64 `json:"CTAX" fieldType:"float | NULL"`
	GP         float64 `json:"GP" fieldType:"float | NULL"`
	OBAL       float64 `json:"OBAL" fieldType:"float | NULL"`
	CBAL       float64 `json:"CBAL" fieldType:"float | NULL"`
	SACNO      int     `json:"SACNO" fieldType:"int | NULL"`
	PACNO      int     `json:"PACNO" fieldType:"int | NULL"`
	STSACNO    int     `json:"STSACNO" fieldType:"int | NULL"`
	STPACNO    int     `json:"STPACNO" fieldType:"int | NULL"`
	IACNO      int     `json:"IACNO" fieldType:"int | NULL"`
	RACNO      int     `json:"RACNO" fieldType:"int | NULL"`
	CCODE      string  `json:"CCODE" fieldType:"string | NULL"`
	STVAL      int     `json:"STVAL" fieldType:"int | NULL" Index:"I"`
	DIV        float64 `json:"DIV" fieldType:"float | NULL"`
	INTO       float64 `json:"INTO" fieldType:"float | NULL"`
	UNITID     int     `json:"UNITID" fieldType:"int | NULL"`
	DBHAV      string  `json:"DBHAV" fieldType:"string | NULL"`
	GWTNWT     int     `json:"GWTNWT" fieldType:"int | NULL"`
	STAMPID    int     `json:"STAMPID" fieldType:"int | NULL"`
	ITYPE      int     `json:"ITYPE" fieldType:"int | NULL" Index:"I"`
	DEL        string  `json:"DEL" fieldType:"string | NULL"`
	TTYPE      int     `json:"TTYPE" fieldType:"int | NULL"`
	ALTTAG     int     `json:"ALTTAG" fieldType:"int | NULL"`
	SRTOTAL    float64 `json:"SRTOTAL" fieldType:"float | NULL"`
	SRDAMT     float64 `json:"SRDAMT" fieldType:"float | NULL"`
	SRSAMT     float64 `json:"SRSAMT" fieldType:"float | NULL"`
	SRLAMT     float64 `json:"SRLAMT" fieldType:"float | NULL"`
	SRMAMT     int     `json:"SRMAMT" fieldType:"int | NULL"`
	EXTOTAL    float64 `json:"EXTOTAL" fieldType:"float | NULL"`
	EXLAMT     float64 `json:"EXLAMT" fieldType:"float | NULL"`
	EXMAMT     int     `json:"EXMAMT" fieldType:"int | NULL"`
	EXDAMT     float64 `json:"EXDAMT" fieldType:"float | NULL"`
	EXSAMT     float64 `json:"EXSAMT" fieldType:"float | NULL"`
	MRPTYPE    int     `json:"MRPTYPE" fieldType:"int | NULL"`
	DIVIDERATE float64 `json:"DIVIDERATE" fieldType:"float | NULL"`
	INTORATE   float64 `json:"INTORATE" fieldType:"float | NULL"`
	PURLESS    float64 `json:"PURLESS" fieldType:"float | NULL"`
	DIS        float64 `json:"DIS" fieldType:"float | NULL"`
	SMDET      string  `json:"SMDET" fieldType:"string | NULL"`
	RTAG       int     `json:"RTAG" fieldType:"int | NULL"`
	MINQTY     float64 `json:"MINQTY" fieldType:"float | NULL"`
	POINT      float64 `json:"POINT" fieldType:"float | NULL"`
	RESDRATE   string  `json:"RESDRATE" fieldType:"string | NULL"`
	RESSRATE   string  `json:"RESSRATE" fieldType:"string | NULL"`
	RESMRATE   string  `json:"RESMRATE" fieldType:"string | NULL"`
	RESLBR     string  `json:"RESLBR" fieldType:"string | NULL"`
	Resigrp    string  `json:"resigrp" fieldType:"string | NULL"`
	CPOINT     float64 `json:"CPOINT" fieldType:"float | NULL"`
	IRLEVEL    int     `json:"IRLEVEL" fieldType:"int | NULL"`
	VRATE      float64 `json:"VRATE" fieldType:"float | NULL"`
	IGTUNCH    float64 `json:"IGTUNCH" fieldType:"float | NULL"`
	IGTGNUM    int     `json:"IGTGNUM" fieldType:"int | NULL"`
	MCXINTO    float64 `json:"MCXINTO" fieldType:"float | NULL"`
	MCXCOMEX   int     `json:"MCXCOMEX" fieldType:"int | NULL"`
	LBRDIS     int     `json:"LBRDIS" fieldType:"int | NULL"`
	ITRATE     int     `json:"ITRATE" fieldType:"int | NULL"`
	UPMRP      string  `json:"UPMRP" fieldType:"string | NULL"`
	IGALADD    int     `json:"IGALADD" fieldType:"int | NULL"`
	RESMRP     int     `json:"RESMRP" fieldType:"int | NULL"`
	RESLESS    int     `json:"RESLESS" fieldType:"int | NULL"`
	CONSUME    int     `json:"CONSUME" fieldType:"int | NULL"`
	UPSALE     string  `json:"UPSALE" fieldType:"string | NULL"`
	RENTACNO   int     `json:"RENTACNO" fieldType:"int | NULL"`
	PSGST      float64 `json:"PSGST" fieldType:"float | NULL"`
	PCGST      float64 `json:"PCGST" fieldType:"float | NULL"`
	PIGST      float64 `json:"PIGST" fieldType:"float | NULL"`
	HSNCODE    string  `json:"HSNCODE" fieldType:"string | NULL"`
	JHSNCODE   string  `json:"JHSNCODE" fieldType:"string | NULL"`
	GSTNAME    string  `json:"GSTNAME" fieldType:"string | NULL"`
	Defino     int     `json:"defino" fieldType:"int | NULL"`
}
