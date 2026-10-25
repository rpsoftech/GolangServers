package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// nativeEngine is the fallback used when the mysqldump binary is missing or
// fails. It reads everything inside one consistent-snapshot transaction (like
// mysqldump --single-transaction) and writes a dump that mysqldump-style
// tools can restore.
type nativeEngine struct{}

func (nativeEngine) Name() string { return engineNative }

// maxInsertBytes caps one multi-row INSERT statement.
const maxInsertBytes = 1 << 20

func (nativeEngine) Dump(ctx context.Context, db *DatabaseConfig, w io.Writer) (err error) {
	m := db.MysqlConfig
	cfg := mysql.NewConfig()
	cfg.User = m.Username
	cfg.Passwd = m.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", m.Host, m.Port)
	cfg.DBName = m.Database
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	// Dates, decimals and floats are read as the server's own text, so no
	// precision is lost and zero dates survive.

	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		return fmt.Errorf("cannot connect: %w", err)
	}

	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "START TRANSACTION WITH CONSISTENT SNAPSHOT"); err != nil {
		return err
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")

	d := &nativeDumper{ctx: ctx, conn: conn, schema: m.Database, out: bufio.NewWriterSize(w, 256<<10)}
	if err := d.run(); err != nil {
		return err
	}
	return d.out.Flush()
}

type nativeDumper struct {
	ctx    context.Context
	conn   *sql.Conn
	schema string
	out    *bufio.Writer
}

func (d *nativeDumper) printf(format string, a ...any) { fmt.Fprintf(d.out, format, a...) }

func (d *nativeDumper) run() error {
	d.printf("-- MySQL dump by mysql-backup-exe (native engine)\n-- Database: %s\n-- Started: %s\n\n", d.schema, time.Now().Format("2006-01-02 15:04:05"))
	d.printf("/*!40101 SET NAMES utf8mb4 */;\n")
	d.printf("/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;\n")
	d.printf("/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;\n")
	d.printf("/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;\n\n")

	tables, views, err := d.listObjects()
	if err != nil {
		return err
	}
	for _, t := range tables {
		if err := d.dumpTable(t); err != nil {
			return fmt.Errorf("table %s: %w", t, err)
		}
	}
	if err := d.dumpViews(views); err != nil {
		return err
	}
	if err := d.dumpRoutines(); err != nil {
		return err
	}
	if err := d.dumpTriggers(); err != nil {
		return err
	}
	if err := d.dumpEvents(); err != nil {
		return err
	}

	d.printf("/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;\n")
	d.printf("/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;\n")
	d.printf("/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;\n\n")
	d.printf("-- %s on %s\n", dumpCompletedMarker, time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func quoteIdent(s string) string { return "`" + strings.ReplaceAll(s, "`", "``") + "`" }

func (d *nativeDumper) listObjects() (tables, views []string, err error) {
	rows, err := d.conn.QueryContext(d.ctx,
		"SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME", d.schema)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return nil, nil, err
		}
		if typ == "VIEW" {
			views = append(views, name)
		} else if typ == "BASE TABLE" {
			tables = append(tables, name)
		}
	}
	return tables, views, rows.Err()
}

// queryStrings runs a query and returns the first column of every row.
func (d *nativeDumper) queryStrings(query string, args ...any) ([]string, error) {
	rows, err := d.conn.QueryContext(d.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// showCreate runs a SHOW CREATE statement and returns the column at idx, or ""
// when the server hides the definition (insufficient privileges).
func (d *nativeDumper) showCreate(kind, name string, idx int) (string, error) {
	rows, err := d.conn.QueryContext(d.ctx, fmt.Sprintf("SHOW CREATE %s %s", kind, quoteIdent(name)))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if !rows.Next() {
		return "", fmt.Errorf("SHOW CREATE %s %s returned nothing", kind, name)
	}
	vals := make([]sql.NullString, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return "", err
	}
	if idx >= len(vals) || !vals[idx].Valid {
		return "", nil
	}
	return vals[idx].String, rows.Err()
}

// --- tables ---

func (d *nativeDumper) dumpTable(table string) error {
	create, err := d.showCreate("TABLE", table, 1)
	if err != nil {
		return err
	}
	if create == "" {
		return fmt.Errorf("no permission to read the table definition")
	}
	d.printf("--\n-- Table structure for %s\n--\n\nDROP TABLE IF EXISTS %s;\n%s;\n\n", quoteIdent(table), quoteIdent(table), create)

	cols, err := d.insertableColumns(table)
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return nil
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
	}
	colList := strings.Join(quoted, ",")

	rows, err := d.conn.QueryContext(d.ctx, fmt.Sprintf("SELECT %s FROM %s", colList, quoteIdent(table)))
	if err != nil {
		return err
	}
	defer rows.Close()
	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	kinds := make([]valueKind, len(types))
	for i, t := range types {
		kinds[i] = kindOf(t.DatabaseTypeName())
	}

	d.printf("--\n-- Data for %s\n--\n\n", quoteIdent(table))
	insertHead := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", quoteIdent(table), colList)
	raw := make([]sql.RawBytes, len(cols))
	ptrs := make([]any, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}

	var stmt strings.Builder
	flush := func() {
		if stmt.Len() > 0 {
			d.out.WriteString(stmt.String())
			d.out.WriteString(";\n")
			stmt.Reset()
		}
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if stmt.Len() == 0 {
			stmt.WriteString(insertHead)
		} else {
			stmt.WriteByte(',')
		}
		stmt.WriteByte('(')
		for i, v := range raw {
			if i > 0 {
				stmt.WriteByte(',')
			}
			writeValue(&stmt, v, kinds[i])
		}
		stmt.WriteByte(')')
		if stmt.Len() >= maxInsertBytes {
			flush()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	flush()
	d.printf("\n")
	return nil
}

// insertableColumns lists a table's columns in order, leaving out generated
// ones, which cannot be inserted into.
func (d *nativeDumper) insertableColumns(table string) ([]string, error) {
	return d.queryStrings(`SELECT COLUMN_NAME FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		AND EXTRA NOT LIKE '%GENERATED%' AND EXTRA NOT LIKE '%PERSISTENT%'
		ORDER BY ORDINAL_POSITION`, d.schema, table)
}

type valueKind int

const (
	kindString valueKind = iota
	kindNumber
	kindBinary
)

func kindOf(dbType string) valueKind {
	t := strings.ToUpper(dbType)
	switch {
	case strings.Contains(t, "INT"), strings.Contains(t, "DECIMAL"), strings.Contains(t, "FLOAT"),
		strings.Contains(t, "DOUBLE"), t == "YEAR":
		return kindNumber
	case strings.Contains(t, "BLOB"), strings.Contains(t, "BINARY"), t == "BIT", t == "GEOMETRY":
		return kindBinary
	}
	return kindString
}

func writeValue(b *strings.Builder, v sql.RawBytes, kind valueKind) {
	if v == nil {
		b.WriteString("NULL")
		return
	}
	switch kind {
	case kindNumber:
		if len(v) > 0 {
			b.Write(v)
			return
		}
	case kindBinary:
		if len(v) == 0 {
			b.WriteString("''")
			return
		}
		b.WriteString("0x")
		b.WriteString(hex.EncodeToString(v))
		return
	}
	writeQuoted(b, v)
}

// writeQuoted writes a string literal using the same escapes as mysqldump.
func writeQuoted(b *strings.Builder, v []byte) {
	b.WriteByte('\'')
	for _, c := range v {
		switch c {
		case 0:
			b.WriteString(`\0`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case 0x1a:
			b.WriteString(`\Z`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
}

// --- views, routines, triggers, events ---

// dumpViews writes views so that a view comes after any view it reads from.
func (d *nativeDumper) dumpViews(views []string) error {
	defs := make(map[string]string, len(views))
	for _, v := range views {
		def, err := d.showCreate("VIEW", v, 1)
		if err != nil {
			return fmt.Errorf("view %s: %w", v, err)
		}
		defs[v] = def
	}
	done := map[string]bool{}
	var order []string
	var visit func(string)
	visit = func(v string) {
		if done[v] {
			return
		}
		done[v] = true // also stops cycles
		for _, other := range views {
			if other != v && strings.Contains(defs[v], quoteIdent(other)) {
				visit(other)
			}
		}
		order = append(order, v)
	}
	sort.Strings(views)
	for _, v := range views {
		visit(v)
	}
	for _, v := range order {
		if defs[v] == "" {
			d.printf("-- view %s skipped: definition hidden (insufficient privileges)\n\n", quoteIdent(v))
			continue
		}
		d.printf("--\n-- View %s\n--\n\nDROP VIEW IF EXISTS %s;\n%s;\n\n", quoteIdent(v), quoteIdent(v), defs[v])
	}
	return nil
}

func (d *nativeDumper) writeDelimited(kind, name, def string) {
	if def == "" {
		d.printf("-- %s %s skipped: definition hidden (insufficient privileges)\n\n", kind, quoteIdent(name))
		return
	}
	d.printf("DROP %s IF EXISTS %s;\nDELIMITER ;;\n%s;;\nDELIMITER ;\n\n", kind, quoteIdent(name), def)
}

func (d *nativeDumper) dumpRoutines() error {
	rows, err := d.conn.QueryContext(d.ctx,
		"SELECT ROUTINE_NAME, ROUTINE_TYPE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? ORDER BY ROUTINE_TYPE, ROUTINE_NAME", d.schema)
	if err != nil {
		return err
	}
	type routine struct{ name, typ string }
	var list []routine
	for rows.Next() {
		var r routine
		if err := rows.Scan(&r.name, &r.typ); err != nil {
			rows.Close()
			return err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range list {
		def, err := d.showCreate(r.typ, r.name, 2)
		if err != nil {
			return fmt.Errorf("%s %s: %w", strings.ToLower(r.typ), r.name, err)
		}
		d.writeDelimited(r.typ, r.name, def)
	}
	return nil
}

func (d *nativeDumper) dumpTriggers() error {
	names, err := d.queryStrings("SELECT TRIGGER_NAME FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME", d.schema)
	if err != nil {
		return err
	}
	for _, n := range names {
		def, err := d.showCreate("TRIGGER", n, 2)
		if err != nil {
			return fmt.Errorf("trigger %s: %w", n, err)
		}
		d.writeDelimited("TRIGGER", n, def)
	}
	return nil
}

func (d *nativeDumper) dumpEvents() error {
	names, err := d.queryStrings("SELECT EVENT_NAME FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? ORDER BY EVENT_NAME", d.schema)
	if err != nil {
		return err
	}
	for _, n := range names {
		def, err := d.showCreate("EVENT", n, 3)
		if err != nil {
			return fmt.Errorf("event %s: %w", n, err)
		}
		d.writeDelimited("EVENT", n, def)
	}
	return nil
}
