package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	SenderEmail  string
	Port         string
	WorkerCount  int
}

func Load() (*Config, error) {
	workerCountStr := os.Getenv("WORKER_COUNT")
	if workerCountStr == "" {
		workerCountStr = "5"
	}
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, fmt.Errorf("ungültiger WORKER_COUNT: %w", err)
	}

	cfg := &Config{
		TenantID:     os.Getenv("TENANT_ID"),
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		SenderEmail:  os.Getenv("SENDER_EMAIL"),
		Port:         os.Getenv("PORT"),
		WorkerCount:  workerCount,
	}

	if cfg.TenantID == "" {
		return nil, fmt.Errorf("TENANT_ID environment variable is not set")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("CLIENT_ID environment variable is not set")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("CLIENT_SECRET environment variable is not set")
	}
	if cfg.SenderEmail == "" {
		return nil, fmt.Errorf("SENDER_EMAIL environment variable is not set")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
