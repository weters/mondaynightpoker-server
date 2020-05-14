package util

import "os"

// Getenv will return an environment variable or a default value
func Getenv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}

	return defaultValue
}
