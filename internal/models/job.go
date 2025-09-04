package models

// Attachment defines the structure for email attachments.
type Attachment struct {
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentBytes string `json:"contentBytes"`
	ContentType  string `json:"contentType"`
}

// EmailJob represents an email sending job received via NATS.
type EmailJob struct {
	Recipients      []string     `json:"recipients"`
	CcRecipients    []string     `json:"cc_recipients,omitempty"`
	BccRecipients   []string     `json:"bcc_recipients,omitempty"`
	Subject         string       `json:"subject"`
	BodyContent     string       `json:"body_content,omitempty"`
	HtmlBodyContent string       `json:"html_body_content,omitempty"`
	Attachments     []Attachment `json:"attachments,omitempty"`
	AppTag          string       `json:"app_tag"`

	// KORREKTUR: Feld zur Aufnahme des Trace-Kontexts von Datadog hinzugefügt.
	TraceContext map[string]string `json:"trace_context,omitempty"`
}

// Sender represents a sender entry from the database.
type Sender struct {
	ID     int64  `json:"id"`
	AppTag string `json:"app_tag"`
	Email  string `json:"email"`
}
