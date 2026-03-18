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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/bson"
)

// TicketAgent runs as a background goroutine, polling for "ready" tickets
// assigned to its projects and autonomously working them via AI.
type TicketAgent struct {
	app          *App
	running      bool
	mu           sync.Mutex
	done         chan struct{}
	agentID      string
	pollInterval time.Duration
	lastPoll     time.Time
	ticketsWorked int
}

// NewTicketAgent creates a new TicketAgent bound to the given App.
// The agent identifies itself using the machine's instance ID.
func NewTicketAgent(app *App) *TicketAgent {
	return &TicketAgent{
		app:          app,
		agentID:      getInstanceID(),
		pollInterval: 5 * time.Minute,
	}
}

// Start begins the background polling goroutine.
func (ta *TicketAgent) Start() {
	ta.mu.Lock()
	if ta.running {
		ta.mu.Unlock()
		return
	}
	ta.running = true
	ta.done = make(chan struct{})
	ta.mu.Unlock()

	log.Printf("[ticket-agent] Started (agentID=%s, interval=%v)", ta.agentID, ta.pollInterval)
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:started", map[string]interface{}{
		"agentId":      ta.agentID,
		"pollInterval": ta.pollInterval.String(),
	})
	go ta.app.LogActivity("started", "ticket_agent", ta.agentID, "Ticket agent started", "")

	go ta.pollLoop()
}

// Stop halts the background polling goroutine.
func (ta *TicketAgent) Stop() {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if !ta.running {
		return
	}

	close(ta.done)
	ta.running = false

	log.Printf("[ticket-agent] Stopped")
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:stopped", map[string]interface{}{
		"agentId":       ta.agentID,
		"ticketsWorked": ta.ticketsWorked,
	})
	go ta.app.LogActivity("stopped", "ticket_agent", ta.agentID, "Ticket agent stopped", "")
}

// IsRunning returns whether the agent is currently active.
func (ta *TicketAgent) IsRunning() bool {
	ta.mu.Lock()
	defer ta.mu.Unlock()
	return ta.running
}

// GetStatus returns the current status of the ticket agent.
func (ta *TicketAgent) GetStatus() map[string]interface{} {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	lastPollStr := ""
	if !ta.lastPoll.IsZero() {
		lastPollStr = ta.lastPoll.Format(time.RFC3339)
	}

	return map[string]interface{}{
		"running":       ta.running,
		"agentId":       ta.agentID,
		"pollInterval":  ta.pollInterval.String(),
		"lastPoll":      lastPollStr,
		"ticketsWorked": ta.ticketsWorked,
	}
}

// pollLoop is the main loop that periodically checks for ready tickets.
func (ta *TicketAgent) pollLoop() {
	// Run an initial poll immediately
	ta.pollReadyTickets()

	ticker := time.NewTicker(ta.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ta.done:
			return
		case <-ticker.C:
			ta.pollReadyTickets()
		}
	}
}

// pollReadyTickets queries for the agent's project assignments and looks
// for tickets with status "ready" in each assigned project.
func (ta *TicketAgent) pollReadyTickets() {
	ta.mu.Lock()
	ta.lastPoll = time.Now()
	ta.mu.Unlock()

	db, err := getDB()
	if err != nil {
		log.Printf("[ticket-agent] DB error during poll: %v", err)
		return
	}

	// Find projects this agent is assigned to
	cursor, err := db.Collection("project_assignments").Find(
		context.Background(),
		bson.M{"agentId": ta.agentID},
	)
	if err != nil {
		log.Printf("[ticket-agent] Failed to query assignments: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var assignments []ProjectAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		log.Printf("[ticket-agent] Failed to decode assignments: %v", err)
		return
	}

	if len(assignments) == 0 {
		log.Printf("[ticket-agent] No project assignments found for %s", ta.agentID)
		runtime.EventsEmit(ta.app.ctx, "ticket-agent:poll", map[string]interface{}{
			"readyTickets": 0,
			"projects":     0,
		})
		return
	}

	// Collect project IDs
	projectIDs := make([]string, len(assignments))
	for i, a := range assignments {
		projectIDs[i] = a.ProjectID
	}

	// Query for ready tickets across all assigned projects
	ticketCursor, err := db.Collection("tickets").Find(
		context.Background(),
		bson.M{
			"projectId": bson.M{"$in": projectIDs},
			"status":    "ready",
		},
	)
	if err != nil {
		log.Printf("[ticket-agent] Failed to query tickets: %v", err)
		return
	}
	defer ticketCursor.Close(context.Background())

	var readyTickets []Ticket
	if err := ticketCursor.All(context.Background(), &readyTickets); err != nil {
		log.Printf("[ticket-agent] Failed to decode tickets: %v", err)
		return
	}

	log.Printf("[ticket-agent] Poll complete: %d ready tickets across %d projects",
		len(readyTickets), len(assignments))
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:poll", map[string]interface{}{
		"readyTickets": len(readyTickets),
		"projects":     len(assignments),
	})

	// Claim and work the first ready ticket found
	if len(readyTickets) > 0 {
		ticket := readyTickets[0]
		if err := ta.claimTicket(ticket); err != nil {
			log.Printf("[ticket-agent] Failed to claim ticket %s: %v", ticket.Code, err)
			return
		}
		ta.workTicket(ticket)
	}
}

// claimTicket marks a ticket as in_progress and assigns it to this agent.
func (ta *TicketAgent) claimTicket(ticket Ticket) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("tickets").UpdateOne(
		context.Background(),
		bson.M{"_id": ticket.ID},
		bson.M{"$set": bson.M{
			"status":     "in_progress",
			"assignedTo": ta.agentID,
			"startedAt":  now,
			"updatedAt":  now,
		}},
	)
	if err != nil {
		return fmt.Errorf("update error: %v", err)
	}

	log.Printf("[ticket-agent] Claimed ticket %s: %s", ticket.Code, ticket.Title)
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:claimed", map[string]interface{}{
		"ticketId":   ticket.ID,
		"ticketCode": ticket.Code,
		"title":      ticket.Title,
		"agentId":    ta.agentID,
	})
	go ta.app.LogActivity("claimed", "ticket", ticket.ID,
		fmt.Sprintf("Ticket agent claimed %s: %s", ticket.Code, ticket.Title), ta.agentID)

	return nil
}

// workTicket performs the actual work on a claimed ticket:
// 1. Builds a prompt from ticket + project context
// 2. Calls the AI for an implementation plan
// 3. If a local repo exists, spawns a coding agent to implement changes
// 4. Updates the ticket with work notes and moves it to review
func (ta *TicketAgent) workTicket(ticket Ticket) {
	log.Printf("[ticket-agent] Working ticket %s: %s", ticket.Code, ticket.Title)
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:working", map[string]interface{}{
		"ticketId":   ticket.ID,
		"ticketCode": ticket.Code,
		"title":      ticket.Title,
	})

	db, err := getDB()
	if err != nil {
		log.Printf("[ticket-agent] DB error while working ticket: %v", err)
		return
	}

	// Look up the project for context
	var project Project
	err = db.Collection("projects").FindOne(
		context.Background(),
		bson.M{"_id": ticket.ProjectID},
	).Decode(&project)
	if err != nil {
		log.Printf("[ticket-agent] Failed to find project %s: %v", ticket.ProjectID, err)
		ta.moveTicketToReview(ticket, "Failed to find project context")
		return
	}

	// Build the work prompt
	workPrompt := buildTicketWorkPrompt(ticket, project)

	// Call OpenClaw API (non-streaming) for the implementation plan
	aiResponse, err := ta.callOpenClaw(workPrompt)
	if err != nil {
		log.Printf("[ticket-agent] AI call failed for %s: %v", ticket.Code, err)
		ta.moveTicketToReview(ticket, fmt.Sprintf("AI analysis failed: %v", err))
		return
	}

	workNotes := fmt.Sprintf("--- AI Implementation Plan (%s) ---\n%s",
		time.Now().Format(time.RFC3339), aiResponse)

	// Attempt to spawn a coding agent if a local repo exists
	codingOutput := ta.tryCodingAgent(ticket, project, workPrompt)
	if codingOutput != "" {
		workNotes += fmt.Sprintf("\n\n--- Coding Agent Output (%s) ---\n%s",
			time.Now().Format(time.RFC3339), codingOutput)
	}

	// Append work notes to the ticket's technical notes
	updatedNotes := ticket.TechnicalNotes
	if updatedNotes != "" {
		updatedNotes += "\n\n"
	}
	updatedNotes += workNotes

	ta.moveTicketToReview(ticket, updatedNotes)

	ta.mu.Lock()
	ta.ticketsWorked++
	ta.mu.Unlock()

	log.Printf("[ticket-agent] Completed ticket %s", ticket.Code)
	runtime.EventsEmit(ta.app.ctx, "ticket-agent:completed", map[string]interface{}{
		"ticketId":   ticket.ID,
		"ticketCode": ticket.Code,
		"title":      ticket.Title,
		"agentId":    ta.agentID,
	})
	go ta.app.LogActivity("completed", "ticket", ticket.ID,
		fmt.Sprintf("Ticket agent completed %s: %s", ticket.Code, ticket.Title), ta.agentID)
}

// moveTicketToReview updates the ticket's technical notes and moves it to review status.
func (ta *TicketAgent) moveTicketToReview(ticket Ticket, notes string) {
	db, err := getDB()
	if err != nil {
		log.Printf("[ticket-agent] DB error moving ticket to review: %v", err)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("tickets").UpdateOne(
		context.Background(),
		bson.M{"_id": ticket.ID},
		bson.M{"$set": bson.M{
			"status":         "review",
			"technicalNotes": notes,
			"updatedAt":      now,
		}},
	)
	if err != nil {
		log.Printf("[ticket-agent] Failed to move ticket %s to review: %v", ticket.Code, err)
	}
}

// callOpenClaw sends a non-streaming request to the OpenClaw API and returns the response text.
func (ta *TicketAgent) callOpenClaw(prompt string) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: "You are an autonomous software engineering agent. Provide detailed, actionable implementation plans."},
		{Role: "user", Content: prompt},
	}

	reqBody := chatRequest{
		Messages: messages,
		Stream:   false,
		User:     openclawUser,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %v", err)
	}

	httpReq, err := http.NewRequest("POST", openclawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("request creation error: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+openclawToken)
	httpReq.Header.Set("x-openclaw-agent-id", openclawAgent)
	httpReq.Header.Set("x-openclaw-session-key", "ticket-agent-"+ta.agentID)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("api request error: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var completion chatCompletion
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return "", fmt.Errorf("unmarshal error: %v", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := completion.Choices[0].Message.Content
	// Strip markdown code blocks if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			// Remove first and last lines (the ``` markers)
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	return content, nil
}

// tryCodingAgent attempts to spawn a Claude coding agent against the local repo
// for the given project. Returns the agent's output or empty string if not available.
func (ta *TicketAgent) tryCodingAgent(ticket Ticket, project Project, workPrompt string) string {
	if project.RepoURL == "" {
		log.Printf("[ticket-agent] No repoUrl for project %s, skipping coding agent", project.Name)
		return ""
	}

	// Determine local repo path: ~/Projects/{project-name}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[ticket-agent] Cannot determine home dir: %v", err)
		return ""
	}

	repoPath := filepath.Join(home, "Projects", project.Name)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		log.Printf("[ticket-agent] Local repo not found at %s, skipping coding agent", repoPath)
		return ""
	}

	log.Printf("[ticket-agent] Found local repo at %s, spawning coding agent", repoPath)

	// Create a branch for this ticket
	branchName := fmt.Sprintf("ticket/%s", strings.ToLower(ticket.Code))

	// Try to create and checkout the branch
	gitCheckout := exec.Command("git", "-C", repoPath, "checkout", "-b", branchName)
	if checkoutOutput, err := gitCheckout.CombinedOutput(); err != nil {
		log.Printf("[ticket-agent] Branch creation failed (may already exist): %s %v",
			string(checkoutOutput), err)
		// Try switching to existing branch
		gitSwitch := exec.Command("git", "-C", repoPath, "switch", branchName)
		if switchOutput, err := gitSwitch.CombinedOutput(); err != nil {
			log.Printf("[ticket-agent] Branch switch also failed: %s %v",
				string(switchOutput), err)
			return ""
		}
	}

	// Build the prompt for the coding agent
	agentPrompt := fmt.Sprintf(`You are working on ticket %s: %s

Project: %s
Stack: %s

%s

IMPORTANT: You are working in the repository at %s on branch %s.
Implement the changes described above. Commit your work when done.`,
		ticket.Code, ticket.Title,
		project.Name, project.Stack,
		workPrompt,
		repoPath, branchName)

	// Spawn the claude coding agent
	cmd := exec.Command("claude", "--print", "--permission-mode", "bypassPermissions", "-p", agentPrompt)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), "CLAUDE_CODE_ENTRYPOINT=ticket-agent")

	log.Printf("[ticket-agent] Running coding agent for %s in %s", ticket.Code, repoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[ticket-agent] Coding agent exited with error: %v", err)
		if len(output) > 0 {
			return fmt.Sprintf("Coding agent error: %v\nOutput:\n%s", err, string(output))
		}
		return fmt.Sprintf("Coding agent error: %v", err)
	}

	result := string(output)
	log.Printf("[ticket-agent] Coding agent completed for %s (%d bytes of output)",
		ticket.Code, len(result))

	return result
}

// --- App integration methods ---

// StartTicketAgent creates and starts the autonomous ticket agent.
func (a *App) StartTicketAgent() {
	if a.ticketAgent != nil && a.ticketAgent.IsRunning() {
		return
	}
	a.ticketAgent = NewTicketAgent(a)
	a.ticketAgent.Start()
}

// StopTicketAgent stops the autonomous ticket agent.
func (a *App) StopTicketAgent() {
	if a.ticketAgent != nil {
		a.ticketAgent.Stop()
	}
}

// GetTicketAgentStatus returns the current status of the ticket agent.
func (a *App) GetTicketAgentStatus() map[string]interface{} {
	if a.ticketAgent == nil {
		return map[string]interface{}{
			"running":       false,
			"agentId":       getInstanceID(),
			"pollInterval":  "5m0s",
			"lastPoll":      "",
			"ticketsWorked": 0,
		}
	}
	return a.ticketAgent.GetStatus()
}
