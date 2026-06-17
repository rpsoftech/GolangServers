package ecommerce_functions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ecommerce_maintables "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/MainTables"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

const SelectUnitTableQuery = "select UNITID,UNIT,`DECIMAL` from units;"
const ServerUnitTable = "ItemUnit"
const UniTableNumberOfColumns = 3
const UpsertUnitTableQuery = `INSERT INTO ItemUnit (itemUnitId, itemUnit, itemDecimal) VALUES %s AS new_row ON DUPLICATE KEY UPDATE itemUnit = new_row.itemUnit, itemDecimal = new_row.itemDecimal`

func SyncItemUnitTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectUnitTableQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.ItemUnitMainTable = []*ecommerce_maintables.ItemUnitMainTable{}
	for rows.Next() {
		row := &ecommerce_maintables.ItemUnitMainTable{}
		err = rows.Scan(
			&row.ItemUnitId,
			&row.ItemUnit,
			&row.ItemDecimal,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Units in Duration of %s\n", len(results), time.Since(startTime))

	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*UniTableNumberOfColumns)
	for _, unit := range results {
		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, unit.ItemUnitId, unit.ItemUnit, unit.ItemDecimal)
	}
	finalQuery := fmt.Sprintf(UpsertUnitTableQuery, strings.Join(valueStrings, ", "))
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert successful! Rows affected metric: %d\n", rowsAffected)

}
