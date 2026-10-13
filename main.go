package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	key, inbox := os.Getenv("INFRAI_API_KEY"), os.Getenv("TEAM_INBOX")
	widgetRecordID := os.Getenv("INFRAI_WIDGET_RECORD_ID")
	if key == "" || inbox == "" || widgetRecordID == "" {
		log.Fatal("INFRAI_API_KEY, INFRAI_WIDGET_RECORD_ID, and TEAM_INBOX are required")
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	service := contactService{api: infraiClient{baseURL: baseURL, key: key, widgetRecordID: widgetRecordID, http: &http.Client{Timeout: 12 * time.Second}}, inbox: inbox}
	mux := http.NewServeMux()
	mux.HandleFunc("/contact", service.submit)
	mux.HandleFunc("/events", service.events)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
