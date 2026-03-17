package main

import (
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
)

// GenerateTicketsRequest is the input from the frontend
type GenerateTicketsRequest struct {
	ProjectID   string `json:"projectId"`
	Description string `json:"description"`
	CreateEpic  bool   `json:"createEpic"`
	Status      string `json:"status"`
}

// GenerateTicketsResult is the return value
type GenerateTicketsResult struct {
	Epic    *Epic    `json:"epic"`
	Tickets []Ticket `json:"tickets"`
}

// aiTicketResponse is the expected JSON shape from the AI
type aiTicketResponse struct {
	Epic *struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"epic"`
	Tickets []struct {
		Title              string   `json:"title"`
		Description        string   `json:"description"`
		Scope              string   `json:"scope"`
		AcceptanceCriteria []string `json:"acceptanceCriteria"`
		TechnicalNotes     string   `json:"technicalNotes"`
		Type               string   `json:"type"`
		Priority           string   `json:"priority"`
		Estimate           string   `json:"estimate"`
		StoryPoints        int      `json:"storyPoints"`
		Tags               []string `json:"tags"`
	} `json:"tickets"`
}

// non-streaming response shape from OpenClaw
type chatCompletion struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateTickets calls the AI to generate structured tickets from a feature description
func (a *App) GenerateTickets(req GenerateTicketsRequest) (GenerateTicketsResult, error) {
	runtime.EventsEmit(a.ctx, "tickets:generating", nil)

	// 1. Fetch project
	db, err := getDB()
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Database error")
		return GenerateTicketsResult{}, fmt.Errorf("db error: %v", err)
	}

	var proj Project
	err = db.Collection("projects").FindOne(context.Background(), bson.M{"_id": req.ProjectID}).Decode(&proj)
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Project not found")
		return GenerateTicketsResult{}, fmt.Errorf("project not found: %v", err)
	}

	// 2. Fetch existing tickets for context
	existingTickets, _ := a.GetTicketsByProject(req.ProjectID)
	var existingTitles []string
	for _, t := range existingTickets {
		existingTitles = append(existingTitles, fmt.Sprintf("- [%s] %s (%s)", t.Code, t.Title, t.Status))
	}

	// 3. Fetch existing epics for context
	existingEpics, _ := a.GetEpicsByProject(req.ProjectID)
	var existingEpicTitles []string
	for _, e := range existingEpics {
		existingEpicTitles = append(existingEpicTitles, fmt.Sprintf("- [%s] %s (%s)", e.Code, e.Title, e.Status))
	}

	// 4. Build system prompt
	systemPrompt := `You are a senior scrum master and technical project manager.
Generate well-structured tickets with clear acceptance criteria in Given/When/Then format.
Each ticket should be independently workable.
Include technical implementation notes relevant to the project's stack.
Estimate using S(1-2h), M(half day), L(1-2 days), XL(3-5 days).
Story points: S=1, M=3, L=5, XL=8.
Break large features into multiple focused tickets.
ONLY respond with valid JSON, no markdown, no explanation.

The JSON must follow this exact schema:
{
  "epic": {
    "title": "string",
    "description": "string"
  },
  "tickets": [
    {
      "title": "string",
      "description": "Detailed description with context",
      "scope": "What's in scope and what's not",
      "acceptanceCriteria": ["Given/When/Then format..."],
      "technicalNotes": "Implementation hints, relevant files, etc.",
      "type": "feature|bug|chore|spike",
      "priority": "critical|high|medium|low",
      "estimate": "S|M|L|XL",
      "storyPoints": 3,
      "tags": ["backend", "auth"]
    }
  ]
}`

	// 5. Build user prompt with project context
	var userPrompt strings.Builder
	userPrompt.WriteString(fmt.Sprintf("Project: %s\n", proj.Name))
	if proj.Description != "" {
		userPrompt.WriteString(fmt.Sprintf("Description: %s\n", proj.Description))
	}
	if proj.Stack != "" {
		userPrompt.WriteString(fmt.Sprintf("Tech Stack: %s\n", proj.Stack))
	}
	if len(existingTitles) > 0 {
		userPrompt.WriteString(fmt.Sprintf("\nExisting tickets:\n%s\n", strings.Join(existingTitles, "\n")))
	}
	if len(existingEpicTitles) > 0 {
		userPrompt.WriteString(fmt.Sprintf("\nExisting epics:\n%s\n", strings.Join(existingEpicTitles, "\n")))
	}
	userPrompt.WriteString(fmt.Sprintf("\n---\nFeature request:\n%s\n", req.Description))
	if req.CreateEpic {
		userPrompt.WriteString("\nPlease include an epic to group these tickets under.\n")
	} else {
		userPrompt.WriteString("\nSet the epic field to null — do not create an epic.\n")
	}

	// 6. Call OpenClaw API (non-streaming)
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt.String()},
	}

	reqBody := chatRequest{
		Messages: messages,
		Stream:   false,
		User:     openclawUser,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to build request")
		return GenerateTicketsResult{}, fmt.Errorf("marshal error: %v", err)
	}

	httpReq, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to create request")
		return GenerateTicketsResult{}, fmt.Errorf("request error: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+openclawToken)
	httpReq.Header.Set("x-openclaw-agent-id", openclawAgent)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "API request failed")
		return GenerateTicketsResult{}, fmt.Errorf("api error: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to read response")
		return GenerateTicketsResult{}, fmt.Errorf("read error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", fmt.Sprintf("API error %d", resp.StatusCode))
		return GenerateTicketsResult{}, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	// 7. Parse the API response to get the content
	var completion chatCompletion
	if err := json.Unmarshal(respBody, &completion); err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to parse API response")
		return GenerateTicketsResult{}, fmt.Errorf("unmarshal completion error: %v", err)
	}

	if len(completion.Choices) == 0 {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Empty response from AI")
		return GenerateTicketsResult{}, fmt.Errorf("no choices in response")
	}

	content := completion.Choices[0].Message.Content

	// Strip markdown code blocks if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	// 8. Parse AI JSON into our struct
	var aiResp aiTicketResponse
	if err := json.Unmarshal([]byte(content), &aiResp); err != nil {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "AI returned malformed JSON")
		return GenerateTicketsResult{}, fmt.Errorf("failed to parse AI response as JSON: %v\nRaw content: %s", err, content)
	}

	if len(aiResp.Tickets) == 0 {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "AI generated no tickets")
		return GenerateTicketsResult{}, fmt.Errorf("AI generated 0 tickets")
	}

	now := time.Now().Format(time.RFC3339)
	status := req.Status
	if status == "" {
		status = "draft"
	}

	result := GenerateTicketsResult{}

	// 9. Create epic if requested
	var epicID string
	if req.CreateEpic && aiResp.Epic != nil {
		epicCode, err := a.generateEpicCode(req.ProjectID, proj.Name)
		if err != nil {
			runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to generate epic code")
			return GenerateTicketsResult{}, fmt.Errorf("epic code error: %v", err)
		}

		epic := Epic{
			ID:          primitive.NewObjectID().Hex(),
			ProjectID:   req.ProjectID,
			Code:        epicCode,
			Title:       aiResp.Epic.Title,
			Description: aiResp.Epic.Description,
			Status:      "open",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		_, err = db.Collection("epics").InsertOne(context.Background(), epic)
		if err != nil {
			runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to save epic")
			return GenerateTicketsResult{}, fmt.Errorf("epic insert error: %v", err)
		}

		epicID = epic.ID
		result.Epic = &epic
		go a.LogActivity("created", "epic", epic.ID, "AI generated epic: "+epic.Code+" "+epic.Title, epic.Code)
	}

	// 10. Create tickets
	for _, at := range aiResp.Tickets {
		ticketCode, err := a.generateTicketCode(req.ProjectID, proj.Name)
		if err != nil {
			continue
		}

		ticketType := at.Type
		if ticketType == "" {
			ticketType = "feature"
		}
		priority := at.Priority
		if priority == "" {
			priority = "medium"
		}
		tags := at.Tags
		if tags == nil {
			tags = []string{}
		}
		ac := at.AcceptanceCriteria
		if ac == nil {
			ac = []string{}
		}

		ticket := Ticket{
			ID:                 primitive.NewObjectID().Hex(),
			ProjectID:          req.ProjectID,
			EpicID:             epicID,
			Code:               ticketCode,
			Title:              at.Title,
			Description:        at.Description,
			Scope:              at.Scope,
			AcceptanceCriteria: ac,
			TechnicalNotes:     at.TechnicalNotes,
			Type:               ticketType,
			Status:             status,
			Priority:           priority,
			Estimate:           at.Estimate,
			StoryPoints:        at.StoryPoints,
			Tags:               tags,
			Order:              0,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		_, err = db.Collection("tickets").InsertOne(context.Background(), ticket)
		if err != nil {
			continue
		}

		result.Tickets = append(result.Tickets, ticket)
		go a.LogActivity("created", "ticket", ticket.ID, "AI generated ticket: "+ticket.Code+" "+ticket.Title, ticket.Code)
	}

	if len(result.Tickets) == 0 {
		runtime.EventsEmit(a.ctx, "tickets:generate-error", "Failed to save any tickets")
		return GenerateTicketsResult{}, fmt.Errorf("failed to save any tickets")
	}

	// 11. Emit success event
	runtime.EventsEmit(a.ctx, "tickets:generated", result)
	go a.LogActivity("generated", "ticket", req.ProjectID, fmt.Sprintf("AI generated %d tickets for project %s", len(result.Tickets), proj.Name), "")

	return result, nil
}
