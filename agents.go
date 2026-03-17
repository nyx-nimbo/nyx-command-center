package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Data Models ---

type Agent struct {
	ID           string   `json:"id" bson:"_id,omitempty"`
	AgentID      string   `json:"agentId" bson:"agentId"`
	Name         string   `json:"name" bson:"name"`
	Type         string   `json:"type" bson:"type"`     // "agent" | "user"
	Status       string   `json:"status" bson:"status"` // "online" | "offline" | "busy"
	Capabilities []string `json:"capabilities" bson:"capabilities"`
	LastSeen     string   `json:"lastSeen" bson:"lastSeen"`
	RegisteredAt string   `json:"registeredAt" bson:"registeredAt"`
}

type ProjectAssignment struct {
	ID         string `json:"id" bson:"_id,omitempty"`
	ProjectID  string `json:"projectId" bson:"projectId"`
	AgentID    string `json:"agentId" bson:"agentId"`
	Role       string `json:"role" bson:"role"` // "developer" | "reviewer" | "lead"
	AssignedAt string `json:"assignedAt" bson:"assignedAt"`
}

// --- Agent CRUD ---

// RegisterAgent upserts an agent: updates lastSeen+status if exists, creates if not
func (a *App) RegisterAgent(agentId, name, agentType string, capabilities []string) (Agent, error) {
	db, err := getDB()
	if err != nil {
		return Agent{}, fmt.Errorf("db error: %v", err)
	}

	if capabilities == nil {
		capabilities = []string{}
	}

	now := time.Now().Format(time.RFC3339)
	col := db.Collection("agents")

	// Try to find existing agent
	var existing Agent
	err = col.FindOne(context.Background(), bson.M{"agentId": agentId}).Decode(&existing)
	if err == nil {
		// Update existing
		_, err = col.UpdateOne(
			context.Background(),
			bson.M{"agentId": agentId},
			bson.M{"$set": bson.M{
				"name":         name,
				"type":         agentType,
				"status":       "online",
				"capabilities": capabilities,
				"lastSeen":     now,
			}},
		)
		if err != nil {
			return Agent{}, fmt.Errorf("update error: %v", err)
		}
		existing.Name = name
		existing.Type = agentType
		existing.Status = "online"
		existing.Capabilities = capabilities
		existing.LastSeen = now
		go a.LogActivity("registered", "agent", existing.ID, "Agent registered: "+name+" ("+agentId+")", agentId)
		return existing, nil
	}

	// Create new agent
	agent := Agent{
		ID:           primitive.NewObjectID().Hex(),
		AgentID:      agentId,
		Name:         name,
		Type:         agentType,
		Status:       "online",
		Capabilities: capabilities,
		LastSeen:     now,
		RegisteredAt: now,
	}

	_, err = col.InsertOne(context.Background(), agent)
	if err != nil {
		return Agent{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("registered", "agent", agent.ID, "Agent registered: "+name+" ("+agentId+")", agentId)
	return agent, nil
}

// GetAgents returns all registered agents
func (a *App) GetAgents() ([]Agent, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("agents").Find(
		context.Background(),
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var agents []Agent
	if err := cursor.All(context.Background(), &agents); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if agents == nil {
		agents = []Agent{}
	}
	return agents, nil
}

// GetAgent returns a single agent by agentId
func (a *App) GetAgent(agentId string) (Agent, error) {
	db, err := getDB()
	if err != nil {
		return Agent{}, fmt.Errorf("db error: %v", err)
	}

	var agent Agent
	err = db.Collection("agents").FindOne(context.Background(), bson.M{"agentId": agentId}).Decode(&agent)
	if err != nil {
		return Agent{}, fmt.Errorf("not found: %v", err)
	}
	return agent, nil
}

// UpdateAgentStatus updates an agent's online/offline/busy status
func (a *App) UpdateAgentStatus(agentId, status string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	result, err := db.Collection("agents").UpdateOne(
		context.Background(),
		bson.M{"agentId": agentId},
		bson.M{"$set": bson.M{"status": status, "lastSeen": now}},
	)
	if err != nil {
		return fmt.Errorf("update error: %v", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("agent not found: %s", agentId)
	}
	go a.LogActivity("status_changed", "agent", agentId, "Agent status changed to "+status+": "+agentId, status)
	return nil
}

// DeleteAgent removes an agent by mongo _id and cleans up assignments
func (a *App) DeleteAgent(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	// Get agent to find agentId for cleanup
	var agent Agent
	err = db.Collection("agents").FindOne(context.Background(), bson.M{"_id": id}).Decode(&agent)
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	_, err = db.Collection("agents").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}

	// Clean up project assignments
	db.Collection("project_assignments").DeleteMany(context.Background(), bson.M{"agentId": agent.AgentID})

	go a.LogActivity("deleted", "agent", id, "Deleted agent: "+agent.Name+" ("+agent.AgentID+")", agent.AgentID)
	return nil
}

// --- Project Assignment ---

// AssignAgentToProject creates an assignment. Validates agent and project exist. Prevents duplicates.
func (a *App) AssignAgentToProject(projectId, agentId, role string) (ProjectAssignment, error) {
	db, err := getDB()
	if err != nil {
		return ProjectAssignment{}, fmt.Errorf("db error: %v", err)
	}

	// Validate project exists
	var proj Project
	err = db.Collection("projects").FindOne(context.Background(), bson.M{"_id": projectId}).Decode(&proj)
	if err != nil {
		return ProjectAssignment{}, fmt.Errorf("project not found: %v", err)
	}

	// Validate agent exists
	var agent Agent
	err = db.Collection("agents").FindOne(context.Background(), bson.M{"agentId": agentId}).Decode(&agent)
	if err != nil {
		return ProjectAssignment{}, fmt.Errorf("agent not found: %v", err)
	}

	// Check for duplicate
	count, err := db.Collection("project_assignments").CountDocuments(
		context.Background(),
		bson.M{"projectId": projectId, "agentId": agentId},
	)
	if err != nil {
		return ProjectAssignment{}, fmt.Errorf("check error: %v", err)
	}
	if count > 0 {
		return ProjectAssignment{}, fmt.Errorf("agent %s is already assigned to this project", agentId)
	}

	now := time.Now().Format(time.RFC3339)
	assignment := ProjectAssignment{
		ID:         primitive.NewObjectID().Hex(),
		ProjectID:  projectId,
		AgentID:    agentId,
		Role:       role,
		AssignedAt: now,
	}

	_, err = db.Collection("project_assignments").InsertOne(context.Background(), assignment)
	if err != nil {
		return ProjectAssignment{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("assigned", "project_assignment", assignment.ID,
		"Assigned "+agent.Name+" ("+agentId+") to project "+proj.Name+" as "+role, projectId)
	return assignment, nil
}

// UnassignAgentFromProject removes an assignment
func (a *App) UnassignAgentFromProject(projectId, agentId string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	result, err := db.Collection("project_assignments").DeleteOne(
		context.Background(),
		bson.M{"projectId": projectId, "agentId": agentId},
	)
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("assignment not found")
	}
	go a.LogActivity("unassigned", "project_assignment", projectId+":"+agentId,
		"Unassigned agent "+agentId+" from project "+projectId, projectId)
	return nil
}

// GetProjectAssignments returns agents assigned to a project (joined with agent data)
func (a *App) GetProjectAssignments(projectId string) ([]Agent, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	// Get assignments for this project
	cursor, err := db.Collection("project_assignments").Find(
		context.Background(),
		bson.M{"projectId": projectId},
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var assignments []ProjectAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if len(assignments) == 0 {
		return []Agent{}, nil
	}

	// Collect agentIds
	agentIds := make([]string, len(assignments))
	for i, pa := range assignments {
		agentIds[i] = pa.AgentID
	}

	// Fetch agents
	agentCursor, err := db.Collection("agents").Find(
		context.Background(),
		bson.M{"agentId": bson.M{"$in": agentIds}},
	)
	if err != nil {
		return nil, fmt.Errorf("agent find error: %v", err)
	}
	defer agentCursor.Close(context.Background())

	var agents []Agent
	if err := agentCursor.All(context.Background(), &agents); err != nil {
		return nil, fmt.Errorf("agent decode error: %v", err)
	}
	if agents == nil {
		agents = []Agent{}
	}
	return agents, nil
}

// GetProjectAssignmentsWithRoles returns assignments with role info for a project
func (a *App) GetProjectAssignmentsWithRoles(projectId string) ([]ProjectAssignment, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("project_assignments").Find(
		context.Background(),
		bson.M{"projectId": projectId},
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var assignments []ProjectAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if assignments == nil {
		assignments = []ProjectAssignment{}
	}
	return assignments, nil
}

// GetAgentProjects returns projects an agent is assigned to
func (a *App) GetAgentProjects(agentId string) ([]Project, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	// Get assignments for this agent
	cursor, err := db.Collection("project_assignments").Find(
		context.Background(),
		bson.M{"agentId": agentId},
	)
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var assignments []ProjectAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if len(assignments) == 0 {
		return []Project{}, nil
	}

	// Collect projectIds
	projectIds := make([]string, len(assignments))
	for i, pa := range assignments {
		projectIds[i] = pa.ProjectID
	}

	// Fetch projects
	projCursor, err := db.Collection("projects").Find(
		context.Background(),
		bson.M{"_id": bson.M{"$in": projectIds}},
	)
	if err != nil {
		return nil, fmt.Errorf("project find error: %v", err)
	}
	defer projCursor.Close(context.Background())

	var projects []Project
	if err := projCursor.All(context.Background(), &projects); err != nil {
		return nil, fmt.Errorf("project decode error: %v", err)
	}
	if projects == nil {
		projects = []Project{}
	}
	return projects, nil
}

// IsAgentAssigned checks if an agent is assigned to a project
func (a *App) IsAgentAssigned(projectId, agentId string) (bool, error) {
	db, err := getDB()
	if err != nil {
		return false, fmt.Errorf("db error: %v", err)
	}

	count, err := db.Collection("project_assignments").CountDocuments(
		context.Background(),
		bson.M{"projectId": projectId, "agentId": agentId},
	)
	if err != nil {
		return false, fmt.Errorf("count error: %v", err)
	}
	return count > 0, nil
}
