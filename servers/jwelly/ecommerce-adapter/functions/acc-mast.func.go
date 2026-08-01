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

const SelectAccountMasterQuery = "SELECT acno, PREFIX, CNAME, PNAME, GROUPID, CITY, LOCATION, MOBILE, PHONE FROM `accmast`;"
const ServerAccMstOfColumns = 9
const upsertAccountMasterPlaceHolder = "(?, ?, ?, ?, ?, ?, ?, ?, ?)"
const AccountMasterUpsertQuery = `INSERT INTO AccountMaster (acno, prefix, Name, pName, aGroupId, city, location, mobile, phone) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
prefix = new_row.prefix, 
Name = new_row.Name, 
pName = new_row.pName, 
aGroupId = new_row.aGroupId, 
city = new_row.city, 
location = new_row.location, 
mobile = new_row.mobile, 
phone = new_row.phone`

func SyncAccountMasterTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	// Increased timeout to 30s to account for large batch inserts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// FIXED: Enforce context timeout
	rows, err := erpDb.Db.QueryContext(ctx, SelectAccountMasterQuery)
	if err != nil {
		log.Printf("Error executing query to read AccountMaster from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.AccountMasterTable
	for rows.Next() {
		row := &ecommerce_maintables.AccountMasterTable{}
		err = rows.Scan(
			&row.Acno,
			&row.Prefix,
			&row.Name,
			&row.PName,
			&row.AGroupId,
			&row.City,
			&row.Location,
			&row.Mobile,
			&row.Phone,
		)
		if err != nil {
			log.Printf("Error scanning row in AccountMaster: %v\n", err)
			continue // Skip corrupted rows instead of failing the whole batch
		}
		results = append(results, row)
	}

	// FIXED: Catch errors that occurred during the loop iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over AccountMaster rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for AccountMaster. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from AccountMaster in %s\n", len(results), time.Since(startTime))

	// FIXED: Implemented Chunking to bypass MySQL's 65,535 placeholder limit
	batchSize := 2000 // 2000 rows * 9 columns = 18,000 placeholders (Well under the 65k limit)
	totalRowsAffected := int64(0)

	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}

		batch := results[i:end]
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*ServerAccMstOfColumns)

		for _, account := range batch {
			valueStrings = append(valueStrings, upsertAccountMasterPlaceHolder)
			valueArgs = append(valueArgs,
				account.Acno,
				account.Prefix,
				account.Name,
				account.PName,
				account.AGroupId,
				account.City,
				account.Location,
				account.Mobile,
				account.Phone,
			)
		}

		finalQuery := fmt.Sprintf(AccountMasterUpsertQuery, strings.Join(valueStrings, ", "))

		// FIXED: Replaced log.Fatalf with log.Printf to keep the cron scheduler alive on failure
		result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
		if err != nil {
			log.Printf("Bulk upsert failed for batch %d to %d: %v\n", i, end, err)
			return // Exit this sync cycle, cron will retry later
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Could not fetch rows affected for batch: %v\n", err)
		} else {
			totalRowsAffected += rowsAffected
		}
	}

	log.Printf("Bulk upsert for AccountMaster successful! Total rows affected metric: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
