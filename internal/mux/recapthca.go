package mux

import (
	grecaptcha "github.com/ezzarghili/recaptcha-go"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/internal/util"
	"time"
)

type recaptcha interface {
	// Verify will verify the token is valid
	Verify(token string) error
}

func newRecaptcha() recaptcha {
	secretKey := util.Getenv("RECAPTCHA_SECRET", "-")
	captcha, err := grecaptcha.NewReCAPTCHA(secretKey, grecaptcha.V3, 10*time.Second)
	if err != nil {
		logrus.WithError(err).Fatal("could not load recaptcha")
	}

	return &captcha
}
