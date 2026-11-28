package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// Levels, lowest to highest.
var levelRank = map[string]int{"info": 1, "warning": 2, "critical": 3}

func validLevel(l string) bool { _, ok := levelRank[l]; return ok }

var levelIcon = map[string]string{"info": "🔵", "warning": "🟠", "critical": "🔴"}

type TelegramChat struct {
	Name     string `json:"name"`
	ChatID   string `json:"chat_id"`
	MinLevel string `json:"min_level"` // receive this level and above
}

type TelegramSettings struct {
	Enabled  bool           `json:"enabled"`
	BotToken string         `json:"bot_token"`
	Chats    []TelegramChat `json:"chats"`
}

const settingTelegram = "telegram"

// Notifier sends alert messages through the Telegram Bot API.
type Notifier struct {
	store   *Store
	client  *http.Client
	apiBase string // overridable for tests

	mu       sync.Mutex
	settings TelegramSettings
	queue    chan outMsg
}

type outMsg struct {
	chat TelegramChat
	text string
}

func NewNotifier(store *Store) *Notifier {
	n := &Notifier{store: store, client: &http.Client{Timeout: 15 * time.Second},
		apiBase: "https://api.telegram.org", queue: make(chan outMsg, 256)}
	var s TelegramSettings
	if ok, _ := store.GetSetting(settingTelegram, &s); ok {
		n.settings = s
	}
	go n.worker()
	return n
}

func (n *Notifier) Settings() TelegramSettings {
	n.mu.Lock()
	defer n.mu.Unlock()
	s := n.settings
	s.Chats = append([]TelegramChat{}, s.Chats...)
	return s
}

// Update stores new settings. An empty BotToken keeps the stored token.
func (n *Notifier) Update(s TelegramSettings) error {
	for i := range s.Chats {
		if s.Chats[i].ChatID == "" {
			return errors.New("chat_id is required")
		}
		if s.Chats[i].MinLevel == "" {
			s.Chats[i].MinLevel = "warning"
		}
		if !validLevel(s.Chats[i].MinLevel) {
			return fmt.Errorf("bad min_level %q", s.Chats[i].MinLevel)
		}
	}
	n.mu.Lock()
	if s.BotToken == "" {
		s.BotToken = n.settings.BotToken
	}
	n.settings = s
	n.mu.Unlock()
	return n.store.PutSetting(settingTelegram, s)
}

// Notify queues a message to every chat whose min level allows it. chatIDs, if
// not empty, restricts delivery to those chats. It returns the chat count.
func (n *Notifier) Notify(level, text string, chatIDs []string) int {
	s := n.Settings()
	if !s.Enabled || s.BotToken == "" {
		return 0
	}
	count := 0
	for _, c := range s.Chats {
		if levelRank[level] < levelRank[c.MinLevel] {
			continue
		}
		if len(chatIDs) > 0 && !contains(chatIDs, c.ChatID) {
			continue
		}
		select {
		case n.queue <- outMsg{chat: c, text: text}:
			count++
		default:
			log.Printf("telegram queue full, dropping message for %s", c.Name)
		}
	}
	return count
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func (n *Notifier) worker() {
	for m := range n.queue {
		for attempt := 0; attempt < 4; attempt++ {
			wait, err := n.send(m.chat.ChatID, m.text)
			if err == nil {
				break
			}
			log.Printf("telegram send to %s failed: %v", m.chat.Name, err)
			if wait == 0 {
				wait = time.Duration(attempt+1) * 2 * time.Second
			}
			time.Sleep(wait)
		}
		time.Sleep(1100 * time.Millisecond) // stay under per-chat rate limits
	}
}

// SendTest sends a message right away and returns the API error, if any.
func (n *Notifier) SendTest(chatID, text string) error {
	_, err := n.send(chatID, text)
	return err
}

// send posts one message. On HTTP 429 it returns the retry_after delay.
func (n *Notifier) send(chatID, text string) (time.Duration, error) {
	s := n.Settings()
	if s.BotToken == "" {
		return 0, errors.New("bot token not set")
	}
	body, _ := json.Marshal(map[string]any{
		"chat_id": chatID, "text": text, "parse_mode": "HTML", "disable_web_page_preview": true,
	})
	resp, err := n.client.Post(n.apiBase+"/bot"+s.BotToken+"/sendMessage", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, errors.New(redact(err.Error(), s.BotToken))
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode == http.StatusOK {
		return 0, nil
	}
	var e struct {
		Description string `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	_ = json.Unmarshal(b, &e)
	wait := time.Duration(e.Parameters.RetryAfter) * time.Second
	return wait, fmt.Errorf("telegram %d: %s", resp.StatusCode, e.Description)
}

func redact(s, token string) string {
	if token == "" {
		return s
	}
	return string(bytes.ReplaceAll([]byte(s), []byte(token), []byte("***")))
}

// FormatAlert builds the HTML message for a firing or resolved alert.
func FormatAlert(state, level, rule, server, proc, detail string) string {
	icon := levelIcon[level]
	head := fmt.Sprintf("%s <b>%s</b> %s", icon, html.EscapeString(levelUpper(level)), html.EscapeString(rule))
	if state == "resolved" {
		head = fmt.Sprintf("✅ <b>RESOLVED</b> %s", html.EscapeString(rule))
	}
	target := html.EscapeString(server)
	if proc != "" {
		target += " / " + html.EscapeString(proc)
	}
	return fmt.Sprintf("%s\n<b>%s</b>\n%s", head, target, html.EscapeString(detail))
}

func levelUpper(l string) string {
	switch l {
	case "critical":
		return "CRITICAL"
	case "warning":
		return "WARNING"
	}
	return "INFO"
}
