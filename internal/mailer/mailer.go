package mailer

import (
	"bytes"
	"embed"
	"text/template"
	"time"

	"github.com/go-mail/mail/v2"
)

// Below we declare a new variable with the type embed.FS (embedded file system)
// to hold our email templates. This has a comment directive in the format
// `//go:embed <path>` IMMEDIATELY ABOVE it, which indicates to Go that we want
// to store the contents of the ./templates directory in the templateFS embedded
// file system variable.
// ↓↓↓

//go:embed "templates"
var templateFS embed.FS

// Mailer holds a mail.Dialer instance (used to connect to a SMTP server) and
// the sender information for your emails (the name and address you want the
// email to be from, such as "Peng-Yu Chen <me@pengyuc.com>")>
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// New returns a Mailer instance containing the dialer and sender information.
func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second
	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send takes the recipient email address, the name of the file containing the
// templates, and any dynamic data for the templates as an any parameter.
func (m Mailer) Send(recipient, templateFile string, data any) error {
	// ParseFS() parses the required template file from the embedded file system.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the named template "subject", passing in the dynamic data and
	// storing the result in a bytes.Buffer variable.
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// Likewise, execute the "plainBody" template.
	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	// Likewise, execute the "htmlBody" template.
	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Note that AddAlternative() should always be called AFTER SetBody().
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Call DialAndSend() on the dialer, passing in the message to send. This
	// opens a connection to the SMTP server, sends the message, then closes the
	// connection. If there's a timeout, it'll return a "dial tcp: i/o timeout"
	// error.
	return m.dialer.DialAndSend(msg)
}
