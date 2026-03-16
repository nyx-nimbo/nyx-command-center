package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	googleClientID     = "" // Set via GOOGLE_CLIENT_ID env var
	googleClientSecret = "" // Set via GOOGLE_CLIENT_SECRET env var
	googleRedirectURL  = "http://localhost:8099"
)

var googleOAuthConfig = &oauth2.Config{
	ClientID:     googleClientID,
	ClientSecret: googleClientSecret,
	RedirectURL:  googleRedirectURL,
	Scopes: []string{
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	Endpoint: google.Endpoint,
}

// GoogleAuthStatus represents the current Google authentication state
type GoogleAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email"`
	ExpiresAt     string `json:"expiresAt"`
}

// GoogleUserInfo represents the authenticated user's Google profile
type GoogleUserInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

func getTokenPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".openclaw", "workspace", ".credentials", "google_token.json"), nil
}

func loadToken() (*oauth2.Token, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

// CheckGoogleAuth checks if the user is authenticated with Google
func (a *App) CheckGoogleAuth() GoogleAuthStatus {
	token, err := loadToken()
	if err != nil {
		return GoogleAuthStatus{Authenticated: false}
	}

	// If token is expired, try to refresh
	if token.Expiry.Before(time.Now()) {
		tokenSource := googleOAuthConfig.TokenSource(context.Background(), token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return GoogleAuthStatus{Authenticated: false}
		}
		token = newToken
		_ = saveToken(token)
	}

	// Get user email
	info, err := a.fetchUserInfo(token)
	if err != nil {
		return GoogleAuthStatus{
			Authenticated: true,
			ExpiresAt:     token.Expiry.Format(time.RFC3339),
		}
	}

	return GoogleAuthStatus{
		Authenticated: true,
		Email:         info.Email,
		ExpiresAt:     token.Expiry.Format(time.RFC3339),
	}
}

// StartGoogleLogin initiates the Google OAuth2 login flow
func (a *App) StartGoogleLogin() GoogleAuthStatus {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			errCh <- fmt.Errorf("%s", errMsg)
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><body style="background:#0f0f0f;color:#e4e4e7;font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;">
				<div style="text-align:center">
					<h2 style="color:#ef4444">Authentication Failed</h2>
					<p>%s</p>
					<p style="color:#71717a">You can close this window.</p>
				</div>
			</body></html>`, errMsg)
			return
		}
		codeCh <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="background:#0f0f0f;color:#e4e4e7;font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;">
			<div style="text-align:center">
				<h2 style="color:#22c55e">Authentication Successful</h2>
				<p>You can close this window and return to Nyx Command Center.</p>
			</div>
		</body></html>`)
	})

	listener, err := net.Listen("tcp", ":8099")
	if err != nil {
		return GoogleAuthStatus{Authenticated: false}
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	authURL := googleOAuthConfig.AuthCodeURL("state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	runtime.BrowserOpenURL(a.ctx, authURL)

	// Wait for callback with timeout
	var authCode string
	select {
	case code := <-codeCh:
		authCode = code
	case authErr := <-errCh:
		_ = server.Shutdown(context.Background())
		runtime.EventsEmit(a.ctx, "google:error", authErr.Error())
		return GoogleAuthStatus{Authenticated: false}
	case <-time.After(5 * time.Minute):
		_ = server.Shutdown(context.Background())
		runtime.EventsEmit(a.ctx, "google:error", "Login timed out")
		return GoogleAuthStatus{Authenticated: false}
	}

	_ = server.Shutdown(context.Background())

	// Exchange code for token
	token, err := googleOAuthConfig.Exchange(context.Background(), authCode)
	if err != nil {
		runtime.EventsEmit(a.ctx, "google:error", fmt.Sprintf("Token exchange failed: %v", err))
		return GoogleAuthStatus{Authenticated: false}
	}

	if err := saveToken(token); err != nil {
		runtime.EventsEmit(a.ctx, "google:error", fmt.Sprintf("Failed to save token: %v", err))
		return GoogleAuthStatus{Authenticated: false}
	}

	// Get user info
	info, err := a.fetchUserInfo(token)
	email := ""
	if err == nil {
		email = info.Email
	}

	runtime.EventsEmit(a.ctx, "google:authenticated", email)

	return GoogleAuthStatus{
		Authenticated: true,
		Email:         email,
		ExpiresAt:     token.Expiry.Format(time.RFC3339),
	}
}

// GetGoogleUserInfo returns the authenticated user's Google profile
func (a *App) GetGoogleUserInfo() GoogleUserInfo {
	token, err := loadToken()
	if err != nil {
		return GoogleUserInfo{}
	}

	info, err := a.fetchUserInfo(token)
	if err != nil {
		return GoogleUserInfo{}
	}

	return *info
}

// LogoutGoogle removes the stored Google OAuth token
func (a *App) LogoutGoogle() bool {
	tokenPath, err := getTokenPath()
	if err != nil {
		return false
	}

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return false
	}

	runtime.EventsEmit(a.ctx, "google:logged-out", nil)
	return true
}

func (a *App) fetchUserInfo(token *oauth2.Token) (*GoogleUserInfo, error) {
	client := googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var info GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}
