package main

import (
	"bufio"
	"fmt"
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
	if v := os.Getenv("OPENCLAW_URL"); v != "" {
		openclawURL = v
	}
	if v := os.Getenv("EREBUS_WS_URL"); v != "" {
		erebusWSURL = v
	}
	if v := os.Getenv("EREBUS_JWT_SECRET"); v != "" {
		erebusJWTSecret = v
	}

	// Debug: log loaded config
	fmt.Printf("[Config] OPENCLAW_URL=%s\n", openclawURL)
	fmt.Printf("[Config] OPENCLAW_TOKEN=%s...\n", truncate(openclawToken, 8))
	fmt.Printf("[Config] MONGODB_URI=%s\n", truncate(mongoURI, 30))
	fmt.Printf("[Config] GOOGLE_CLIENT_ID=%s\n", truncate(googleClientID, 20))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func loadEnvFile() {
	// Search for .env in multiple locations
	paths := []string{}

	// 1. Executable directory
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths = append(paths, filepath.Join(exeDir, ".env"))
		// 2. Parent of executable dir (for build/bin/ structure)
		paths = append(paths, filepath.Join(filepath.Dir(exeDir), ".env"))
		// 3. Grandparent (for build/bin/app.exe)
		paths = append(paths, filepath.Join(filepath.Dir(filepath.Dir(exeDir)), ".env"))
	}

	// 4. Current working directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".env"))
	}

	// 5. Home directory openclaw workspace
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".openclaw", "workspace", "nyx-command-center", ".env"))
	}

	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		fmt.Printf("[Config] Loaded .env from: %s\n", p)
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
		f.Close()
		return // only load first found
	}
	fmt.Printf("[Config] WARNING: No .env file found. Searched: %v\n", paths)
}
