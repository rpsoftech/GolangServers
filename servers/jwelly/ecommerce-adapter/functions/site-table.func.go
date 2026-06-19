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

const SelectSiteQuery = "select SITEID,SITE,`PATH`,PREFIX from `site`;"
const ServerSiteOfColumns = 4
const upsertSitePlaceHolder = "(?, ?, ?, ?)"
const SiteUpsertQuery = `INSERT INTO Site (siteId, siteName, siteAddress, sPrefix) VALUES %s AS new_row ON DUPLICATE KEY UPDATE siteName = new_row.siteName,siteAddress = new_row.siteAddress,sPrefix = new_row.sPrefix;`

func SyncSiteTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectSiteQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.SiteTable = []*ecommerce_maintables.SiteTable{}
	for rows.Next() {
		row := &ecommerce_maintables.SiteTable{}
		err = rows.Scan(
			&row.SiteId,
			&row.SiteName,
			&row.SiteAddress,
			&row.SPrefix,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Site in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerSiteOfColumns)
	for _, site := range results {
		valueStrings = append(valueStrings, upsertSitePlaceHolder)
		valueArgs = append(valueArgs,
			site.SiteId,
			site.SiteName,
			site.SiteAddress,
			site.SPrefix,
		)
	}
	finalQuery := fmt.Sprintf(SiteUpsertQuery, strings.Join(valueStrings, ", "))
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for Site successful!  Rows affected metric: %d\n", rowsAffected)

}
