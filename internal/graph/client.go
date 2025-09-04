package graph

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"email-microservice/internal/config"
)

type Client struct {
	cfg    *config.Config
	client *http.Client
}

// Attachment definiert die Struktur für Anhänge, wie sie vom Worker übergeben werden.
type Attachment struct {
	Name     string
	Content  []byte
	MimeType string
}

// oAuthTokenResponse wird verwendet, um die Antwort des Token-Endpunkts zu parsen.
type oAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// emailMessage ist die Hauptstruktur für die an die Graph-API gesendete JSON-Payload.
type emailMessage struct {
	Message struct {
		Subject       string       `json:"subject"`
		Body          body         `json:"body"`
		ToRecipients  []recipient  `json:"toRecipients"`
		CcRecipients  []recipient  `json:"ccRecipients,omitempty"`
		BccRecipients []recipient  `json:"bccRecipients,omitempty"`
		Attachments   []attachment `json:"attachments,omitempty"`
	} `json:"message"`
	SaveToSentItems bool `json:"saveToSentItems"`
}

type body struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type recipient struct {
	EmailAddress emailAddress `json:"emailAddress"`
}

type emailAddress struct {
	Address string `json:"address"`
}

// attachment ist die interne Struktur für Anhänge im Graph-API-Format.
type attachment struct {
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"`
}

// NewClient erstellt eine neue Instanz des Graph-API-Clients.
func NewClient(cfg *config.Config) *Client {
	log.Printf("[DEBUG] Graph-Client wird initialisiert mit TenantID: %s, ClientID: %s", cfg.TenantID, cfg.ClientID)
	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// SendEmail wurde angepasst. Die UserID wurde entfernt, der Absender wird aus der Konfiguration (cfg.SenderEmail) bezogen.
func (c *Client) SendEmail(
	recipients, ccRecipients, bccRecipients []string,
	subject, bodyContent, contentType string,
	attachments []Attachment) (*http.Response, error) {

	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// GEÄNDERT: Die URL wird jetzt mit der SENDER_EMAIL aus der Konfiguration erstellt.
	graphAPIURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", c.cfg.SenderEmail)
	log.Printf("[DEBUG] Sending email via Graph API endpoint: %s", graphAPIURL)

	var toRecipients, ccRecipientsPayload, bccRecipientsPayload []recipient
	for _, email := range recipients {
		toRecipients = append(toRecipients, recipient{EmailAddress: emailAddress{Address: email}})
	}
	for _, email := range ccRecipients {
		ccRecipientsPayload = append(ccRecipientsPayload, recipient{EmailAddress: emailAddress{Address: email}})
	}
	for _, email := range bccRecipients {
		bccRecipientsPayload = append(bccRecipientsPayload, recipient{EmailAddress: emailAddress{Address: email}})
	}

	var graphAttachments []attachment
	for _, attach := range attachments {
		base64Content := base64.StdEncoding.EncodeToString(attach.Content)
		graphAttachments = append(graphAttachments, attachment{
			ODataType:    "#microsoft.graph.fileAttachment",
			Name:         attach.Name,
			ContentType:  attach.MimeType,
			ContentBytes: base64Content,
		})
	}

	email := emailMessage{
		Message: struct {
			Subject       string       `json:"subject"`
			Body          body         `json:"body"`
			ToRecipients  []recipient  `json:"toRecipients"`
			CcRecipients  []recipient  `json:"ccRecipients,omitempty"`
			BccRecipients []recipient  `json:"bccRecipients,omitempty"`
			Attachments   []attachment `json:"attachments,omitempty"`
		}{
			Subject:       subject,
			Body:          body{ContentType: contentType, Content: bodyContent},
			ToRecipients:  toRecipients,
			CcRecipients:  ccRecipientsPayload,
			BccRecipients: bccRecipientsPayload,
			Attachments:   graphAttachments,
		},
		SaveToSentItems: true,
	}

	emailBytes, err := json.Marshal(email)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email message: %w", err)
	}

	req, err := http.NewRequest("POST", graphAPIURL, bytes.NewBuffer(emailBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	return c.client.Do(req)
}

// getAccessToken ruft ein OAuth2-Zugriffstoken von Microsoft Identity Platform ab.
func (c *Client) getAccessToken() (string, error) {
	log.Printf("[DEBUG] Versuche Token abzurufen mit TenantID: [%s] und ClientID: [%s]", c.cfg.TenantID, c.cfg.ClientID)
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.cfg.TenantID)
	data := new(bytes.Buffer)
	_, err := data.WriteString(fmt.Sprintf("client_id=%s&scope=https://graph.microsoft.com/.default&client_secret=%s&grant_type=client_credentials",
		c.cfg.ClientID, c.cfg.ClientSecret))
	if err != nil {
		return "", fmt.Errorf("failed to create request body: %w", err)
	}

	req, err := http.NewRequest("POST", tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse oAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}
