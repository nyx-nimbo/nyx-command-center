package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Data Models ---

type Ticket struct {
	ID                 string   `json:"id" bson:"_id,omitempty"`
	ProjectID          string   `json:"projectId" bson:"projectId"`
	EpicID             string   `json:"epicId" bson:"epicId"`
	Code               string   `json:"code" bson:"code"`
	Title              string   `json:"title" bson:"title"`
	Description        string   `json:"description" bson:"description"`
	Scope              string   `json:"scope" bson:"scope"`
	AcceptanceCriteria []string `json:"acceptanceCriteria" bson:"acceptanceCriteria"`
	TechnicalNotes     string   `json:"technicalNotes" bson:"technicalNotes"`
	Type               string   `json:"type" bson:"type"`
	Status             string   `json:"status" bson:"status"`
	Priority           string   `json:"priority" bson:"priority"`
	Estimate           string   `json:"estimate" bson:"estimate"`
	StoryPoints        int      `json:"storyPoints" bson:"storyPoints"`
	AssignedTo         string   `json:"assignedTo" bson:"assignedTo"`
	Tags               []string `json:"tags" bson:"tags"`
	Order              int      `json:"order" bson:"order"`
	CreatedAt          string   `json:"createdAt" bson:"createdAt"`
	UpdatedAt          string   `json:"updatedAt" bson:"updatedAt"`
	StartedAt          string   `json:"startedAt" bson:"startedAt"`
	CompletedAt        string   `json:"completedAt" bson:"completedAt"`
}

type Epic struct {
	ID          string `json:"id" bson:"_id,omitempty"`
	ProjectID   string `json:"projectId" bson:"projectId"`
	Code        string `json:"code" bson:"code"`
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	Status      string `json:"status" bson:"status"`
	CreatedAt   string `json:"createdAt" bson:"createdAt"`
	UpdatedAt   string `json:"updatedAt" bson:"updatedAt"`
}

type TicketStats struct {
	Draft      int `json:"draft"`
	Ready      int `json:"ready"`
	InProgress int `json:"inProgress"`
	Review     int `json:"review"`
	Done       int `json:"done"`
	Total      int `json:"total"`
}

// --- Ticket Code Generation ---

var nonAlpha = regexp.MustCompile(`[^A-Z]`)

func generateProjectPrefix(projectName string) string {
	upper := strings.ToUpper(projectName)
	upper = nonAlpha.ReplaceAllString(upper, "")
	if len(upper) > 4 {
		upper = upper[:4]
	}
	if upper == "" {
		upper = "TKT"
	}
	return upper
}

func (a *App) generateTicketCode(projectID string, projectName string) (string, error) {
	db, err := getDB()
	if err != nil {
		return "", err
	}

	prefix := generateProjectPrefix(projectName)

	// Find the highest existing code number for this project
	cursor, err := db.Collection("tickets").Find(
		context.Background(),
		bson.M{"projectId": projectID},
		options.Find().SetProjection(bson.M{"code": 1}),
	)
	if err != nil {
		return prefix + "-001", nil
	}
	defer cursor.Close(context.Background())

	maxNum := 0
	for cursor.Next(context.Background()) {
		var t struct {
			Code string `bson:"code"`
		}
		if err := cursor.Decode(&t); err != nil {
			continue
		}
		parts := strings.Split(t.Code, "-")
		if len(parts) == 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}

	return fmt.Sprintf("%s-%03d", prefix, maxNum+1), nil
}

func (a *App) generateEpicCode(projectID string, projectName string) (string, error) {
	db, err := getDB()
	if err != nil {
		return "", err
	}

	prefix := generateProjectPrefix(projectName) + "E"

	cursor, err := db.Collection("epics").Find(
		context.Background(),
		bson.M{"projectId": projectID},
		options.Find().SetProjection(bson.M{"code": 1}),
	)
	if err != nil {
		return prefix + "-001", nil
	}
	defer cursor.Close(context.Background())

	maxNum := 0
	for cursor.Next(context.Background()) {
		var e struct {
			Code string `bson:"code"`
		}
		if err := cursor.Decode(&e); err != nil {
			continue
		}
		parts := strings.Split(e.Code, "-")
		if len(parts) == 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}

	return fmt.Sprintf("%s-%03d", prefix, maxNum+1), nil
}

// --- Ticket CRUD ---

func (a *App) CreateTicket(ticket Ticket) (Ticket, error) {
	db, err := getDB()
	if err != nil {
		return Ticket{}, fmt.Errorf("db error: %v", err)
	}

	// Lookup project name for code generation
	var proj Project
	err = db.Collection("projects").FindOne(context.Background(), bson.M{"_id": ticket.ProjectID}).Decode(&proj)
	if err != nil {
		return Ticket{}, fmt.Errorf("project not found: %v", err)
	}

	code, err := a.generateTicketCode(ticket.ProjectID, proj.Name)
	if err != nil {
		return Ticket{}, fmt.Errorf("code generation error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	ticket.ID = primitive.NewObjectID().Hex()
	ticket.Code = code
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	if ticket.Status == "" {
		ticket.Status = "draft"
	}
	if ticket.Priority == "" {
		ticket.Priority = "medium"
	}
	if ticket.Type == "" {
		ticket.Type = "feature"
	}
	if ticket.Tags == nil {
		ticket.Tags = []string{}
	}
	if ticket.AcceptanceCriteria == nil {
		ticket.AcceptanceCriteria = []string{}
	}

	_, err = db.Collection("tickets").InsertOne(context.Background(), ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "ticket", ticket.ID, "Created ticket: "+ticket.Code+" "+ticket.Title, ticket.Code)
	return ticket, nil
}

func (a *App) GetTicket(id string) (Ticket, error) {
	db, err := getDB()
	if err != nil {
		return Ticket{}, fmt.Errorf("db error: %v", err)
	}

	var ticket Ticket
	err = db.Collection("tickets").FindOne(context.Background(), bson.M{"_id": id}).Decode(&ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("not found: %v", err)
	}
	return ticket, nil
}

func (a *App) UpdateTicket(ticket Ticket) (Ticket, error) {
	db, err := getDB()
	if err != nil {
		return Ticket{}, fmt.Errorf("db error: %v", err)
	}

	ticket.UpdatedAt = time.Now().Format(time.RFC3339)
	if ticket.Tags == nil {
		ticket.Tags = []string{}
	}
	if ticket.AcceptanceCriteria == nil {
		ticket.AcceptanceCriteria = []string{}
	}

	_, err = db.Collection("tickets").ReplaceOne(context.Background(), bson.M{"_id": ticket.ID}, ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "ticket", ticket.ID, "Updated ticket: "+ticket.Code+" "+ticket.Title, ticket.Code)
	return ticket, nil
}

func (a *App) DeleteTicket(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("tickets").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	go a.LogActivity("deleted", "ticket", id, "Deleted ticket: "+id, "")
	return nil
}

func (a *App) GetTicketsByProject(projectId string) ([]Ticket, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("tickets").Find(
		context.Background(),
		bson.M{"projectId": projectId},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var tickets []Ticket
	if err := cursor.All(context.Background(), &tickets); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if tickets == nil {
		tickets = []Ticket{}
	}
	return tickets, nil
}

func (a *App) GetTicketsByStatus(projectId, status string) ([]Ticket, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	filter := bson.M{"projectId": projectId, "status": status}
	cursor, err := db.Collection("tickets").Find(
		context.Background(),
		filter,
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var tickets []Ticket
	if err := cursor.All(context.Background(), &tickets); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if tickets == nil {
		tickets = []Ticket{}
	}
	return tickets, nil
}

func (a *App) MoveTicket(ticketId, newStatus string) (Ticket, error) {
	db, err := getDB()
	if err != nil {
		return Ticket{}, fmt.Errorf("db error: %v", err)
	}

	// When moving to in_progress, validate agent assignment if ticket has assignee
	if newStatus == "in_progress" {
		var t Ticket
		err = db.Collection("tickets").FindOne(context.Background(), bson.M{"_id": ticketId}).Decode(&t)
		if err != nil {
			return Ticket{}, fmt.Errorf("ticket not found: %v", err)
		}
		if t.AssignedTo != "" {
			assigned, err := a.IsAgentAssigned(t.ProjectID, t.AssignedTo)
			if err != nil {
				return Ticket{}, fmt.Errorf("assignment check error: %v", err)
			}
			if !assigned {
				return Ticket{}, fmt.Errorf("agent not assigned to this project")
			}
		}
	}

	now := time.Now().Format(time.RFC3339)
	update := bson.M{
		"$set": bson.M{
			"status":    newStatus,
			"updatedAt": now,
		},
	}

	if newStatus == "in_progress" {
		update["$set"].(bson.M)["startedAt"] = now
	}
	if newStatus == "done" {
		update["$set"].(bson.M)["completedAt"] = now
	}

	_, err = db.Collection("tickets").UpdateOne(
		context.Background(),
		bson.M{"_id": ticketId},
		update,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("move error: %v", err)
	}

	var ticket Ticket
	err = db.Collection("tickets").FindOne(context.Background(), bson.M{"_id": ticketId}).Decode(&ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("fetch error: %v", err)
	}
	go a.LogActivity("moved", "ticket", ticket.ID, "Moved ticket "+ticket.Code+" to "+newStatus, ticket.Code)
	return ticket, nil
}

// AssignTicket assigns an agent to a ticket, validating project assignment first
func (a *App) AssignTicket(ticketId, agentId string) (Ticket, error) {
	db, err := getDB()
	if err != nil {
		return Ticket{}, fmt.Errorf("db error: %v", err)
	}

	// Fetch the ticket
	var ticket Ticket
	err = db.Collection("tickets").FindOne(context.Background(), bson.M{"_id": ticketId}).Decode(&ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("ticket not found: %v", err)
	}

	// Validate agent is assigned to the project
	assigned, err := a.IsAgentAssigned(ticket.ProjectID, agentId)
	if err != nil {
		return Ticket{}, fmt.Errorf("assignment check error: %v", err)
	}
	if !assigned {
		return Ticket{}, fmt.Errorf("agent not assigned to this project")
	}

	now := time.Now().Format(time.RFC3339)
	update := bson.M{
		"$set": bson.M{
			"assignedTo": agentId,
			"updatedAt":  now,
		},
	}

	// Auto-move from ready to in_progress
	if ticket.Status == "ready" {
		update["$set"].(bson.M)["status"] = "in_progress"
		update["$set"].(bson.M)["startedAt"] = now
	}

	_, err = db.Collection("tickets").UpdateOne(
		context.Background(),
		bson.M{"_id": ticketId},
		update,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("update error: %v", err)
	}

	// Fetch updated ticket
	err = db.Collection("tickets").FindOne(context.Background(), bson.M{"_id": ticketId}).Decode(&ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("fetch error: %v", err)
	}
	go a.LogActivity("assigned", "ticket", ticket.ID, "Assigned ticket "+ticket.Code+" to "+agentId, agentId)
	return ticket, nil
}

func (a *App) BulkUpdateTicketStatus(ticketIds []string, status string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("tickets").UpdateMany(
		context.Background(),
		bson.M{"_id": bson.M{"$in": ticketIds}},
		bson.M{"$set": bson.M{"status": status, "updatedAt": now}},
	)
	if err != nil {
		return fmt.Errorf("bulk update error: %v", err)
	}
	go a.LogActivity("bulk_updated", "ticket", strings.Join(ticketIds, ","), fmt.Sprintf("Bulk moved %d tickets to %s", len(ticketIds), status), "")
	return nil
}

func (a *App) ReorderTicket(ticketId string, newOrder int) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("tickets").UpdateOne(
		context.Background(),
		bson.M{"_id": ticketId},
		bson.M{"$set": bson.M{"order": newOrder, "updatedAt": time.Now().Format(time.RFC3339)}},
	)
	if err != nil {
		return fmt.Errorf("reorder error: %v", err)
	}
	return nil
}

func (a *App) GetTicketStats(projectId string) (TicketStats, error) {
	db, err := getDB()
	if err != nil {
		return TicketStats{}, fmt.Errorf("db error: %v", err)
	}

	col := db.Collection("tickets")
	ctx := context.Background()
	filter := bson.M{"projectId": projectId}

	total, _ := col.CountDocuments(ctx, filter)
	draft, _ := col.CountDocuments(ctx, bson.M{"projectId": projectId, "status": "draft"})
	ready, _ := col.CountDocuments(ctx, bson.M{"projectId": projectId, "status": "ready"})
	ip, _ := col.CountDocuments(ctx, bson.M{"projectId": projectId, "status": "in_progress"})
	review, _ := col.CountDocuments(ctx, bson.M{"projectId": projectId, "status": "review"})
	done, _ := col.CountDocuments(ctx, bson.M{"projectId": projectId, "status": "done"})

	return TicketStats{
		Draft:      int(draft),
		Ready:      int(ready),
		InProgress: int(ip),
		Review:     int(review),
		Done:       int(done),
		Total:      int(total),
	}, nil
}

// --- Epic CRUD ---

func (a *App) CreateEpic(epic Epic) (Epic, error) {
	db, err := getDB()
	if err != nil {
		return Epic{}, fmt.Errorf("db error: %v", err)
	}

	var proj Project
	err = db.Collection("projects").FindOne(context.Background(), bson.M{"_id": epic.ProjectID}).Decode(&proj)
	if err != nil {
		return Epic{}, fmt.Errorf("project not found: %v", err)
	}

	code, err := a.generateEpicCode(epic.ProjectID, proj.Name)
	if err != nil {
		return Epic{}, fmt.Errorf("code generation error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	epic.ID = primitive.NewObjectID().Hex()
	epic.Code = code
	epic.CreatedAt = now
	epic.UpdatedAt = now
	if epic.Status == "" {
		epic.Status = "open"
	}

	_, err = db.Collection("epics").InsertOne(context.Background(), epic)
	if err != nil {
		return Epic{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "epic", epic.ID, "Created epic: "+epic.Code+" "+epic.Title, epic.Code)
	return epic, nil
}

func (a *App) GetEpic(id string) (Epic, error) {
	db, err := getDB()
	if err != nil {
		return Epic{}, fmt.Errorf("db error: %v", err)
	}

	var epic Epic
	err = db.Collection("epics").FindOne(context.Background(), bson.M{"_id": id}).Decode(&epic)
	if err != nil {
		return Epic{}, fmt.Errorf("not found: %v", err)
	}
	return epic, nil
}

func (a *App) UpdateEpic(epic Epic) (Epic, error) {
	db, err := getDB()
	if err != nil {
		return Epic{}, fmt.Errorf("db error: %v", err)
	}

	epic.UpdatedAt = time.Now().Format(time.RFC3339)
	_, err = db.Collection("epics").ReplaceOne(context.Background(), bson.M{"_id": epic.ID}, epic)
	if err != nil {
		return Epic{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "epic", epic.ID, "Updated epic: "+epic.Code+" "+epic.Title, epic.Code)
	return epic, nil
}

func (a *App) DeleteEpic(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("epics").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	// Clear epicId from tickets that referenced this epic
	db.Collection("tickets").UpdateMany(
		context.Background(),
		bson.M{"epicId": id},
		bson.M{"$set": bson.M{"epicId": ""}},
	)
	go a.LogActivity("deleted", "epic", id, "Deleted epic: "+id, "")
	return nil
}

func (a *App) GetEpicsByProject(projectId string) ([]Epic, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("epics").Find(
		context.Background(),
		bson.M{"projectId": projectId},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var epics []Epic
	if err := cursor.All(context.Background(), &epics); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if epics == nil {
		epics = []Epic{}
	}
	return epics, nil
}

func (a *App) GetTicketsByEpic(epicId string) ([]Ticket, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("tickets").Find(
		context.Background(),
		bson.M{"epicId": epicId},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var tickets []Ticket
	if err := cursor.All(context.Background(), &tickets); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if tickets == nil {
		tickets = []Ticket{}
	}

	// Sort by order then by createdAt descending
	sort.SliceStable(tickets, func(i, j int) bool {
		if tickets[i].Order != tickets[j].Order {
			return tickets[i].Order < tickets[j].Order
		}
		return tickets[i].CreatedAt > tickets[j].CreatedAt
	})

	return tickets, nil
}
