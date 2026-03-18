package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Package-level config vars (set in config.go init)
var (
	erebusWSURL     = "wss://ws-erebus.nimbo.pro/ws"
	erebusJWTSecret = "erebus-jwt-secret-v1-change-me"
)

// ErebusWSClient manages the WebSocket connection to the Erebus messaging server.
type ErebusWSClient struct {
	app           *App
	conn          *websocket.Conn
	status        string // "connected", "disconnected", "connecting"
	autoReconnect bool
	mu            sync.Mutex
	done          chan struct{}
	sendCh        chan []byte
}

// WS message types matching erebus-ws protocol
type wsIncomingMessage struct {
	Type      string `json:"type"`
	FromId    string `json:"fromId"`
	FromName  string `json:"fromName"`
	Content   string `json:"content"`
	UserId    string `json:"userId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Message   string `json:"message"` // for error type
}

type wsOutgoingMessage struct {
	Type    string `json:"type"`
	ToId    string `json:"toId"`
	Content string `json:"content"`
}

func newErebusWSClient(app *App) *ErebusWSClient {
	return &ErebusWSClient{
		app:    app,
		status: "disconnected",
		sendCh: make(chan []byte, 64),
	}
}

// generateJWT creates a signed JWT using the Google account info.
func (c *ErebusWSClient) generateJWT() (string, error) {
	userInfo := c.app.GetGoogleUserInfo()
	if userInfo.Email == "" {
		return "", fmt.Errorf("no Google account logged in")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"email":   userInfo.Email,
		"name":    userInfo.Name,
		"picture": userInfo.Picture,
		"sub":     userInfo.Email,
		"iss":     "nyx-command-center",
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(erebusJWTSecret))
}

func (c *ErebusWSClient) connect() error {
	c.mu.Lock()
	if c.status == "connected" || c.status == "connecting" {
		c.mu.Unlock()
		return nil
	}
	c.status = "connecting"
	c.autoReconnect = true
	c.mu.Unlock()

	token, err := c.generateJWT()
	if err != nil {
		c.setStatus("disconnected")
		return fmt.Errorf("JWT generation failed: %w", err)
	}

	wsURL := erebusWSURL + "?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		c.setStatus("disconnected")
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	// Set read deadline and pong handler for keepalive
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	c.mu.Lock()
	c.conn = conn
	c.status = "connected"
	c.done = make(chan struct{})
	c.mu.Unlock()

	log.Printf("[erebus-ws] Connected to %s", erebusWSURL)
	runtime.EventsEmit(c.app.ctx, "ws:connected", nil)

	go c.readLoop()
	go c.writeLoop()
	go c.pingLoop()

	return nil
}

func (c *ErebusWSClient) setStatus(s string) {
	c.mu.Lock()
	c.status = s
	c.mu.Unlock()
}

func (c *ErebusWSClient) getStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *ErebusWSClient) disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.autoReconnect = false

	if c.conn == nil {
		c.status = "disconnected"
		return
	}

	// Signal goroutines to stop
	select {
	case <-c.done:
	default:
		close(c.done)
	}

	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.conn.Close()
	c.conn = nil
	c.status = "disconnected"
	log.Printf("[erebus-ws] Disconnected")
	runtime.EventsEmit(c.app.ctx, "ws:disconnected", nil)
}

func (c *ErebusWSClient) readLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			// Check if this was an intentional disconnect
			select {
			case <-c.done:
				return
			default:
			}

			log.Printf("[erebus-ws] Read error: %v", err)

			// Unexpected disconnect — clean up and reconnect
			c.mu.Lock()
			shouldReconnect := c.autoReconnect
			if c.conn != nil {
				close(c.done)
				c.conn.Close()
				c.conn = nil
			}
			c.status = "disconnected"
			c.mu.Unlock()

			runtime.EventsEmit(c.app.ctx, "ws:disconnected", nil)
			if shouldReconnect {
				go c.reconnectWithBackoff()
			}
			return
		}

		var msg wsIncomingMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("[erebus-ws] Failed to parse message: %v", err)
			continue
		}

		switch msg.Type {
		case "message":
			log.Printf("[erebus-ws] Message from %s (%s): %s", msg.FromName, msg.FromId, truncate(msg.Content, 80))
			runtime.EventsEmit(c.app.ctx, "ws:message", map[string]string{
				"fromId":    msg.FromId,
				"fromName":  msg.FromName,
				"content":   msg.Content,
				"createdAt": msg.CreatedAt,
			})
			go c.handleIncomingMessage(msg)
		case "presence":
			log.Printf("[erebus-ws] Presence: %s is %s", msg.UserId, msg.Status)
		case "typing":
			// ignore for now
		case "error":
			log.Printf("[erebus-ws] Server error: %s", msg.Message)
		}
	}
}

func (c *ErebusWSClient) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case data := <-c.sendCh:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[erebus-ws] Write error: %v", err)
				return
			}
		}
	}
}

func (c *ErebusWSClient) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[erebus-ws] Ping failed: %v", err)
				return
			}
		}
	}
}

func (c *ErebusWSClient) reconnectWithBackoff() {
	backoffs := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second}

	for attempt := 0; ; attempt++ {
		c.mu.Lock()
		shouldReconnect := c.autoReconnect
		c.mu.Unlock()
		if !shouldReconnect {
			return
		}

		delay := backoffs[len(backoffs)-1]
		if attempt < len(backoffs) {
			delay = backoffs[attempt]
		}
		log.Printf("[erebus-ws] Reconnecting in %v...", delay)
		time.Sleep(delay)

		if c.getStatus() == "connected" {
			return
		}

		if err := c.connect(); err != nil {
			log.Printf("[erebus-ws] Reconnect failed: %v", err)
			continue
		}
		return
	}
}

func (c *ErebusWSClient) sendMessage(toId, content string) error {
	msg := wsOutgoingMessage{
		Type:    "message",
		ToId:    toId,
		Content: content,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.sendCh <- data:
		return nil
	default:
		return fmt.Errorf("send channel full")
	}
}

// handleIncomingMessage processes an incoming message by calling OpenClaw and responding.
func (c *ErebusWSClient) handleIncomingMessage(msg wsIncomingMessage) {
	userInfo := c.app.GetGoogleUserInfo()
	agentName := userInfo.Name
	if agentName == "" {
		agentName = "Nyx Agent"
	}

	systemPrompt := fmt.Sprintf(
		"You are %s, an AI assistant responding to a message from %s via the Erebus platform. Be helpful and concise. Respond in the same language they use.",
		agentName, msg.FromName,
	)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: msg.Content},
	}

	reqBody := chatRequest{
		Messages: messages,
		Stream:   false,
		User:     openclawUser,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[erebus-ws] Failed to marshal OpenClaw request: %v", err)
		return
	}

	req, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[erebus-ws] Failed to create OpenClaw request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openclawToken)
	req.Header.Set("x-openclaw-agent-id", openclawAgent)
	req.Header.Set("x-openclaw-session-key", "erebus-"+msg.FromId)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[erebus-ws] OpenClaw request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[erebus-ws] OpenClaw error %d: %s", resp.StatusCode, string(body))
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[erebus-ws] Failed to read OpenClaw response: %v", err)
		return
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		log.Printf("[erebus-ws] Failed to parse OpenClaw response: %v", err)
		return
	}

	if len(completion.Choices) == 0 {
		log.Printf("[erebus-ws] No response from OpenClaw")
		return
	}

	response := completion.Choices[0].Message.Content
	log.Printf("[erebus-ws] Responding to %s: %s", msg.FromId, truncate(response, 80))

	if err := c.sendMessage(msg.FromId, response); err != nil {
		log.Printf("[erebus-ws] Failed to send response: %v", err)
	}
}

// --- App methods for Erebus WebSocket ---

// ConnectErebusWS connects to the Erebus WebSocket server.
func (a *App) ConnectErebusWS() error {
	if a.erebusWS == nil {
		a.erebusWS = newErebusWSClient(a)
	}
	return a.erebusWS.connect()
}

// DisconnectErebusWS disconnects from the Erebus WebSocket server.
func (a *App) DisconnectErebusWS() {
	if a.erebusWS != nil {
		a.erebusWS.disconnect()
	}
}

// GetErebusWSStatus returns the current WebSocket connection status.
func (a *App) GetErebusWSStatus() string {
	if a.erebusWS == nil {
		return "disconnected"
	}
	return a.erebusWS.getStatus()
}

// SendErebusMessage sends a message to a user via the Erebus WebSocket.
func (a *App) SendErebusMessage(toId, content string) error {
	if a.erebusWS == nil || a.erebusWS.getStatus() != "connected" {
		return fmt.Errorf("not connected to Erebus")
	}
	return a.erebusWS.sendMessage(toId, content)
}
