package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestKindOf(t *testing.T) {
	cases := map[string]valueKind{
		"INT": kindNumber, "BIGINT": kindNumber, "TINYINT": kindNumber, "SMALLINT": kindNumber,
		"MEDIUMINT": kindNumber, "UNSIGNED INT": kindNumber, "UNSIGNED BIGINT": kindNumber,
		"DECIMAL": kindNumber, "DOUBLE": kindNumber, "FLOAT": kindNumber, "YEAR": kindNumber,
		"int":  kindNumber,
		"BLOB": kindBinary, "TINYBLOB": kindBinary, "LONGBLOB": kindBinary, "VARBINARY": kindBinary,
		"BINARY": kindBinary, "BIT": kindBinary, "GEOMETRY": kindBinary,
		"VARCHAR": kindString, "CHAR": kindString, "TEXT": kindString, "JSON": kindString,
		"DATETIME": kindString, "DATE": kindString, "TIME": kindString, "TIMESTAMP": kindString,
		"ENUM": kindString, "SET": kindString,
	}
	for in, want := range cases {
		if got := kindOf(in); got != want {
			t.Errorf("kindOf(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestQuoteIdent(t *testing.T) {
	cases := map[string]string{
		"t":       "`t`",
		"a`b":     "`a``b`",
		"``":      "``````",
		"":        "``",
		"version": "`version`",
	}
	for in, want := range cases {
		if got := quoteIdent(in); got != want {
			t.Errorf("quoteIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteQuoted(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", `'plain'`},
		{"", `''`},
		{"a\x00b", `'a\0b'`},
		{"a\nb", `'a\nb'`},
		{"a\rb", `'a\rb'`},
		{`a\b`, `'a\\b'`},
		{"it's", `'it\'s'`},
		{"a\x1ab", `'a\Zb'`},
		{`say "hi"`, `'say "hi"'`},
		{"héllo 日本語", "'héllo 日本語'"},
	}
	for _, c := range cases {
		var b strings.Builder
		writeQuoted(&b, []byte(c.in))
		if b.String() != c.want {
			t.Errorf("writeQuoted(%q) = %s, want %s", c.in, b.String(), c.want)
		}
	}
}

func TestWriteValue(t *testing.T) {
	cases := []struct {
		name string
		v    sql.RawBytes
		kind valueKind
		want string
	}{
		{"null string", nil, kindString, "NULL"},
		{"null number", nil, kindNumber, "NULL"},
		{"null binary", nil, kindBinary, "NULL"},
		{"empty string", sql.RawBytes{}, kindString, "''"},
		{"empty binary", sql.RawBytes{}, kindBinary, "''"},
		{"hex binary", sql.RawBytes{0x00, 0xff, 0x10}, kindBinary, "0x00ff10"},
		{"number", sql.RawBytes("-12.3400"), kindNumber, "-12.3400"},
		{"empty number falls back to string", sql.RawBytes{}, kindNumber, "''"},
		{"string escape", sql.RawBytes("a'b"), kindString, `'a\'b'`},
	}
	for _, c := range cases {
		var b strings.Builder
		writeValue(&b, c.v, c.kind)
		if b.String() != c.want {
			t.Errorf("%s: got %s, want %s", c.name, b.String(), c.want)
		}
	}
}

// --- integration test against a real MySQL server ---

// splitSQL splits a dump into statements the way the mysql CLI would: it
// honours quotes, comments and DELIMITER directives.
func splitSQL(script string) []string {
	var out []string
	delim := ";"
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	i, n := 0, len(script)
	for i < n {
		if strings.TrimSpace(cur.String()) == "" && strings.HasPrefix(script[i:], "DELIMITER ") {
			end := strings.IndexByte(script[i:], '\n')
			if end < 0 {
				end = n - i
			}
			delim = strings.TrimSpace(script[i+len("DELIMITER ") : i+end])
			i += end
			continue
		}
		rest := script[i:]
		c := script[i]
		switch {
		case strings.HasPrefix(rest, "--") && (len(rest) == 2 || rest[2] == ' ' || rest[2] == '\n' || rest[2] == '\r'):
			end := strings.IndexByte(rest, '\n')
			if end < 0 {
				end = len(rest)
			}
			i += end
		case c == '\'' || c == '"' || c == '`':
			j := i + 1
			for j < n && script[j] != c {
				if script[j] == '\\' && c != '`' {
					j++
				}
				j++
			}
			if j >= n {
				j = n - 1
			}
			cur.WriteString(script[i : j+1])
			i = j + 1
		case strings.HasPrefix(rest, "/*"):
			end := strings.Index(rest, "*/")
			if end < 0 {
				end = len(rest) - 2
			}
			cur.WriteString(rest[:end+2])
			i += end + 2
		case strings.HasPrefix(rest, delim):
			flush()
			i += len(delim)
		default:
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

func TestSplitSQL(t *testing.T) {
	script := "-- c\nSET @a=1;\nINSERT INTO t VALUES ('a;b\\'c;', \"x;y\");\nDELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2; END;;\nDELIMITER ;\n/*!40101 SET NAMES utf8mb4 */;\nSELECT `a;b`;\n"
	got := splitSQL(script)
	want := []string{
		"SET @a=1",
		"INSERT INTO t VALUES ('a;b\\'c;', \"x;y\")",
		"CREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2; END",
		"/*!40101 SET NAMES utf8mb4 */",
		"SELECT `a;b`",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d statements: %q", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stmt %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// tableDigest returns every row of schema.table, ordered by id, with every
// column rendered as HEX() so NULL, empty strings and binary data compare
// exactly.
func tableDigest(t *testing.T, db *sql.DB, schema, table, orderBy string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? ORDER BY ORDINAL_POSITION`, schema, table)
	if err != nil {
		t.Fatal(err)
	}
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, "HEX("+quoteIdent(c)+")")
	}
	rows.Close()
	if len(cols) == 0 {
		t.Fatalf("no columns for %s.%s", schema, table)
	}
	q := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY %s", strings.Join(cols, ","), quoteIdent(schema), quoteIdent(table), orderBy)
	r, err := db.Query(q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			if v.Valid {
				parts[i] = "[" + v.String + "]"
			} else {
				parts[i] = "NULL"
			}
		}
		out = append(out, strings.Join(parts, "|"))
	}
	if err := r.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNativeEngineRoundTrip(t *testing.T) {
	dsn := os.Getenv("BACKUP_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("BACKUP_TEST_MYSQL_DSN not set")
	}
	base, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, err := net.SplitHostPort(base.Addr)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portStr)

	adminCfg := base.Clone()
	adminCfg.DBName = ""
	admin, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	if err := admin.Ping(); err != nil {
		t.Fatalf("cannot reach MySQL: %v", err)
	}

	stamp := time.Now().UnixNano()
	src := fmt.Sprintf("bk_test_%d_src", stamp)
	dst := fmt.Sprintf("bk_test_%d_dst", stamp)
	for _, name := range []string{src, dst} {
		mustExec(t, admin, "CREATE DATABASE "+quoteIdent(name)+" CHARACTER SET utf8mb4")
		name := name
		t.Cleanup(func() { admin.Exec("DROP DATABASE IF EXISTS " + quoteIdent(name)) })
	}

	srcCfg := base.Clone()
	srcCfg.DBName = src
	srcDB, err := sql.Open("mysql", srcCfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srcDB.Close() })

	for _, q := range []string{
		"CREATE TABLE `version` (id INT AUTO_INCREMENT PRIMARY KEY, v VARCHAR(50)) ENGINE=InnoDB",
		"CREATE TABLE t_text (id INT AUTO_INCREMENT PRIMARY KEY, s TEXT, n VARCHAR(20) NULL, `we``ird` INT NULL) ENGINE=InnoDB",
		"CREATE TABLE t_bin (id INT AUTO_INCREMENT PRIMARY KEY, b BLOB, vb VARBINARY(16), bits BIT(10)) ENGINE=InnoDB",
		"CREATE TABLE t_misc (id INT AUTO_INCREMENT PRIMARY KEY, d DECIMAL(12,4), dt DATETIME, f DOUBLE, u BIGINT UNSIGNED, j JSON, e ENUM('a','b c'), g VARCHAR(20) AS (CONCAT(e,'!')) VIRTUAL) ENGINE=InnoDB",
		"CREATE VIEW v1 AS SELECT id, v FROM `version`",
		"CREATE VIEW a_v2 AS SELECT id, UPPER(v) AS uv FROM v1",
		"CREATE TRIGGER trg_ver BEFORE UPDATE ON `version` FOR EACH ROW SET NEW.v = TRIM(NEW.v)",
		"CREATE PROCEDURE p_count() BEGIN SELECT COUNT(*) FROM `version`; SELECT 'a;b'; END",
		"CREATE FUNCTION f_double(x INT) RETURNS INT DETERMINISTIC RETURN x*2",
	} {
		mustExec(t, srcDB, q)
	}

	mustExec(t, srcDB, "INSERT INTO `version` (v) VALUES (?),(?),(NULL)", "8.0.1", "  spaced  ")
	tricky := "back\\slash 'single' \"double\" line1\nline2\r\nline3 tab\t ctrlZ\x1a nul\x00end héllo 日本語 ;--; /* c */"
	mustExec(t, srcDB, "INSERT INTO t_text (s, n, `we``ird`) VALUES (?, NULL, 1), ('', '', NULL), (NULL, 'NULL', 3), (?, ?, 4)",
		tricky, "日本語", "it's")
	mustExec(t, srcDB, "INSERT INTO t_bin (b, vb, bits) VALUES (?, ?, b'1010101010'), (?, ?, b'0'), (NULL, NULL, NULL)",
		[]byte{0x00, 0xff, 0x10}, []byte{0x1a, 0x27, 0x5c}, []byte{}, []byte{})
	mustExec(t, srcDB, "INSERT INTO t_misc (d, dt, f, u, j, e) VALUES (12345678.1234, '2024-02-29 23:59:59', 3.14159265358979, 18446744073709551615, '{\"k\": [1, 2, {\"x\": \"y'\"'\"'z\"}]}', 'b c'), (-0.0001, '1999-12-31 00:00:00', -1e300, 0, 'null', 'a'), (NULL, NULL, NULL, NULL, NULL, NULL)")

	// Dump with the native engine through the production entry point.
	outDir := t.TempDir()
	cfg := &DatabaseConfig{
		Name:   "scratch",
		Engine: engineNative,
		MysqlConfig: MysqlConfig{
			Host: host, Port: port, Username: base.User, Password: base.Passwd, Database: src,
		},
	}
	path, used, err := DumpToFile(context.Background(), cfg, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if used != engineNative {
		t.Fatalf("engine used = %q", used)
	}
	dump := gunzipFile(t, path)
	if !strings.Contains(firstLine(lastLines(dump, 1)), dumpCompletedMarker) {
		t.Errorf("dump does not end with the %q marker: %q", dumpCompletedMarker, lastLines(dump, 2))
	}

	// The a_v2 view reads from v1 and sorts first alphabetically; v1 must be
	// created before it.
	if a, b := strings.Index(dump, "DROP VIEW IF EXISTS `a_v2`"), strings.Index(dump, "DROP VIEW IF EXISTS `v1`"); a < 0 || b < 0 || b > a {
		t.Errorf("view order wrong: v1 at %d, a_v2 at %d", b, a)
	}

	// Restore into the second schema.
	dstCfg := base.Clone()
	dstCfg.DBName = dst
	dstDB, err := sql.Open("mysql", dstCfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dstDB.Close() })
	conn, err := dstDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, stmt := range splitSQL(dump) {
		if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
			t.Fatalf("restore failed on statement:\n%s\nerror: %v", stmt, err)
		}
	}
	conn.Close()

	// Views must read from the restored schema, not the source.
	for _, v := range []string{"v1", "a_v2"} {
		var def string
		if err := admin.QueryRow("SELECT VIEW_DEFINITION FROM information_schema.VIEWS WHERE TABLE_SCHEMA=? AND TABLE_NAME=?", dst, v).Scan(&def); err != nil {
			t.Fatalf("view %s missing after restore: %v", v, err)
		}
		if strings.Contains(def, src) {
			t.Errorf("restored view %s still references the source schema: %s", v, def)
		}
	}

	// Table lists must match.
	tableNames := func(schema string) string {
		rows, err := admin.Query("SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA=? ORDER BY TABLE_NAME", schema)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var parts []string
		for rows.Next() {
			var n, ty string
			rows.Scan(&n, &ty)
			parts = append(parts, n+":"+ty)
		}
		return strings.Join(parts, ",")
	}
	if a, b := tableNames(src), tableNames(dst); a != b || a == "" {
		t.Fatalf("object lists differ:\nsrc: %s\ndst: %s", a, b)
	}

	// Row-for-row comparison.
	for _, table := range []string{"version", "t_text", "t_bin", "t_misc", "v1", "a_v2"} {
		a := tableDigest(t, admin, src, table, "id")
		b := tableDigest(t, admin, dst, table, "id")
		if len(a) == 0 {
			t.Errorf("%s: source has no rows", table)
		}
		if len(a) != len(b) {
			t.Errorf("%s: row count src=%d dst=%d", table, len(a), len(b))
			continue
		}
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("%s row %d differs:\n src %s\n dst %s", table, i, a[i], b[i])
			}
		}
	}

	// Spot-check a few exact values after the round trip.
	var gotS string
	if err := admin.QueryRow("SELECT s FROM " + quoteIdent(dst) + ".t_text ORDER BY id LIMIT 1").Scan(&gotS); err != nil {
		t.Fatal(err)
	}
	if gotS != tricky {
		t.Errorf("tricky string changed: %q", gotS)
	}
	var emptyN, nullN int
	admin.QueryRow("SELECT SUM(s = ''), SUM(s IS NULL) FROM "+quoteIdent(dst)+".t_text").Scan(&emptyN, &nullN)
	if emptyN != 1 || nullN != 1 {
		t.Errorf("empty/NULL distinction lost: empty=%d null=%d", emptyN, nullN)
	}

	// Triggers, procedure and function exist and work in the restored schema.
	count := func(q string, args ...any) int {
		var n int
		if err := admin.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if count("SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=? AND TRIGGER_NAME='trg_ver'", dst) != 1 {
		t.Error("trigger trg_ver missing after restore")
	}
	if count("SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=? AND ROUTINE_NAME='p_count' AND ROUTINE_TYPE='PROCEDURE'", dst) != 1 {
		t.Error("procedure p_count missing after restore")
	}
	if count("SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=? AND ROUTINE_NAME='f_double' AND ROUTINE_TYPE='FUNCTION'", dst) != 1 {
		t.Error("function f_double missing after restore")
	}
	if got := count("SELECT " + quoteIdent(dst) + ".f_double(21)"); got != 42 {
		t.Errorf("f_double(21) = %d", got)
	}
	mustExec(t, dstDB, "UPDATE `version` SET v = '  trimmed  ' WHERE id = 2")
	var v string
	if err := dstDB.QueryRow("SELECT v FROM `version` WHERE id = 2").Scan(&v); err != nil || v != "trimmed" {
		t.Errorf("restored trigger did not fire: %q, %v", v, err)
	}
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
