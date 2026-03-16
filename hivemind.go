package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Instance ID ---

var (
	instanceID     string
	instanceIDOnce sync.Once
)

func getInstanceID() string {
	instanceIDOnce.Do(func() {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		instanceID = hostname + "-nyx"
	})
	return instanceID
}

// GetInstanceId returns a unique identifier for this machine
func (a *App) GetInstanceId() string {
	return getInstanceID()
}

// --- Embedding Generation ---

type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func generateEmbedding(text string) ([]float64, error) {
	reqBody := ollamaEmbeddingRequest{
		Model:  "bge-m3",
		Prompt: text,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("http://localhost:11434/api/embeddings", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("ollama request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	return result.Embedding, nil
}

// --- Data Models ---

type ActivityLogEntry struct {
	ID         string    `json:"id" bson:"_id,omitempty"`
	InstanceID string    `json:"instanceId" bson:"instanceId"`
	Action     string    `json:"action" bson:"action"`
	EntityType string    `json:"entityType" bson:"entityType"`
	EntityID   string    `json:"entityId" bson:"entityId"`
	Summary    string    `json:"summary" bson:"summary"`
	Data       string    `json:"data" bson:"data"`
	Embedding  []float64 `json:"embedding,omitempty" bson:"embedding,omitempty"`
	Timestamp  string    `json:"timestamp" bson:"timestamp"`
	Synced     bool      `json:"synced" bson:"synced"`
}

type KnowledgeEntry struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	Type      string    `json:"type" bson:"type"`
	Title     string    `json:"title" bson:"title"`
	Content   string    `json:"content" bson:"content"`
	Tags      []string  `json:"tags" bson:"tags"`
	ProjectID string    `json:"projectId,omitempty" bson:"projectId,omitempty"`
	Embedding []float64 `json:"embedding,omitempty" bson:"embedding,omitempty"`
	CreatedBy string    `json:"createdBy" bson:"createdBy"`
	CreatedAt string    `json:"createdAt" bson:"createdAt"`
	UpdatedAt string    `json:"updatedAt" bson:"updatedAt"`
}

type KnowledgeSearchResult struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Type      string   `json:"type"`
	Tags      []string `json:"tags"`
	Score     float64  `json:"score"`
	CreatedAt string   `json:"createdAt"`
}

type SyncState struct {
	LastSyncTime   string `json:"lastSyncTime"`
	PendingChanges int    `json:"pendingChanges"`
}

// --- Activity Logging ---

// LogActivity records an action in the activity log with an embedding
func (a *App) LogActivity(action, entityType, entityId, summary, data string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	// Generate embedding (non-blocking failure)
	embedding, _ := generateEmbedding(summary)

	entry := ActivityLogEntry{
		ID:         primitive.NewObjectID().Hex(),
		InstanceID: getInstanceID(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityId,
		Summary:    summary,
		Data:       data,
		Embedding:  embedding,
		Timestamp:  time.Now().Format(time.RFC3339),
		Synced:     true,
	}

	_, err = db.Collection("activity_log").InsertOne(context.Background(), entry)
	if err != nil {
		return fmt.Errorf("insert error: %v", err)
	}

	// Emit event for real-time UI updates
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "hivemind:new-activity", entry)
	}

	return nil
}

// --- Knowledge Base ---

// AddKnowledge adds a new entry to the knowledge base
func (a *App) AddKnowledge(knowledgeType, title, content, tags, projectId string) (KnowledgeEntry, error) {
	db, err := getDB()
	if err != nil {
		return KnowledgeEntry{}, fmt.Errorf("db error: %v", err)
	}

	// Parse tags from comma-separated string
	var tagList []string
	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tagList = append(tagList, trimmed)
			}
		}
	}
	if tagList == nil {
		tagList = []string{}
	}

	// Generate embedding from title + content
	embedding, _ := generateEmbedding(title + " " + content)

	now := time.Now().Format(time.RFC3339)
	entry := KnowledgeEntry{
		ID:        primitive.NewObjectID().Hex(),
		Type:      knowledgeType,
		Title:     title,
		Content:   content,
		Tags:      tagList,
		ProjectID: projectId,
		Embedding: embedding,
		CreatedBy: getInstanceID(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = db.Collection("knowledge").InsertOne(context.Background(), entry)
	if err != nil {
		return KnowledgeEntry{}, fmt.Errorf("insert error: %v", err)
	}

	return entry, nil
}

// SearchKnowledge performs semantic search over the knowledge base using cosine similarity
func (a *App) SearchKnowledge(query string, limit int) ([]KnowledgeSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	// Generate query embedding
	queryEmb, err := generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("embedding error: %v", err)
	}

	// Fetch all knowledge docs
	cursor, err := db.Collection("knowledge").Find(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var entries []KnowledgeEntry
	if err := cursor.All(context.Background(), &entries); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	// Compute cosine similarity and rank
	type scored struct {
		entry KnowledgeEntry
		score float64
	}
	var scoredEntries []scored

	for _, e := range entries {
		if len(e.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(queryEmb, e.Embedding)
		scoredEntries = append(scoredEntries, scored{entry: e, score: sim})
	}

	// Sort by score descending
	for i := 0; i < len(scoredEntries); i++ {
		for j := i + 1; j < len(scoredEntries); j++ {
			if scoredEntries[j].score > scoredEntries[i].score {
				scoredEntries[i], scoredEntries[j] = scoredEntries[j], scoredEntries[i]
			}
		}
	}

	// Take top N
	if len(scoredEntries) > limit {
		scoredEntries = scoredEntries[:limit]
	}

	var results []KnowledgeSearchResult
	for _, s := range scoredEntries {
		results = append(results, KnowledgeSearchResult{
			Title:     s.entry.Title,
			Content:   s.entry.Content,
			Type:      s.entry.Type,
			Tags:      s.entry.Tags,
			Score:     math.Round(s.score*1000) / 1000,
			CreatedAt: s.entry.CreatedAt,
		})
	}

	if results == nil {
		results = []KnowledgeSearchResult{}
	}

	return results, nil
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// --- Activity Queries ---

// GetRecentActivity returns the latest activity log entries
func (a *App) GetRecentActivity(limit int) ([]ActivityLogEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))
	cursor, err := db.Collection("activity_log").Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var entries []ActivityLogEntry
	if err := cursor.All(context.Background(), &entries); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if entries == nil {
		entries = []ActivityLogEntry{}
	}

	return entries, nil
}

// GetActivityForEntity returns activity log entries for a specific entity
func (a *App) GetActivityForEntity(entityType, entityId string) ([]ActivityLogEntry, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	filter := bson.M{"entityType": entityType, "entityId": entityId}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := db.Collection("activity_log").Find(context.Background(), filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var entries []ActivityLogEntry
	if err := cursor.All(context.Background(), &entries); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if entries == nil {
		entries = []ActivityLogEntry{}
	}

	return entries, nil
}

// --- Smart Polling Sync ---

var (
	syncMu       sync.Mutex
	syncRunning  bool
	syncStop     chan struct{}
	lastSyncTime time.Time
)

// GetSyncState returns the current sync state
func (a *App) GetSyncState() SyncState {
	syncMu.Lock()
	defer syncMu.Unlock()

	lt := ""
	if !lastSyncTime.IsZero() {
		lt = lastSyncTime.Format(time.RFC3339)
	}

	// Count pending (unsynced from other instances since last sync)
	pending := 0
	if !lastSyncTime.IsZero() {
		db, err := getDB()
		if err == nil {
			filter := bson.M{
				"instanceId": bson.M{"$ne": getInstanceID()},
				"timestamp":  bson.M{"$gt": lastSyncTime.Format(time.RFC3339)},
			}
			count, err := db.Collection("activity_log").CountDocuments(context.Background(), filter)
			if err == nil {
				pending = int(count)
			}
		}
	}

	return SyncState{
		LastSyncTime:   lt,
		PendingChanges: pending,
	}
}

// StartSync begins background polling for new activity from other instances
func (a *App) StartSync() {
	syncMu.Lock()
	if syncRunning {
		syncMu.Unlock()
		return
	}
	syncRunning = true
	syncStop = make(chan struct{})
	if lastSyncTime.IsZero() {
		lastSyncTime = time.Now()
	}
	syncMu.Unlock()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-syncStop:
				return
			case <-ticker.C:
				a.pollNewActivity()
			}
		}
	}()
}

// StopSync stops the background sync polling
func (a *App) StopSync() {
	syncMu.Lock()
	defer syncMu.Unlock()

	if syncRunning && syncStop != nil {
		close(syncStop)
		syncRunning = false
	}
}

func (a *App) pollNewActivity() {
	db, err := getDB()
	if err != nil {
		return
	}

	syncMu.Lock()
	since := lastSyncTime.Format(time.RFC3339)
	syncMu.Unlock()

	filter := bson.M{
		"instanceId": bson.M{"$ne": getInstanceID()},
		"timestamp":  bson.M{"$gt": since},
	}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := db.Collection("activity_log").Find(context.Background(), filter, opts)
	if err != nil {
		return
	}
	defer cursor.Close(context.Background())

	var newEntries []ActivityLogEntry
	if err := cursor.All(context.Background(), &newEntries); err != nil {
		return
	}

	if len(newEntries) > 0 {
		// Emit event for frontend
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "hivemind:new-activity", newEntries)
		}

		syncMu.Lock()
		lastSyncTime = time.Now()
		syncMu.Unlock()
	}
}
