package models

import "email-microservice/internal/graph"

type EmailJob struct {
	Recipients      []string           `json:"recipients"`
	CcRecipients    []string           `json:"cc_recipients,omitempty"`
	BccRecipients   []string           `json:"bcc_recipients,omitempty"`
	Subject         string             `json:"subject"`
	BodyContent     string             `json:"body_content,omitempty"`
	HtmlBodyContent string             `json:"html_body_content,omitempty"`
	AppTag          string             `json:"app_tag"` // Zuweisung welche Mailbox das versenden soll , der Tag wird von der Applikation mitgeliefert, der Microservice setzt dann den entsprechenden Absender
	Attachments     []graph.Attachment `json:"attachments,omitempty"`
}
