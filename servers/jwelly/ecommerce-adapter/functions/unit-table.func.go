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

const SelectUnitTableQuery = "SELECT UNITID, UNIT, `DECIMAL` FROM units;"
const ServerUnitTable = "ItemUnit"
const UniTableNumberOfColumns = 3
const UpsertUnitTableQuery = `INSERT INTO ItemUnit (itemUnitId, itemUnit, itemDecimal) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
itemUnit = new_row.itemUnit, 
itemDecimal = new_row.itemDecimal`

func SyncItemUnitTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// FIXED: Bound query to context
	rows, err := erpDb.Db.QueryContext(ctx, SelectUnitTableQuery)
	if err != nil {
		log.Printf("Error executing query to read Unit from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.ItemUnitMainTable
	for rows.Next() {
		row := &ecommerce_maintables.ItemUnitMainTable{}
		err := rows.Scan(
			&row.ItemUnitId,
			&row.ItemUnit,
			&row.ItemDecimal,
		)
		if err != nil {
			log.Printf("Error scanning row in Unit: %v\n", err)
			continue
		}
		results = append(results, row)
	}

	// FIXED: Catch mid-loop iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over Unit rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for Unit. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from Units in Duration of %s\n", len(results), time.Since(startTime))

	// FIXED: Standardized batching loop
	batchSize := 5000
	totalRowsAffected := int64(0)

	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}

		batch := results[i:end]
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*UniTableNumberOfColumns)

		for _, unit := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, unit.ItemUnitId, unit.ItemUnit, unit.ItemDecimal)
		}

		finalQuery := fmt.Sprintf(UpsertUnitTableQuery, strings.Join(valueStrings, ", "))

		// FIXED: Graceful error handling instead of Fatalf
		result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
		if err != nil {
			log.Printf("Bulk upsert failed for batch %d to %d: %v\n", i, end, err)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Could not fetch rows affected for batch: %v\n", err)
		} else {
			totalRowsAffected += rowsAffected
		}
	}

	log.Printf("Bulk upsert for Units successful! Total rows affected: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
