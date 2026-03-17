package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Idea Data Model ---

type IdeaResearchEntry struct {
	Type      string `json:"type" bson:"type"`
	Title     string `json:"title" bson:"title"`
	Content   string `json:"content" bson:"content"`
	Source    string `json:"source" bson:"source"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
	CreatedBy string `json:"createdBy" bson:"createdBy"`
}

type IdeaSuggestedTask struct {
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	Status      string `json:"status" bson:"status"`
	CreatedAt   string `json:"createdAt" bson:"createdAt"`
}

type Idea struct {
	ID               string              `json:"id" bson:"_id,omitempty"`
	Title            string              `json:"title" bson:"title"`
	Description      string              `json:"description" bson:"description"`
	Status           string              `json:"status" bson:"status"`
	Category         string              `json:"category" bson:"category"`
	Priority         string              `json:"priority" bson:"priority"`
	Tags             []string            `json:"tags" bson:"tags"`
	Research         []IdeaResearchEntry `json:"research" bson:"research"`
	SuggestedTasks   []IdeaSuggestedTask `json:"suggestedTasks" bson:"suggestedTasks"`
	Notes            string              `json:"notes" bson:"notes"`
	EstimatedEffort  string              `json:"estimatedEffort" bson:"estimatedEffort"`
	PotentialRevenue string              `json:"potentialRevenue" bson:"potentialRevenue"`
	Embedding        []float64           `json:"embedding,omitempty" bson:"embedding,omitempty"`
	ProjectID        string              `json:"projectId,omitempty" bson:"projectId,omitempty"`
	CreatedAt        string              `json:"createdAt" bson:"createdAt"`
	UpdatedAt        string              `json:"updatedAt" bson:"updatedAt"`
	LastResearchedAt string              `json:"lastResearchedAt,omitempty" bson:"lastResearchedAt,omitempty"`
	CreatedBy        string              `json:"createdBy" bson:"createdBy"`
}

// --- CRUD ---

func (a *App) CreateIdea(title, description, category, priority string, tags []string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	if tags == nil {
		tags = []string{}
	}

	now := time.Now().Format(time.RFC3339)
	idea := Idea{
		ID:             primitive.NewObjectID().Hex(),
		Title:          title,
		Description:    description,
		Status:         "new",
		Category:       category,
		Priority:       priority,
		Tags:           tags,
		Research:       []IdeaResearchEntry{},
		SuggestedTasks: []IdeaSuggestedTask{},
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      getInstanceID(),
	}

	embedding, _ := generateEmbedding(title + " " + description)
	idea.Embedding = embedding

	_, err = db.Collection("ideas").InsertOne(context.Background(), idea)
	if err != nil {
		return Idea{}, fmt.Errorf("insert error: %v", err)
	}

	go a.LogActivity("created", "idea", idea.ID, "Created idea: "+title, title)
	return idea, nil
}

func (a *App) GetIdeas(statusFilter string) ([]Idea, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	filter := bson.M{}
	if statusFilter != "" && statusFilter != "all" {
		filter["status"] = statusFilter
	} else {
		filter["status"] = bson.M{"$ne": "discarded"}
	}

	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	cursor, err := db.Collection("ideas").Find(context.Background(), filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var ideas []Idea
	if err := cursor.All(context.Background(), &ideas); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if ideas == nil {
		ideas = []Idea{}
	}
	return ideas, nil
}

func (a *App) GetIdea(id string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	var idea Idea
	err = db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": id}).Decode(&idea)
	if err != nil {
		return Idea{}, fmt.Errorf("not found: %v", err)
	}
	return idea, nil
}

func (a *App) UpdateIdea(id, title, description, category, priority, status string, tags []string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	if tags == nil {
		tags = []string{}
	}

	var existing Idea
	err = db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": id}).Decode(&existing)
	if err != nil {
		return Idea{}, fmt.Errorf("not found: %v", err)
	}

	existing.Title = title
	existing.Description = description
	existing.Category = category
	existing.Priority = priority
	existing.Status = status
	existing.Tags = tags
	existing.UpdatedAt = time.Now().Format(time.RFC3339)

	embedding, _ := generateEmbedding(title + " " + description)
	if embedding != nil {
		existing.Embedding = embedding
	}

	_, err = db.Collection("ideas").ReplaceOne(context.Background(), bson.M{"_id": id}, existing)
	if err != nil {
		return Idea{}, fmt.Errorf("update error: %v", err)
	}

	go a.LogActivity("updated", "idea", id, "Updated idea: "+title, title)
	return existing, nil
}

func (a *App) UpdateIdeaNotes(id, notes, estimatedEffort, potentialRevenue string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"notes":            notes,
			"estimatedEffort":  estimatedEffort,
			"potentialRevenue": potentialRevenue,
			"updatedAt":        now,
		}},
	)
	if err != nil {
		return Idea{}, fmt.Errorf("update error: %v", err)
	}

	var idea Idea
	db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": id}).Decode(&idea)
	return idea, nil
}

func (a *App) DeleteIdea(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("ideas").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}

	go a.LogActivity("deleted", "idea", id, "Deleted idea: "+id, "")
	return nil
}

// --- Research ---

func (a *App) AddResearch(ideaId, researchType, title, content, source string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	entry := IdeaResearchEntry{
		Type:      researchType,
		Title:     title,
		Content:   content,
		Source:    source,
		CreatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: getInstanceID(),
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{
			"$push": bson.M{"research": entry},
			"$set":  bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return Idea{}, fmt.Errorf("update error: %v", err)
	}

	var idea Idea
	db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": ideaId}).Decode(&idea)

	go a.LogActivity("added_research", "idea", ideaId, "Added research: "+title, content)
	return idea, nil
}

func (a *App) AddSuggestedTask(ideaId, title, description string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	task := IdeaSuggestedTask{
		Title:       title,
		Description: description,
		Status:      "pending",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{
			"$push": bson.M{"suggestedTasks": task},
			"$set":  bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return Idea{}, fmt.Errorf("update error: %v", err)
	}

	var idea Idea
	db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": ideaId}).Decode(&idea)

	go a.LogActivity("added_task", "idea", ideaId, "Added suggested task: "+title, description)
	return idea, nil
}

func (a *App) UpdateSuggestedTaskStatus(ideaId string, taskIndex int, status string) (Idea, error) {
	db, err := getDB()
	if err != nil {
		return Idea{}, fmt.Errorf("db error: %v", err)
	}

	field := fmt.Sprintf("suggestedTasks.%d.status", taskIndex)
	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{"$set": bson.M{field: status, "updatedAt": now}},
	)
	if err != nil {
		return Idea{}, fmt.Errorf("update error: %v", err)
	}

	var idea Idea
	db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": ideaId}).Decode(&idea)
	return idea, nil
}

// --- Convert to Project ---

func (a *App) ConvertIdeaToProject(ideaId string) (Project, error) {
	db, err := getDB()
	if err != nil {
		return Project{}, fmt.Errorf("db error: %v", err)
	}

	var idea Idea
	err = db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": ideaId}).Decode(&idea)
	if err != nil {
		return Project{}, fmt.Errorf("idea not found: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	project := Project{
		ID:          primitive.NewObjectID().Hex(),
		Name:        idea.Title,
		Description: idea.Description,
		Status:      "active",
		Priority:    idea.Priority,
		Ports:       []PortEntry{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = db.Collection("projects").InsertOne(context.Background(), project)
	if err != nil {
		return Project{}, fmt.Errorf("create project error: %v", err)
	}

	for _, st := range idea.SuggestedTasks {
		if st.Status == "accepted" {
			task := Task{
				ID:          primitive.NewObjectID().Hex(),
				ProjectID:   project.ID,
				Title:       st.Title,
				Description: st.Description,
				Status:      "todo",
				Priority:    "medium",
				Tags:        []string{},
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			db.Collection("tasks").InsertOne(context.Background(), task)
		}
	}

	db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{"$set": bson.M{
			"projectId": project.ID,
			"status":    "developing",
			"updatedAt": now,
		}},
	)

	go a.LogActivity("converted", "idea", ideaId, "Converted idea to project: "+idea.Title, project.ID)
	return project, nil
}

// --- Search ---

func (a *App) SearchIdeas(query string) ([]Idea, error) {
	if query == "" {
		return a.GetIdeas("")
	}

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	queryEmb, embErr := generateEmbedding(query)
	if embErr != nil || len(queryEmb) == 0 {
		return textSearchIdeas(db, query)
	}

	cursor, err := db.Collection("ideas").Find(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var allIdeas []Idea
	if err := cursor.All(context.Background(), &allIdeas); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	type scored struct {
		idea  Idea
		score float64
	}
	var results []scored
	for _, idea := range allIdeas {
		if len(idea.Embedding) > 0 {
			sim := cosineSimilarity(queryEmb, idea.Embedding)
			results = append(results, scored{idea: idea, score: sim})
		}
	}

	// Sort descending by score
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	limit := 20
	if len(results) < limit {
		limit = len(results)
	}

	var ideas []Idea
	for _, r := range results[:limit] {
		if r.score > 0.3 {
			ideas = append(ideas, r.idea)
		}
	}
	if ideas == nil {
		ideas = []Idea{}
	}
	return ideas, nil
}

func textSearchIdeas(db *mongo.Database, query string) ([]Idea, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"title": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := db.Collection("ideas").Find(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("search error: %v", err)
	}
	defer cursor.Close(context.Background())

	var ideas []Idea
	if err := cursor.All(context.Background(), &ideas); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if ideas == nil {
		ideas = []Idea{}
	}
	return ideas, nil
}

// --- Autonomous Research ---

type ideaChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (a *App) AutoResearchIdea(ideaId string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	var idea Idea
	err = db.Collection("ideas").FindOne(context.Background(), bson.M{"_id": ideaId}).Decode(&idea)
	if err != nil {
		return fmt.Errorf("idea not found: %v", err)
	}

	// Mark as researching
	now := time.Now().Format(time.RFC3339)
	db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{"$set": bson.M{"status": "researching", "updatedAt": now}},
	)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "idea:status-changed", ideaId, "researching")
	}

	prompt := fmt.Sprintf(`Research this product/project idea thoroughly:

Title: %s
Description: %s

Provide your analysis in these sections:

1. COMPETITORS: List 2-4 similar existing products/services with brief descriptions and pricing.

2. TECH STACK: Recommend a specific technology stack for building this, with reasoning.

3. COMPLEXITY: Estimate development complexity (simple/moderate/complex) and rough MVP timeline.

4. MARKET: Analyze market size and opportunity. Who are the target users?

5. DIFFERENTIATORS: Suggest 3-5 unique features that could make this stand out.

Be specific, concise, and actionable.`, idea.Title, idea.Description)

	messages := []ChatMessage{
		{Role: "system", Content: "You are a startup research analyst. Provide concise, actionable research about product ideas. Be specific about competitors, tech choices, and market opportunities."},
		{Role: "user", Content: prompt},
	}

	reqBody := chatRequest{
		Messages: messages,
		Stream:   false,
		User:     "nyx-idea-research",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("marshal error: %v", err)
	}

	req, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openclawToken)
	req.Header.Set("x-openclaw-agent-id", "nyx-idea-research")
	req.Header.Set("x-openclaw-session-key", "nyx-idea-research")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("api error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("read error: %v", err)
	}

	// Try non-streaming JSON response first, fallback to SSE
	var responseText string
	var completionResp ideaChatCompletionResponse
	if err := json.Unmarshal(respBytes, &completionResp); err == nil && len(completionResp.Choices) > 0 {
		responseText = completionResp.Choices[0].Message.Content
	} else {
		scanner := bufio.NewScanner(bytes.NewReader(respBytes))
		var sb strings.Builder
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
			if json.Unmarshal([]byte(data), &chunk) == nil {
				for _, choice := range chunk.Choices {
					sb.WriteString(choice.Delta.Content)
				}
			}
		}
		responseText = sb.String()
	}

	if responseText == "" {
		a.revertIdeaStatus(ideaId, idea.Status)
		return fmt.Errorf("empty response from AI")
	}

	// Parse into research entries
	now = time.Now().Format(time.RFC3339)
	entries := parseResearchSections(responseText, now, getInstanceID())

	for _, entry := range entries {
		db.Collection("ideas").UpdateOne(
			context.Background(),
			bson.M{"_id": ideaId},
			bson.M{"$push": bson.M{"research": entry}},
		)
	}

	// Regenerate embedding with enriched content
	enrichedText := idea.Title + " " + idea.Description + " " + responseText
	newEmbedding, _ := generateEmbedding(enrichedText)

	updateFields := bson.M{
		"status":           "researched",
		"lastResearchedAt": now,
		"updatedAt":        now,
	}
	if newEmbedding != nil {
		updateFields["embedding"] = newEmbedding
	}

	db.Collection("ideas").UpdateOne(
		context.Background(),
		bson.M{"_id": ideaId},
		bson.M{"$set": updateFields},
	)

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "idea:researched", ideaId)
	}

	summary := responseText
	if len(summary) > 500 {
		summary = summary[:500]
	}
	go a.LogActivity("researched", "idea", ideaId, "Auto-researched idea: "+idea.Title, summary)
	return nil
}

func (a *App) revertIdeaStatus(ideaId, status string) {
	db, _ := getDB()
	if db != nil {
		db.Collection("ideas").UpdateOne(
			context.Background(),
			bson.M{"_id": ideaId},
			bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now().Format(time.RFC3339)}},
		)
	}
}

func parseResearchSections(response, timestamp, instanceID string) []IdeaResearchEntry {
	type sectionDef struct {
		marker string
		rtype  string
		title  string
	}

	defs := []sectionDef{
		{"COMPETITORS", "competitor", "Competitor Analysis"},
		{"TECH STACK", "suggestion", "Recommended Tech Stack"},
		{"COMPLEXITY", "finding", "Complexity Assessment"},
		{"MARKET", "finding", "Market Analysis"},
		{"DIFFERENTIATORS", "suggestion", "Unique Differentiators"},
	}

	upper := strings.ToUpper(response)

	type foundSection struct {
		def   sectionDef
		start int
	}

	var found []foundSection
	for _, def := range defs {
		idx := strings.Index(upper, def.marker)
		if idx != -1 {
			found = append(found, foundSection{def: def, start: idx})
		}
	}

	if len(found) == 0 {
		return []IdeaResearchEntry{{
			Type:      "finding",
			Title:     "AI Research Analysis",
			Content:   response,
			Source:    "agent_analysis",
			CreatedAt: timestamp,
			CreatedBy: instanceID,
		}}
	}

	// Sort by position in text
	for i := 0; i < len(found); i++ {
		for j := i + 1; j < len(found); j++ {
			if found[j].start < found[i].start {
				found[i], found[j] = found[j], found[i]
			}
		}
	}

	var entries []IdeaResearchEntry
	for i, f := range found {
		endIdx := len(response)
		if i+1 < len(found) {
			endIdx = found[i+1].start
		}

		content := strings.TrimSpace(response[f.start:endIdx])
		// Skip past the marker line
		if nlIdx := strings.Index(content, "\n"); nlIdx != -1 {
			content = strings.TrimSpace(content[nlIdx:])
		}

		if content != "" {
			entries = append(entries, IdeaResearchEntry{
				Type:      f.def.rtype,
				Title:     f.def.title,
				Content:   content,
				Source:    "agent_analysis",
				CreatedAt: timestamp,
				CreatedBy: instanceID,
			})
		}
	}

	return entries
}
