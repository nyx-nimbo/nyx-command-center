package main

import _ "embed"

//go:embed AGENT.md
var embeddedAgentManual string

// GetCapabilities returns the embedded agent manual describing
// everything this application can do. Any OpenClaw agent can call
// this method to discover available tools, APIs, and workflows.
func (a *App) GetCapabilities() string {
	return embeddedAgentManual
}

// GetCapabilitiesSummary returns a short summary of available features
func (a *App) GetCapabilitiesSummary() map[string]interface{} {
	return map[string]interface{}{
		"app":     "Nyx Command Center",
		"version": "0.1.0",
		"features": []string{
			"chat:multi-session streaming via OpenClaw",
			"email:gmail read/send/list via Google API",
			"calendar:events read/create/delete via Google API",
			"clients:CRUD for clients/business_units",
			"projects:CRUD with kanban board",
			"tasks:CRUD with claim/assign system",
			"health:auto-check and repair on startup",
			"auth:Google OAuth2 login",
		},
		"methods": map[string][]string{
			"chat":     {"StreamChat", "CreateChatSession", "ListChatSessions", "SwitchSession", "DeleteSession", "ClearChatHistory", "GetChatHistory"},
			"email":    {"GetEmails", "GetEmail", "SendEmail", "MarkAsRead"},
			"calendar": {"GetTodayEvents", "GetUpcomingEvents", "CreateEvent", "DeleteEvent"},
			"clients":  {"CreateClient", "GetClients", "GetClient", "UpdateClient", "DeleteClient"},
			"business": {"CreateBusinessUnit", "GetBusinessUnits", "UpdateBusinessUnit", "DeleteBusinessUnit"},
			"projects": {"CreateProject", "GetProjects", "UpdateProject", "DeleteProject", "GetProjectStats"},
			"tasks":    {"CreateTask", "GetTasks", "UpdateTask", "DeleteTask", "ClaimTask"},
			"auth":     {"CheckGoogleAuth", "StartGoogleLogin", "GetGoogleUserInfo", "LogoutGoogle"},
			"health":   {"CheckHealth"},
			"system":   {"GetAppInfo", "GetCapabilities", "GetCapabilitiesSummary"},
		},
	}
}
