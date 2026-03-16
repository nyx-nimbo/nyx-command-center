package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- MongoDB Singleton ---

var (
	dbOnce   sync.Once
	dbClient *mongo.Client
	nyxDB    *mongo.Database
	dbErr    error
)

func getDB() (*mongo.Database, error) {
	dbOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		dbClient, dbErr = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		if dbErr != nil {
			return
		}
		nyxDB = dbClient.Database("nyx")
	})
	if dbErr != nil {
		return nil, dbErr
	}
	return nyxDB, nil
}

// --- Data Models ---

type Client struct {
	ID           string `json:"id" bson:"_id,omitempty"`
	Name         string `json:"name" bson:"name"`
	ContactName  string `json:"contactName" bson:"contactName"`
	ContactEmail string `json:"contactEmail" bson:"contactEmail"`
	Phone        string `json:"phone" bson:"phone"`
	Notes        string `json:"notes" bson:"notes"`
	CreatedAt    string `json:"createdAt" bson:"createdAt"`
	UpdatedAt    string `json:"updatedAt" bson:"updatedAt"`
}

type BusinessUnit struct {
	ID        string `json:"id" bson:"_id,omitempty"`
	ClientID  string `json:"clientId" bson:"clientId"`
	Name      string `json:"name" bson:"name"`
	RFC       string `json:"rfc" bson:"rfc"`
	Address   string `json:"address" bson:"address"`
	Notes     string `json:"notes" bson:"notes"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
	UpdatedAt string `json:"updatedAt" bson:"updatedAt"`
}

type Project struct {
	ID             string `json:"id" bson:"_id,omitempty"`
	ClientID       string `json:"clientId" bson:"clientId"`
	BusinessUnitID string `json:"businessUnitId" bson:"businessUnitId"`
	Name           string `json:"name" bson:"name"`
	Description    string `json:"description" bson:"description"`
	Status         string `json:"status" bson:"status"`
	Stack          string `json:"stack" bson:"stack"`
	RepoURL        string `json:"repoUrl" bson:"repoUrl"`
	Priority       string `json:"priority" bson:"priority"`
	StartDate      string `json:"startDate" bson:"startDate"`
	DueDate        string `json:"dueDate" bson:"dueDate"`
	CreatedAt      string `json:"createdAt" bson:"createdAt"`
	UpdatedAt      string `json:"updatedAt" bson:"updatedAt"`
}

type Task struct {
	ID             string   `json:"id" bson:"_id,omitempty"`
	ProjectID      string   `json:"projectId" bson:"projectId"`
	Title          string   `json:"title" bson:"title"`
	Description    string   `json:"description" bson:"description"`
	Status         string   `json:"status" bson:"status"`
	Priority       string   `json:"priority" bson:"priority"`
	AssignedTo     string   `json:"assignedTo" bson:"assignedTo"`
	EstimatedHours float64  `json:"estimatedHours" bson:"estimatedHours"`
	Tags           []string `json:"tags" bson:"tags"`
	CreatedAt      string   `json:"createdAt" bson:"createdAt"`
	UpdatedAt      string   `json:"updatedAt" bson:"updatedAt"`
	CompletedAt    string   `json:"completedAt" bson:"completedAt"`
}

type ProjectStats struct {
	Todo       int `json:"todo"`
	InProgress int `json:"inProgress"`
	InReview   int `json:"inReview"`
	Done       int `json:"done"`
	Total      int `json:"total"`
}

// --- Client CRUD ---

func (a *App) CreateClient(client Client) (Client, error) {
	db, err := getDB()
	if err != nil {
		return Client{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	client.ID = primitive.NewObjectID().Hex()
	client.CreatedAt = now
	client.UpdatedAt = now

	_, err = db.Collection("clients").InsertOne(context.Background(), client)
	if err != nil {
		return Client{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "client", client.ID, "Created client: "+client.Name, client.Name)
	return client, nil
}

func (a *App) GetClients() ([]Client, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("clients").Find(context.Background(), bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var clients []Client
	if err := cursor.All(context.Background(), &clients); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if clients == nil {
		clients = []Client{}
	}
	return clients, nil
}

func (a *App) GetClient(id string) (Client, error) {
	db, err := getDB()
	if err != nil {
		return Client{}, fmt.Errorf("db error: %v", err)
	}

	var client Client
	err = db.Collection("clients").FindOne(context.Background(), bson.M{"_id": id}).Decode(&client)
	if err != nil {
		return Client{}, fmt.Errorf("not found: %v", err)
	}
	return client, nil
}

func (a *App) UpdateClient(client Client) (Client, error) {
	db, err := getDB()
	if err != nil {
		return Client{}, fmt.Errorf("db error: %v", err)
	}

	client.UpdatedAt = time.Now().Format(time.RFC3339)
	_, err = db.Collection("clients").ReplaceOne(context.Background(), bson.M{"_id": client.ID}, client)
	if err != nil {
		return Client{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "client", client.ID, "Updated client: "+client.Name, client.Name)
	return client, nil
}

func (a *App) DeleteClient(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("clients").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	// Cascade: delete business units and projects for this client
	db.Collection("business_units").DeleteMany(context.Background(), bson.M{"clientId": id})
	db.Collection("projects").DeleteMany(context.Background(), bson.M{"clientId": id})
	go a.LogActivity("deleted", "client", id, "Deleted client: "+id, "")
	return nil
}

// --- BusinessUnit CRUD ---

func (a *App) CreateBusinessUnit(bu BusinessUnit) (BusinessUnit, error) {
	db, err := getDB()
	if err != nil {
		return BusinessUnit{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	bu.ID = primitive.NewObjectID().Hex()
	bu.CreatedAt = now
	bu.UpdatedAt = now

	_, err = db.Collection("business_units").InsertOne(context.Background(), bu)
	if err != nil {
		return BusinessUnit{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "business_unit", bu.ID, "Created business unit: "+bu.Name, bu.Name)
	return bu, nil
}

func (a *App) GetBusinessUnits(clientId string) ([]BusinessUnit, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	cursor, err := db.Collection("business_units").Find(context.Background(), bson.M{"clientId": clientId}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var units []BusinessUnit
	if err := cursor.All(context.Background(), &units); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if units == nil {
		units = []BusinessUnit{}
	}
	return units, nil
}

func (a *App) UpdateBusinessUnit(bu BusinessUnit) (BusinessUnit, error) {
	db, err := getDB()
	if err != nil {
		return BusinessUnit{}, fmt.Errorf("db error: %v", err)
	}

	bu.UpdatedAt = time.Now().Format(time.RFC3339)
	_, err = db.Collection("business_units").ReplaceOne(context.Background(), bson.M{"_id": bu.ID}, bu)
	if err != nil {
		return BusinessUnit{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "business_unit", bu.ID, "Updated business unit: "+bu.Name, bu.Name)
	return bu, nil
}

func (a *App) DeleteBusinessUnit(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("business_units").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	// Cascade: delete projects for this business unit
	db.Collection("projects").DeleteMany(context.Background(), bson.M{"businessUnitId": id})
	go a.LogActivity("deleted", "business_unit", id, "Deleted business unit: "+id, "")
	return nil
}

// --- Project CRUD ---

func (a *App) CreateProject(project Project) (Project, error) {
	db, err := getDB()
	if err != nil {
		return Project{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	project.ID = primitive.NewObjectID().Hex()
	project.CreatedAt = now
	project.UpdatedAt = now
	if project.Status == "" {
		project.Status = "active"
	}
	if project.Priority == "" {
		project.Priority = "medium"
	}

	_, err = db.Collection("projects").InsertOne(context.Background(), project)
	if err != nil {
		return Project{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "project", project.ID, "Created project: "+project.Name, project.Name)
	return project, nil
}

func (a *App) GetProjects(clientId string, businessUnitId string) ([]Project, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	filter := bson.M{}
	if clientId != "" {
		filter["clientId"] = clientId
	}
	if businessUnitId != "" {
		filter["businessUnitId"] = businessUnitId
	}

	cursor, err := db.Collection("projects").Find(context.Background(), filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var projects []Project
	if err := cursor.All(context.Background(), &projects); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if projects == nil {
		projects = []Project{}
	}
	return projects, nil
}

func (a *App) GetProject(id string) (Project, error) {
	db, err := getDB()
	if err != nil {
		return Project{}, fmt.Errorf("db error: %v", err)
	}

	var project Project
	err = db.Collection("projects").FindOne(context.Background(), bson.M{"_id": id}).Decode(&project)
	if err != nil {
		return Project{}, fmt.Errorf("not found: %v", err)
	}
	return project, nil
}

func (a *App) UpdateProject(project Project) (Project, error) {
	db, err := getDB()
	if err != nil {
		return Project{}, fmt.Errorf("db error: %v", err)
	}

	project.UpdatedAt = time.Now().Format(time.RFC3339)
	_, err = db.Collection("projects").ReplaceOne(context.Background(), bson.M{"_id": project.ID}, project)
	if err != nil {
		return Project{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "project", project.ID, "Updated project: "+project.Name, project.Name)
	return project, nil
}

func (a *App) DeleteProject(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("projects").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	// Cascade: delete tasks for this project
	db.Collection("tasks").DeleteMany(context.Background(), bson.M{"projectId": id})
	go a.LogActivity("deleted", "project", id, "Deleted project: "+id, "")
	return nil
}

// --- Task CRUD ---

func (a *App) CreateTask(task Task) (Task, error) {
	db, err := getDB()
	if err != nil {
		return Task{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	task.ID = primitive.NewObjectID().Hex()
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Tags == nil {
		task.Tags = []string{}
	}

	_, err = db.Collection("tasks").InsertOne(context.Background(), task)
	if err != nil {
		return Task{}, fmt.Errorf("insert error: %v", err)
	}
	go a.LogActivity("created", "task", task.ID, "Created task: "+task.Title, task.Title)
	return task, nil
}

func (a *App) GetTasks(projectId string, statusFilter string) ([]Task, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	filter := bson.M{"projectId": projectId}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	cursor, err := db.Collection("tasks").Find(context.Background(), filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find error: %v", err)
	}
	defer cursor.Close(context.Background())

	var tasks []Task
	if err := cursor.All(context.Background(), &tasks); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if tasks == nil {
		tasks = []Task{}
	}
	return tasks, nil
}

func (a *App) UpdateTask(task Task) (Task, error) {
	db, err := getDB()
	if err != nil {
		return Task{}, fmt.Errorf("db error: %v", err)
	}

	task.UpdatedAt = time.Now().Format(time.RFC3339)
	if task.Status == "done" && task.CompletedAt == "" {
		task.CompletedAt = time.Now().Format(time.RFC3339)
	}
	if task.Tags == nil {
		task.Tags = []string{}
	}

	_, err = db.Collection("tasks").ReplaceOne(context.Background(), bson.M{"_id": task.ID}, task)
	if err != nil {
		return Task{}, fmt.Errorf("update error: %v", err)
	}
	go a.LogActivity("updated", "task", task.ID, "Updated task: "+task.Title, task.Title)
	return task, nil
}

func (a *App) DeleteTask(id string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = db.Collection("tasks").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}
	go a.LogActivity("deleted", "task", id, "Deleted task: "+id, "")
	return nil
}

func (a *App) ClaimTask(taskId string, assignedTo string) (Task, error) {
	db, err := getDB()
	if err != nil {
		return Task{}, fmt.Errorf("db error: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Collection("tasks").UpdateOne(
		context.Background(),
		bson.M{"_id": taskId},
		bson.M{"$set": bson.M{"assignedTo": assignedTo, "updatedAt": now}},
	)
	if err != nil {
		return Task{}, fmt.Errorf("claim error: %v", err)
	}

	var task Task
	err = db.Collection("tasks").FindOne(context.Background(), bson.M{"_id": taskId}).Decode(&task)
	if err != nil {
		return Task{}, fmt.Errorf("fetch error: %v", err)
	}
	return task, nil
}

// --- Stats ---

func (a *App) GetProjectStats(projectId string) (ProjectStats, error) {
	db, err := getDB()
	if err != nil {
		return ProjectStats{}, fmt.Errorf("db error: %v", err)
	}

	col := db.Collection("tasks")
	ctx := context.Background()
	filter := bson.M{"projectId": projectId}

	total, _ := col.CountDocuments(ctx, filter)

	todoFilter := bson.M{"projectId": projectId, "status": "todo"}
	todo, _ := col.CountDocuments(ctx, todoFilter)

	ipFilter := bson.M{"projectId": projectId, "status": "in_progress"}
	ip, _ := col.CountDocuments(ctx, ipFilter)

	irFilter := bson.M{"projectId": projectId, "status": "in_review"}
	ir, _ := col.CountDocuments(ctx, irFilter)

	doneFilter := bson.M{"projectId": projectId, "status": "done"}
	done, _ := col.CountDocuments(ctx, doneFilter)

	return ProjectStats{
		Todo:       int(todo),
		InProgress: int(ip),
		InReview:   int(ir),
		Done:       int(done),
		Total:      int(total),
	}, nil
}
