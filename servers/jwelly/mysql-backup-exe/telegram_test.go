package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testBotToken = "123456:SECRET-token"

type tgCall struct {
	path string
	body map[string]string
}

// fakeTelegram answers like the Bot API and records the calls. failFirst
// requests get HTTP 500 before it starts answering 200.
func fakeTelegram(t *testing.T, failFirst int, status int) (*[]tgCall, *atomic.Int32) {
	t.Helper()
	var calls []tgCall
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		data, _ := io.ReadAll(r.Body)
		body := map[string]string{}
		json.Unmarshal(data, &body)
		calls = append(calls, tgCall{r.URL.Path, body})
		if int(n.Load()) <= failFirst {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(status)
		io.WriteString(w, `{"ok":false,"description":"Unauthorized"}`)
	}))
	t.Cleanup(srv.Close)
	old := telegramBaseURL
	telegramBaseURL = srv.URL
	t.Cleanup(func() { telegramBaseURL = old })
	return &calls, &n
}

func okResult() *Result {
	return &Result{Name: "shop", Engine: "mysqldump", Size: 2048, Duration: 3 * time.Second,
		Dests: []DestResult{{Label: "local D:/b"}}}
}

func TestSendTelegramPostsMessage(t *testing.T) {
	noBackoff(t)
	calls, _ := fakeTelegram(t, 0, 200)
	tg := &TelegramConfig{BotToken: testBotToken, ChatID: "-100123"}
	if err := sendTelegram(context.Background(), tg, "hello"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 {
		t.Fatalf("calls = %d", len(*calls))
	}
	c := (*calls)[0]
	if c.path != "/bot"+testBotToken+"/sendMessage" || c.body["chat_id"] != "-100123" || c.body["text"] != "hello" {
		t.Errorf("unexpected request: %+v", c)
	}
}

func TestSendTelegramRetriesServerErrors(t *testing.T) {
	noBackoff(t)
	_, n := fakeTelegram(t, 2, 200)
	tg := &TelegramConfig{BotToken: testBotToken, ChatID: "1"}
	if err := sendTelegram(context.Background(), tg, "x"); err != nil {
		t.Fatalf("should succeed on the third try: %v", err)
	}
	if n.Load() != 3 {
		t.Errorf("requests = %d, want 3", n.Load())
	}
}

func TestSendTelegramErrorsNeverHoldTheToken(t *testing.T) {
	noBackoff(t)
	fakeTelegram(t, 0, 401)
	tg := &TelegramConfig{BotToken: testBotToken, ChatID: "1"}
	err := sendTelegram(context.Background(), tg, "x")
	if err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("want an Unauthorized error, got %v", err)
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Errorf("token leaked in %q", err)
	}

	// An unreachable server puts the URL, and so the token, in the raw error.
	telegramBaseURL = "http://127.0.0.1:1"
	err = sendTelegram(context.Background(), tg, "x")
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Errorf("token leaked in %q", err)
	}
}

func TestTelegramText(t *testing.T) {
	ok := telegramText(okResult())
	for _, want := range []string{"✅ shop backup OK", "2.0 KB", "mysqldump", "→ local D:/b"} {
		if !strings.Contains(ok, want) {
			t.Errorf("OK message missing %q:\n%s", want, ok)
		}
	}
	bad := okResult()
	bad.Dests = []DestResult{{Label: "r2 bkt", Err: errors.New("denied\nsecond line")}}
	msg := telegramText(bad)
	if !strings.Contains(msg, "❌ shop backup FAILED") || !strings.Contains(msg, "✗ r2 bkt: denied") {
		t.Errorf("failure message wrong:\n%s", msg)
	}
	if strings.Contains(msg, "second line") {
		t.Errorf("only the first line of an error belongs in the message:\n%s", msg)
	}
	dump := &Result{Name: "shop", Err: errors.New("mysqldump not found")}
	if m := telegramText(dump); !strings.Contains(m, "dump: mysqldump not found") {
		t.Errorf("dump failure missing:\n%s", m)
	}
	long := &Result{Name: "shop", Err: errors.New(strings.Repeat("x", 9000))}
	if n := len([]rune(telegramText(long))); n > 4096 {
		t.Errorf("message of %d characters exceeds Telegram's limit", n)
	}
}

func TestNotifyTelegramOnlyWhenConfigured(t *testing.T) {
	noBackoff(t)
	calls, _ := fakeTelegram(t, 0, 200)
	notifyTelegram(context.Background(), &Config{}, okResult())
	if len(*calls) != 0 {
		t.Fatal("sent a message without telegram configured")
	}
	cfg := &Config{Telegram: &TelegramConfig{BotToken: testBotToken, ChatID: "1"}}
	notifyTelegram(context.Background(), cfg, okResult())
	if len(*calls) != 1 || !strings.Contains((*calls)[0].body["text"], "shop backup OK") {
		t.Errorf("calls = %+v", *calls)
	}
}

func TestNotifyTelegramFailureDoesNotPanicOrBlock(t *testing.T) {
	noBackoff(t)
	telegramBaseURL = "http://127.0.0.1:1"
	t.Cleanup(func() { telegramBaseURL = "https://api.telegram.org" })
	cfg := &Config{Telegram: &TelegramConfig{BotToken: testBotToken, ChatID: "1"}}
	notifyTelegram(context.Background(), cfg, okResult()) // must just log
}

func TestTelegramConfigParsingAndValidation(t *testing.T) {
	for _, in := range []string{`{"botToken":"1:a","chatId":-100123}`, `{"botToken":"1:a","chatId":"-100123"}`} {
		var tg TelegramConfig
		if err := json.Unmarshal([]byte(in), &tg); err != nil || tg.ChatID != "-100123" {
			t.Errorf("%s: chatId %q err %v", in, tg.ChatID, err)
		}
	}
	var bad TelegramConfig
	if err := json.Unmarshal([]byte(`{"chatId":true}`), &bad); err == nil {
		t.Error("a boolean chatId was accepted")
	}

	db := &DatabaseConfig{
		Name:         "shop",
		MysqlConfig:  MysqlConfig{Host: "h", Port: 3306, Username: "u", Database: "d"},
		Destinations: []Destination{{Type: destLocal, Path: "x"}},
	}
	cfg := func(tg *TelegramConfig) *Config { return &Config{Telegram: tg, Databases: []*DatabaseConfig{db}} }
	if err := cfg(&TelegramConfig{BotToken: "1:a", ChatID: "5"}).Validate(); err != nil {
		t.Errorf("valid telegram rejected: %v", err)
	}
	if err := cfg(nil).Validate(); err != nil {
		t.Errorf("no telegram must stay valid: %v", err)
	}
	for name, tg := range map[string]*TelegramConfig{
		"missing token":  {ChatID: "5"},
		"missing chat":   {BotToken: "1:a"},
		"token no colon": {BotToken: "abc", ChatID: "5"},
		"placeholder":    {BotToken: "<token>:x", ChatID: "5"},
	} {
		if err := cfg(tg).Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestUpdateHelpers(t *testing.T) {
	if shouldCheckUpdate(false, "") {
		t.Error("an unpublished build must not update")
	}
	if shouldCheckUpdate(true, "1") {
		t.Error("the process started by an update must not check again")
	}
	if !shouldCheckUpdate(true, "") {
		t.Error("a published build should check")
	}
	got := restartArgs([]string{"--config", "c.json"})
	if got[0] != "--no-pause" || len(got) != 3 || got[2] != "c.json" {
		t.Errorf("restartArgs = %v", got)
	}
}

// TestHelperProcess is the child that TestRunChild starts; it is not a test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	code, _ := strconv.Atoi(os.Getenv("HELPER_EXIT"))
	if os.Getenv(skipUpdateEnv) != "1" {
		code = 99
	}
	os.Exit(code)
}

func TestRunChildPassesExitCodeAndSkipsUpdate(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_EXIT", "7")
	if got := runChild(os.Args[0], []string{"-test.run=TestHelperProcess"}); got != 7 {
		t.Errorf("exit code = %d, want 7 (99 means the skip-update variable was missing)", got)
	}
	t.Setenv("HELPER_EXIT", "0")
	if got := runChild(os.Args[0], []string{"-test.run=TestHelperProcess"}); got != 0 {
		t.Errorf("exit code = %d, want 0", got)
	}
	if got := runChild("definitely-not-a-real-binary-xyz", nil); got != 1 {
		t.Errorf("missing executable gave exit code %d, want 1", got)
	}
}
