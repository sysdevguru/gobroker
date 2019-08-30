package mailgun

import (
	"bytes"
	"html/template"
	"strconv"
	"sync"
	"time"

	"github.com/alpacahq/gopaca/env"
	mg "github.com/mailgun/mailgun-go"
)

var (
	once    sync.Once
	enabled = false
	domain  = "mg.alpaca.markets"
	privKey = "key-a3cc258c45a82c0dd62cd1f511e420dd"
	pubKey  = "pubkey-d2bb0ee04b62cb59d8daa2a547c5426f"
)

// Email includes all of the fields required to send
// an email using Mailgun.
type Email struct {
	Sender     string
	Subject    string
	PlainText  string
	HTML       string
	Recipient  string
	DeliverAt  *time.Time
	Bcc        string
	Attachment *Attachment
}

type Attachment struct {
	Data []byte
	Name string
}

// Send an email using Mailgun.
func Send(email Email) error {
	once.Do(func() {
		enabled, _ = strconv.ParseBool(env.GetVar("EMAILS_ENABLED"))
	})

	if !enabled {
		return nil
	}

	mgc := mg.NewMailgun(domain, privKey, pubKey)

	msg := mgc.NewMessage(
		email.Sender,
		email.Subject,
		email.PlainText,
		email.Recipient,
	)

	if email.Attachment != nil {
		msg.AddBufferAttachment(email.Attachment.Name, email.Attachment.Data)
	}

	if email.Bcc != "" {
		msg.AddBCC(email.Bcc)
	}

	if email.HTML != "" {
		msg.SetHtml(email.HTML)
	}

	if email.DeliverAt != nil {
		msg.SetDeliveryTime(*email.DeliverAt)
	}

	_, _, err := mgc.Send(msg)

	return err
}

// ParseTemplate takes a path to an HTML template, parses
// it, then executes it with the template data provided.
func ParseTemplate(templateFile string, data interface{}) (*string, error) {
	t, err := template.ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	if err := t.Execute(buf, data); err != nil {
		return nil, err
	}

	body := buf.String()

	return &body, nil
}
