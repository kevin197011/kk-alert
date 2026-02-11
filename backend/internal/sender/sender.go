package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
)

// TelegramConfig from channel config JSON.
type TelegramConfig struct {
	Token  string `json:"token"`
	ChatID string `json:"chat_id"`
}

// LarkConfig from channel config JSON (webhook).
type LarkConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// larkRateLimiter implements a token bucket rate limiter for Lark webhook API
// Limits: 5 requests per second with burst of 3
type larkRateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	lastTime time.Time
	rate     float64 // tokens per second
	burst    float64 // max burst size
}

var larkLimiter = &larkRateLimiter{
	rate:   5, // 5 requests per second
	burst:  3, // burst of 3
	tokens: 3, // start with full bucket
}

func (rl *larkRateLimiter) acquire() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.lastTime = now

	// Add tokens based on elapsed time
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	// If no tokens available, wait
	if rl.tokens < 1 {
		sleepTime := time.Duration((1 - rl.tokens) / rl.rate * float64(time.Second))
		log.Printf("[lark rate limiter] waiting %v for token (tokens=%.2f)", sleepTime, rl.tokens)
		rl.mu.Unlock()
		time.Sleep(sleepTime)
		rl.mu.Lock()
		rl.tokens = 0
	}

	rl.tokens--
}

var labelRe = regexp.MustCompile(`\{\{\.Labels\.(\w+)\}\}`)

// AlertTemplateData is the struct passed to Go templates for alert notification body.
// Supports {{.AlertID}}, {{.Title}}, {{.Severity}}, {{.StartAt}}, {{.SentAt}}, {{.SourceType}}, {{.Description}}, {{.Value}}, {{range .Labels}}, {{if .IsRecovery}} etc.
type AlertTemplateData struct {
	AlertID     string
	Title       string
	Severity    string
	Labels      map[string]string
	StartAt     string
	SourceType  string
	Description string
	// Value is the trigger value (e.g. PromQL result), for use in template as {{.Value}} (e.g. 当前值/阈值).
	Value string
	// IsRecovery is true when rendering a recovery notification; use {{if .IsRecovery}} in template to show different style.
	IsRecovery  bool
	// ResolvedAt is the resolution time (e.g. "2006-01-02 15:04:05"), empty when firing.
	ResolvedAt  string
	// RuleDescription is the rule's description (purpose/usage), for use in template as {{.RuleDescription}}.
	RuleDescription string
	// SentAt is when this notification is sent (e.g. "2006-01-02 15:04:05" in Asia/Shanghai), for {{.SentAt}} in template.
	SentAt string
}

// RenderTemplate renders the body with text/template so {{.StartAt}}, {{range .Labels}}, {{.Description}} etc. work.
func RenderTemplate(body string, data AlertTemplateData) (string, error) {
	if data.Labels == nil {
		data.Labels = make(map[string]string)
	}
	tpl, err := template.New("alert").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderBody replaces {{.AlertID}}, {{.Title}}, {{.Severity}}, {{.Labels.xxx}} in body with alert data.
// Used as fallback when RenderTemplate fails or for the default template.
func RenderBody(body string, labels map[string]string, alertID, title, severity string) string {
	out := []byte(body)
	out = bytes.ReplaceAll(out, []byte("{{.AlertID}}"), []byte(alertID))
	out = bytes.ReplaceAll(out, []byte("{{.Title}}"), []byte(title))
	out = bytes.ReplaceAll(out, []byte("{{.Severity}}"), []byte(severity))
	out = labelRe.ReplaceAllFunc(out, func(m []byte) []byte {
		sub := labelRe.FindSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		if v, ok := labels[string(sub[1])]; ok {
			return []byte(v)
		}
		return []byte("")
	})
	return string(out)
}

const maxSendRetries = 3
const retryDelay = time.Second

// Send delivers a message to the channel with automatic retry (max 3 times) to avoid losing alerts. isRecovery: when true, Lark uses green card header; when false, red (alert).
func Send(channelType, configJSON, title, body string, isRecovery bool) error {
	var lastErr error
	for attempt := 1; attempt <= maxSendRetries; attempt++ {
		switch channelType {
		case "telegram":
			lastErr = sendTelegram(configJSON, title, body, isRecovery)
		case "lark":
			lastErr = sendLark(configJSON, title, body, isRecovery)
		default:
			return fmt.Errorf("unsupported channel type: %s", channelType)
		}
		if lastErr == nil {
			return nil
		}
		if attempt < maxSendRetries {
			log.Printf("[sender] send failed (attempt %d/%d): %v; retrying in %v", attempt, maxSendRetries, lastErr, retryDelay*time.Duration(attempt))
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}
	return fmt.Errorf("send failed after %d attempts: %w", maxSendRetries, lastErr)
}

func sendTelegram(configJSON, _ string, body string, isRecovery bool) error {
	var cfg TelegramConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.Token == "" || cfg.ChatID == "" {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	header := "告警通知"
	if isRecovery {
		header = "恢复通知"
	}
	text := header
	if body != "" {
		text = header + "\n" + strings.TrimLeft(body, "\n\r\t ")
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Token)
	payload := map[string]interface{}{
		"chat_id": cfg.ChatID,
		"text":    text,
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api %d: %s", resp.StatusCode, string(bb))
	}
	return nil
}

func sendLark(configJSON, title, body string, isRecovery bool) error {
	var cfg LarkConfig
	raw := strings.TrimSpace(configJSON)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		cfg.WebhookURL = raw
	} else if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.WebhookURL == "" {
		return fmt.Errorf("invalid lark config: use JSON {\"webhook_url\":\"...\"} or paste the webhook URL directly: %w", err)
	}

	log.Printf("[lark] waiting for rate limiter, webhook: %s...", cfg.WebhookURL[:50])
	larkLimiter.acquire()
	log.Printf("[lark] rate limiter acquired, sending message")

	// Use interactive card so alert=red header, recovery=green header for visual distinction
	headerTemplate := "red"
	headerTitle := "告警通知"
	if isRecovery {
		headerTemplate = "green"
		headerTitle = "恢复通知"
	}
	// Card header already shows "告警通知"/"恢复"; body content only, trim leading blank lines
	content := strings.TrimLeft(body, "\n\r\t ")
	if content == "" {
		content = title
	}
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"config": map[string]interface{}{"wide_screen_mode": true},
			"header": map[string]interface{}{
				"template": headerTemplate,
				"title":    map[string]interface{}{"tag": "plain_text", "content": headerTitle},
			},
			"elements": []map[string]interface{}{
				{"tag": "div", "text": map[string]interface{}{"tag": "lark_md", "content": content}},
			},
		},
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lark read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lark api %d: %s", resp.StatusCode, string(bb))
	}
	// Lark/Feishu returns HTTP 200 even on failure; real result is in body: {"code":0,"msg":"success"} or {"code":19001,"msg":"..."}
	var larkResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(bb, &larkResp); err == nil && larkResp.Code != 0 {
		return fmt.Errorf("lark api error: code=%d msg=%s", larkResp.Code, larkResp.Msg)
	}
	return nil
}
