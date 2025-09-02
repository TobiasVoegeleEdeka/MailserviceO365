package main

import (
	"log"
	"os"

	"email-microservice/internal/config"
	"email-microservice/internal/graph"
	natsclient "email-microservice/internal/nats"
	"email-microservice/internal/worker"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found.")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, js := natsclient.Setup(natsURL)
	defer nc.Close()

	graphClient := graph.NewClient(cfg)

	emailWorker, err := worker.New(js, graphClient)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	emailWorker.Run()
}
