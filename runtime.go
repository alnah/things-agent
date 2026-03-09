package main

import (
	"os"
	"strings"

	thingslib "github.com/alnah/things-agent/internal/things"
)

func resolveDataDir() (string, error) {
	return thingslib.ResolveDataDir(thingsDataPattern)
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func escapeApple(value string) string {
	return thingslib.EscapeApple(value)
}
