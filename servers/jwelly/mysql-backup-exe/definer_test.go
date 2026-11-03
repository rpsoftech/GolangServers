package main

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestStripDefiner(t *testing.T) {
	cases := map[string]string{
		"CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY DEFINER VIEW `v` AS select 1": "CREATE ALGORITHM=UNDEFINED SQL SECURITY DEFINER VIEW `v` AS select 1",
		"CREATE DEFINER=`erp`@`%` TRIGGER `t` BEFORE INSERT ON `a` FOR EACH ROW SET NEW.x = 1":            "CREATE TRIGGER `t` BEFORE INSERT ON `a` FOR EACH ROW SET NEW.x = 1",
		"CREATE DEFINER=root@localhost PROCEDURE `p`() BEGIN SELECT 'DEFINER=`a`@`b` x'; END":             "CREATE PROCEDURE `p`() BEGIN SELECT 'DEFINER=`a`@`b` x'; END",
		"CREATE DEFINER=`we``ird`@`host` FUNCTION `f`() RETURNS int RETURN 1":                             "CREATE FUNCTION `f`() RETURNS int RETURN 1",
		"CREATE DEFINER='erp'@'10.0.0.%' EVENT `e` ON SCHEDULE EVERY 1 DAY DO SELECT 1":                   "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY DO SELECT 1",
		"CREATE PROCEDURE `p`() SELECT 1": "CREATE PROCEDURE `p`() SELECT 1",
	}
	for in, want := range cases {
		if got := stripDefiner(in); got != want {
			t.Errorf("stripDefiner(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

// mysqldumpSample has one header line of each kind mysqldump writes, plus data
// and body lines that hold DEFINER text which must survive.
const mysqldumpSample = "-- MySQL dump\n" +
	"/*!40101 SET NAMES utf8mb4 */;\n" +
	"INSERT INTO `notes` VALUES (1,'CREATE DEFINER=`a`@`b` PROCEDURE');\n" +
	"/*!50001 CREATE ALGORITHM=UNDEFINED */\n" +
	"/*!50013 DEFINER=`root`@`localhost` SQL SECURITY DEFINER */\n" +
	"/*!50001 VIEW `v` AS select 1 */;\n" +
	"/*!50003 CREATE*/ /*!50017 DEFINER=`root`@`%`*/ /*!50003 TRIGGER `t` BEFORE INSERT ON `a` FOR EACH ROW SET NEW.x = 1 */;;\n" +
	"CREATE DEFINER=`erp`@`10.0.0.%` PROCEDURE `p`()\n" +
	"BEGIN SELECT 'DEFINER=`a`@`b`'; END ;;\n" +
	"/*!50106 CREATE*/ /*!50117 DEFINER=`root`@`localhost`*/ /*!50106 EVENT `e` ON SCHEDULE EVERY 1 DAY DO SELECT 1 */ ;;\n" +
	"-- Dump completed on x"

const mysqldumpSampleStripped = "-- MySQL dump\n" +
	"/*!40101 SET NAMES utf8mb4 */;\n" +
	"INSERT INTO `notes` VALUES (1,'CREATE DEFINER=`a`@`b` PROCEDURE');\n" +
	"/*!50001 CREATE ALGORITHM=UNDEFINED */\n" +
	"/*!50013 SQL SECURITY DEFINER */\n" +
	"/*!50001 VIEW `v` AS select 1 */;\n" +
	"/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER `t` BEFORE INSERT ON `a` FOR EACH ROW SET NEW.x = 1 */;;\n" +
	"CREATE PROCEDURE `p`()\n" +
	"BEGIN SELECT 'DEFINER=`a`@`b`'; END ;;\n" +
	"/*!50106 CREATE*/ /*!50117 */ /*!50106 EVENT `e` ON SCHEDULE EVERY 1 DAY DO SELECT 1 */ ;;\n" +
	"-- Dump completed on x"

func TestDefinerFilter(t *testing.T) {
	// Write sizes that split lines and prefixes at every position.
	for _, size := range []int{1, 2, 5, 7, 64, len(mysqldumpSample)} {
		var out bytes.Buffer
		f := &definerFilter{w: &out}
		for s := mysqldumpSample; len(s) > 0; {
			n := min(size, len(s))
			if k, err := f.Write([]byte(s[:n])); err != nil || k != n {
				t.Fatalf("size %d: Write = %d, %v", size, k, err)
			}
			s = s[n:]
		}
		if err := f.Flush(); err != nil {
			t.Fatal(err)
		}
		if out.String() != mysqldumpSampleStripped {
			t.Errorf("size %d:\n got %q\nwant %q", size, out.String(), mysqldumpSampleStripped)
		}
	}
}

func TestDefinerFilterLongLines(t *testing.T) {
	// A long data line streams through without being held, and an oversized
	// header-like line is passed on unchanged rather than buffered without
	// limit.
	data := "INSERT INTO `t` VALUES ('" + strings.Repeat("x", 3*maxHeaderLine) + "');\n"
	header := "CREATE DEFINER=`a`@`b` VIEW v AS SELECT '" + strings.Repeat("y", 2*maxHeaderLine) + "';\n"
	in := data + header + "CREATE DEFINER=`a`@`b` VIEW w AS SELECT 1;\n"
	var out bytes.Buffer
	f := &definerFilter{w: &out}
	if _, err := f.Write([]byte(in)); err != nil {
		t.Fatal(err)
	}
	if err := f.Flush(); err != nil {
		t.Fatal(err)
	}
	want := data + header + "CREATE VIEW w AS SELECT 1;\n"
	if out.String() != want {
		t.Errorf("long lines changed (got %d bytes, want %d)", out.Len(), len(want))
	}
	if cap(f.line) > 2*maxHeaderLine {
		t.Errorf("filter buffered %d bytes", cap(f.line))
	}
}

func TestMysqldumpStripDefiner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake mysqldump prints backticks through sh")
	}
	dir := t.TempDir()
	script := writeFakeMysqldump(t, dir, "cat <<'EOF'\n"+mysqldumpSample+"\nEOF\n", "")
	for _, strip := range []bool{false, true} {
		db := testDB(engineMysqldump, script)
		db.StripDefiner = strip
		path, _, err := DumpToFile(context.Background(), db, t.TempDir())
		if err != nil {
			t.Fatalf("stripDefiner=%v: %v", strip, err)
		}
		want := mysqldumpSample + "\n"
		if strip {
			want = mysqldumpSampleStripped + "\n"
		}
		if got := gunzipFile(t, path); got != want {
			t.Errorf("stripDefiner=%v:\n got %q\nwant %q", strip, got, want)
		}
	}
}
