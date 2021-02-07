package mux

import (
	grecaptcha "github.com/ezzarghili/recaptcha-go"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/internal/config"
	"time"
)

type recaptcha interface {
	// Verify will verify the token is valid
	Verify(token string) error
}

func newRecaptcha() recaptcha {
	captcha, err := grecaptcha.NewReCAPTCHA(config.Instance().RecaptchaSecret, grecaptcha.V3, 10*time.Second)
	if err != nil {
		logrus.WithError(err).Fatal("could not load recaptcha")
	}

	return &captcha
}
