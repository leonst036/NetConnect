package main

import (
	"os"
	"strings"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	data, err := os.ReadFile(".env")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, key+"=") {
				return strings.Trim(strings.TrimPrefix(line, key+"="), `"' `)
			}
		}
	}
	return fallback
}
