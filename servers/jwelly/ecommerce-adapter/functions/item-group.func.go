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

const SelectItemGroupQuery = "SELECT igroupid, IGROUP, PNAME, UNITID FROM igroup;"
const ServerItemGroupTable = "ItemGroup"
const ServerItemGroupOfColumns = 4
const ItemGroupUpsertQuery = `INSERT INTO ItemGroup (itemGroupId, itemGroup, itemPrintName, itemGroupUnitId) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
itemGroup = new_row.itemGroup, 
itemPrintName = new_row.itemPrintName, 
itemGroupUnitId = new_row.itemGroupUnitId`

func SyncItemGroupTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	// 30s timeout to handle potentially slower network I/O during batching
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// FIXED: Bound query to context
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemGroupQuery)
	if err != nil {
		log.Printf("Error executing query to read ItemGroup from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool memory leaks
	defer rows.Close()

	var results []*ecommerce_maintables.ItemGroupMainTable
	for rows.Next() {
		row := &ecommerce_maintables.ItemGroupMainTable{}
		err = rows.Scan(
			&row.ItemGroupId,
			&row.ItemGroup,
			&row.ItemPrintName,
			&row.ItemUnitId,
		)
		if err != nil {
			log.Printf("Error scanning row in ItemGroup: %v\n", err)
			continue
		}
		results = append(results, row)
	}

	// FIXED: Catch mid-loop iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over ItemGroup rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for ItemGroup. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from ItemGroup in %s\n", len(results), time.Since(startTime))

	// FIXED: Batching to prevent 65,535 placeholder limit crashes
	batchSize := 5000 // 5000 rows * 4 columns = 20,000 placeholders
	totalRowsAffected := int64(0)

	for i := 0; i < len(results); i += batchSize {
		end := min(i+batchSize, len(results))

		batch := results[i:end]
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]any, 0, len(batch)*ServerItemGroupOfColumns)

		for _, group := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			valueArgs = append(valueArgs, group.ItemGroupId, group.ItemGroup, group.ItemPrintName, group.ItemUnitId)
		}

		finalQuery := fmt.Sprintf(ItemGroupUpsertQuery, strings.Join(valueStrings, ", "))

		// FIXED: Replaced log.Fatalf with log.Printf so cron jobs gracefully fail without crashing the API
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

	log.Printf("Bulk upsert for ItemGroup successful! Total rows affected: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
