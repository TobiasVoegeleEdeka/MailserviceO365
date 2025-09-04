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
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	// 1. Datadog Tracer initialisieren
	tracer.Start(
		tracer.WithService("email-api"),
		tracer.WithEnv(os.Getenv("DD_ENV")), // z.B. "development" oder "production"
	)
	defer tracer.Stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	_, js := natsclient.Setup(natsURL)

	// 2. HTTP-Router als Datadog-instrumentierten ServeMux erstellen
	// KORREKTUR: NewServeMux verwenden, um den Router automatisch zu instrumentieren.
	mux := httptrace.NewServeMux()
	mux.HandleFunc("/send-email", sendEmailHandler(js))

	log.Printf("API service starting on port %s", port)
	// KORREKTUR: Den instrumentierten mux direkt übergeben.
	log.Fatal(http.ListenAndServe(":"+port, mux))
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

		if len(job.Recipients) == 0 {
			http.Error(w, "Recipients field is required and must not be empty", http.StatusBadRequest)
			return
		}

		// 3. Trace-Kontext für die Weitergabe an den Worker vorbereiten
		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			job.TraceContext = make(map[string]string)
			carrier := tracer.TextMapCarrier(job.TraceContext)
			// Injizieren des Kontexts in die Job-Struktur
			if err := tracer.Inject(span.Context(), carrier); err != nil {
				log.Printf("ERROR: Could not inject trace context: %v", err)
			}
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
