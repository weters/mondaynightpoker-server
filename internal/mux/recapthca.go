package mux

import (
	"time"

	grecaptcha "github.com/ezzarghili/recaptcha-go"
	"github.com/sirupsen/logrus"
)

type recaptcha interface {
	// Verify will verify the token is valid
	Verify(token string) error
}

func newRecaptcha(secret string) recaptcha {
	captcha, err := grecaptcha.NewReCAPTCHA(secret, grecaptcha.V3, 10*time.Second)
	if err != nil {
		logrus.WithError(err).Fatal("could not load recaptcha")
	}

	return &captcha
}
