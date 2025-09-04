package config

import (
	"email-microservice/internal/db" // Wichtig: DB-Paket importieren
	"errors"
	"os"
	"strconv"
)

// Config speichert die gesamte Konfiguration für den Microservice.
type Config struct {
	// Microsoft Graph API-Konfiguration
	TenantID     string
	ClientID     string
	ClientSecret string
	SenderEmail  string

	// Server- und Worker-Konfiguration
	Port        string
	WorkerCount int

	// Datenbank-Konfiguration wird aus dem db-Paket eingebettet.
	DB db.Config
}

// Load liest die gesamte Konfiguration aus den Umgebungsvariablen.
func Load() (*Config, error) {
	// Lade zuerst die separate DB-Konfiguration.
	dbCfg, err := db.Load()
	if err != nil {
		return nil, err
	}

	workerCountStr := os.Getenv("WORKER_COUNT")
	if workerCountStr == "" {
		workerCountStr = "1" // Standardwert auf 1 gesetzt
	}
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, errors.New("ungültiger WORKER_COUNT: " + err.Error())
	}

	cfg := &Config{
		TenantID:     os.Getenv("TENANT_ID"),
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		SenderEmail:  os.Getenv("SENDER_EMAIL"),
		Port:         os.Getenv("PORT"),
		WorkerCount:  workerCount,
		DB:           *dbCfg, // Weise die geladene DB-Konfiguration zu.
	}

	// Überprüfen, ob die Applikations-spezifischen Variablen gesetzt sind.
	if cfg.TenantID == "" {
		return nil, errors.New("umgebungsvariable TENANT_ID ist nicht gesetzt")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("umgebungsvariable CLIENT_ID ist nicht gesetzt")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("umgebungsvariable CLIENT_SECRET ist nicht gesetzt")
	}
	// SenderEmail ist optional, da es pro Job gesetzt wird
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
