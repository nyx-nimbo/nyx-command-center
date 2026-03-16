package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// ServiceStatus represents the health of a single service
type ServiceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`  // "checking", "ok", "repairing", "error"
	Message string `json:"message"`
}

// HealthReport is the overall health report
type HealthReport struct {
	Overall   string          `json:"overall"` // "ok", "degraded", "error"
	Services  []ServiceStatus `json:"services"`
	Timestamp string          `json:"timestamp"`
}

const (
	mongoURI        = "" // Set via MONGODB_URI env var
	gatewayCheckURL = "http://localhost:18789/"
)

// CheckHealth runs health checks against all services and emits events
func (a *App) CheckHealth() HealthReport {
	runtime.EventsEmit(a.ctx, "health:checking", nil)

	services := []ServiceStatus{
		{Name: "OpenClaw Gateway", Status: "checking", Message: "Checking..."},
		{Name: "MongoDB", Status: "checking", Message: "Checking..."},
		{Name: "Google OAuth", Status: "checking", Message: "Checking..."},
	}

	report := HealthReport{
		Overall:   "checking",
		Services:  services,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Emit initial checking state
	runtime.EventsEmit(a.ctx, "health:report", report)

	// Check OpenClaw Gateway
	services[0] = a.checkOpenClawGateway()
	report.Services = services
	runtime.EventsEmit(a.ctx, "health:report", report)

	// Check MongoDB
	services[1] = a.checkMongoDB()
	report.Services = services
	runtime.EventsEmit(a.ctx, "health:report", report)

	// Check Google OAuth
	services[2] = a.checkGoogleOAuth()
	report.Services = services

	// Determine overall status
	hasError := false
	hasRepairing := false
	for _, s := range services {
		if s.Status == "error" {
			hasError = true
		}
		if s.Status == "repairing" {
			hasRepairing = true
		}
	}

	if hasError {
		report.Overall = "error"
	} else if hasRepairing {
		report.Overall = "degraded"
	} else {
		report.Overall = "ok"
	}

	report.Timestamp = time.Now().Format(time.RFC3339)
	runtime.EventsEmit(a.ctx, "health:report", report)

	return report
}

func (a *App) checkOpenClawGateway() ServiceStatus {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(gatewayCheckURL)
	if err != nil {
		// Try auto-repair
		return a.AutoRepair("OpenClaw Gateway")
	}
	defer resp.Body.Close()

	return ServiceStatus{
		Name:    "OpenClaw Gateway",
		Status:  "ok",
		Message: fmt.Sprintf("Running (HTTP %d)", resp.StatusCode),
	}
}

func (a *App) checkMongoDB() ServiceStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return ServiceStatus{
			Name:    "MongoDB",
			Status:  "error",
			Message: fmt.Sprintf("Connection failed: %v", err),
		}
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return ServiceStatus{
			Name:    "MongoDB",
			Status:  "error",
			Message: fmt.Sprintf("Ping failed: %v", err),
		}
	}

	return ServiceStatus{
		Name:    "MongoDB",
		Status:  "ok",
		Message: "Connected to Atlas cluster",
	}
}

func (a *App) checkGoogleOAuth() ServiceStatus {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ServiceStatus{
			Name:    "Google OAuth",
			Status:  "error",
			Message: "Cannot determine home directory",
		}
	}

	credPath := filepath.Join(homeDir, ".openclaw", "workspace", ".credentials", "google_token.json")
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return ServiceStatus{
			Name:    "Google OAuth",
			Status:  "error",
			Message: "Token file not found at " + credPath,
		}
	}

	return ServiceStatus{
		Name:    "Google OAuth",
		Status:  "ok",
		Message: "Token file present",
	}
}

// AutoRepair attempts to repair a service
func (a *App) AutoRepair(service string) ServiceStatus {
	switch service {
	case "OpenClaw Gateway":
		runtime.EventsEmit(a.ctx, "health:repairing", service)

		cmd := exec.Command("openclaw", "gateway", "start")
		if err := cmd.Start(); err != nil {
			return ServiceStatus{
				Name:    "OpenClaw Gateway",
				Status:  "error",
				Message: fmt.Sprintf("Auto-start failed: %v", err),
			}
		}

		// Wait a moment for the gateway to start
		time.Sleep(3 * time.Second)

		// Verify it started
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(gatewayCheckURL)
		if err != nil {
			return ServiceStatus{
				Name:    "OpenClaw Gateway",
				Status:  "error",
				Message: "Started but not responding yet. May need more time.",
			}
		}
		defer resp.Body.Close()

		runtime.EventsEmit(a.ctx, "health:repaired", service)
		return ServiceStatus{
			Name:    "OpenClaw Gateway",
			Status:  "ok",
			Message: "Auto-started successfully",
		}

	default:
		return ServiceStatus{
			Name:    service,
			Status:  "error",
			Message: "No auto-repair available for this service",
		}
	}
}
