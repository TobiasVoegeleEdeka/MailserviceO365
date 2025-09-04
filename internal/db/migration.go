package db

import (
	"database/sql"
	"fmt"
	"log"
)

// Migrate führt die Datenbankmigrationen aus, um sicherzustellen,
// dass alle erforderlichen Tabellen und Indizes vorhanden sind.
func (c *Client) Migrate() error {
	// 1. Tabelle für die Absender-Konfigurationen erstellen.
	// Die Spalte 'user_id' wurde entfernt, da sie nicht mehr benötigt wird.
	const createSendersTableSQL = `
    CREATE TABLE IF NOT EXISTS senders (
        id SERIAL PRIMARY KEY,
        app_tag VARCHAR(50) NOT NULL UNIQUE,
        email VARCHAR(255) NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );`

	if _, err := c.db.Exec(createSendersTableSQL); err != nil {
		return fmt.Errorf("fehler beim Erstellen der 'senders'-Tabelle: %w", err)
	}
	log.Println("Tabelle 'senders' ist bereit.")

	// Ein Index auf 'app_tag' beschleunigt die Suche erheblich.
	const createIndexSQL = `CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_app_tag ON senders(app_tag);`
	if _, err := c.db.Exec(createIndexSQL); err != nil {
		log.Printf("Warnung: Fehler beim Erstellen des Index für 'senders': %v", err)
	}

	// 2. Tabelle zum Protokollieren von E-Mail-Aufträgen erstellen.
	const createMailJobsTableSQL = `
    CREATE TABLE IF NOT EXISTS mail_jobs (
        id SERIAL PRIMARY KEY,
        recipients TEXT[],
        cc_recipients TEXT[],
        bcc_recipients TEXT[],
        subject TEXT,
        body_content TEXT,
        html_body_content TEXT,
        app_tag TEXT,
        status TEXT,
        error_message TEXT,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        processed_at TIMESTAMPTZ
    );`

	if _, err := c.db.Exec(createMailJobsTableSQL); err != nil {
		return fmt.Errorf("fehler beim Erstellen der 'mail_jobs'-Tabelle: %w", err)
	}
	log.Println("Tabelle 'mail_jobs' ist bereit.")

	log.Println("Datenbankmigration erfolgreich überprüft/abgeschlossen.")
	return nil
}

// GetDB gibt die zugrundeliegende SQL-DB-Verbindung zurück.
// Nützlich für Operationen, die nicht vom generischen Client abgedeckt werden.
func (c *Client) GetDB() *sql.DB {
	return c.db
}
