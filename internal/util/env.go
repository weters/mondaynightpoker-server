package util

import "os"

// SetEnv will set an environment variable and return a function for unsetting it
func SetEnv(key, value string) func() {
	origVal, found := os.LookupEnv(key)
	_ = os.Setenv(key, value)

	return func() {
		if found {
			_ = os.Setenv(key, origVal)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}
