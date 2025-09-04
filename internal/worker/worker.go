package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"email-microservice/internal/db"
	"email-microservice/internal/graph"
	"email-microservice/internal/models"
	natsclient "email-microservice/internal/nats"

	"github.com/nats-io/nats.go"
)

const (
	ConsumerName     = "EMAIL_WORKER"
	maxRetries       = 3
	SendersTableName = "senders"
)

type Worker struct {
	js             nats.JetStreamContext
	sub            *nats.Subscription
	graphClient    *graph.Client
	dbClient       *db.Client
	processedCount uint64
	throttledCount uint64
	failedCount    uint64
}

func New(js nats.JetStreamContext, graphClient *graph.Client, dbClient *db.Client) (*Worker, error) {
	sub, err := js.PullSubscribe(natsclient.EmailToSend, ConsumerName)
	if err != nil {
		return nil, err
	}
	return &Worker{
		js:             js,
		sub:            sub,
		graphClient:    graphClient,
		dbClient:       dbClient,
		processedCount: 0,
		throttledCount: 0,
		failedCount:    0,
	}, nil
}

func (w *Worker) Run() {
	log.Println("Worker started. Waiting for email jobs...")
	go w.logSummary()

	for {
		msgs, err := w.sub.Fetch(1, nats.MaxWait(10*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			log.Printf("Error fetching message: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, msg := range msgs {
			w.processMessage(msg)
		}
	}
}

func (w *Worker) logSummary() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		streamInfo, sErr := w.js.StreamInfo(natsclient.StreamName)
		consumerInfo, cErr := w.js.ConsumerInfo(natsclient.StreamName, ConsumerName)

		log.Println("----------- WORKER SUMMARY -----------")
		if sErr == nil && cErr == nil {
			log.Printf("Queue Status -> Total in Stream: %d, Pending for Workers: %d", streamInfo.State.Msgs, consumerInfo.NumPending)
		} else {
			log.Println("Queue Status -> Could not retrieve NATS stream/consumer info")
		}
		log.Printf("This Worker -> Processed: %d, Throttled: %d, Failed: %d", w.processedCount, w.throttledCount, w.failedCount)
		log.Println("------------------------------------")
	}
}

func (w *Worker) getSenderByAppTag(appTag string) (*models.Sender, error) {
	if appTag == "" {
		return nil, fmt.Errorf("appTag is empty")
	}

	var senders []models.Sender
	whereClause := "app_tag = $1"
	err := w.dbClient.Read(SendersTableName, &senders, whereClause, appTag)
	if err != nil {
		return nil, fmt.Errorf("database query failed for appTag '%s': %w", appTag, err)
	}

	if len(senders) == 0 {
		return nil, fmt.Errorf("no sender found for appTag '%s'", appTag)
	}

	return &senders[0], nil
}

func (w *Worker) processMessage(msg *nats.Msg) {
	var job models.EmailJob
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("ERROR: Could not unmarshal message, discarding: %v", err)
		msg.Ack()
		return
	}

	sender, err := w.getSenderByAppTag(job.AppTag)
	if err != nil {
		log.Printf("ERROR: Permanent failure for job with appTag '%s', discarding: %v", job.AppTag, err)
		w.failedCount++
		msg.Ack()
		return
	}

	allRecipients := append(job.Recipients, job.CcRecipients...)
	allRecipients = append(allRecipients, job.BccRecipients...)

	log.Printf("Processing email with appTag '%s' from sender '%s' to %v", job.AppTag, sender.Email, allRecipients)

	var bodyContent string
	var contentType string
	if job.HtmlBodyContent != "" {
		bodyContent = job.HtmlBodyContent
		contentType = "HTML"
	} else {
		bodyContent = job.BodyContent
		contentType = "Text"
	}

	// KORREKTUR: Anhänge von models.Attachment (mit base64-String) zu graph.Attachment (mit byte slice) konvertieren.
	graphAttachments := make([]graph.Attachment, len(job.Attachments))
	for i, att := range job.Attachments {
		decodedContent, err := base64.StdEncoding.DecodeString(att.ContentBytes)
		if err != nil {
			log.Printf("ERROR: Permanent failure for job with appTag '%s', could not decode attachment '%s': %v", job.AppTag, att.Name, err)
			w.failedCount++
			msg.Ack()
			return
		}

		graphAttachments[i] = graph.Attachment{
			Name:     att.Name,
			Content:  decodedContent,
			MimeType: att.ContentType,
		}
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// GEÄNDERT: sender.UserID wird nicht mehr übergeben. Das konvertierte Slice 'graphAttachments' wird hier verwendet.
		resp, err := w.graphClient.SendEmail(job.Recipients, job.CcRecipients, job.BccRecipients, job.Subject, bodyContent, contentType, graphAttachments)
		if err != nil {
			log.Printf("ERROR (Attempt %d) sending email from '%s' to %v: %v", attempt+1, sender.Email, allRecipients, err)
			time.Sleep(time.Duration(2+attempt) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusAccepted {
			w.processedCount++
			log.Printf("Successfully sent email from '%s' to %v", sender.Email, allRecipients)
			msg.Ack()
			resp.Body.Close()
			return
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			w.throttledCount++
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfter, _ := strconv.Atoi(retryAfterStr)
			if retryAfter <= 0 {
				retryAfter = 5 // Fallback
			}
			log.Printf("WARN: Throttled (429) on attempt %d for sender '%s'. Waiting %d seconds...", attempt+1, sender.Email, retryAfter)
			resp.Body.Close()
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("ERROR: Unexpected status %d on attempt %d from sender '%s': %s", resp.StatusCode, attempt+1, sender.Email, string(bodyBytes))
		resp.Body.Close()
		break
	}

	w.failedCount++
	log.Printf("ERROR: All retries failed for email from '%s' to %v. Releasing job for later retry.", sender.Email, allRecipients)
	msg.Nak()
}
