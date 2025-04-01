package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type OutlineWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Document struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"document"`
	} `json:"data"`
}

func sendToZulip(title string, docURL string, zulipStream string, zulipTopic string, zulipWebhookURL string) {
	message := fmt.Sprintf("Document Updated: [%s](%s)", title, docURL)

	form := url.Values{}
	form.Set("type", "stream")
	form.Set("to", zulipStream)
	form.Set("topic", zulipTopic)
	form.Set("content", message)

	resp, err := http.PostForm(zulipWebhookURL, form)
	if err != nil {
		log.Printf("Failed to send message to Zulip: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Zulip responded with status: %s", resp.Status)
	}
}

func outlineWebhookHandler(zulipStream, zulipTopic, zulipWebhookURL, webhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("🔍 Incoming headers:")
		for k, v := range r.Header {
			log.Printf("%s: %v", k, v)
		}

		// Read the raw request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Reset body for JSON decoding
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		// Validate HMAC signature
		sigHeader := r.Header.Get("X-Outline-Signature")
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(body)
		expectedSig := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expectedSig), []byte(sigHeader)) {
			log.Println("Invalid Outline webhook signature")
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}

		var payload OutlineWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("Invalid webhook payload: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if payload.Event == "documents.create" || payload.Event == "documents.update" {
			title := payload.Data.Document.Title
			docURL := payload.Data.Document.URL

			log.Printf("Received '%s' for document: %s", payload.Event, title)
			sendToZulip(title, docURL, zulipStream, zulipTopic, zulipWebhookURL)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func main() {
	zulipWebhook := os.Getenv("ZULIP_WEBHOOK_URL")
	zulipStream := os.Getenv("ZULIP_STREAM")
	zulipTopic := os.Getenv("ZULIP_TOPIC")
	webhookSecret := os.Getenv("OUTLINE_WEBHOOK_SECRET")

	listenPort := os.Getenv("PORT")
	if listenPort == "" {
		listenPort = "8484"
	}

	if zulipWebhook == "" || zulipStream == "" || zulipTopic == "" || webhookSecret == "" {
		log.Fatal("Missing required environment variables: ZULIP_WEBHOOK_URL, ZULIP_STREAM, ZULIP_TOPIC, or OUTLINE_WEBHOOK_SECRET")
	}

	http.HandleFunc("/outline-webhook", outlineWebhookHandler(zulipStream, zulipTopic, zulipWebhook, webhookSecret))

	log.Printf("Listening on :%s for Outline webhooks...", listenPort)
	if err := http.ListenAndServe(":"+listenPort, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
