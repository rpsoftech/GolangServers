package mysql_to_surreal_functions

import (
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

const TgMasterTableName = "tg_master"

var (
	GetTgMasterSqlCommand = ""
)

func init() {
	GetTgMasterSqlCommand = fmt.Sprintf("SELECT * FROM %s", TgMasterTableName)

}

const addTgMasterQuery = `DEFINE TABLE  IF NOT EXISTS %s TYPE NORMAL as
    SELECT 
        SurrealINO.IDESC AS idesc,
        TGNO AS tgno,
        vtgno AS vtgno,
        STATUS,
        SurrealINO.tpre AS tpre,
        REMARKS AS remarks,
        TDATE AS tdate,
        GWT AS gwt,
        (GWT - SWT) AS lesswt,
        SWT AS wt,
        SDIAWT AS sdiawt,
        (SSTNWT / 5) AS sstnwt,
        GOLDWT AS goldwt,
        SILWT AS silwt,
        PLATWT AS platwt,
        OTHWT AS othwt,
        SLBR AS slbr,
        slbr2 AS slbr2,
        SLBR3 AS slbr3,
        (IF STATUS = ' ' OR STATUS = '' THEN 'IN STOCK'  
        ELSE IF STATUS = 'AI' THEN 'APPROVAL'  
        ELSE IF STATUS = 'ITM' THEN 'MODIFY'  
        ELSE IF STATUS != ' ' AND STATUS != 'AI' AND STATUS != 'ITM' THEN 'SOLD'
        ELSE {"N/A"} END) as status,
        STUNCH AS stunch,
        SWSTG AS swstg,
        SBEEDS AS sbeeds,
        (calmrp - ((((csmamt + cslamt) + SDAMT) + SSAMT) - cstdmamt)) AS sothers,
        othrem AS othrem,
        design AS design,
        karigar AS karigar,
        MRATE AS mrate,
        GWT1 AS gwt1,
        GWT2 AS gwt2,
        SurrealSTAMPID.STAMP AS stamp,
        size AS size,
        QUALITY AS quality,
        COLOUR AS color,
        CLARITY AS clarity,
        -- REPEAT(' ', 20) AS site,
        LINKTGNO AS linktgno,
        DIAPC AS diapc,
        STNPC AS stnpc,
        MRP1 AS mrp1,
        MRP2 AS mrp2,
        SDAMT AS sdamt,
        SSAMT AS ssamt,
        SLAMT AS slamt,
        SMAMT AS smamt,
        0 AS scamt,
        MRP AS mrp,
        hm AS hm,
        certno AS certno,
        SPOLISH AS spolish,
        SPOLISHWT AS spolishwt,
        PC AS pc,
        calmrp AS salemrp,
        DIAWT1 AS diawt1,
        DIAWT2 AS diawt2,
        STNWT1 AS stnwt1,
        STNWT2 AS stnwt2,
        LAKHWT AS lakhwt,
        SHAPE AS shape,
        SurrealINO.SurrealIGROUPID.ITNAME AS itname,
        SurrealINO.category AS category,
        SurrealINO.gcode AS gcode,
        tsno AS tsno,
        ' ' AS designid,
        SurrealINO.PNAME AS pname,
        -- REPEAT(' ', 20) AS branch,
        0 AS tgmastid,
        -- REPEAT(' ', 20) AS diacut,
        -- REPEAT(' ', 20) AS diapol,
        -- REPEAT(' ', 20) AS diasymm,
        PDIS AS pdis,
        SAMPLE AS cindex,
        PHOTOPATH AS photopath,
        SurrealINO.igroup AS igroup,
        RFID AS rfid,
        gst AS gst,
        (gst + calmrp) AS billamt,
        BILLTYPE AS billtype,
        SAMPLE AS tgimage,
        INO AS ino,
        (csmamt - cstdmamt) AS csmamt,
        cslamt AS cslamt,
        SurrealINO.itype AS itype,
        SurrealUNITID.UNIT AS unit,
        srate AS srate,
        pgst AS pgst,
        SITEID AS siteid
    FROM tgm1;`

func removeAndInsertTgMaster(c *ConfigWithConnection) {
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf("Remove Table %s", TgMasterTableName), nil)
	surrealdb.Query[any](c.DbConnections.SurrealDbConncetion.Db, fmt.Sprintf(addTgMasterQuery, TgMasterTableName), nil)
	fmt.Printf("Removed And Created %s\n", TgMasterTableName)
}
