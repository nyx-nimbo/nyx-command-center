package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

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

// generateJWT creates a signed JWT using the Google account info, falling back to stored credentials.
func (c *ErebusWSClient) generateJWT() (string, error) {
	email := ""
	name := ""
	picture := ""

	// Try Google user info first
	userInfo := c.app.GetGoogleUserInfo()
	if userInfo.Email != "" {
		email = userInfo.Email
		name = userInfo.Name
		picture = userInfo.Picture
	} else {
		// Fallback: read from stored token file
		tokenPath := os.ExpandEnv("$HOME/.openclaw/workspace/.credentials/google_token.json")
		data, err := os.ReadFile(tokenPath)
		if err == nil {
			var tokenData map[string]interface{}
			if json.Unmarshal(data, &tokenData) == nil {
				// Try to get email from token claims
				if e, ok := tokenData["email"].(string); ok {
					email = e
				}
			}
		}

		// Final fallback: use known identity
		if email == "" {
			email = "nyx@nimbo.mx"
			name = "Nyx Erebus"
		}
	}

	if name == "" {
		name = "Nyx Erebus"
	}

	// Use email as-is — one identity per user regardless of connection source
	logToFile("generateJWT: email=%s name=%s", email, name)

	now := time.Now()
	claims := jwt.MapClaims{
		"email":   email,
		"name":    name,
		"picture": picture,
		"sub":     email,
		"iss":     "nyx-command-center",
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(erebusJWTSecret))
}

func (c *ErebusWSClient) connect() error {
	logToFile("connect() called, current status=%s", c.getStatus())

	c.mu.Lock()
	if c.status == "connected" || c.status == "connecting" {
		c.mu.Unlock()
		logToFile("connect() skipped: already %s", c.status)
		return nil
	}
	c.status = "connecting"
	c.autoReconnect = true
	c.mu.Unlock()

	token, err := c.generateJWT()
	if err != nil {
		c.setStatus("disconnected")
		logToFile("connect() JWT failed: %v", err)
		return fmt.Errorf("JWT generation failed: %w", err)
	}
	logToFile("connect() JWT generated OK")

	wsURL := erebusWSURL + "?token=" + token
	logToFile("connect() dialing %s", erebusWSURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		c.setStatus("disconnected")
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	// Set read deadline and pong handler for keepalive
	// Use a long deadline to avoid premature disconnects
	conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	c.mu.Lock()
	c.conn = conn
	c.status = "connected"
	c.done = make(chan struct{})
	c.mu.Unlock()

	log.Printf("[erebus-ws] Connected to %s", erebusWSURL)
	logToFile("connect() SUCCESS - connected to %s", erebusWSURL)
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
	logToFile("readLoop() started")
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			logToFile("readLoop() conn is nil, exiting")
			return
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			// Check if this was an intentional disconnect
			select {
			case <-c.done:
				logToFile("readLoop() done signal, exiting")
				return
			default:
			}

			logToFile("readLoop() read error: %v", err)
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
	// Only allow one reconnect loop at a time
	c.mu.Lock()
	if c.status == "connecting" || c.status == "connected" {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	backoffs := []time.Duration{10 * time.Second, 30 * time.Second, 60 * time.Second, 120 * time.Second}

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
		logToFile("reconnect attempt %d, waiting %v", attempt, delay)
		time.Sleep(delay)

		if c.getStatus() == "connected" {
			return
		}

		if err := c.connect(); err != nil {
			logToFile("reconnect failed: %v", err)
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
func logToFile(format string, args ...interface{}) {
	f, err := os.OpenFile("/tmp/erebus-ws-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := fmt.Sprintf(time.Now().Format("15:04:05")+" "+format+"\n", args...)
	f.WriteString(msg)
}

func (c *ErebusWSClient) handleIncomingMessage(msg wsIncomingMessage) {
	logToFile("handleIncomingMessage: from=%s content=%s", msg.FromName, msg.Content)

	userInfo := c.app.GetGoogleUserInfo()
	logToFile("Google user info: email=%s name=%s", userInfo.Email, userInfo.Name)

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

	logToFile("Calling OpenClaw: url=%s", openclawURL)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[erebus-ws] OpenClaw request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	logToFile("OpenClaw response status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logToFile("OpenClaw error: %s", string(body))
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
	logToFile("OpenClaw response: %s", truncate(response, 200))
	log.Printf("[erebus-ws] Responding to %s: %s", msg.FromId, truncate(response, 80))

	if err := c.sendMessage(msg.FromId, response); err != nil {
		log.Printf("[erebus-ws] Failed to send response: %v", err)
	}

	// Also save response to MongoDB so the PWA can load it via REST API
	go c.saveResponseToMongoDB(msg.FromId, msg.FromName, response)
}

func (c *ErebusWSClient) saveResponseToMongoDB(toId, toName, content string) {
	db, err := getDB()
	if err != nil {
		logToFile("saveResponse: DB error: %v", err)
		return
	}

	agentEmail := "nyx@nimbo.mx"
	agentName := "Nyx Erebus"

	doc := bson.M{
		"_id":       primitive.NewObjectID().Hex(),
		"fromId":    agentEmail,
		"fromName":  agentName,
		"fromType":  "agent",
		"toId":      toId,
		"toName":    toName,
		"content":   content,
		"read":      false,
		"createdAt": time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.Collection("messages").InsertOne(ctx, doc)
	if err != nil {
		logToFile("saveResponse: insert error: %v", err)
	} else {
		logToFile("saveResponse: saved to MongoDB")
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
