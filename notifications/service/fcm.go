package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FCMSender sends push notifications via Firebase Cloud Messaging legacy HTTP API.
type FCMSender struct {
	serverKey  string
	httpClient *http.Client
}

func NewFCMSender(serverKey string) *FCMSender {
	return &FCMSender{
		serverKey:  serverKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *FCMSender) SendToToken(ctx context.Context, token, title, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload := map[string]any{
		"to": token,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("fcm: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fcm.googleapis.com/fcm/send", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("fcm: build request: %w", err)
	}
	req.Header.Set("Authorization", "key="+f.serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fcm: unexpected status %d", resp.StatusCode)
	}
	return nil
}
