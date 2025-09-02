package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

const (
	StreamName  = "EMAILS"
	StreamSubj  = "EMAILS.*"
	EmailToSend = "EMAILS.send"
)

func Setup(natsURL string) (*nats.Conn, nats.JetStreamContext) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Error creating JetStream context: %v", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     StreamName,
		Subjects: []string{StreamSubj},
	})
	if err != nil {
		log.Printf("Warning: Could not create stream (it likely already exists): %v", err)
	}

	return nc, js
}
