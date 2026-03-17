package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// getDBContext returns a summary of current database state for chat context injection
func (a *App) getDBContext() string {
	db, err := getDB()
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sb strings.Builder

	// Clients summary
	clientCursor, err := db.Collection("clients").Find(ctx, bson.M{}, options.Find().SetLimit(50))
	if err == nil {
		var clients []bson.M
		if clientCursor.All(ctx, &clients) == nil && len(clients) > 0 {
			sb.WriteString(fmt.Sprintf("## Clients (%d)\n", len(clients)))
			for _, c := range clients {
				name := getString(c, "name")
				email := getString(c, "contactEmail")
				sb.WriteString(fmt.Sprintf("- %s (ID: %v, Email: %s)\n", name, c["_id"], email))
			}
			sb.WriteString("\n")
		}
	}

	// Projects summary
	projCursor, err := db.Collection("projects").Find(ctx, bson.M{}, options.Find().SetLimit(50))
	if err == nil {
		var projects []bson.M
		if projCursor.All(ctx, &projects) == nil && len(projects) > 0 {
			sb.WriteString(fmt.Sprintf("## Projects (%d)\n", len(projects)))
			for _, p := range projects {
				name := getString(p, "name")
				status := getString(p, "status")
				stack := getString(p, "stack")
				sb.WriteString(fmt.Sprintf("- %s (Status: %s, Stack: %s, ID: %v)\n", name, status, stack, p["_id"]))
			}
			sb.WriteString("\n")
		}
	}

	// Tasks summary (count by status)
	taskCursor, err := db.Collection("tasks").Find(ctx, bson.M{}, options.Find().SetLimit(100))
	if err == nil {
		var tasks []bson.M
		if taskCursor.All(ctx, &tasks) == nil && len(tasks) > 0 {
			statusCount := map[string]int{}
			for _, t := range tasks {
				s := getString(t, "status")
				if s == "" {
					s = "unknown"
				}
				statusCount[s]++
			}
			sb.WriteString(fmt.Sprintf("## Tasks (%d total)\n", len(tasks)))
			for status, count := range statusCount {
				sb.WriteString(fmt.Sprintf("- %s: %d\n", status, count))
			}
			sb.WriteString("\n")
		}
	}

	// Ideas summary (with status counts and research info)
	ideaCursor, err := db.Collection("ideas").Find(ctx, bson.M{"status": bson.M{"$ne": "discarded"}}, options.Find().SetLimit(50))
	if err == nil {
		var ideas []bson.M
		if ideaCursor.All(ctx, &ideas) == nil && len(ideas) > 0 {
			ideaStatusCount := map[string]int{}
			for _, idea := range ideas {
				s := getString(idea, "status")
				if s == "" {
					s = "new"
				}
				ideaStatusCount[s]++
			}
			sb.WriteString(fmt.Sprintf("## Ideas (%d total)\n", len(ideas)))
			sb.WriteString("Status breakdown: ")
			first := true
			for status, count := range ideaStatusCount {
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s: %d", status, count))
				first = false
			}
			sb.WriteString("\n")
			for _, idea := range ideas {
				title := getString(idea, "title")
				status := getString(idea, "status")
				category := getString(idea, "category")
				priority := getString(idea, "priority")
				researchCount := 0
				if r, ok := idea["research"]; ok {
					if arr, ok := r.(primitive.A); ok {
						researchCount = len(arr)
					}
				}
				projectId := getString(idea, "projectId")
				extra := ""
				if researchCount > 0 {
					extra += fmt.Sprintf(", Research: %d entries", researchCount)
				}
				if projectId != "" {
					extra += fmt.Sprintf(", ProjectID: %s", projectId)
				}
				sb.WriteString(fmt.Sprintf("- %s (Status: %s, Category: %s, Priority: %s%s, ID: %v)\n", title, status, category, priority, extra, idea["_id"]))
			}
			sb.WriteString("\n")
		}
	}

	// Recent activity (last 5)
	actOpts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(5)
	actCursor, err := db.Collection("activity_log").Find(ctx, bson.M{}, actOpts)
	if err == nil {
		var activities []bson.M
		if actCursor.All(ctx, &activities) == nil && len(activities) > 0 {
			sb.WriteString("## Recent Activity\n")
			for _, act := range activities {
				summary := getString(act, "summary")
				instance := getString(act, "instanceId")
				ts := getString(act, "timestamp")
				sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", instance, summary, ts))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func getString(m bson.M, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
