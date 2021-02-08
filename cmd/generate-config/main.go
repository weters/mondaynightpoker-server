package main

import (
	"gopkg.in/yaml.v2"
	"mondaynightpoker-server/internal/config"
	"os"
)

func main() {
	if err := yaml.NewEncoder(os.Stdout).Encode(config.DefaultConfig()); err != nil {
		panic(err)
	}
}
