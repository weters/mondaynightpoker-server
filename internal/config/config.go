package config

import (
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	"mondaynightpoker-server/internal/util"
	"os"
)

// Config provides configuration for Monday Night Poker
type Config struct {
	loaded         bool
	PGDSN          string `yaml:"pgDsn" envconfig:"pg_dsn"`
	MigrationsPath string `yaml:"migrationsPath" envconfig:"migrations_path"`
	JWT            struct {
		PublicKey  string `yaml:"publicKey" envconfig:"public_key"`
		PrivateKey string `yaml:"privateKey" envconfig:"private_key"`
	}
	RecaptchaSecret string `yaml:"recaptchaSecret" envconfig:"recaptcha_secret"`
	StartGameDelay  int    `yaml:"startGameDelay" envconfig:"start_game_delay"`
	Email           struct {
		From, Sender, Username, Password, Host string
	}
}

var config Config

// Instance returns a singleton instance
// If the config hasn't been loaded, it will be loaded
func Instance() Config {
	if !config.loaded {
		if err := Load(); err != nil {
			panic(err)
		}
	}

	return config
}

// Load will load the configuration
func Load() error {
	configFile := util.Getenv("MNP_CONFIG_FILE", "config.yaml")
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	config = Config{}
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return err
	}

	if err := envconfig.Process("mnp", &config); err != nil {
		return err
	}

	config.loaded = true
	return nil
}
