package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type PostEvent struct {
	Event       string `json:"event"`
	BoardSlug   string `json:"board_slug"`
	ThreadID    int    `json:"thread_id"`
	PostID      int64  `json:"post_id,omitempty"`
	TextPreview string `json:"text_preview,omitempty"`
}

func PostJSON(url, secret string, ev PostEvent) {
	if url == "" {
		return
	}
	go func() {
		b, err := json.Marshal(ev)
		if err != nil {
			return
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			req.Header.Set("X-Webhook-Secret", secret)
		}
		c := &http.Client{Timeout: 6 * time.Second}
		_, _ = c.Do(req)
	}()
}
