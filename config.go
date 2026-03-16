package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Load .env file if it exists
	loadEnvFile()

	// Override package-level vars from environment
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		googleClientID = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		googleClientSecret = v
	}

	// Update OAuth config since it was initialized before env vars were loaded
	googleOAuthConfig.ClientID = googleClientID
	googleOAuthConfig.ClientSecret = googleClientSecret
	if v := os.Getenv("MONGODB_URI"); v != "" {
		mongoURI = v
	}
	if v := os.Getenv("OPENCLAW_TOKEN"); v != "" {
		openclawToken = v
	}
}

func loadEnvFile() {
	// Try .env in executable directory, then working directory
	paths := []string{}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
	}
	paths = append(paths, ".env")

	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		break // only load first found
	}
}
