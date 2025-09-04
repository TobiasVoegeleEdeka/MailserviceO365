package main

import (
	"log"
	"os"

	"email-microservice/internal/config"
	"email-microservice/internal/db"
	"email-microservice/internal/graph"
	natsclient "email-microservice/internal/nats"
	"email-microservice/internal/worker"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	// 1. Datadog Tracer initialisieren
	tracer.Start(
		tracer.WithService("email-worker"),
		tracer.WithEnv(os.Getenv("DD_ENV")),
	)
	defer tracer.Stop()

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found.")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dbClient, err := db.NewClient(cfg.DB.Driver, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, js := natsclient.Setup(natsURL)
	defer nc.Close()

	graphClient := graph.NewClient(cfg)

	emailWorker, err := worker.New(js, graphClient, dbClient)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	emailWorker.Run()
}
