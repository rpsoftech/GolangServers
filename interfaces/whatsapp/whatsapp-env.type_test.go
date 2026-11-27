package whatsapp_interfaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func newTestConfig(t *testing.T) *IServerConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.config.json")
	if err := os.WriteFile(path, []byte(`{"tokens":{"existing":"jid-1"},"JID":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	return ReadConfigFileAndReturniserverConfig(path)
}

func readSaved(t *testing.T, path string) map[string]map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved file is not valid JSON: %v\n%s", err, data)
	}
	return saved
}

func TestReadConfigInitialisesJID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.config.json")
	if err := os.WriteFile(path, []byte(`{"tokens":{"a":""}}`), 0644); err != nil {
		t.Fatal(err)
	}
	sc := ReadConfigFileAndReturniserverConfig(path)
	sc.SetJID("a", "jid") // must not panic on a nil map
	if got := sc.GetJID("a"); got != "jid" {
		t.Fatalf("GetJID = %q, want jid", got)
	}
	if sc.GetConfigPath() != path {
		t.Fatalf("GetConfigPath = %q, want %q", sc.GetConfigPath(), path)
	}
}

func TestAddTokenRefusesDuplicates(t *testing.T) {
	sc := newTestConfig(t)
	if sc.AddToken("existing", "other") {
		t.Fatal("AddToken accepted an existing token")
	}
	if v, _ := sc.GetToken("existing"); v != "jid-1" {
		t.Fatalf("existing token value = %q, want jid-1 (must not be overwritten)", v)
	}
	if !sc.AddToken("new", "v") {
		t.Fatal("AddToken refused a new token")
	}
	if sc.AddToken("new", "v2") {
		t.Fatal("AddToken accepted the same new token twice")
	}
	if v, ok := sc.GetToken("new"); !ok || v != "v" {
		t.Fatalf("GetToken(new) = %q, %v; want v, true", v, ok)
	}
	if _, ok := sc.GetToken("missing"); ok {
		t.Fatal("GetToken found a missing token")
	}
}

func TestAddTokenConcurrentDuplicatesOnlyOneWins(t *testing.T) {
	sc := newTestConfig(t)
	const workers = 32
	var wg sync.WaitGroup
	results := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- sc.AddToken("race", "v")
		}()
	}
	wg.Wait()
	close(results)
	wins := 0
	for ok := range results {
		if ok {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("%d AddToken calls won, want exactly 1", wins)
	}
}

func TestSaveWritesValidJSON(t *testing.T) {
	sc := newTestConfig(t)
	sc.AddToken("t2", "")
	sc.SetJID("existing", "123@s.whatsapp.net")
	sc.Save()

	saved := readSaved(t, sc.GetConfigPath())
	if saved["tokens"]["existing"] != "jid-1" || saved["tokens"]["t2"] != "" {
		t.Fatalf("saved tokens = %v", saved["tokens"])
	}
	if _, ok := saved["tokens"]["t2"]; !ok {
		t.Fatal("saved tokens miss t2")
	}
	if saved["JID"]["existing"] != "123@s.whatsapp.net" {
		t.Fatalf("saved JID = %v", saved["JID"])
	}
	if len(saved) != 2 {
		t.Fatalf("saved keys = %v, want only tokens and JID", saved)
	}

	// The file must load back.
	again := ReadConfigFileAndReturniserverConfig(sc.GetConfigPath())
	if v, ok := again.GetToken("existing"); !ok || v != "jid-1" {
		t.Fatalf("reloaded token = %q, %v", v, ok)
	}
}

func TestSaveToNewPath(t *testing.T) {
	sc := newTestConfig(t)
	path := filepath.Join(t.TempDir(), "copy.json")
	sc.SetConfigPath(path).Save()
	readSaved(t, path)
}

// Run with -race: readers, writers and saves on one config at the same time.
func TestConcurrentAccess(t *testing.T) {
	sc := newTestConfig(t)
	const workers = 16
	const rounds = 50
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				token := fmt.Sprintf("tok-%d-%d", w, i)
				sc.AddToken(token, "")
				sc.GetToken(token)
				sc.GetToken("existing")
				sc.SetJID(token, "jid")
				sc.GetJID(token)
				if i%10 == 0 {
					sc.DeleteJID(token)
					for range sc.TokensSnapshot() {
					}
					sc.Save()
				}
			}
		}(w)
	}
	wg.Wait()
	sc.Save()

	saved := readSaved(t, sc.GetConfigPath())
	if got, want := len(saved["tokens"]), workers*rounds+1; got != want {
		t.Fatalf("saved %d tokens, want %d", got, want)
	}
	if got, want := len(saved["JID"]), workers*(rounds-rounds/10); got != want {
		t.Fatalf("saved %d JIDs, want %d", got, want)
	}
}
