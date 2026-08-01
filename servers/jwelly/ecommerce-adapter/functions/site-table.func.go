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

const SelectSiteQuery = "SELECT SITEID, SITE, `PATH`, PREFIX FROM `site`;"
const ServerSiteOfColumns = 4
const upsertSitePlaceHolder = "(?, ?, ?, ?)"
const SiteUpsertQuery = `INSERT INTO Site (siteId, siteName, siteAddress, sPrefix) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
siteName = new_row.siteName, 
siteAddress = new_row.siteAddress, 
sPrefix = new_row.sPrefix;`

func SyncSiteTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	// 10-second timeout is plenty for a small Site table
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// FIXED: Bound query to context
	rows, err := erpDb.Db.QueryContext(ctx, SelectSiteQuery)
	if err != nil {
		log.Printf("Error executing query to read Site from ERP: %v\n", err)
		return
	}
	// FIXED: Prevent connection pool leaks
	defer rows.Close()

	var results []*ecommerce_maintables.SiteTable
	for rows.Next() {
		row := &ecommerce_maintables.SiteTable{}
		err := rows.Scan(
			&row.SiteId,
			&row.SiteName,
			&row.SiteAddress,
			&row.SPrefix,
		)
		if err != nil {
			log.Printf("Error scanning row in Site: %v\n", err)
			continue
		}
		results = append(results, row)
	}

	// FIXED: Catch mid-loop iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over Site rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for Site. Exiting sync.")
		return
	}

	log.Printf("Fetched Total %d rows from Site in %s\n", len(results), time.Since(startTime))

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
		valueArgs := make([]interface{}, 0, len(batch)*ServerSiteOfColumns)

		for _, site := range batch {
			valueStrings = append(valueStrings, upsertSitePlaceHolder)
			valueArgs = append(valueArgs,
				site.SiteId,
				site.SiteName,
				site.SiteAddress,
				site.SPrefix,
			)
		}

		finalQuery := fmt.Sprintf(SiteUpsertQuery, strings.Join(valueStrings, ", "))

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

	log.Printf("Bulk upsert for Site successful! Total rows affected: %d. Total Sync Time: %s\n", totalRowsAffected, time.Since(startTime))
}
