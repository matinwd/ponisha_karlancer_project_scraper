package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yaa110/go-persian-calendar"

	"ponisha-go/internal/model"
)

type Sender struct {
	token    string
	chat     string
	threadID *int

	client       *http.Client
	queue        chan string
	minInterval  time.Duration
	lastSentTime time.Time
}

func NewSender(token, chat string, threadID *int) *Sender {
	s := &Sender{
		token:       token,
		chat:        chat,
		threadID:    threadID,
		client:      &http.Client{Timeout: 15 * time.Second},
		queue:       make(chan string, 100),
		minInterval: 1200 * time.Millisecond,
	}

	go s.worker()
	return s
}

func (s *Sender) SendAlert(project model.ScrapedProject) {
	message := formatMessage(project)
	for _, part := range splitMessage(message, 4096) {
		s.queue <- part
	}
}

func (s *Sender) worker() {
	for msg := range s.queue {
		s.sendWithRateLimit(msg)
	}
}

func (s *Sender) sendWithRateLimit(text string) {
	wait := time.Until(s.lastSentTime.Add(s.minInterval))
	if wait > 0 {
		time.Sleep(wait)
	}

	retryAfter, err := s.postMessage(text)
	if err != nil {
		if retryAfter > 0 {
			log.Printf("Telegram rate limit hit. Retrying after %s", retryAfter)
			time.Sleep(retryAfter)
			if _, retryErr := s.postMessage(text); retryErr != nil {
				log.Printf("Telegram retry failed: %v", retryErr)
				return
			}
			s.lastSentTime = time.Now()
			log.Printf("Telegram alert sent successfully (after retry)")
			return
		}

		log.Printf("Telegram send error: %v", err)
		return
	}

	s.lastSentTime = time.Now()
	log.Printf("Telegram alert sent successfully")
}

func (s *Sender) postMessage(text string) (time.Duration, error) {
	payload := map[string]any{
		"chat_id":    s.chat,
		"text":       text,
		"parse_mode": "HTML",
	}
	if s.threadID != nil {
		payload["message_thread_id"] = *s.threadID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var parsed telegramResponse
	_ = json.NewDecoder(resp.Body).Decode(&parsed)

	if resp.StatusCode == http.StatusTooManyRequests && parsed.Parameters.RetryAfter > 0 {
		return time.Duration(parsed.Parameters.RetryAfter) * time.Second, fmt.Errorf("rate limited")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("telegram error: %d %s", resp.StatusCode, parsed.Description)
	}

	return 0, nil
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func formatMessage(project model.ScrapedProject) string {
	skillList := "—"
	if len(project.Skills) > 0 {
		skillList = joinSkills(project.Skills)
	}

	approvedAt := formatPersianTime(project.ApprovedAt)
	biddingClosedAt := formatPersianTime(project.BiddingClosedAt)

	message := fmt.Sprintf("📢 %s\n🌐 منبع: %s\n💰 بودجه: %s\n", project.Title, project.Source, project.BudgetText)
	if project.Description != "" {
		message += fmt.Sprintf("📝 توضیحات: %s\n", project.Description)
	}
	message += fmt.Sprintf("🛠 مهارت‌ها: %s\n", skillList)
	if approvedAt != "" {
		message += fmt.Sprintf("✅ تایید شده: %s\n", approvedAt)
	}
	if biddingClosedAt != "" {
		message += fmt.Sprintf("⏰ پایان مناقصه: %s\n", biddingClosedAt)
	}
	if project.BidsCount != nil {
		message += fmt.Sprintf("📦 تعداد پیشنهادها: %d\n", *project.BidsCount)
	}
	message += fmt.Sprintf("🔗 لینک: %s", project.Link)
	return message
}

func formatPersianTime(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := parseTime(value)
	if err != nil {
		return ""
	}
	pt := ptime.New(parsed)
	return pt.Format("yyyy/MM/dd HH:mm")
}

func parseTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

func joinSkills(skills []string) string {
	if len(skills) == 0 {
		return "—"
	}
	out := ""
	for i, skill := range skills {
		if i > 0 {
			out += ", "
		}
		out += skill
	}
	return out
}

func splitMessage(message string, limit int) []string {
	runes := []rune(message)
	if len(runes) <= limit {
		return []string{message}
	}

	parts := []string{}
	for start := 0; start < len(runes); start += limit {
		end := start + limit
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[start:end]))
	}
	return parts
}
