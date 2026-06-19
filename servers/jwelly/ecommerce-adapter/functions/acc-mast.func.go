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

const SelectAccountMasterQuery = "select acno,PREFIX,CNAME,PNAME,GROUPID,CITY,LOCATION,MOBILE,PHONE from `accmast`;"
const ServerAccMstOfColumns = 9
const upsertAccountMasterPlaceHolder = "(?, ?, ?, ?, ?, ?, ?, ?, ?)"
const AccountMasterUpsertQuery = `INSERT INTO AccountMaster (acno,prefix,Name,pName,aGroupId,city,location,mobile,phone) VALUES %s AS new_row ON DUPLICATE KEY UPDATE prefix = new_row.prefix,Name = new_row.Name,pName = new_row.pName,aGroupId = new_row.aGroupId,city = new_row.city,location = new_row.location,mobile = new_row.mobile,phone = new_row.phone`

func SyncAccountMasterTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectAccountMasterQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.AccountMasterTable = []*ecommerce_maintables.AccountMasterTable{}
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
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from AccountMaster in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerAccMstOfColumns)
	for _, account := range results {
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
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Fatalf("Bulk upsert failed: %v", err)
		log.Fatalf("Query : %s", finalQuery)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bulk upsert for AccountMaster successful!  Rows affected metric: %d\n", rowsAffected)

}
