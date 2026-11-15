package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

// newKey writes a key file and returns its path and public key.
func newKey(t *testing.T) (path, pub string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "key.txt")
	pub, err := generateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, pub
}

func recipientsOf(t *testing.T, pubs ...string) []age.Recipient {
	t.Helper()
	r, err := parseRecipients(pubs)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, pub := newKey(t)
	dir := t.TempDir()
	plain := bytes.Repeat([]byte("INSERT INTO t VALUES (1,'x');\n"), 50000) // > one age chunk
	src := writeFile(t, dir, "d.sql.gz", plain)
	enc := filepath.Join(dir, "d.sql.gz.age")
	out := filepath.Join(dir, "back.sql.gz")

	if err := encryptFile(src, enc, recipientsOf(t, pub)); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(enc); bytes.Contains(got, []byte("INSERT INTO")) {
		t.Fatal("encrypted file contains plaintext")
	}
	if err := decryptFile(enc, out, key); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, plain) {
		t.Fatal("round trip changed the data")
	}
	if _, err := os.Stat(enc + ".part"); err == nil {
		t.Error("leftover .part file")
	}
}

func TestEncryptMultipleRecipients(t *testing.T) {
	keyA, pubA := newKey(t)
	keyB, pubB := newKey(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "d", []byte("secret data"))
	enc := filepath.Join(dir, "d.age")
	if err := encryptFile(src, enc, recipientsOf(t, pubA, pubB)); err != nil {
		t.Fatal(err)
	}
	for name, key := range map[string]string{"a": keyA, "b": keyB} {
		out := filepath.Join(dir, "out-"+name)
		if err := decryptFile(enc, out, key); err != nil {
			t.Errorf("key %s cannot decrypt: %v", name, err)
		}
	}
}

func TestDecryptWrongKeyLeavesNothing(t *testing.T) {
	_, pub := newKey(t)
	wrongKey, _ := newKey(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "d", []byte("secret data"))
	enc := filepath.Join(dir, "d.age")
	if err := encryptFile(src, enc, recipientsOf(t, pub)); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := decryptFile(enc, out, wrongKey); err == nil {
		t.Fatal("wrong key decrypted the file")
	}
	for _, p := range []string{out, out + ".part"} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s left behind after a failed decrypt", p)
		}
	}
}

func TestDecryptTamperedFileFails(t *testing.T) {
	key, pub := newKey(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "d", bytes.Repeat([]byte("abc"), 100000))
	enc := filepath.Join(dir, "d.age")
	if err := encryptFile(src, enc, recipientsOf(t, pub)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(enc)
	data[len(data)-20] ^= 0xff
	os.WriteFile(enc, data, 0o644)
	out := filepath.Join(dir, "out")
	if err := decryptFile(enc, out, key); err == nil {
		t.Fatal("tampered file decrypted without error")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("output left behind for a tampered file")
	}
}

func TestDecryptRefusesToOverwrite(t *testing.T) {
	key, pub := newKey(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "d", []byte("x"))
	enc := filepath.Join(dir, "d.age")
	if err := encryptFile(src, enc, recipientsOf(t, pub)); err != nil {
		t.Fatal(err)
	}
	existing := writeFile(t, dir, "out", []byte("keep me"))
	if err := decryptFile(enc, existing, key); err == nil {
		t.Fatal("overwrote an existing file")
	}
	if got, _ := os.ReadFile(existing); string(got) != "keep me" {
		t.Errorf("existing file changed to %q", got)
	}
}

func TestKeygenRefusesToOverwrite(t *testing.T) {
	path, _ := newKey(t)
	if _, err := generateKeyFile(path); err == nil {
		t.Fatal("keygen overwrote an existing key file")
	}
}

func TestKeyFileWithCommentsParses(t *testing.T) {
	path, _ := newKey(t)
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "# created:") {
		t.Fatalf("key file has no comment header: %q", data)
	}
	// decryptFile must cope with the comment lines.
	dir := t.TempDir()
	src := writeFile(t, dir, "d", []byte("x"))
	pubFromFile := strings.TrimSpace(strings.TrimPrefix(strings.Split(string(data), "\n")[1], "# public key:"))
	enc := filepath.Join(dir, "d.age")
	if err := encryptFile(src, enc, recipientsOf(t, pubFromFile)); err != nil {
		t.Fatal(err)
	}
	if err := decryptFile(enc, filepath.Join(dir, "out"), path); err != nil {
		t.Fatal(err)
	}
}

func TestParseRecipientsRejectsBadKey(t *testing.T) {
	if _, err := parseRecipients([]string{"not-a-key"}); err == nil {
		t.Fatal("accepted a bad key")
	}
}

func TestConfigValidatesEncryptTo(t *testing.T) {
	_, pub := newKey(t)
	db := func(keys ...string) *DatabaseConfig {
		return &DatabaseConfig{
			Name:         "shop",
			MysqlConfig:  MysqlConfig{Host: "h", Port: 3306, Username: "u", Database: "d"},
			EncryptTo:    keys,
			Destinations: []Destination{{Type: destLocal, Path: "x"}},
		}
	}
	if p := db(pub).validate(); len(p) != 0 {
		t.Errorf("valid key rejected: %v", p)
	}
	if p := db("bogus").validate(); len(p) == 0 {
		t.Error("bad key accepted")
	}
	if p := db().validate(); len(p) != 0 {
		t.Errorf("no encryption must stay valid: %v", p)
	}
}

func TestRunBackupEncryptsForEveryDestination(t *testing.T) {
	key, pub := newKey(t)
	localDir := filepath.Join(t.TempDir(), "local")
	otherDir := filepath.Join(t.TempDir(), "other")
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0),
		Destination{Type: destLocal, Path: localDir},
		Destination{Type: destLocal, Path: otherDir})
	db.EncryptTo = []string{pub}

	res := RunBackup(context.Background(), db, &recordingNotifier{})
	if !res.OK() {
		t.Fatalf("backup failed: dump=%v dests=%+v", res.Err, res.Dests)
	}
	if !res.Encrypted {
		t.Error("result not marked encrypted")
	}
	for _, dir := range []string{localDir, otherDir} {
		if plain, _ := filepath.Glob(filepath.Join(dir, "*.sql.gz")); len(plain) != 0 {
			t.Errorf("%s holds a plain backup: %v", dir, plain)
		}
		files, _ := filepath.Glob(filepath.Join(dir, "shop-*.sql.gz.age"))
		if len(files) != 1 {
			t.Fatalf("want one encrypted backup in %s, got %v", dir, files)
		}
		out := filepath.Join(t.TempDir(), "restored.sql.gz")
		if err := decryptFile(files[0], out, key); err != nil {
			t.Fatal(err)
		}
		f, _ := os.Open(out)
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("decrypted file is not gzip: %v", err)
		}
		body, _ := io.ReadAll(zr)
		f.Close()
		if !strings.Contains(string(body), "Dump completed") {
			t.Errorf("restored dump is incomplete: %q", body)
		}
	}
}

func TestRunBackupWithoutEncryptionStaysPlain(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "b")
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0), Destination{Type: destLocal, Path: dir})
	res := RunBackup(context.Background(), db, &recordingNotifier{})
	if !res.OK() || res.Encrypted {
		t.Fatalf("ok=%v encrypted=%v", res.OK(), res.Encrypted)
	}
	if enc, _ := filepath.Glob(filepath.Join(dir, "*.age")); len(enc) != 0 {
		t.Errorf("unexpected encrypted file: %v", enc)
	}
}

func TestPruneLocalCoversEncryptedBackups(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{
		"db-20260101-000001.sql.gz.age",
		"db-20260102-000001.sql.gz.age",
		"db-20260103-000001.sql.gz.age",
		"db-eu-20260101-000001.sql.gz.age", // another database, must survive
	} {
		writeFile(t, dir, n, []byte("x"))
	}
	if err := pruneLocal(dir, "db", 2, 0); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(listNames(t, dir), ",")
	want := "db-20260102-000001.sql.gz.age,db-20260103-000001.sql.gz.age,db-eu-20260101-000001.sql.gz.age"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestSubcommands(t *testing.T) {
	var out, errb bytes.Buffer
	if handled, _ := runSubcommand([]string{"backup-now"}, &out, &errb); handled {
		t.Error("unknown word treated as a subcommand")
	}
	keyPath := filepath.Join(t.TempDir(), "k.txt")
	if handled, code := runSubcommand([]string{"keygen", "-o", keyPath}, &out, &errb); !handled || code != 0 {
		t.Fatalf("keygen: handled=%v code=%d err=%s", handled, code, errb.String())
	}
	pub := strings.TrimSpace(strings.Split(out.String(), "\n")[1])
	if !strings.HasPrefix(pub, "age1") {
		t.Fatalf("keygen printed %q instead of a public key", pub)
	}
	if _, code := runSubcommand([]string{"keygen"}, &out, &errb); code != 2 {
		t.Errorf("keygen without -o: code %d, want 2", code)
	}

	dir := t.TempDir()
	src := writeFile(t, dir, "d.sql.gz", []byte("hello"))
	enc := src + encryptedSuffix
	if err := encryptFile(src, enc, recipientsOf(t, pub)); err != nil {
		t.Fatal(err)
	}
	os.Remove(src)
	if _, code := runSubcommand([]string{"decrypt", "--identity", keyPath, enc}, &out, &errb); code != 0 {
		t.Fatalf("decrypt failed: %s", errb.String())
	}
	if got, _ := os.ReadFile(src); string(got) != "hello" {
		t.Errorf("decrypted to %q", got)
	}
	if _, code := runSubcommand([]string{"decrypt", "--identity", keyPath, src}, &out, &errb); code != 2 {
		t.Errorf("decrypt of a non-.age file without -o: code %d, want 2", code)
	}
}
