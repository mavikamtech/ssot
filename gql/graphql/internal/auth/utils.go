package auth

import "os"

// GetCurrentEnv returns the current environment from ENV variable
func GetCurrentEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "staging" // default environment
	}
	return env
}
