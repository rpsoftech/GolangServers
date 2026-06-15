package ecommerece_functions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ecommerce_maintables "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/MainTables"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

// const SelectStampQuery
const SelectStampQuery = "select STAMPID,STAMP,STUNCH,STKTUNCH from stamp;"
const ServerStampTable = "Stamp"
const ServerStampOfColums = 4
const StampUpsertQuery = `INSERT INTO Stamp (stampId,STAMP,tunch,stockTunch) VALUES %s AS new_row ON DUPLICATE KEY UPDATE stampId=new_row.stampId,STAMP=new_row.STAMP,tunch=new_row.tunch,stockTunch=new_row.stockTunch`

func SyncStampTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initalTime := time.Now()
	startTime := initalTime
	rows, err := erpDb.Db.Query(SelectStampQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.StampMainTable = []*ecommerce_maintables.StampMainTable{}
	for rows.Next() {
		row := &ecommerce_maintables.StampMainTable{}
		err = rows.Scan(
			&row.StampId,
			&row.Stamp,
			&row.Tunch,
			&row.StockTunch,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Stamp in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerStampOfColums)
	for _, stamp := range results {
		valueStrings = append(valueStrings, "(?, ?, ? ,?)")
		valueArgs = append(valueArgs,
			stamp.StampId,
			stamp.Stamp,
			stamp.Tunch,
			stamp.StockTunch,
		)
	}
	finalQuery := fmt.Sprintf(StampUpsertQuery, strings.Join(valueStrings, ", "))
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for Stamp successful!  Rows affected metric: %d\n", rowsAffected)

}
