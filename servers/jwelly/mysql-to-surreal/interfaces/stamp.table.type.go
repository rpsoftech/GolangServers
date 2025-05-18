package mysql_to_surreal_interfaces

type StampTableStruct struct {
	SurrealId  int     `json:"id" Index:"U"`
	STAMPID    int     `json:"STAMPID" Index:"U" fieldType:"int"`
	STAMP      string  `json:"STAMP" Index:"I" fieldType:"string | NULL"`
	STUNCH     float64 `json:"STUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	GROUP      string  `json:"GROUP" fieldType:"string | NULL"`
	REM1       string  `json:"REM1" fieldType:"string | NULL"`
	REM2       string  `json:"REM2" fieldType:"string | NULL"`
	DIAST      int     `json:"DIAST" fieldType:"int" defaultValue:"0"`
	PRATE      float64 `json:"PRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	SRATE      float64 `json:"SRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	DIVIDE     float64 `json:"DIVIDE" fieldType:"float | NULL" defaultValue:"0.00"`
	MRATE      int     `json:"MRATE" fieldType:"int" defaultValue:"0"`
	Colour     string  `json:"colour" fieldType:"string | NULL"`
	Clarity    string  `json:"clarity" fieldType:"string | NULL"`
	Shape      string  `json:"shape" fieldType:"string | NULL"`
	SIZE       string  `json:"SIZE" fieldType:"string | NULL"`
	TUNCH      float64 `json:"TUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	INO        int     `json:"INO" fieldType:"int" defaultValue:"0"`
	STKRATE    float64 `json:"STKRATE" fieldType:"float | NULL" defaultValue:"0.00"`
	PTUNCH     float64 `json:"PTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	STKTUNCH   float64 `json:"STKTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	BHAVTUNCH  float64 `json:"BHAVTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	TUNCHF     float64 `json:"TUNCHF" fieldType:"float | NULL" defaultValue:"0.00"`
	TUNCHT     float64 `json:"TUNCHT" fieldType:"float | NULL" defaultValue:"0.00"`
	Pcwt       float64 `json:"pcwt" fieldType:"float | NULL" defaultValue:"0.00"`
	ASSORT     int     `json:"ASSORT" fieldType:"int" defaultValue:"0"`
	PPROFIT    float64 `json:"PPROFIT" fieldType:"float | NULL" defaultValue:"0.00"`
	VPROFIT    int     `json:"VPROFIT" fieldType:"int" defaultValue:"0"`
	Pname      string  `json:"pname" fieldType:"string | NULL"`
	SIZEID     int     `json:"SIZEID" fieldType:"int" defaultValue:"0"`
	SRNO       int     `json:"SRNO" fieldType:"int" defaultValue:"0"`
	DISPBHAV   string  `json:"DISPBHAV" fieldType:"string | NULL"`
	PBHAVTUNCH float64 `json:"PBHAVTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	TSTAMP     string  `json:"TSTAMP" fieldType:"string | NULL"`
	ADVPER     float64 `json:"ADVPER" fieldType:"float | NULL" defaultValue:"0.00"`
	OLDLESS    float64 `json:"OLDLESS" fieldType:"float | NULL" defaultValue:"0.00"`
	OLDTUNCH   float64 `json:"OLDTUNCH" fieldType:"float | NULL" defaultValue:"0.00"`
	HIDE       string  `json:"HIDE" fieldType:"string | NULL"`
	Bhavroff   int     `json:"bhavroff" fieldType:"int" defaultValue:"0"`
	Webstamp   string  `json:"webstamp" fieldType:"string | NULL"`
}
