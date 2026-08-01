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

const SelectAccountGroupQuery = "SELECT GROUPID, GNAME, UNDERID, GRTYPE FROM `group`;"
const ServerAccGrpOfColumns = 4

// FIXED: Update clause maps to correct column names to prevent data corruption
const AccountGroupUpsertQuery = `INSERT INTO AccountGroup (groupId, groupName, underId, gType) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
groupName = new_row.groupName, 
underId = new_row.underId, 
gType = new_row.gType`

func SyncAccountGroupTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// FIXED: Used QueryContext to enforce the 10-second timeout
	rows, err := erpDb.Db.QueryContext(ctx, SelectAccountGroupQuery)
	if err != nil {
		log.Printf("Error executing query to read AccountGroup from ERP: %v\n", err)
		return
	}
	// FIXED: Crucial memory leak prevention
	defer rows.Close()

	var results []*ecommerce_maintables.AccountGroupTable

	for rows.Next() {
		row := &ecommerce_maintables.AccountGroupTable{}
		err := rows.Scan(
			&row.GroupId,
			&row.GroupName,
			&row.UnderId,
			&row.GType,
		)
		if err != nil {
			log.Printf("Error scanning row in AccountGroup: %v\n", err)
			continue // Skip bad rows instead of crashing
		}
		results = append(results, row)
	}

	// FIXED: Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over AccountGroup rows: %v\n", err)
		return
	}

	if len(results) == 0 {
		log.Println("No records fetched from ERP for AccountGroup. Exiting sync.")
		return
	}

	// FIXED: Cleaned up copy-paste logging errors
	log.Printf("Fetched Total %d rows from AccountGroup in %s\n", len(results), time.Since(startTime))

	valueStrings := make([]string, 0, len(results))
	valueArgs := make([]interface{}, 0, len(results)*ServerAccGrpOfColumns)

	for _, grp := range results {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			grp.GroupId,
			grp.GroupName,
			grp.UnderId,
			grp.GType,
		)
	}

	finalQuery := fmt.Sprintf(AccountGroupUpsertQuery, strings.Join(valueStrings, ", "))

	// ExecContext already used correctly, but now the query won't corrupt the database
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		log.Printf("Bulk upsert failed: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Could not fetch rows affected: %v\n", err)
		return
	}

	log.Printf("Bulk upsert for AccountGroup successful! Rows affected: %d\n", rowsAffected)
}
