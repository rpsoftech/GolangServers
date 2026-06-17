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

const SelectItemTagVariationQuery = "SELECT tsno, STAMPID, GWT, LESSWT, WT, STATUS, TUNCH, WSTG, STUNCH, SWSTG, karigar FROM tgm1 where vtgno != 0 and STATUS != 'ITM' and TPRE !='REP' ORDER BY tsno LIMIT ? OFFSET ?;"
const ServerItemTagVariationOfColumns = 11
const ItemTagVariationUpsertQuery = `INSERT INTO ItemTagVariation (vTagId,vStampId,vGrossWt,vLessWeight,vNetWt,vStatus,vTunch,vWstg,vSellTunch,vSellWstg,vKarigarCode) VALUES %s AS new_row ON DUPLICATE KEY UPDATE vTagId = new_row.vTagId,vStampId = new_row.vStampId,vGrossWt = new_row.vGrossWt,vLessWeight = new_row.vLessWeight,vNetWt = new_row.vNetWt,vStatus = new_row.vStatus,vTunch = new_row.vTunch,vWstg = new_row.vWstg,vSellTunch = new_row.vSellTunch,vSellWstg = new_row.vSellWstg,vKarigarCode = new_row.vKarigarCode`

// func SyncItemTagVariationTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
// 	initialTime := time.Now()
// 	startTime := initialTime
// 	batchSize := 1000 // You can change this to 2000 if your server handles it well
// 	offset := 0
// 	var totalRowsAffected int64 = 0
// 	rows, err := erpDb.Db.Query(SelectItemTagVariationQuery)
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	if err != nil {
// 		log.Printf("Error in Reading Unit From Server")
// 		log.Println(err.Error())
// 		return
// 	}
// 	var results []*ecommerce_maintables.TagVariationMainTable = []*ecommerce_maintables.TagVariationMainTable{}
// 	for rows.Next() {
// 		row := &ecommerce_maintables.TagVariationMainTable{}
// 		err = rows.Scan(
// 			&row.VTagId,
// 			&row.VStampId,
// 			&row.VGrossWt,
// 			&row.VLessWeight,
// 			&row.VNetWt,
// 			&row.VStatusString,
// 			&row.VTunch,
// 			&row.VWstg,
// 			&row.VSellTunch,
// 			&row.VSellWstg,
// 			&row.VKarigarCode,
// 		)
// 		if row.VStatusString == "" || row.VStatusString == "AI" {
// 			row.VStatus = true
// 		} else {
// 			row.VStatus = false
// 		}
// 		results = append(results, row)
// 	}
// 	log.Printf("Fetched Total %d rows from Item Tag Variation in Duration of %s\n", len(results), time.Since(startTime))
// 	valueStrings := make([]string, 0, len(results))
// 	valueArgs := make([]interface{}, 0, len(results)*ServerItemTagVariationOfColumns)

// 	for _, variation := range results {
// 		valueStrings = append(valueStrings, "(?,?,?,?,?,?,?,?,?,?,?)")
// 		valueArgs = append(valueArgs,
// 			variation.VTagId,
// 			variation.VStampId,
// 			variation.VGrossWt,
// 			variation.VLessWeight,
// 			variation.VNetWt,
// 			variation.VStatusString,
// 			variation.VTunch,
// 			variation.VWstg,
// 			variation.VSellTunch,
// 			variation.VSellWstg,
// 			variation.VKarigarCode,
// 		)
// 	}
// 	finalQuery := fmt.Sprintf(ItemTagVariationUpsertQuery, strings.Join(valueStrings, ", "))
// 	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
// 	if err != nil {
// 		log.Fatalf("Bulk upsert failed: %v", err)
// 		log.Fatalf("Query : %s", finalQuery)
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Printf("Bulk upsert for Item Tag Variation successful!  Rows affected metric: %d\n", rowsAffected)

// }
func SyncItemTagVariationTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	batchSize := 1000 // You can change this to 2000 if your server handles it well
	offset := 0
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Tag Variations...")

	for {
		// Process a single batch
		rowsFetched, rowsAffected, err := processItemTagVariationBatch(serverDb, erpDb, batchSize, offset)
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

func processItemTagVariationBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, offset int) (int, int64, error) {
	startTime := time.Now()

	// 1. Fetch chunk from ERP
	rows, err := erpDb.Db.Query(SelectItemTagVariationQuery, limit, offset)
	if err != nil {
		return 0, 0, fmt.Errorf("failed reading from ERP (Limit: %d, Offset: %d): %w", limit, offset, err)
	}
	// CRITICAL: Ensure rows are closed immediately after this batch finishes
	defer rows.Close()

	var results []*ecommerce_maintables.TagVariationMainTable

	for rows.Next() {
		row := &ecommerce_maintables.TagVariationMainTable{}
		err = rows.Scan(
			&row.VTagId,
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
			return 0, 0, fmt.Errorf("row scan error: %w", err)
		}

		// Calculate the boolean status based on the string
		if row.VStatusString == "" || row.VStatusString == "AI" {
			row.VStatus = true
		} else {
			row.VStatus = false
		}

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
	valueArgs := make([]interface{}, 0, fetchedCount*ServerItemTagVariationOfColumns)

	for _, variation := range results {
		valueStrings = append(valueStrings, "(?,?,?,?,?,?,?,?,?,?,?)")
		valueArgs = append(valueArgs,
			variation.VTagId,
			variation.VStampId,
			variation.VGrossWt,
			variation.VLessWeight,
			variation.VNetWt,
			variation.VStatus, // FIX: Inserting the calculated boolean, not the raw string
			variation.VTunch,
			variation.VWstg,
			variation.VSellTunch,
			variation.VSellWstg,
			variation.VKarigarCode,
		)
	}

	finalQuery := fmt.Sprintf(ItemTagVariationUpsertQuery, strings.Join(valueStrings, ", "))

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
