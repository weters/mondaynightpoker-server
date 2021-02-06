package email

import (
	"github.com/stretchr/testify/assert"
	"net/smtp"
	"testing"
)

func TestClient_Send(t *testing.T) {
	a := assert.New(t)
	client, err := NewClient("Test <test@test.com>", "test@test.com", "username@test.com", "pw123", "localhost:123")
	a.NoError(err)
	a.NotNil(client)

	called := 0
	defaultSender = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		called++
		a.Equal(1, called)
		a.Equal("localhost:123", addr)
		a.Equal(smtp.PlainAuth("", "username@test.com", "pw123", "localhost"), auth)
		a.Equal("test@test.com", from)
		a.Equal([]string{"to1@test.com", "to2@test.com", "cc1@test.com", "cc2@test.com", "bcc1@test.com", "bcc2@test.com"}, to)
		a.Equal(`To: to1@test.com,to2@test.com
Cc: cc1@test.com,cc2@test.com
From: Test <test@test.com>
Subject: my subject
Content-Type: text/html

<p>Test Message</p>`, string(msg))
		return nil
	}

	a.NoError(
		client.Send([]string{"to1@test.com", "to2@test.com"},
			[]string{"cc1@test.com", "cc2@test.com"},
			[]string{"bcc1@test.com", "bcc2@test.com"}, "my subject", "<p>Test Message</p>"),
	)
	a.Equal(1, called)
}

func TestClient_SendSimple(t *testing.T) {
	a := assert.New(t)
	client, err := NewClient("Test <test@test.com>", "test@test.com", "username@test.com", "pw123", "localhost:123")
	a.NoError(err)
	a.NotNil(client)

	called := 0
	defaultSender = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		called++
		a.Equal(1, called)
		a.Equal("localhost:123", addr)
		a.Equal(smtp.PlainAuth("", "username@test.com", "pw123", "localhost"), auth)
		a.Equal("test@test.com", from)
		a.Equal([]string{"to@test.com"}, to)
		a.Equal(`To: to@test.com
From: Test <test@test.com>
Subject: My Subject
Content-Type: text/html

<p>Test</p>`, string(msg))
		return nil
	}

	a.NoError(client.SendSimple("to@test.com", "My Subject", "<p>Test</p>"))
	a.Equal(1, called)
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("Test <test@test.com>", "test@test.com", "user@test.com", "pw123", "localhost")
	assert.Nil(t, client)
	assert.EqualError(t, err, "host must have a port")
}
