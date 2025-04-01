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
	"strings"
)

type OutlineWebhookPayload struct {
	Event   string `json:"event"`
	Payload struct {
		ID    string `json:"id"`
		Model struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Text      string `json:"text"`
			UpdatedBy struct {
				Name string `json:"name"`
			} `json:"updatedBy"`
		} `json:"model"`
	} `json:"payload"`
}

func formatZulipMessage(payload OutlineWebhookPayload, baseURL string) string {
	title := payload.Payload.Model.Title
	docURL := fmt.Sprintf("%s%s", baseURL, payload.Payload.Model.URL)
	updatedBy := payload.Payload.Model.UpdatedBy.Name
	textSnippet := strings.Split(payload.Payload.Model.Text, "\n")[0] // Get first paragraph

	if textSnippet != "" {
		return fmt.Sprintf("%s: [%s](%s) was updated by %s\n\n%s\n\n_(Click the title to view the full document.)_", payload.Event, title, docURL, updatedBy, textSnippet)
	}

	return fmt.Sprintf("%s: [%s](%s) was updated by %s", payload.Event, title, docURL, updatedBy)
}

func sendToZulip(message string, zulipStream string, zulipTopic string, zulipWebhookURL string) {
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

func outlineWebhookHandler(zulipStream, zulipTopic, zulipWebhookURL, webhookSecret, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Println("🔍 Incoming headers:")
		//for k, v := range r.Header {
		//	log.Printf("%s: %v", k, v)
		//}

		// Read the raw request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Reset body for JSON decoding
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		log.Printf("Body: '%s'", string(body))

		// Get the signature header from the request
		sigHeader := r.Header.Get("Outline-Signature")

		var actualSig, timestamp string
		for _, part := range bytes.Split([]byte(sigHeader), []byte{','}) {
			kv := bytes.SplitN(part, []byte{'='}, 2)
			if len(kv) != 2 {
				continue
			}
			key := string(bytes.TrimSpace(kv[0]))
			value := string(bytes.TrimSpace(kv[1]))

			switch key {
			case "s":
				actualSig = value
			case "t":
				timestamp = value
			}
		}

		if actualSig == "" || timestamp == "" {
			log.Println("Signature or timestamp missing from Outline-Signature header")
			http.Error(w, "invalid signature header", http.StatusForbidden)
			return
		}

		// Construct payload: timestamp.body
		signedPayload := fmt.Sprintf("%s.%s", timestamp, string(body))

		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write([]byte(signedPayload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
			log.Printf("Signature mismatch\nExpected: %s\nActual  : %s", expectedSig, actualSig)
			http.Error(w, "invalid signature", http.StatusForbidden)
			//w.WriteHeader(http.StatusOK)
			//_, _ = w.Write([]byte("ok"))
			return
		}

		var payload OutlineWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("Invalid webhook payload: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		//if payload.Event == "documents.create" || payload.Event == "documents.update" {
		message := formatZulipMessage(payload, baseURL)
		log.Printf("Received '%s' for document: %s", payload.Event, payload.Payload.Model.Title)
		sendToZulip(message, zulipStream, zulipTopic, zulipWebhookURL)
		//} else {
		//	log.Printf("Ignoring event: %s", payload.Event)
		//}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func main() {
	zulipWebhook := os.Getenv("ZULIP_WEBHOOK_URL")
	zulipStream := os.Getenv("ZULIP_STREAM")
	zulipTopic := os.Getenv("ZULIP_TOPIC")
	webhookSecret := os.Getenv("OUTLINE_WEBHOOK_SECRET")
	baseURL := os.Getenv("OUTLINE_BASE_URL")

	listenPort := os.Getenv("PORT")
	if listenPort == "" {
		listenPort = "8484"
	}

	if zulipWebhook == "" || zulipStream == "" || zulipTopic == "" || webhookSecret == "" {
		log.Fatal("Missing required environment variables: ZULIP_WEBHOOK_URL, ZULIP_STREAM, ZULIP_TOPIC, or OUTLINE_WEBHOOK_SECRET")
	}

	http.HandleFunc("/outline-webhook", outlineWebhookHandler(zulipStream, zulipTopic, zulipWebhook, webhookSecret, baseURL))

	log.Printf("Listening on :%s for Outline webhooks...", listenPort)
	if err := http.ListenAndServe(":"+listenPort, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
