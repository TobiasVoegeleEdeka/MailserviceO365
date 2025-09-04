E-Mail Microservice für Microsoft O365
Dieser Microservice stellt eine robuste und skalierbare Lösung zum Versenden von E-Mails über die Microsoft Graph API bereit. Das System ist entkoppelt konzipiert und besteht aus einem API-Endpunkt sowie einem asynchronen Worker-Dienst, um eine hohe Ausfallsicherheit und Performance zu gewährleisten.

Inhaltsverzeichnis
Architektur

Technologien & Sprachen

Voraussetzungen

Einrichtung & Konfiguration

Verwendung

Monitoring mit Datadog

Architektur
Der Service folgt einem entkoppelten Architekturmuster, um Anfragen schnell entgegenzunehmen und die Verarbeitung im Hintergrund zuverlässig abzuwickeln.

Ablauf einer Anfrage:

Ein Client (z.B. eine Webanwendung) sendet eine POST-Anfrage mit den E-Mail-Daten an den API-Service.

Der API-Service validiert die Anfrage, nimmt den Auftrag sofort an und publiziert ihn als Job in eine NATS JetStream Queue.

Ein oder mehrere Worker-Services hören auf diese Queue, holen sich die Jobs ab und verarbeiten sie.

Der Worker liest die Absender-Konfiguration aus der PostgreSQL-Datenbank und versendet die E-Mail über die Microsoft Graph API.

Der gesamte Prozess wird mit Datadog überwacht, um verteiltes Tracing von der API bis zum Worker zu ermöglichen.

Technologien & Sprachen
Sprache: Go (Golang)

Backend-Services:

API-Service: Nimmt HTTP-Anfragen entgegen.

Worker-Service: Verarbeitet die E-Mail-Jobs asynchron.

Messaging Queue: NATS JetStream

Dient zur Entkopplung von API und Worker.

Ermöglicht Persistenz und Ausfallsicherheit der E-Mail-Jobs.

Datenbank: PostgreSQL

Speichert die Konfiguration der zulässigen Absender (app_tag zu E-Mail-Zuordnung).

E-Mail-Versand: Microsoft Graph API

Authentifizierung via OAuth 2.0 Client Credentials Flow.

Containerisierung: Docker & Docker Compose

Definiert und orchestriert alle Services für eine einfache Bereitstellung.

Monitoring & Observability: Datadog

Ermöglicht verteiltes APM-Tracing über alle Service-Grenzen hinweg.

Voraussetzungen
Docker und Docker Compose müssen installiert sein.

Go (aktuelle Version) für die lokale Entwicklung und die Ausführung des Admin-Tools.

Ein Microsoft 365 / Azure Account mit:

Einer App-Registrierung in Azure AD.

Konfigurierten API-Berechtigungen (Mail.Send für die Anwendung).

Tenant ID, Client ID und einem Client Secret.

Ein Datadog Account und ein API-Schlüssel.

Einrichtung & Konfiguration
Repository klonen:

git clone <repository-url>
cd MailserviceO365

.env-Datei erstellen:
Im Hauptverzeichnis wird eine .env-Datei erstellt und mit den folgenden Werten gefüllt:

# Docker-Compose-Projektname
COMPOSE_PROJECT_NAME=mailservice

# Microsoft Graph API Anmeldeinformationen
TENANT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
CLIENT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SENDER_EMAIL=absender@ihre-domain.de

# Datenbank-Konfiguration (passend zur docker-compose.yml)
DB_DRIVER=postgres
DB_DSN=host=localhost port=5432 user=mailservice_user password=mysecretpassword dbname=mailservice_db sslmode=disable

# Datadog Konfiguration
DD_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
DD_ENV=development

Verwendung
1. Services starten
Zum Starten werden die Docker-Images gebaut und alle Container gestartet:

docker-compose up --build -d

2. Datenbank initialisieren und Absender verwalten (Admin CLI)
Das Admin-Tool muss lokal kompiliert und ausgeführt werden, um die Datenbank im Docker-Container zu verwalten.

Kompilieren (nur einmalig oder bei Änderungen):

go build -o admin-tool ./cmd/admin/main.go

Datenbank-Tabellen erstellen (erster Schritt):

./admin-tool init

Einen neuen Absender hinzufügen:

./admin-tool add -tag "rechnungssystem" -email "rechnungen@ihre-domain.de"

Alle Absender auflisten:

./admin-tool list

3. E-Mail senden (API-Aufruf)
Eine E-Mail wird über eine POST-Anfrage an den API-Endpunkt gesendet:

curl -X POST \
  http://localhost:8080/send-email \
  -H 'Content-Type: application/json' \
  -d '{
    "recipients": ["empfaenger@example.com"],
    "subject": "Test E-Mail",
    "body_content": "Dies ist eine Testnachricht.",
    "app_tag": "rechnungssystem"
  }'

Monitoring mit Datadog
Nachdem eine Anfrage gesendet wurde, kann man den gesamten Ablauf im Datadog-Account verfolgen:

Login bei Datadog.

Navigation zu APM > Traces.

Suche nach Traces des Services email-api.

Man sieht einen verteilten Trace, der die Anfrage vom email-api Service bis zur Verarbeitung im email-worker Service und den Aufrufen an die Datenbank und die Graph API visualisiert.