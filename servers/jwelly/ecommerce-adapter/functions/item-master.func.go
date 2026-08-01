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

const SelectItemMasterQuery = "SELECT ino, IDESC, IGROUPID, PNAME, UNITID, tpre FROM itemmast;"
const ServerItemMasterTable = "ItemMaster"

// FIXED: Changed from 4 to 6 to perfectly match the number of inserted columns
const ServerItemMasterOfColumns = 6

const ItemMasterUpsertQuery = `INSERT INTO ItemMaster (itemId, itemName, iGroupId, itemPrintName, iUnitId, itemTagPrefix) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
itemName = new_row.itemName, 
iGroupId = new_row.iGroupId, 
itemPrintName = new_row.itemPrintName, 
iUnitId = new_row.iUnitId, 
itemTagPrefix = new_row.itemTagPrefix`

func SyncItemMasterTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	// Increased timeout for batching
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// FIXED: Bound query to context
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemMasterQuery)
	if err != nil {
		log.Printf("Error executing query to read ItemMaster from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.ItemMasterTable
	for rows.Next() {
		row := &ecommerce_maintables.ItemMasterTable{}
		err = rows.Scan(
			&row.ItemId,
			&row.ItemName,
			&row.IGroupId,
			&row.ItemPrintName,
			&row.IUnitId,
			&row.ItemTagPrefix,
		)
		if err != nil {
			log.Printf("Error scanning row in ItemMaster: %v\n", err)
			continue
		}
		results = append(results, row)
	}

	// FIXED: Catch mid-loop iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over ItemMaster rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for ItemMaster. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from Item Master in %s\n", len(results), time.Since(startTime))

	// FIXED: Batching to prevent 65,535 placeholder limit crashes
	batchSize := 2000 // 2000 rows * 6 columns = 12,000 placeholders per batch
	totalRowsAffected := int64(0)

	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}

		batch := results[i:end]
		valueStrings := make([]string, 0, len(batch))

		// Memory allocation is now perfectly sized
		valueArgs := make([]interface{}, 0, len(batch)*ServerItemMasterOfColumns)

		for _, item := range batch {
			valueStrings = append(valueStrings, "(?,?,?,?,?,?)")
			valueArgs = append(valueArgs,
				item.ItemId,
				item.ItemName,
				item.IGroupId,
				item.ItemPrintName,
				item.IUnitId,
				item.ItemTagPrefix,
			)
		}

		finalQuery := fmt.Sprintf(ItemMasterUpsertQuery, strings.Join(valueStrings, ", "))

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

	log.Printf("Bulk upsert for Item Master successful! Total rows affected: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
