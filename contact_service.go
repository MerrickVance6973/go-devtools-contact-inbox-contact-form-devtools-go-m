package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type contactForm struct {
	SubmissionID string `json:"submission_id"`
	Kind         string `json:"kind"`
	Email        string `json:"email"`
	Title        string `json:"title"`
	Details      string `json:"details"`
	CaptchaToken string `json:"captcha_token"`
}

type contactService struct {
	api   infraiClient
	inbox string
}

func (s contactService) submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var form contactForm
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&form); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if len(form.SubmissionID) < 8 || len(form.SubmissionID) > 128 || strings.ContainsAny(form.SubmissionID, "\r\n") || !strings.Contains(form.Email, "@") || len(form.Email) > 254 || strings.ContainsAny(form.Email, "\r\n") || strings.TrimSpace(form.Title) == "" || len(form.Title) > 160 || strings.TrimSpace(form.Details) == "" || form.CaptchaToken == "" {
		http.Error(w, "invalid form fields", http.StatusBadRequest)
		return
	}
	labels := map[string]string{"build": "Build event", "release": "Release operation", "diagnostic": "Developer diagnostic"}
	label, valid := labels[form.Kind]
	if !valid {
		http.Error(w, "kind must be build, release, or diagnostic", http.StatusBadRequest)
		return
	}
	if err := s.api.verifyCaptcha(r.Context(), form.CaptchaToken); err != nil {
		respondAPIError(w, err)
		return
	}
	subject := "[Devtools / " + label + "] " + cleanLine(form.Title)
	message := fmt.Sprintf("Category: %s\nContact: %s\nSubmission: %s\n\n%s", label, cleanLine(form.Email), form.SubmissionID, form.Details)
	id, err := s.api.sendEmail(r.Context(), s.inbox, subject, message, form.SubmissionID)
	if err != nil {
		respondAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message_id": id})
}

func (s contactService) events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+s.api.key {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("message_id")
	if id == "" {
		http.Error(w, "message_id required", http.StatusBadRequest)
		return
	}
	data, err := s.api.emailEvents(r.Context(), id)
	if err != nil {
		respondAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]json.RawMessage{"events": data})
}

func respondAPIError(w http.ResponseWriter, err error) {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		status := http.StatusBadGateway
		if strings.HasPrefix(apiErr.Code, "CAPTCHA_") || apiErr.Code == "INVALID_ARGUMENT" {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, apiErr.Error(), status)
		return
	}
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}
