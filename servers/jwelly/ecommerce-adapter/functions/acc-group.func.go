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

const SelectAccountGroupQuery = "select GROUPID,GNAME,UNDERID,GRTYPE from `group`;"
const ServerAccGrpOfColumns = 4
const AccountGroupUpsertQuery = `INSERT INTO AccountGroup (groupId,groupName,underId,gType) VALUES %s AS new_row ON DUPLICATE KEY UPDATE groupName=new_row.groupId,underId=new_row.groupId,gType=new_row.groupId`

func SyncAccountGroupTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()
	startTime := initialTime
	rows, err := erpDb.Db.Query(SelectAccountGroupQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Printf("Error in Reading Unit From Server")
		log.Println(err.Error())
		return
	}
	var results []*ecommerce_maintables.AccountGroupTable = []*ecommerce_maintables.AccountGroupTable{}
	for rows.Next() {
		row := &ecommerce_maintables.AccountGroupTable{}
		err = rows.Scan(
			&row.GroupId,
			&row.GroupName,
			&row.UnderId,
			&row.GType,
		)
		results = append(results, row)
	}
	log.Printf("Fetched Total %d rows from Stamp in Duration of %s\n", len(results), time.Since(startTime))
	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerAccGrpOfColumns)
	for _, stamp := range results {
		valueStrings = append(valueStrings, "(?, ?, ? ,?)")
		valueArgs = append(valueArgs,
			stamp.GroupId,
			stamp.GroupName,
			stamp.UnderId,
			stamp.GType,
		)
	}
	finalQuery := fmt.Sprintf(AccountGroupUpsertQuery, strings.Join(valueStrings, ", "))
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
