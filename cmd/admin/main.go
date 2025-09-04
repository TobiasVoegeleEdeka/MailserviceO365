package main

import (
	"email-microservice/internal/db"
	"email-microservice/internal/models"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Laden der .env Datei für die Datenbankverbindung
	if err := godotenv.Load(); err != nil {
		log.Println("Warnung: .env Datei nicht gefunden. Lese Konfiguration aus Umgebung.")
	}

	// Erstellen des DB-Clients
	dbCfg, err := db.Load()
	if err != nil {
		log.Fatalf("Fehler beim Laden der DB-Konfiguration: %v", err)
	}
	dbClient, err := db.NewClient(dbCfg.Driver, dbCfg.DSN)
	if err != nil {
		log.Fatalf("Fehler beim Verbinden mit der Datenbank: %v", err)
	}

	// Befehls-Parsing
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		handleInit(dbClient)
	case "add":
		handleAdd(dbClient)
	case "list":
		handleList(dbClient)
	case "delete":
		handleDelete(dbClient)
	default:
		fmt.Printf("Unbekannter Befehl: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// handleInit führt die Datenbankmigration aus.
func handleInit(client *db.Client) {
	if err := client.Migrate(); err != nil {
		log.Fatalf("Fehler bei der Datenbankmigration: %v", err)
	}
}

// handleAdd wurde angepasst und benötigt das -userid Flag nicht mehr.
func handleAdd(client *db.Client) {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	appTag := addCmd.String("tag", "", "Einzigartiger App-Tag (z.B. 'invoicing-system')")
	email := addCmd.String("email", "", "E-Mail-Adresse des Absenders")
	addCmd.Parse(os.Args[2:])

	if *appTag == "" || *email == "" {
		log.Println("Alle Flags (-tag, -email) sind erforderlich.")
		addCmd.Usage()
		return
	}

	sender := models.Sender{
		AppTag: *appTag,
		Email:  *email,
	}

	id, err := client.Create("senders", sender)
	if err != nil {
		log.Fatalf("Fehler beim Hinzufügen des Senders: %v", err)
	}
	fmt.Printf("Sender erfolgreich mit ID %d hinzugefügt.\n", id)
}

// handleList wurde angepasst und zeigt die USER ID nicht mehr an.
func handleList(client *db.Client) {
	var senders []models.Sender
	if err := client.Read("senders", &senders, ""); err != nil {
		log.Fatalf("Fehler beim Abrufen der Sender: %v", err)
	}

	if len(senders) == 0 {
		fmt.Println("Keine Sender in der Datenbank gefunden.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tAPP TAG\tEMAIL")
	fmt.Fprintln(w, "--\t-------\t-----")
	for _, s := range senders {
		fmt.Fprintf(w, "%d\t%s\t%s\n", s.ID, s.AppTag, s.Email)
	}
	w.Flush()
}

// handleDelete löscht einen Sender anhand seines App-Tags.
func handleDelete(client *db.Client) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	appTag := deleteCmd.String("tag", "", "Der App-Tag des zu löschenden Senders")
	deleteCmd.Parse(os.Args[2:])

	if *appTag == "" {
		log.Println("Das Flag -tag ist erforderlich.")
		deleteCmd.Usage()
		return
	}

	rowsAffected, err := client.Delete("senders", "app_tag = $1", *appTag)
	if err != nil {
		log.Fatalf("Fehler beim Löschen des Senders: %v", err)
	}

	if rowsAffected == 0 {
		fmt.Printf("Kein Sender mit dem App-Tag '%s' gefunden.\n", *appTag)
		return
	}
	fmt.Printf("Sender mit App-Tag '%s' erfolgreich gelöscht.\n", *appTag)
}

// printUsage wurde angepasst.
func printUsage() {
	fmt.Println("Admin-Tool zur Verwaltung der E-Mail-Sender-Datenbank.")
	fmt.Println("\nVerwendung:")
	fmt.Println("  go run ./cmd/admin/main.go <befehl> [argumente]")
	fmt.Println("\nBefehle:")
	fmt.Println("  init          Initialisiert die Datenbank und erstellt die Tabellen.")
	fmt.Println("  add           Fügt einen neuen Sender hinzu.")
	fmt.Println("  list          Zeigt alle vorhandenen Sender an.")
	fmt.Println("  delete        Löscht einen Sender.")
	fmt.Println("\nFür Hilfe zu einem Befehl, rufen Sie ihn ohne Argumente auf, z.B.:")
	fmt.Println("  go run ./cmd/admin/main.go add")
}
