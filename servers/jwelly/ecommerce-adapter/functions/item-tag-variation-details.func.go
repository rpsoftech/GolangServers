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

// OPTIMIZED: Replaced OFFSET with Keyset Pagination (td.TGDID > ?) for massive JOIN performance gains
const SelectItemTagVariationDetailsQuery = "SELECT td.TGDID, td.TSNO, td.INO, td.STAMPID, td.WT, td.EXWT, td.remarks, td.PC, td.RATE, td.SPRICE, td.UNITID FROM tgd1 as td JOIN tgm1 as tg ON td.tsno = tg.tsno WHERE tg.vtgno != 0 AND tg.STATUS != 'ITM' AND tg.TPRE !='REP' AND td.TGDID > ? ORDER BY td.TGDID ASC LIMIT ?;"

// FIXED: Changed from 11 to 12 to match the exact number of inserted columns
const ServerItemTagVariationDetailsOfColumns = 12

const ItemTagVariationDetailsUpsertQuery = `INSERT INTO ItemTagVariationDetails (itemTagDetailsId, dItemTagId, dItemId, dSTAMPID, dWeight, dExWeight, dFinalWeight, dRemarks, dPcs, dRate, dSaleValue, dUnitId) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
dItemTagId = new_row.dItemTagId, 
dItemId = new_row.dItemId, 
dSTAMPID = new_row.dSTAMPID, 
dWeight = new_row.dWeight, 
dExWeight = new_row.dExWeight, 
dFinalWeight = new_row.dFinalWeight, 
dRemarks = new_row.dRemarks, 
dPcs = new_row.dPcs, 
dRate = new_row.dRate, 
dSaleValue = new_row.dSaleValue, 
dUnitId = new_row.dUnitId;`

func SyncItemTagVariationDetailsTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	batchSize := 2000 // 2000 * 12 = 24,000 placeholders (Well under 65k limit)
	lastId := 0       // High-performance cursor tracking TGDID
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Tag Variation Details using Keyset Pagination...")

	for {
		rowsFetched, rowsAffected, nextId, err := processItemTagVariationDetailsBatch(serverDb, erpDb, batchSize, lastId)
		if err != nil {
			log.Printf("Sync halted due to error after cursor ID %d: %v", lastId, err)
			break
		}

		totalRowsAffected += rowsAffected

		if rowsFetched < batchSize {
			break
		}

		lastId = nextId // Move cursor forward
	}

	log.Printf("Sync completed successfully! Total Duration: %s | Total Rows Affected Metric: %d\n", time.Since(initialTime), totalRowsAffected)
}

func processItemTagVariationDetailsBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, lastId int) (int, int64, int, error) {
	startTime := time.Now()

	// FIXED: Wrap Read and Write in context to prevent infinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Slightly longer for JOIN operations
	defer cancel()

	// 1. Fetch chunk from ERP using QueryContext and Cursor
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemTagVariationDetailsQuery, lastId, limit)
	if err != nil {
		return 0, 0, lastId, fmt.Errorf("failed reading from ERP (Limit: %d, Cursor: %d): %w", limit, lastId, err)
	}
	defer rows.Close()

	var results []*ecommerce_maintables.ItemTagVariationDetails
	maxId := lastId // Track highest TGDID in this batch

	for rows.Next() {
		row := &ecommerce_maintables.ItemTagVariationDetails{}
		err = rows.Scan(
			&row.ItemTagDetailsId, // Maps to td.TGDID
			&row.DItemTagId,
			&row.DItemId,
			&row.DSTAMPID,
			&row.DGrossWeight,
			&row.DExWeight,
			&row.DRemarks,
			&row.DPcs,
			&row.DRate,
			&row.DSaleValue,
			&row.DUnitId,
		)
		if err != nil {
			return 0, 0, lastId, fmt.Errorf("row scan error: %w", err)
		}

		// Calculate final weight
		row.DFinalWeight = row.DGrossWeight - row.DExWeight

		// Update cursor
		if int(row.ItemTagDetailsId) > maxId {
			maxId = int(row.ItemTagDetailsId)
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return 0, 0, lastId, fmt.Errorf("rows iteration error: %w", err)
	}

	fetchedCount := len(results)
	if fetchedCount == 0 {
		return 0, 0, maxId, nil
	}

	// 2. Prepare chunk for bulk upsert
	valueStrings := make([]string, 0, fetchedCount)
	valueArgs := make([]any, 0, fetchedCount*ServerItemTagVariationDetailsOfColumns)

	for _, detail := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?,?,?,?,?,?,?,?)")
		valueArgs = append(valueArgs,
			detail.ItemTagDetailsId,
			detail.DItemTagId,
			detail.DItemId,
			detail.DSTAMPID,
			detail.DGrossWeight,
			detail.DExWeight,
			detail.DFinalWeight, // Calculated value injected safely
			detail.DRemarks,
			detail.DPcs,
			detail.DRate,
			detail.DSaleValue,
			detail.DUnitId,
		)
	}

	finalQuery := fmt.Sprintf(ItemTagVariationDetailsUpsertQuery, strings.Join(valueStrings, ", "))

	// 3. Execute bulk upsert
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		return fetchedCount, 0, maxId, fmt.Errorf("bulk upsert failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fetchedCount, 0, maxId, fmt.Errorf("could not get rows affected: %w", err)
	}

	log.Printf("Batch processed: %d rows fetched | %d rows affected | Duration: %s", fetchedCount, rowsAffected, time.Since(startTime))

	return fetchedCount, rowsAffected, maxId, nil
}
