package config

import (
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

var defaultConfig = Config{
	Host: "https://mondaynight.bid",
	AllowedOrigins: []string{
		"https://mondaynight.bid",
		"https://beta.mondaynight.bid",
	},
	Log: Log{
		DisableAccessLogs: false,
		Level:             "info",
	},
	Database: Database{
		DSN:            "postgres://postgres@localhost:5432/postgres?sslmode=disable",
		MigrationsPath: "./sql",
	},
	JWT: JWT{
		PublicKey:  ".keys/public.pem",
		PrivateKey: ".keys/private.key",
	},
	RecaptchaSecret:   "-",
	StartGameDelay:    10,
	PlayerCreateDelay: 60,
	Email: Email{
		From:         "Monday Night Poker <dealer@mondaynight.bid>",
		Sender:       "dealer@mondaynight.bid",
		Username:     "dealer@mondaynight.bid",
		Password:     "",
		Host:         "mail.privateemail.com:587",
		TemplatesDir: "templates",
	},
}

// Config provides configuration for Monday Night Poker
type Config struct {
	// Host is the player-facing website's base URL (e.g. https://mondaynight.bid).
	// It is used to build links in emails (account verification, password reset)
	// and, when AllowedOrigins is unset, as the BrowserOrigins() fallback. It is
	// NOT necessarily where this API server itself is reachable; see PublicURL.
	Host string
	// AllowedOrigins is the single list of browser origins permitted to make
	// cross-origin requests. It governs BOTH the HTTP CORS layer and the
	// WebSocket upgrade origin check, so the two policies never drift apart.
	// If empty, only Host is allowed.
	AllowedOrigins []string `yaml:"allowedOrigins" envconfig:"allowed_origins"`
	// PublicURL is the public base URL of THIS API server (e.g.
	// https://api.mondaynight.bid), as opposed to Host, which is the
	// player-facing website. It is used as the OAuth issuer and MCP resource
	// identifier via APIBaseURL(). When unset, APIBaseURL() falls back to Host,
	// which is correct for single-host/dev deployments where the API and the
	// website share an origin.
	PublicURL         string `yaml:"publicURL" envconfig:"public_url"`
	Log               Log
	Database          Database
	JWT               JWT
	RecaptchaSecret   string `yaml:"recaptchaSecret" envconfig:"recaptcha_secret"`
	StartGameDelay    int    `yaml:"startGameDelay" envconfig:"start_game_delay"`
	PlayerCreateDelay int    `yaml:"playerCreateDelay" envconfig:"player_create_delay"`
	Email             Email
}

// BrowserOrigins returns the browser origins permitted to make cross-origin
// requests. The same list is shared by the HTTP CORS layer and the WebSocket
// upgrade origin check so they cannot diverge. When AllowedOrigins is unset it
// falls back to Host.
func (c Config) BrowserOrigins() []string {
	if len(c.AllowedOrigins) > 0 {
		return c.AllowedOrigins
	}

	return []string{c.Host}
}

// APIBaseURL returns the public base URL of this API server, used as the
// OAuth issuer and MCP resource identifier. It falls back to Host when
// PublicURL is unset, which is correct for single-host/dev deployments.
func (c Config) APIBaseURL() string {
	if c.PublicURL != "" {
		return strings.TrimRight(c.PublicURL, "/")
	}

	return strings.TrimRight(c.Host, "/")
}

// Log represents logging configuration
type Log struct {
	DisableAccessLogs bool `yaml:"disableAccessLogs" envconfig:"disable_access_logs"`
	Level             string
}

// Database represents database configuration
type Database struct {
	DSN            string
	MigrationsPath string `yaml:"migrationsPath" envconfig:"migrations_path"`
}

// JWT represents JWT configuration
type JWT struct {
	PublicKey  string `yaml:"publicKey" envconfig:"public_key"`
	PrivateKey string `yaml:"privateKey" envconfig:"private_key"`
}

// Email represents configuration for sending emails
type Email struct {
	From, Sender, Username, Password, Host string
	TemplatesDir                           string `yaml:"templatesDir" envconfig:"templates_dir"`
	// if true, do not send emails
	Disable bool
}

// Load loads and returns the configuration
func Load() (Config, error) {
	config := defaultConfig

	if cfgFile, ok := getConfigFile(); ok {
		defer cfgFile.Close()

		if err := yaml.NewDecoder(cfgFile).Decode(&config); err != nil {
			return Config{}, err
		}
	}

	if err := envconfig.Process("mnp", &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return defaultConfig
}

func getConfigFile() (*os.File, bool) {
	paths := []string{os.Getenv("MNP_CONFIG_FILE"), "config.yaml", "testdata/config.yaml"}
	for _, path := range paths {
		if path == "" {
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			continue
		}

		return file, true
	}

	return nil, false
}
