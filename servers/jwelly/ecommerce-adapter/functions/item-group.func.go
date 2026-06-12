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

// const SelectItemGroupQuery
const SelectItemGroupQuery = "select igroupid,IGROUP,PNAME,UNITID from igroup;"
const ServerItemGroupTable = "ItemGroup"
const ServerItemGroupOfColums = 4
const ItemGroupUpsertQuery = `INSERT INTO ItemGroup (itemGroupId, itemGroup, itemPrintName,itemGroupUnitId) VALUES %s AS new_row ON DUPLICATE KEY UPDATE itemGroup=new_row.itemGroup,itemPrintName=new_row.itemPrintName,itemGroupUnitId=new_row.itemGroupUnitId`

func SyncItemGroupTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initalTime := time.Now()
	startTime := initalTime
	rows, err := erpDb.Db.Query(SelectItemGroupQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.ItemGroupMainTable = []*ecommerce_maintables.ItemGroupMainTable{}
	for rows.Next() {
		row := &ecommerce_maintables.ItemGroupMainTable{}
		err = rows.Scan(
			&row.ItemGroupId,
			&row.ItemGroup,
			&row.ItemPrintName,
			&row.ItemUnitId,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from ItemGroup in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerItemGroupOfColums)
	for _, group := range results {
		valueStrings = append(valueStrings, "(?, ?, ? ,?)")
		valueArgs = append(valueArgs, group.ItemGroupId, group.ItemGroup, group.ItemPrintName, group.ItemUnitId)
	}
	finalQuery := fmt.Sprintf(ItemGroupUpsertQuery, strings.Join(valueStrings, ", "))
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for Item Group successful!  Rows affected metric: %d\n", rowsAffected)

}
