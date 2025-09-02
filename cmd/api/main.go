package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"email-microservice/internal/models"
	natsclient "email-microservice/internal/nats"

	"github.com/nats-io/nats.go"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	_, js := natsclient.Setup(natsURL)

	http.HandleFunc("/send-email", sendEmailHandler(js))

	log.Printf("API service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func sendEmailHandler(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		var job models.EmailJob
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Überprüfen, ob mindestens ein Empfänger vorhanden ist
		if len(job.Recipients) == 0 {
			http.Error(w, "Recipients field is required and must not be empty", http.StatusBadRequest)
			return
		}

		jobJSON, _ := json.Marshal(job)
		if _, err := js.Publish(natsclient.EmailToSend, jobJSON); err != nil {
			log.Printf("ERROR: Failed to publish job to NATS: %v", err)
			http.Error(w, "Failed to queue email job", http.StatusInternalServerError)
			return
		}

		log.Printf("Accepted job to send email to: %s", strings.Join(job.Recipients, ", "))
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(fmt.Sprintf("Email job accepted for recipients: %s", strings.Join(job.Recipients, ", "))))
	}
}
