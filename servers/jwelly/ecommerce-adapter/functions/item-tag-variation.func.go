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

// OPTIMIZED: Replaced OFFSET with Keyset Pagination (tsno > ?) for massive performance gains
const SelectItemTagVariationQuery = "SELECT tsno, STAMPID, GWT, LESSWT, WT, STATUS, TUNCH, WSTG, STUNCH, SWSTG, karigar FROM tgm1 WHERE vtgno != 0 AND STATUS != 'ITM' AND TPRE != 'REP' AND tsno > ? ORDER BY tsno ASC LIMIT ?;"
const ServerItemTagVariationOfColumns = 11
const ItemTagVariationUpsertQuery = `INSERT INTO ItemTagVariation (vTagId, vStampId, vGrossWt, vLessWeight, vNetWt, vStatus, vTunch, vWstg, vSellTunch, vSellWstg, vKarigarCode) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
vGrossWt = new_row.vGrossWt, 
vLessWeight = new_row.vLessWeight, 
vNetWt = new_row.vNetWt, 
vStatus = new_row.vStatus, 
vTunch = new_row.vTunch, 
vWstg = new_row.vWstg, 
vSellTunch = new_row.vSellTunch, 
vSellWstg = new_row.vSellWstg, 
vKarigarCode = new_row.vKarigarCode`

func SyncItemTagVariationTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	batchSize := 2000 // Safe to increase because 2000 * 11 = 22,000 placeholders (limit is 65k)
	lastTsno := 0     // Acts as our high-performance cursor
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Tag Variations using Keyset Pagination...")

	for {
		rowsFetched, rowsAffected, nextTsno, err := processItemTagVariationBatch(serverDb, erpDb, batchSize, lastTsno)
		if err != nil {
			log.Printf("Sync halted due to error after cursor tsno %d: %v", lastTsno, err)
			break
		}

		totalRowsAffected += rowsAffected

		if rowsFetched < batchSize {
			break // End of table reached
		}

		lastTsno = nextTsno // Move cursor forward
	}

	log.Printf("Sync completed successfully! Total Duration: %s | Total Rows Affected Metric: %d\n", time.Since(initialTime), totalRowsAffected)
}

func processItemTagVariationBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, lastTsno int) (int, int64, int, error) {
	startTime := time.Now()

	// FIXED: Wrapping BOTH Read and Write in a context to prevent infinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Fetch chunk from ERP using QueryContext and the cursor
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemTagVariationQuery, lastTsno, limit)
	if err != nil {
		return 0, 0, lastTsno, fmt.Errorf("failed reading from ERP (Limit: %d, Cursor: %d): %w", limit, lastTsno, err)
	}
	defer rows.Close()

	var results []*ecommerce_maintables.TagVariationMainTable
	maxTsno := lastTsno // Track the highest ID in this batch

	for rows.Next() {
		row := &ecommerce_maintables.TagVariationMainTable{}
		err = rows.Scan(
			&row.VTagId, // Assuming VTagId receives the 'tsno' from the ERP query based on your struct mapping
			&row.VStampId,
			&row.VGrossWt,
			&row.VLessWeight,
			&row.VNetWt,
			&row.VStatusString,
			&row.VTunch,
			&row.VWstg,
			&row.VSellTunch,
			&row.VSellWstg,
			&row.VKarigarCode,
		)
		if err != nil {
			return 0, 0, lastTsno, fmt.Errorf("row scan error: %w", err)
		}

		// Parse status
		if row.VStatusString == "" || row.VStatusString == "AI" {
			row.VStatus = true
		} else {
			row.VStatus = false
		}

		// Update our cursor
		if int(row.VTagId) > maxTsno {
			maxTsno = int(row.VTagId)
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return 0, 0, lastTsno, fmt.Errorf("rows iteration error: %w", err)
	}

	fetchedCount := len(results)
	if fetchedCount == 0 {
		return 0, 0, maxTsno, nil
	}

	// 2. Prepare chunk for bulk upsert
	valueStrings := make([]string, 0, fetchedCount)
	valueArgs := make([]interface{}, 0, fetchedCount*ServerItemTagVariationOfColumns)

	for _, variation := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?,?,?,?,?,?,?)")
		valueArgs = append(valueArgs,
			variation.VTagId,
			variation.VStampId,
			variation.VGrossWt,
			variation.VLessWeight,
			variation.VNetWt,
			variation.VStatus,
			variation.VTunch,
			variation.VWstg,
			variation.VSellTunch,
			variation.VSellWstg,
			variation.VKarigarCode,
		)
	}

	finalQuery := fmt.Sprintf(ItemTagVariationUpsertQuery, strings.Join(valueStrings, ", "))

	// 3. Execute bulk upsert
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		return fetchedCount, 0, maxTsno, fmt.Errorf("bulk upsert failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fetchedCount, 0, maxTsno, fmt.Errorf("could not get rows affected: %w", err)
	}

	log.Printf("Batch processed: %d rows fetched | %d rows affected | Duration: %s", fetchedCount, rowsAffected, time.Since(startTime))

	return fetchedCount, rowsAffected, maxTsno, nil
}
