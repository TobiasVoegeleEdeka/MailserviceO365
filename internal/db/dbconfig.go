package db

import (
	"errors"
	"os"
)

// Config speichert die reine Datenbank-Konfiguration.
// Die Werte werden aus Umgebungsvariablen geladen.
type Config struct {
	Driver string
	DSN    string // Data Source Name (Connection-String)
}

// Load lädt die DB-Konfiguration aus den Umgebungsvariablen.
func Load() (*Config, error) {
	cfg := &Config{
		Driver: os.Getenv("DB_DRIVER"),
		DSN:    os.Getenv("DB_DSN"),
	}

	if cfg.Driver == "" {
		return nil, errors.New("umgebungsvariable DB_DRIVER ist nicht gesetzt")
	}

	if cfg.DSN == "" {
		return nil, errors.New("umgebungsvariable DB_DSN ist nicht gesetzt")
	}

	return cfg, nil
}
