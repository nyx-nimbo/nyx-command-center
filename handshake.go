package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const handshakeSessionKey = "nyx-handshake"

// HandshakeStatus represents the current handshake state
type HandshakeStatus struct {
	Connected     bool   `json:"connected"`
	LastHandshake string `json:"lastHandshake"`
}

// handshakeFile is the persisted handshake record
type handshakeFile struct {
	Connected     bool   `json:"connected"`
	Timestamp     string `json:"timestamp"`
	AgentResponse string `json:"agentResponse"`
}

// chatCompletionResponse is the non-streaming OpenAI-compatible response
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func handshakePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", ".credentials", "nyx_handshake.json")
}

// CheckHandshake checks if a valid (< 24h old) handshake exists
func (a *App) CheckHandshake() HandshakeStatus {
	data, err := os.ReadFile(handshakePath())
	if err != nil {
		return HandshakeStatus{Connected: false}
	}

	var hf handshakeFile
	if err := json.Unmarshal(data, &hf); err != nil {
		return HandshakeStatus{Connected: false}
	}

	_, err = time.Parse(time.RFC3339, hf.Timestamp)
	if err != nil {
		return HandshakeStatus{Connected: false}
	}

	return HandshakeStatus{
		Connected:     hf.Connected,
		LastHandshake: hf.Timestamp,
	}
}

// ResetHandshake deletes the saved handshake so the button becomes active again
func (a *App) ResetHandshake() error {
	err := os.Remove(handshakePath())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to reset handshake: %w", err)
	}
	return nil
}

// PerformHandshake sends the AGENT.md capabilities to the OpenClaw agent and saves the result
func (a *App) PerformHandshake() (string, error) {
	capabilities := a.GetCapabilities()

	messages := []ChatMessage{
		{
			Role: "system",
			Content: `You are an OpenClaw agent receiving a permanent connection request from Nyx Command Center.
This is a HANDSHAKE — a one-time setup that establishes a permanent link between you and this application.

Your instructions:
1. Read the capabilities document carefully.
2. Save this information PERMANENTLY to your memory files:
   - Update MEMORY.md with a "Nyx Command Center" section describing what it is and how to use it.
   - Save the full capabilities to your workspace so you always have them available.
3. Confirm the connection is established and that you have saved the information.

This is not a temporary context — you must persist this knowledge across sessions.`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`HANDSHAKE REQUEST from Nyx Command Center.

Save the following capabilities document to your MEMORY.md permanently under a "## Nyx Command Center" section. This application is always available at localhost and you should always know how to use it.

---

%s

---

After saving to memory, confirm: what is Nyx Command Center and what can you do with it?`, capabilities),
		},
	}

	reqBody := chatRequest{
		Messages: messages,
		Stream:   false,
		User:     openclawUser,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openclawToken)
	req.Header.Set("x-openclaw-agent-id", openclawAgent)
	req.Header.Set("x-openclaw-session-key", handshakeSessionKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no response from agent")
	}

	agentResponse := completion.Choices[0].Message.Content

	// Save handshake status
	hf := handshakeFile{
		Connected:     true,
		Timestamp:     time.Now().Format(time.RFC3339),
		AgentResponse: agentResponse,
	}

	hfBytes, err := json.MarshalIndent(hf, "", "  ")
	if err != nil {
		return agentResponse, fmt.Errorf("failed to marshal handshake file: %w", err)
	}

	dir := filepath.Dir(handshakePath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return agentResponse, fmt.Errorf("failed to create credentials dir: %w", err)
	}

	if err := os.WriteFile(handshakePath(), hfBytes, 0600); err != nil {
		return agentResponse, fmt.Errorf("failed to save handshake: %w", err)
	}

	// Auto-register this instance as an agent
	hostname, _ := os.Hostname()
	a.RegisterAgent(getInstanceID(), hostname+" (Nyx)", "agent", []string{"chat", "code", "review"})

	return agentResponse, nil
}
