package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// telegramBaseURL is a variable so tests can point it at a local server.
var telegramBaseURL = "https://api.telegram.org"

// chatID is a Telegram chat id. JSON may give it as a number (123456, or
// -1001234567890 for a group) or as a string ("@mychannel").
type chatID string

func (c *chatID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*c = chatID(strings.TrimSpace(s))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("chatId must be a number or a string")
	}
	*c = chatID(n.String())
	return nil
}

// TelegramConfig sends every backup result to a Telegram chat through a bot.
type TelegramConfig struct {
	BotToken string `json:"botToken"`
	ChatID   chatID `json:"chatId"`
}

func (t *TelegramConfig) validate() []string {
	var p []string
	if t.BotToken == "" || t.ChatID == "" {
		p = append(p, "telegram needs botToken and chatId")
	}
	if t.BotToken != "" && !strings.Contains(t.BotToken, ":") {
		p = append(p, "telegram botToken looks wrong (expected 123456:ABC...)")
	}
	if strings.ContainsAny(t.BotToken+string(t.ChatID), "<>") {
		p = append(p, "telegram still holds a placeholder value")
	}
	return p
}

// sendTelegram posts text to the chat, retrying a few times. The bot token is
// never part of a returned error.
func sendTelegram(ctx context.Context, t *TelegramConfig, text string) error {
	body, err := json.Marshal(map[string]string{"chat_id": string(t.ChatID), "text": text})
	if err != nil {
		return err
	}
	endpoint := telegramBaseURL + "/bot" + t.BotToken + "/sendMessage"
	err = withRetries(ctx, defaultRetries, func() error {
		reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			io.Copy(io.Discard, resp.Body)
			return nil
		}
		var reply struct {
			Description string `json:"description"`
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		json.Unmarshal(data, &reply)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, reply.Description)
	})
	if err != nil {
		return fmt.Errorf("telegram: %s", strings.ReplaceAll(err.Error(), t.BotToken, "***"))
	}
	return nil
}

// telegramText is the message for one backup result.
func telegramText(r *Result) string {
	host, _ := os.Hostname()
	var b strings.Builder
	if r.OK() {
		fmt.Fprintf(&b, "✅ %s backup OK", r.Name)
	} else {
		fmt.Fprintf(&b, "❌ %s backup FAILED", r.Name)
	}
	if host != "" {
		fmt.Fprintf(&b, " (%s)", host)
	}
	b.WriteString("\n")
	if r.Err == nil {
		fmt.Fprintf(&b, "%s via %s in %s%s\n", humanSize(r.Size), r.Engine, r.Duration.Round(100*time.Millisecond), encryptedNote(r))
	} else {
		fmt.Fprintf(&b, "dump: %s\n", clip(firstLine(r.Err.Error()), 300))
	}
	for _, d := range r.Dests {
		if d.Err != nil {
			fmt.Fprintf(&b, "✗ %s: %s\n", d.Label, clip(firstLine(d.Err.Error()), 300))
		} else if r.Err == nil {
			fmt.Fprintf(&b, "→ %s\n", d.Label)
		}
	}
	return clip(strings.TrimSpace(b.String()), 3500)
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// notifyTelegram reports one backup result. It never fails the backup: a
// problem sending is only logged.
func notifyTelegram(ctx context.Context, cfg *Config, r *Result) {
	if cfg.Telegram == nil {
		return
	}
	// A shutdown signal must not stop the report of the backup it interrupted.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	if err := sendTelegram(ctx, cfg.Telegram, telegramText(r)); err != nil {
		log.Printf("[%s] %v", r.Name, err)
	}
}
