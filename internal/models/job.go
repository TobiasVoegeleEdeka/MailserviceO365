package models

// Attachment definiert die Struktur für E-Mail-Anhänge, wie sie im Job-Payload erwartet wird.
type Attachment struct {
	ODataType    string `json:"@odata.type,omitempty"` // Oft "#microsoft.graph.fileAttachment"
	Name         string `json:"name"`
	ContentBytes string `json:"contentBytes"` // Wichtig: Inhalt wird als Base64-kodierter String erwartet
	ContentType  string `json:"contentType"`
}

// EmailJob repräsentiert einen E-Mail-Sendeauftrag, der über NATS empfangen wird.
type EmailJob struct {
	Recipients      []string     `json:"recipients"`
	CcRecipients    []string     `json:"cc_recipients,omitempty"`
	BccRecipients   []string     `json:"bcc_recipients,omitempty"`
	Subject         string       `json:"subject"`
	BodyContent     string       `json:"body_content,omitempty"`
	HtmlBodyContent string       `json:"html_body_content,omitempty"`
	Attachments     []Attachment `json:"attachments,omitempty"`

	// AppTag weist den Auftrag einem konfigurierten Absender-Postfach zu.
	// Der Tag wird von der aufrufenden Applikation bereitgestellt.
	AppTag string `json:"app_tag"`
}

// Sender repräsentiert einen Absender-Eintrag aus der Datenbank.
// In der neuen Konfiguration wird die UserID nicht mehr benötigt.
type Sender struct {
	ID     int64  `json:"id"`
	AppTag string `json:"app_tag"`

	Email string `json:"email"`
}
