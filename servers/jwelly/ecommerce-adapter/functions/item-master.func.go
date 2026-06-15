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

const SelectItemMasterQuery = "select ino,IDESC,IGROUPID,PNAME,UNITID,tpre from itemmast;"
const ServerItemMasterTable = "ItemMaster"
const ServerItemMasterOfColumns = 4
const ItemMasterUpsertQuery = `INSERT INTO ItemMaster (itemId, itemName, iGroupId,itemPrintName,iUnitId,itemTagPrefix) VALUES %s AS new_row ON DUPLICATE KEY UPDATE itemName = new_row.itemName, iGroupId = new_row.iGroupId, itemPrintName = new_row.itemPrintName, iUnitId = new_row.iUnitId, itemTagPrefix = new_row.itemTagPrefix`

func SyncItemMasterTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectItemMasterQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.ItemMasterTable = []*ecommerce_maintables.ItemMasterTable{}
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
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Item Master in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerItemMasterOfColumns)

	for _, item := range results {
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
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for Item Master successful!  Rows affected metric: %d\n", rowsAffected)

}
