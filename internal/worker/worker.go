package worker

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"email-microservice/internal/graph"
	"email-microservice/internal/models"
	natsclient "email-microservice/internal/nats"

	"github.com/nats-io/nats.go"
)

const (
	ConsumerName = "EMAIL_WORKER"
	maxRetries   = 3
)

type Worker struct {
	js             nats.JetStreamContext
	sub            *nats.Subscription
	graphClient    *graph.Client
	processedCount uint64
	throttledCount uint64
	failedCount    uint64
}

func New(js nats.JetStreamContext, client *graph.Client) (*Worker, error) {
	sub, err := js.PullSubscribe(natsclient.EmailToSend, ConsumerName)
	if err != nil {
		return nil, err
	}
	return &Worker{
		js:             js,
		sub:            sub,
		graphClient:    client,
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

func (w *Worker) processMessage(msg *nats.Msg) {
	var job models.EmailJob
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("ERROR: Could not unmarshal message, discarding: %v", err)
		msg.Ack()
		return
	}

	allRecipients := append(job.Recipients, job.CcRecipients...)
	allRecipients = append(allRecipients, job.BccRecipients...)

	log.Printf("Processing email to %v", allRecipients)

	var bodyContent string
	var contentType string
	if job.HtmlBodyContent != "" {
		bodyContent = job.HtmlBodyContent
		contentType = "HTML"
	} else {
		bodyContent = job.BodyContent
		contentType = "Text"
	}

	attachments := job.Attachments

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := w.graphClient.SendEmail(job.Recipients, job.CcRecipients, job.BccRecipients, job.Subject, bodyContent, contentType, attachments)
		if err != nil {
			log.Printf("ERROR (Attempt %d) sending email to %v: %v", attempt+1, allRecipients, err)
			time.Sleep(time.Duration(2+attempt) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusAccepted {
			w.processedCount++
			log.Printf("Successfully sent email to %v", allRecipients)
			msg.Ack()
			resp.Body.Close()
			return
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			w.throttledCount++
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfter, _ := strconv.Atoi(retryAfterStr)
			if retryAfter <= 0 {
				retryAfter = 5
			}
			log.Printf("WARN: Throttled (429) on attempt %d. Waiting %d seconds...", attempt+1, retryAfter)
			resp.Body.Close()
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("ERROR: Unexpected status %d on attempt %d: %s", resp.StatusCode, attempt+1, string(bodyBytes))
		resp.Body.Close()
		break
	}

	w.failedCount++
	log.Printf("ERROR: All retries failed for email to %v. Releasing job.", allRecipients)
	msg.Nak()
}
