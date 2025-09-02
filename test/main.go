package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type EmailJob struct {
	Recipient   string `json:"recipient"`
	Subject     string `json:"subject"`
	BodyContent string `json:"bodyContent"`
}

const StreamName = "EMAILS"

func main() {

	emailCount := flag.Int("count", 100, "Total number of emails to send")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	recipient := flag.String("recipient", "test@example.com", "Recipient email address for the test")
	apiURL := flag.String("url", "http://localhost:8080/send-email", "URL of the mailservice API")
	natsURL := flag.String("nats", "nats://localhost:4222", "URL of the NATS server")
	purgeQueue := flag.Bool("purge", false, "If set, purge the NATS queue before running the test")
	flag.Parse()

	if *purgeQueue {
		purgeNatsQueue(*natsURL)
	}

	runLoadTest(*emailCount, *concurrency, *recipient, *apiURL)
}

func purgeNatsQueue(natsURL string) {
	log.Printf("Connecting to NATS at %s to purge queue...", natsURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Error creating JetStream context: %v", err)
	}

	log.Printf("Purging stream '%s'...", StreamName)
	err = js.PurgeStream(StreamName)
	if err != nil {
		log.Fatalf("Failed to purge stream: %v", err)
	}
	log.Printf("Stream '%s' successfully purged.", StreamName)
}

func runLoadTest(emailCount, concurrency int, recipient, apiURL string) {
	log.Printf("Starting load test: %d emails with %d concurrent workers to %s", emailCount, concurrency, apiURL)

	jobs := make(chan EmailJob, emailCount)
	results := make(chan bool, emailCount)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(i+1, apiURL, jobs, results, &wg)
	}

	startTime := time.Now()
	for i := 0; i < emailCount; i++ {
		jobs <- EmailJob{
			Recipient:   recipient,
			Subject:     fmt.Sprintf("API Load Test Email %d/%d", i+1, emailCount),
			BodyContent: fmt.Sprintf("This is an automatically generated test email at %v", time.Now()),
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	duration := time.Since(startTime)
	successCount := 0
	for r := range results {
		if r {
			successCount++
		}
	}

	log.Println("----------- Load Test Complete -----------")
	log.Printf("Total Requests: %d", emailCount)
	log.Printf("Successful:     %d", successCount)
	log.Printf("Failed:         %d", emailCount-successCount)
	log.Printf("Duration:       %.2f minutes", duration.Minutes())
	log.Printf("RPS (Requests Per Second): %.2f", float64(emailCount)/duration.Seconds())
	log.Println("-------------------------------------------")
}

func worker(id int, apiURL string, jobs <-chan EmailJob, results chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 10 * time.Second}
	for job := range jobs {
		err := sendRequestToAPI(client, apiURL, job)
		if err != nil {
			log.Printf("ERROR (Worker %d): %v", id, err)
			results <- false
		} else {
			log.Printf("OK (Worker %d): Request for %s accepted.", id, job.Recipient)
			results <- true
		}
	}
}

func sendRequestToAPI(client *http.Client, apiURL string, job EmailJob) error {
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("API returned non-202 status: %s", resp.Status)
	}
	return nil
}
