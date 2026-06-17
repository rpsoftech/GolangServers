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

const SelectItemTagVariationDetailsQuery = "SELECT td.TGDID, td.TSNO, td.INO, td.STAMPID, td.WT,td.EXWT,td.remarks, td.PC, td.RATE, td.SPRICE, td.UNITID FROM tgd1 as td join tgm1 as tg on td.tsno = tg.tsno where tg.vtgno != 0 and tg.STATUS != 'ITM' and tg.TPRE !='REP' ORDER BY td.TSNO LIMIT ? OFFSET ?;"
const ServerItemTagVariationDetailsOfColumns = 11
const ItemTagVariationDetailsUpsertQuery = `INSERT INTO ItemTagVariationDetails (itemTagDetailsId,dItemTagId,dItemId,dSTAMPID,dWeight,dExWeight,dFinalWeight,dRemarks,dPcs,dRate,dSaleValue,dUnitId) VALUES %s AS new_row ON DUPLICATE KEY UPDATE dItemTagId = new_row.itemTagDetailsId,dItemId = new_row.itemTagDetailsId,dSTAMPID = new_row.itemTagDetailsId,dWeight = new_row.itemTagDetailsId,dExWeight = new_row.itemTagDetailsId,dFinalWeight = new_row.itemTagDetailsId,dRemarks = new_row.itemTagDetailsId,dPcs = new_row.itemTagDetailsId,dRate = new_row.itemTagDetailsId,dSaleValue = new_row.itemTagDetailsId,dUnitId = new_row.itemTagDetailsId`

func SyncItemTagVariationDetailsTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	batchSize := 1000 // You can change this to 2000 if your server handles it well
	offset := 0
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Tag Variations...")

	for {
		// Process a single batch
		rowsFetched, rowsAffected, err := processItemTagVariationDetailsBatch(serverDb, erpDb, batchSize, offset)
		if err != nil {
			log.Printf("Sync halted due to error at offset %d: %v", offset, err)
			break // Stop the loop on error, but keep the app alive
		}

		totalRowsAffected += rowsAffected

		// If we fetched fewer rows than the batch size, we've reached the end of the table
		if rowsFetched < batchSize {
			break
		}

		// Move the offset forward for the next iteration
		offset += batchSize
	}

	log.Printf("Sync completed successfully! Total Duration: %s | Total Rows Affected Metric: %d\n", time.Since(initialTime), totalRowsAffected)
}

func processItemTagVariationDetailsBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, offset int) (int, int64, error) {
	startTime := time.Now()

	// 1. Fetch chunk from ERP
	rows, err := erpDb.Db.Query(SelectItemTagVariationDetailsQuery, limit, offset)
	if err != nil {
		return 0, 0, fmt.Errorf("failed reading from ERP (Limit: %d, Offset: %d): %w", limit, offset, err)
	}
	// CRITICAL: Ensure rows are closed immediately after this batch finishes
	defer rows.Close()

	var results []*ecommerce_maintables.ItemTagVariationDetails

	for rows.Next() {
		row := &ecommerce_maintables.ItemTagVariationDetails{}
		err = rows.Scan(
			&row.ItemTagDetailsId,
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
			return 0, 0, fmt.Errorf("row scan error: %w", err)
		}

		// Calculate the boolean status based on the string
		row.DFinalWeight = row.DGrossWeight - row.DExWeight
		results = append(results, row)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	fetchedCount := len(results)
	if fetchedCount == 0 {
		return 0, 0, nil // Nothing to process
	}

	// 2. Prepare chunk for bulk upsert
	valueStrings := make([]string, 0, fetchedCount)
	valueArgs := make([]interface{}, 0, fetchedCount*ServerItemTagVariationDetailsOfColumns)

	for _, detail := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?,?,?,?,?,?,?,?)")
		valueArgs = append(valueArgs,
			detail.ItemTagDetailsId,
			detail.DItemTagId,
			detail.DItemId,
			detail.DSTAMPID,
			detail.DGrossWeight,
			detail.DExWeight,
			detail.DFinalWeight,
			detail.DRemarks,
			detail.DPcs,
			detail.DRate,
			detail.DSaleValue,
			detail.DUnitId,
		)
	}

	finalQuery := fmt.Sprintf(ItemTagVariationDetailsUpsertQuery, strings.Join(valueStrings, ", "))

	// 3. Execute bulk upsert for this specific chunk
	// Give each chunk a reasonable 10-second timeout to execute
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		return fetchedCount, 0, fmt.Errorf("bulk upsert failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fetchedCount, 0, fmt.Errorf("could not get rows affected: %w", err)
	}

	log.Printf("Batch processed: %d rows fetched | %d rows affected | Duration: %s", fetchedCount, rowsAffected, time.Since(startTime))

	return fetchedCount, rowsAffected, nil
}
