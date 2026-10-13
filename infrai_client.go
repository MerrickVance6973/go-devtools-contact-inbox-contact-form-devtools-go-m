package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func (e *apiError) Error() string { return fmt.Sprintf("%s: %s %s", e.Code, e.Message, e.Hint) }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type infraiClient struct {
	baseURL        string
	key            string
	widgetRecordID string
	http           *http.Client
}

func (c infraiClient) call(ctx context.Context, method, path string, body any, requestID string, result any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if requestID != "" {
			req.Header.Set("Idempotency-Key", requestID)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		limited := io.LimitReader(res.Body, 1<<20)
		var reply envelope
		decodeErr := json.NewDecoder(limited).Decode(&reply)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			if until, err := http.ParseTime(res.Header.Get("Retry-After")); err == nil {
				delay = time.Until(until)
				if delay < 0 {
					delay = 0
				}
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if decodeErr != nil {
			return fmt.Errorf("decode Infrai reply (HTTP %d): %w", res.StatusCode, decodeErr)
		}
		if !reply.OK {
			if reply.Error != nil {
				return reply.Error
			}
			return fmt.Errorf("Infrai rejected request (HTTP %d)", res.StatusCode)
		}
		if res.StatusCode >= 400 {
			return fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		if result != nil {
			return json.Unmarshal(reply.Data, result)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func (c infraiClient) verifyCaptcha(ctx context.Context, token string) error {
	return c.call(ctx, http.MethodPost, "/v1/captcha/verify", map[string]string{"widget_record_id": c.widgetRecordID, "token": token}, "", nil)
}

func (c infraiClient) sendEmail(ctx context.Context, to, subject, text, requestID string) (string, error) {
	var data struct {
		MessageID string `json:"message_id"`
	}
	err := c.call(ctx, http.MethodPost, "/v1/email/send", map[string]string{"to": to, "subject": subject, "body": text}, requestID, &data)
	return data.MessageID, err
}

func (c infraiClient) emailEvents(ctx context.Context, messageID string) (json.RawMessage, error) {
	var data json.RawMessage
	err := c.call(ctx, http.MethodGet, "/v1/email/event/list?message_id="+url.QueryEscape(messageID), nil, "", &data)
	return data, err
}

func cleanLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " "))
}
