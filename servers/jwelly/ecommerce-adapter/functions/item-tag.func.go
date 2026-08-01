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

// OPTIMIZED: Added Keyset Pagination (tsno > ?) and ORDER BY tsno ASC LIMIT ?
const SelectItemTagQuery = "SELECT tsno, TGNO, vtgno, INO, TDATE FROM tgm1 WHERE vtgno != 0 AND STATUS != 'ITM' AND TPRE != 'REP' AND tsno > ? ORDER BY tsno ASC LIMIT ?;"
const ServerItemTagOfColumns = 5
const ItemTagUpsertQuery = `INSERT INTO ItemsTag (itemTagId, itemTag, itemVTagId, tItemId, tagCreatedDate) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
itemVTagId = new_row.itemVTagId, 
tItemId = new_row.tItemId, 
tagCreatedDate = new_row.tagCreatedDate`

func SyncItemTagTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	batchSize := 5000 // 5000 * 5 = 25,000 placeholders (Well under 65k limit)
	lastTsno := 0     // High-performance cursor tracking tsno
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Tag Main using Keyset Pagination...")

	for {
		rowsFetched, rowsAffected, nextTsno, err := processItemTagBatch(serverDb, erpDb, batchSize, lastTsno)
		if err != nil {
			log.Printf("Sync halted due to error after cursor tsno %d: %v", lastTsno, err)
			break // Stop the loop on error, but keep the app alive
		}

		totalRowsAffected += rowsAffected

		if rowsFetched < batchSize {
			break // Reached the end of the table
		}

		lastTsno = nextTsno // Move cursor forward for the next iteration
	}

	log.Printf("Sync completed successfully! Total Duration: %s | Total Rows Affected Metric: %d\n", time.Since(initialTime), totalRowsAffected)
}

func processItemTagBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, lastTsno int) (int, int64, int, error) {
	startTime := time.Now()

	// FIXED: Wrap Read and Write in context to prevent infinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Fetch chunk from ERP using QueryContext and Cursor
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemTagQuery, lastTsno, limit)
	if err != nil {
		return 0, 0, lastTsno, fmt.Errorf("failed reading from ERP (Limit: %d, Cursor: %d): %w", limit, lastTsno, err)
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.ItemTagMainTable
	maxTsno := lastTsno // Track highest tsno in this batch

	for rows.Next() {
		row := &ecommerce_maintables.ItemTagMainTable{}
		err = rows.Scan(
			&row.ItemTagId, // Maps to tsno
			&row.ItemTag,
			&row.ItemVTagId,
			&row.TItemId,
			&row.TagCreatedDate,
		)
		if err != nil {
			return 0, 0, lastTsno, fmt.Errorf("row scan error: %w", err)
		}

		// Update cursor
		if int(row.ItemTagId) > maxTsno {
			maxTsno = int(row.ItemTagId)
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
	valueArgs := make([]interface{}, 0, fetchedCount*ServerItemTagOfColumns)

	for _, tag := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?)")
		valueArgs = append(valueArgs,
			tag.ItemTagId,
			tag.ItemTag,
			tag.ItemVTagId,
			tag.TItemId,
			tag.TagCreatedDate,
		)
	}

	finalQuery := fmt.Sprintf(ItemTagUpsertQuery, strings.Join(valueStrings, ", "))

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
