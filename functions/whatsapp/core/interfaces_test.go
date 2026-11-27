package whatsapp_core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// useTempLog points the send log at a daily file in a temp dir and installs
// a fixed clock. It restores the globals when the test ends.
func useTempLog(t *testing.T, start time.Time) (dir string, setNow func(time.Time)) {
	t.Helper()
	dir = t.TempDir()
	var mu sync.Mutex
	current := start
	oldPath, oldNow := OutPutFilePath, now
	OutPutFilePath = currentOutPutFilePath(filepath.Join(dir, "0.log.csv"), start)
	now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return current
	}
	t.Cleanup(func() {
		OutPutFilePath, now = oldPath, oldNow
	})
	return dir, func(next time.Time) {
		mu.Lock()
		defer mu.Unlock()
		current = next
	}
}

func dailyFile(dir string, t time.Time) string {
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
	return filepath.Join(dir, fmt.Sprintf("%d.log.csv", midnight))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestAppendToOutPutFileRotatesDaily(t *testing.T) {
	day1 := time.Date(2026, 11, 27, 23, 59, 0, 0, time.Local)
	day2 := day1.Add(2 * time.Minute)
	dir, setNow := useTempLog(t, day1)

	AppendToOutPutFile("a,true\n")
	setNow(day2)
	AppendToOutPutFile("b,true\n")

	if got := readFile(t, dailyFile(dir, day1)); got != "a,true\n" {
		t.Errorf("day 1 file = %q, want %q", got, "a,true\n")
	}
	if got := readFile(t, dailyFile(dir, day2)); got != "b,true\n" {
		t.Errorf("day 2 file = %q, want %q", got, "b,true\n")
	}
}

func TestAppendToOutPutFileConcurrent(t *testing.T) {
	day := time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local)
	dir, _ := useTempLog(t, day)

	const writers, lines = 8, 50
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range lines {
				AppendToOutPutFile(fmt.Sprintf("%d-%d,true\n", w, i))
			}
		}(w)
	}
	wg.Wait()

	got := strings.Split(strings.TrimSuffix(readFile(t, dailyFile(dir, day)), "\n"), "\n")
	if len(got) != writers*lines {
		t.Fatalf("got %d lines, want %d", len(got), writers*lines)
	}
	for _, line := range got {
		if !strings.HasSuffix(line, ",true") {
			t.Fatalf("interleaved line %q", line)
		}
	}
}

func TestCurrentOutPutFilePath(t *testing.T) {
	ts := time.Date(2026, 11, 27, 15, 0, 0, 0, time.Local)
	want := dailyFile("/logs", ts)
	cases := map[string]string{
		"/logs/1700000000.log.csv": want,
		"/logs/custom.csv":         "/logs/custom.csv",
		"":                         "",
	}
	for base, expected := range cases {
		if got := currentOutPutFilePath(base, ts); got != expected {
			t.Errorf("currentOutPutFilePath(%q) = %q, want %q", base, got, expected)
		}
	}
}

func TestMaskToken(t *testing.T) {
	cases := map[string]string{
		"":                 "***",
		"abc":              "***",
		"abcd":             "***",
		"abcde":            "abcd***",
		"secret-api-token": "secr***",
	}
	for in, want := range cases {
		if got := maskToken(in); got != want {
			t.Errorf("maskToken(%q) = %q, want %q", in, got, want)
		}
		if len(in) > 4 && strings.Contains(maskToken(in), in) {
			t.Errorf("maskToken(%q) leaks the full token", in)
		}
	}
}

func TestSleepCtx(t *testing.T) {
	if !sleepCtx(context.Background(), time.Millisecond) {
		t.Error("sleepCtx with live ctx returned false")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if sleepCtx(ctx, time.Hour) {
		t.Error("sleepCtx with cancelled ctx returned true")
	}
	if time.Since(start) > time.Second {
		t.Error("sleepCtx did not return promptly on cancel")
	}
}

func TestSendMediaFileFromURLsStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		// A cancelled ctx must return before any fetch or delay.
		(&WhatsappConnection{}).SendMediaFileFromURLs(ctx, []string{"1"}, []string{"http://127.0.0.1:1/a", "http://127.0.0.1:1/b"}, "")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SendMediaFileFromURLs ignored ctx cancellation")
	}
}

type sentMessage struct {
	to  types.JID
	msg *waE2E.Message
}

type fakeClient struct {
	notOnWhatsApp map[string]bool
	uploadErr     error
	sendErr       error

	uploads int
	sent    []sentMessage
}

func (f *fakeClient) IsOnWhatsApp(_ context.Context, phones []string) ([]types.IsOnWhatsAppResponse, error) {
	number := strings.TrimPrefix(phones[0], "+")
	return []types.IsOnWhatsAppResponse{{
		Query: phones[0],
		JID:   types.NewJID(number, types.DefaultUserServer),
		IsIn:  !f.notOnWhatsApp[number],
	}}, nil
}

func (f *fakeClient) Upload(context.Context, []byte, whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	f.uploads++
	if f.uploadErr != nil {
		return whatsmeow.UploadResponse{}, f.uploadErr
	}
	return whatsmeow.UploadResponse{URL: "https://example.invalid/m", DirectPath: "/m"}, nil
}

func (f *fakeClient) SendMessage(_ context.Context, to types.JID, message *waE2E.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	if f.sendErr != nil {
		return whatsmeow.SendResponse{}, f.sendErr
	}
	f.sent = append(f.sent, sentMessage{to: to, msg: message})
	return whatsmeow.SendResponse{}, nil
}

func TestSendMediaFileUploadFailureFailsFast(t *testing.T) {
	dir, _ := useTempLog(t, time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local))
	client := &fakeClient{uploadErr: errors.New("upload boom")}
	to := []string{"111", "222", "333"}

	got := *sendMediaFileWith(context.Background(), client, to, []byte("x"), "a.txt", "hi")

	if client.uploads != 1 {
		t.Errorf("uploads = %d, want 1", client.uploads)
	}
	if len(got) != len(to) {
		t.Fatalf("result has %d entries, want %d: %v", len(got), len(to), got)
	}
	for _, number := range to {
		if v, ok := got[number]; !ok || v {
			t.Errorf("result[%s] = %v (present %v), want false", number, v, ok)
		}
	}
	if len(client.sent) != 0 {
		t.Errorf("sent %d messages, want 0", len(client.sent))
	}
	log := readFile(t, dailyFile(dir, now()))
	if n := strings.Count(log, "Error While Uploading"); n != 1 {
		t.Errorf("upload error logged %d times, want 1", n)
	}
}

func TestSendMediaFileUploadsOnceAndSendsAll(t *testing.T) {
	useTempLog(t, time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local))
	client := &fakeClient{notOnWhatsApp: map[string]bool{"222": true}}
	to := []string{"111", "222", "333"}

	got := *sendMediaFileWith(context.Background(), client, to, []byte("x"), "a.txt", "hi")

	want := map[string]bool{"111": true, "222": false, "333": true}
	for number, v := range want {
		if got[number] != v {
			t.Errorf("result[%s] = %v, want %v", number, got[number], v)
		}
	}
	if client.uploads != 1 {
		t.Errorf("uploads = %d, want 1", client.uploads)
	}
	if len(client.sent) != 2 {
		t.Fatalf("sent %d messages, want 2", len(client.sent))
	}
	if c := client.sent[0].msg.GetDocumentMessage().GetCaption(); c != "hi" {
		t.Errorf("document caption = %q, want %q", c, "hi")
	}
}

func TestSendMediaFileSendErrorIsFalse(t *testing.T) {
	useTempLog(t, time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local))
	client := &fakeClient{sendErr: errors.New("send boom")}

	got := *sendMediaFileWith(context.Background(), client, []string{"111"}, []byte("x"), "a.txt", "")

	if v, ok := got["111"]; !ok || v {
		t.Errorf("result[111] = %v (present %v), want false", v, ok)
	}
}

func TestSendMediaFileAudioSendsMessageAsText(t *testing.T) {
	useTempLog(t, time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local))
	client := &fakeClient{}

	got := *sendMediaFileWith(context.Background(), client, []string{"111", "222"}, []byte("x"), "a.mp3", "listen")

	if !got["111"] || !got["222"] {
		t.Fatalf("result = %v, want both true", got)
	}
	if len(client.sent) != 4 {
		t.Fatalf("sent %d messages, want 4 (audio + text per recipient)", len(client.sent))
	}
	for i := 0; i < len(client.sent); i += 2 {
		audio, text := client.sent[i], client.sent[i+1]
		if audio.msg.GetAudioMessage() == nil {
			t.Errorf("message %d is not audio", i)
		}
		if text.msg.GetConversation() != "listen" {
			t.Errorf("message %d text = %q, want %q", i+1, text.msg.GetConversation(), "listen")
		}
		if audio.to != text.to {
			t.Errorf("audio sent to %v but text to %v", audio.to, text.to)
		}
	}
}

func TestSendMediaFileAudioWithoutMessage(t *testing.T) {
	useTempLog(t, time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local))
	client := &fakeClient{}

	sendMediaFileWith(context.Background(), client, []string{"111"}, []byte("x"), "a.mp3", "")

	if len(client.sent) != 1 || client.sent[0].msg.GetAudioMessage() == nil {
		t.Fatalf("sent %v, want one audio message only", client.sent)
	}
}
