package main

import (
	"flag"
	"github.com/gorilla/handlers"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/mux"
	"mondaynightpoker-server/pkg/db"
	"strings"
	"time"
)

const readTimeout = time.Second * 5
const writeTimeout = time.Second * 10

// Version is the server version
var Version = "v0.0.0-dev"

var addr = flag.String("addr", ":5000", "the listen address")

func main() {
	flag.Parse()
	setupLogger()

	// fail fast
	jwt.LoadKeys()

	// run the db migrations
	db.Migrate()

	c := cors.New(cors.Options{
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization"},
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handlers.CombinedLoggingHandler(os.Stdout, c.Handler(mux.NewMux(Version))),
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
	}

	logrus.WithField("addr", srv.Addr).Info("listening")
	logrus.Fatal(srv.ListenAndServe())
}

func setupLogger() {
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		level, err := logrus.ParseLevel(lvl)
		if err != nil {
			logrus.WithError(err).Fatal("could not parse level")
		}

		logrus.SetLevel(level)
	}

	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}
