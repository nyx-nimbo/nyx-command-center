package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// App struct
type App struct {
	ctx         context.Context
	sessions    map[string]*ChatSession
	sessMu      sync.Mutex
	erebusWS    *ErebusWSClient
	ticketAgent *TicketAgent
}

// ChatSession represents a single chat session/channel
type ChatSession struct {
	Key          string        `json:"key" bson:"_id"`
	Name         string        `json:"name" bson:"name"`
	Icon         string        `json:"icon" bson:"icon"`
	SystemPrompt string        `json:"systemPrompt" bson:"systemPrompt"`
	History      []ChatMessage `json:"history" bson:"history"`
	LastMessage  string        `json:"lastMessage" bson:"lastMessage"`
	LastTime     string        `json:"lastTime" bson:"lastTime"`
	Unread       int           `json:"unread" bson:"unread"`
	UpdatedAt    string        `json:"updatedAt" bson:"updatedAt"`
}

// ChatSessionInfo is the summary sent to the frontend (no full history)
type ChatSessionInfo struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	SystemPrompt string `json:"systemPrompt"`
	LastMessage  string `json:"lastMessage"`
	LastTime     string `json:"lastTime"`
	Unread       int    `json:"unread"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		sessions: make(map[string]*ChatSession),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initDefaultSessions()
}

// initDefaultSessions loads sessions from MongoDB, or creates defaults if none exist
func (a *App) initDefaultSessions() {
	// Try to load from MongoDB first
	loaded := a.LoadSessions()

	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	if loaded > 0 {
		// Always refresh the General system prompt to pick up embedded manual changes
		if s, ok := a.sessions["general"]; ok {
			s.SystemPrompt = "You are Nix ⚡, an AI assistant operating inside the Nyx Command Center desktop application. You have access to Gmail, Google Calendar, MongoDB (clients/projects/tasks), and chat. You are helping Ernesto, a Principal Engineer. Be direct, efficient, with personality. Respond in the same language the user writes in.\n\n" + embeddedAgentManual
		}
		return
	}

	// No sessions in DB — create defaults
	now := time.Now().Format(time.RFC3339)
	a.sessions["general"] = &ChatSession{
		Key:          "general",
		Name:         "General",
		Icon:         "💬",
		SystemPrompt: "You are Nix ⚡, an AI assistant operating inside the Nyx Command Center desktop application. You have access to Gmail, Google Calendar, MongoDB (clients/projects/tasks), and chat. You are helping Ernesto, a Principal Engineer. Be direct, efficient, with personality. Respond in the same language the user writes in.\n\n" + embeddedAgentManual,
		History:      []ChatMessage{},
		LastMessage:  "Welcome to Nyx Command Center",
		LastTime:     now,
		Unread:       0,
		UpdatedAt:    now,
	}
	a.sessions["ideas"] = &ChatSession{
		Key:          "ideas",
		Name:         "Ideas",
		Icon:         "💡",
		SystemPrompt: "You are a creative research assistant focused on brainstorming, evaluating, and refining ideas. Help the user explore concepts, find related research, and develop actionable plans from their ideas.",
		History:      []ChatMessage{},
		LastMessage:  "Ready for idea brainstorming",
		LastTime:     now,
		Unread:       0,
		UpdatedAt:    now,
	}

	// Persist defaults to MongoDB
	for key := range a.sessions {
		go a.saveSessionToDB(key)
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// AppInfo contains basic application metadata
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// GetAppInfo returns application name and version
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Name:    "Nyx Command Center",
		Version: "0.1.0",
		Status:  "online",
	}
}


// --- Chat Persistence (MongoDB) ---

const maxChatHistory = 50

// SaveSession persists a single session to MongoDB (callable from frontend)
func (a *App) SaveSession(key string) {
	a.saveSessionToDB(key)
}

// saveSessionToDB is the internal method that upserts a session to MongoDB
func (a *App) saveSessionToDB(key string) {
	db, err := getDB()
	if err != nil {
		log.Printf("[chat-persist] DB error: %v", err)
		return
	}

	a.sessMu.Lock()
	session, exists := a.sessions[key]
	if !exists {
		a.sessMu.Unlock()
		return
	}

	// Build a copy of history trimmed to last N messages
	history := session.History
	if len(history) > maxChatHistory {
		history = history[len(history)-maxChatHistory:]
	}

	doc := bson.M{
		"_id":          session.Key,
		"name":         session.Name,
		"icon":         session.Icon,
		"systemPrompt": session.SystemPrompt,
		"history":      history,
		"lastMessage":  session.LastMessage,
		"lastTime":     session.LastTime,
		"unread":       session.Unread,
		"updatedAt":    time.Now().Format(time.RFC3339),
	}
	a.sessMu.Unlock()

	col := db.Collection("chat_sessions")
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(
		context.Background(),
		bson.M{"_id": key},
		bson.M{"$set": doc},
		opts,
	)
	if err != nil {
		log.Printf("[chat-persist] save %s failed: %v", key, err)
	}
}

// LoadSessions loads all sessions from MongoDB into memory. Returns count loaded.
func (a *App) LoadSessions() int {
	db, err := getDB()
	if err != nil {
		log.Printf("[chat-persist] DB error on load: %v", err)
		return 0
	}

	col := db.Collection("chat_sessions")
	cursor, err := col.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("[chat-persist] find failed: %v", err)
		return 0
	}
	defer cursor.Close(context.Background())

	var sessions []ChatSession
	if err := cursor.All(context.Background(), &sessions); err != nil {
		log.Printf("[chat-persist] decode failed: %v", err)
		return 0
	}

	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	for i := range sessions {
		s := &sessions[i]
		if s.History == nil {
			s.History = []ChatMessage{}
		}
		a.sessions[s.Key] = s
	}

	return len(sessions)
}

// deleteSessionFromDB removes a session from MongoDB
func (a *App) deleteSessionFromDB(key string) {
	db, err := getDB()
	if err != nil {
		log.Printf("[chat-persist] DB error: %v", err)
		return
	}
	col := db.Collection("chat_sessions")
	_, err = col.DeleteOne(context.Background(), bson.M{"_id": key})
	if err != nil {
		log.Printf("[chat-persist] delete %s failed: %v", key, err)
	}
}

// --- Chat types and methods ---

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	Role    string      `json:"role" bson:"role"`
	Content interface{} `json:"content" bson:"content"`
}

// ChatContentPart represents a content part for multimodal messages
type ChatContentPart struct {
	Type     string    `json:"type" bson:"type"`
	Text     string    `json:"text,omitempty" bson:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty" bson:"image_url,omitempty"`
}

// ImageURL represents an image URL in a content part
type ImageURL struct {
	URL string `json:"url" bson:"url"`
}

// chatRequest is the request body for the OpenAI-compatible API
type chatRequest struct {
	Model    string        `json:"model,omitempty"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	User     string        `json:"user"`
}

// sseChoice represents a choice in an SSE chunk
type sseChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

// sseChunk represents an SSE data chunk
type sseChunk struct {
	Choices []sseChoice `json:"choices"`
}

var openclawToken = "" // Set via OPENCLAW_TOKEN env var

var openclawURL = "http://localhost:18789/v1/chat/completions" // Override via OPENCLAW_URL env var

const (
	openclawAgent = "main"
	openclawUser  = "nyx-dashboard"
)

// --- Multi-Session Methods ---

// CreateChatSession creates a new chat session with the given name and optional system prompt
func (a *App) CreateChatSession(name string, systemPrompt string) ChatSessionInfo {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	key := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	// Ensure unique key
	baseKey := key
	counter := 1
	for {
		if _, exists := a.sessions[key]; !exists {
			break
		}
		counter++
		key = fmt.Sprintf("%s-%d", baseKey, counter)
	}

	now := time.Now().Format(time.RFC3339)
	session := &ChatSession{
		Key:          key,
		Name:         name,
		Icon:         "📝",
		SystemPrompt: systemPrompt,
		History:      []ChatMessage{},
		LastMessage:  "New session created",
		LastTime:     now,
		Unread:       0,
		UpdatedAt:    now,
	}
	a.sessions[key] = session

	go a.saveSessionToDB(key)
	go a.LogActivity("create", "chat_session", key, "Created chat session: "+name, "")

	return ChatSessionInfo{
		Key:          session.Key,
		Name:         session.Name,
		Icon:         session.Icon,
		SystemPrompt: session.SystemPrompt,
		LastMessage:  session.LastMessage,
		LastTime:     session.LastTime,
		Unread:       session.Unread,
	}
}

// ListChatSessions returns info about all chat sessions
func (a *App) ListChatSessions() []ChatSessionInfo {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	var result []ChatSessionInfo
	for _, s := range a.sessions {
		result = append(result, ChatSessionInfo{
			Key:          s.Key,
			Name:         s.Name,
			Icon:         s.Icon,
			SystemPrompt: s.SystemPrompt,
			LastMessage:  s.LastMessage,
			LastTime:     s.LastTime,
			Unread:       s.Unread,
		})
	}
	return result
}

// SwitchSession returns the full history for a session and resets its unread count
func (a *App) SwitchSession(key string) []ChatMessage {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	session, exists := a.sessions[key]
	if !exists {
		return []ChatMessage{}
	}

	session.Unread = 0
	result := make([]ChatMessage, len(session.History))
	copy(result, session.History)
	return result
}

// DeleteSession removes a chat session
func (a *App) DeleteSession(key string) bool {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	if key == "general" {
		return false // Cannot delete the default session
	}

	if _, exists := a.sessions[key]; exists {
		delete(a.sessions, key)
		go a.deleteSessionFromDB(key)
		go a.LogActivity("delete", "chat_session", key, "Deleted chat session: "+key, "")
		return true
	}
	return false
}

// StreamChat sends a message to OpenClaw and streams the response via Wails events
func (a *App) StreamChat(sessionKey string, message string) {
	a.sessMu.Lock()
	session, exists := a.sessions[sessionKey]
	if !exists {
		a.sessMu.Unlock()
		runtime.EventsEmit(a.ctx, "chat:error", "Session not found: "+sessionKey)
		return
	}

	session.History = append(session.History, ChatMessage{
		Role:    "user",
		Content: message,
	})

	// Build messages with optional system prompt
	var messages []ChatMessage
	if session.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: session.SystemPrompt,
		})
	}

	// Inject live DB context so the agent knows current data
	if dbCtx := a.getDBContext(); dbCtx != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: "[Live Database Context]\n" + dbCtx,
		})
	}

	messages = append(messages, session.History...)
	a.sessMu.Unlock()

	go a.doStreamChat(sessionKey, messages)
}

// StreamChatWithImages sends a message with optional images to OpenClaw
func (a *App) StreamChatWithImages(sessionKey string, message string, imageDataURLs []string) {
	a.sessMu.Lock()
	session, exists := a.sessions[sessionKey]
	if !exists {
		a.sessMu.Unlock()
		runtime.EventsEmit(a.ctx, "chat:error", "Session not found: "+sessionKey)
		return
	}

	var content interface{}
	if len(imageDataURLs) > 0 {
		parts := []ChatContentPart{
			{Type: "text", Text: message},
		}
		for _, dataURL := range imageDataURLs {
			parts = append(parts, ChatContentPart{
				Type:     "image_url",
				ImageURL: &ImageURL{URL: dataURL},
			})
		}
		content = parts
	} else {
		content = message
	}

	session.History = append(session.History, ChatMessage{
		Role:    "user",
		Content: content,
	})

	var messages []ChatMessage
	if session.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: session.SystemPrompt,
		})
	}

	// Inject live DB context so the agent knows current data
	if dbCtx := a.getDBContext(); dbCtx != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: "[Live Database Context]\n" + dbCtx,
		})
	}

	messages = append(messages, session.History...)
	a.sessMu.Unlock()

	go a.doStreamChat(sessionKey, messages)
}

func (a *App) doStreamChat(sessionKey string, messages []ChatMessage) {
	reqBody := chatRequest{
		Messages: messages,
		Stream:   true,
		User:     openclawUser,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		runtime.EventsEmit(a.ctx, "chat:error", fmt.Sprintf("Failed to marshal request: %v", err))
		return
	}

	req, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		runtime.EventsEmit(a.ctx, "chat:error", fmt.Sprintf("Failed to create request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openclawToken)
	req.Header.Set("x-openclaw-agent-id", openclawAgent)
	req.Header.Set("x-openclaw-session-key", sessionKey)

	client := &http.Client{Timeout: 0} // No timeout for SSE streaming
	resp, err := client.Do(req)
	if err != nil {
		runtime.EventsEmit(a.ctx, "chat:error", fmt.Sprintf("Request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		runtime.EventsEmit(a.ctx, "chat:error", fmt.Sprintf("API error %d: %s", resp.StatusCode, string(body)))
		return
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fullResponse.WriteString(choice.Delta.Content)
				runtime.EventsEmit(a.ctx, "chat:chunk", choice.Delta.Content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		runtime.EventsEmit(a.ctx, "chat:error", fmt.Sprintf("Stream read error: %v", err))
		return
	}

	responseText := fullResponse.String()

	a.sessMu.Lock()
	if session, exists := a.sessions[sessionKey]; exists {
		session.History = append(session.History, ChatMessage{
			Role:    "assistant",
			Content: responseText,
		})
		// Update last message preview
		preview := responseText
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		session.LastMessage = preview
		session.LastTime = time.Now().Format(time.RFC3339)
		session.UpdatedAt = session.LastTime
	}
	a.sessMu.Unlock()

	go a.saveSessionToDB(sessionKey)

	runtime.EventsEmit(a.ctx, "chat:done", responseText)
}

// ClearChatHistory resets the conversation history for a session
func (a *App) ClearChatHistory(sessionKey string) {
	a.sessMu.Lock()
	if session, exists := a.sessions[sessionKey]; exists {
		session.History = nil
		session.LastMessage = "Chat cleared"
		session.LastTime = time.Now().Format(time.RFC3339)
		session.UpdatedAt = session.LastTime
	}
	a.sessMu.Unlock()

	go a.saveSessionToDB(sessionKey)
}

// GetChatHistory returns the current conversation messages for a session
func (a *App) GetChatHistory(sessionKey string) []ChatMessage {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()

	session, exists := a.sessions[sessionKey]
	if !exists {
		return []ChatMessage{}
	}

	result := make([]ChatMessage, len(session.History))
	copy(result, session.History)
	return result
}
