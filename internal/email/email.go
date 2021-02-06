package email

import (
	"bytes"
	"errors"
	"net/smtp"
	"strings"
)

type smtpSender func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

var defaultSender smtpSender = smtp.SendMail

// Client is a client capable of sending email
type Client struct {
	auth               smtp.Auth
	from, sender, host string
}

// NewClient returns a new email client
func NewClient(from, sender, username, password, host string) (*Client, error) {
	hostParts := strings.Split(host, ":")
	if len(hostParts) != 2 {
		return nil, errors.New("host must have a port")
	}

	return &Client{
		auth:   smtp.PlainAuth("", username, password, hostParts[0]),
		from:   from,
		sender: sender,
		host:   host,
	}, nil
}

// SendSimple sends a text/html email
func (c *Client) SendSimple(to, subject, msg string) error {
	return c.Send([]string{to}, nil, nil, subject, msg)
}

// Send sends a text/html email
func (c *Client) Send(to, cc, bcc []string, subject, msg string) error {
	recipients := make([]string, 0, len(to)+len(cc)+len(bcc))
	recipients = append(recipients, to...)
	recipients = append(recipients, cc...)
	recipients = append(recipients, bcc...)

	toString := strings.Join(to, ",")
	ccString := strings.Join(cc, ",")

	buffer := bytes.Buffer{}
	if toString != "" {
		buffer.WriteString("To: " + toString + "\n")
	}

	if ccString != "" {
		buffer.WriteString("Cc: " + ccString + "\n")
	}

	buffer.WriteString("From: " + c.from + "\n")
	buffer.WriteString("Subject: " + subject + "\n")
	buffer.WriteString("Content-Type: text/html\n\n")
	buffer.WriteString(msg)

	return defaultSender(c.host, c.auth, c.sender, recipients, buffer.Bytes())
}
