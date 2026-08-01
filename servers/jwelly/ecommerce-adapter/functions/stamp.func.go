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

const SelectStampQuery = "SELECT STAMPID, STAMP, STUNCH, STKTUNCH FROM stamp;"
const ServerStampOfColumns = 4

// FIXED: Removed the redundant stampId=new_row.stampId from the UPDATE clause
const StampUpsertQuery = `INSERT INTO Stamp (stampId, STAMP, tunch, stockTunch) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
STAMP = new_row.STAMP, 
tunch = new_row.tunch, 
stockTunch = new_row.stockTunch`

func SyncStampTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	// 10 seconds is plenty of time for a small lookup table
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// FIXED: Bound query to context
	rows, err := erpDb.Db.QueryContext(ctx, SelectStampQuery)
	if err != nil {
		log.Printf("Error executing query to read Stamp from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.StampMainTable
	for rows.Next() {
		row := &ecommerce_maintables.StampMainTable{}
		err := rows.Scan(
			&row.StampId,
			&row.Stamp,
			&row.Tunch,
			&row.StockTunch,
		)
		if err != nil {
			log.Printf("Error scanning row in Stamp: %v\n", err)
			continue
		}
		results = append(results, row)
	}

	// FIXED: Catch mid-loop iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over Stamp rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for Stamp. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from Stamp in %s\n", len(results), time.Since(startTime))

	// FIXED: Standardized batching to prevent placeholder limit crashes
	batchSize := 5000 // 5000 rows * 4 columns = 20,000 placeholders
	totalRowsAffected := int64(0)

	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}

		batch := results[i:end]
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*ServerStampOfColumns)

		for _, stamp := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				stamp.StampId,
				stamp.Stamp,
				stamp.Tunch,
				stamp.StockTunch,
			)
		}

		finalQuery := fmt.Sprintf(StampUpsertQuery, strings.Join(valueStrings, ", "))

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

	log.Printf("Bulk upsert for Stamp successful! Total rows affected: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
