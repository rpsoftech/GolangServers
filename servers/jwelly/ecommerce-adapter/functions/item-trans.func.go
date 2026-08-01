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

// OPTIMIZED: Replaced OFFSET with Keyset Pagination (ITRNID > ?) for massive view performance gains
const SelectItemTransQuery = "SELECT ITRNID, TRNID, VONO, TNO, SNTRNID, TDATE, INO, REMARKS, GWT, WT, LESSWT, PC, rate, TUNCH, WSTG, MAMT, TYPE, STOCK, UNITID, STAMPID, SITEID, TVALUE, DAMT, SAMT, LAMT, BAMT, OTHERS, LBR, size, FINE1, TGNO, vtgno, TPRE, TSNO, ORGRATE, ORGGWT, ORGTOTAL, TOTAL, KACNO, karigar, metalwt FROM itrnview WHERE ITRNID > ? ORDER BY ITRNID ASC LIMIT ?;"
const ItemTransactionColumns = 41
const upsertItemTransPlaceHolder = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

const ItemTransUpsertQuery = `INSERT INTO ItemTransaction (itransId, trnId, vono, tno, snTranId, tdate, ino, remarks, gwt, wt, lesswt, pc, rate, tunch, wstg, mamt, type, stock, unitId, stampId, siteId, tValue, damt, samt, lamt, bamt, others, lbr, size, fine1, tgno, vtgno, tpre, tsno, orgRate, orgGwt, orgTotal, total, kacno, karigar, metalwt) 
VALUES %s AS new_row 
ON DUPLICATE KEY UPDATE 
trnId = new_row.trnId, vono = new_row.vono, tno = new_row.tno, snTranId = new_row.snTranId, tdate = new_row.tdate, ino = new_row.ino, remarks = new_row.remarks, gwt = new_row.gwt, wt = new_row.wt, lesswt = new_row.lesswt, pc = new_row.pc, rate = new_row.rate, tunch = new_row.tunch, wstg = new_row.wstg, mamt = new_row.mamt, type = new_row.type, stock = new_row.stock, unitId = new_row.unitId, stampId = new_row.stampId, siteId = new_row.siteId, tValue = new_row.tValue, damt = new_row.damt, samt = new_row.samt, lamt = new_row.lamt, bamt = new_row.bamt, others = new_row.others, lbr = new_row.lbr, size = new_row.size, fine1 = new_row.fine1, tgno = new_row.tgno, vtgno = new_row.vtgno, tpre = new_row.tpre, tsno = new_row.tsno, orgRate = new_row.orgRate, orgGwt = new_row.orgGwt, orgTotal = new_row.orgTotal, total = new_row.total, kacno = new_row.kacno, karigar = new_row.karigar, metalwt = new_row.metalwt;`

func SyncItemTransTable(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct) {
	initialTime := time.Now()

	// CRITICAL FIX: Max safe batch size is 1500 (1500 * 41 columns = 61,500 placeholders).
	// Do not exceed 1500 or MySQL will hard crash.
	batchSize := 1500
	lastId := 0 // High-performance cursor tracking ITRNID
	var totalRowsAffected int64 = 0

	log.Println("Starting chunked sync for Item Transactions using Keyset Pagination...")

	for {
		rowsFetched, rowsAffected, nextId, err := processItemTransBatch(serverDb, erpDb, batchSize, lastId)
		if err != nil {
			log.Printf("Sync halted due to error after cursor ID %d: %v", lastId, err)
			break
		}

		totalRowsAffected += rowsAffected

		if rowsFetched < batchSize {
			break // End of table reached
		}

		lastId = nextId // Move cursor forward
	}

	log.Printf("Sync completed successfully! Total Duration: %s | Total Rows Affected Metric: %d\n", time.Since(initialTime), totalRowsAffected)
}

func processItemTransBatch(serverDb *mysqldb.MysqlDBStruct, erpDb *mysqldb.MysqlDBStruct, limit int, lastId int) (int, int64, int, error) {
	startTime := time.Now()

	// Increased timeout because pulling from `itrnview` with 41 columns is heavy
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Fetch chunk from ERP using QueryContext and Cursor
	rows, err := erpDb.Db.QueryContext(ctx, SelectItemTransQuery, lastId, limit)
	if err != nil {
		return 0, 0, lastId, fmt.Errorf("failed reading from ERP (Limit: %d, Cursor: %d): %w", limit, lastId, err)
	}
	defer rows.Close()

	var results []*ecommerce_maintables.ItemTransactionTable
	maxId := lastId // Track highest ITRNID in this batch

	for rows.Next() {
		row := &ecommerce_maintables.ItemTransactionTable{}
		err = rows.Scan(
			&row.ItransId, // Maps to ITRNID
			&row.TrnId,
			&row.VONO,
			&row.TNO,
			&row.SNTranId,
			&row.TDATE,
			&row.INO,
			&row.Remarks,
			&row.GWT,
			&row.WT,
			&row.LESSWT,
			&row.PC,
			&row.Rate,
			&row.TUNCH,
			&row.WSTG,
			&row.MAMT,
			&row.TYPE,
			&row.Stock,
			&row.UnitId,
			&row.StampId,
			&row.SiteId,
			&row.TValue,
			&row.DAmt,
			&row.SAmt,
			&row.LAmt,
			&row.BAmt,
			&row.Others,
			&row.LBR,
			&row.Size,
			&row.FINE1,
			&row.TGNO,
			&row.VTGNO,
			&row.TPRE,
			&row.TSNO,
			&row.ORGRate,
			&row.ORGGwt,
			&row.ORGTotal,
			&row.Total,
			&row.KACNO,
			&row.Karigar,
			&row.Metalwt,
		)
		if err != nil {
			return 0, 0, lastId, fmt.Errorf("row scan error: %w", err)
		}

		// Update cursor
		if int(row.ItransId) > maxId {
			maxId = int(row.ItransId)
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return 0, 0, lastId, fmt.Errorf("rows iteration error: %w", err)
	}

	fetchedCount := len(results)
	if fetchedCount == 0 {
		return 0, 0, maxId, nil
	}

	// 2. Prepare chunk for bulk upsert
	valueStrings := make([]string, 0, fetchedCount)
	valueArgs := make([]interface{}, 0, fetchedCount*ItemTransactionColumns)

	for _, trans := range results {
		valueStrings = append(valueStrings, upsertItemTransPlaceHolder)
		valueArgs = append(valueArgs,
			trans.ItransId,
			trans.TrnId,
			trans.VONO,
			trans.TNO,
			trans.SNTranId,
			trans.TDATE,
			trans.INO,
			trans.Remarks,
			trans.GWT,
			trans.WT,
			trans.LESSWT,
			trans.PC,
			trans.Rate,
			trans.TUNCH,
			trans.WSTG,
			trans.MAMT,
			trans.TYPE,
			trans.Stock,
			trans.UnitId,
			trans.StampId,
			trans.SiteId,
			trans.TValue,
			trans.DAmt,
			trans.SAmt,
			trans.LAmt,
			trans.BAmt,
			trans.Others,
			trans.LBR,
			trans.Size,
			trans.FINE1,
			trans.TGNO,
			trans.VTGNO,
			trans.TPRE,
			trans.TSNO,
			trans.ORGRate,
			trans.ORGGwt,
			trans.ORGTotal,
			trans.Total,
			trans.KACNO,
			trans.Karigar,
			trans.Metalwt,
		)
	}

	finalQuery := fmt.Sprintf(ItemTransUpsertQuery, strings.Join(valueStrings, ", "))

	// 3. Execute bulk upsert
	result, err := serverDb.Db.ExecContext(ctx, finalQuery, valueArgs...)
	if err != nil {
		return fetchedCount, 0, maxId, fmt.Errorf("bulk upsert failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fetchedCount, 0, maxId, fmt.Errorf("could not get rows affected: %w", err)
	}

	log.Printf("Batch processed: %d rows fetched | %d rows affected | Duration: %s", fetchedCount, rowsAffected, time.Since(startTime))

	return fetchedCount, rowsAffected, maxId, nil
}
