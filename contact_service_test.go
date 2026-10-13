package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContactRouting(t *testing.T) {
	for _, tc := range []struct {
		name, captchaReply                   string
		captchaStatus, wantStatus, wantSends int
	}{
		{"accepted build", `{"ok":true,"data":{}}`, 200, 202, 1},
		{"rejected bot", `{"ok":false,"error":{"code":"CAPTCHA_SCORE_TOO_LOW","message":"score rejected"}}`, 422, 422, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sends := 0
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("missing credential")
				}
				switch r.URL.Path {
				case "/v1/captcha/verify":
					if r.Method != http.MethodPost {
						t.Errorf("captcha method: %s", r.Method)
					}
					var payload map[string]string
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["widget_record_id"] != "widget-42" || payload["token"] != "verified-token" || len(payload) != 2 {
						t.Errorf("captcha payload: %#v (decode error: %v)", payload, err)
					}
					w.WriteHeader(tc.captchaStatus)
					w.Write([]byte(tc.captchaReply))
				case "/v1/email/send":
					sends++
					if r.Method != http.MethodPost || r.Header.Get("Idempotency-Key") != "build-0042" {
						t.Error("send boundary changed")
					}
					var payload map[string]string
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["body"] == "" || payload["text"] != "" || len(payload) != 3 {
						t.Errorf("email payload: %#v (decode error: %v)", payload, err)
					}
					w.Write([]byte(`{"ok":true,"data":{"message_id":"msg-42"}}`))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			}))
			defer api.Close()
			service := contactService{api: infraiClient{baseURL: api.URL, key: "test-key", widgetRecordID: "widget-42", http: api.Client()}, inbox: "team@example.org"}
			body := `{"submission_id":"build-0042","kind":"build","email":"dev@example.org","title":"CI failed","details":"Build 42 needs review","captcha_token":"verified-token"}`
			rec := httptest.NewRecorder()
			service.submit(rec, httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(body)))
			if rec.Code != tc.wantStatus || sends != tc.wantSends {
				t.Fatalf("status=%d sends=%d body=%s", rec.Code, sends, rec.Body.String())
			}
			if tc.wantSends == 1 && !strings.Contains(rec.Body.String(), "msg-42") {
				t.Fatal("missing message ID")
			}
		})
	}
}
