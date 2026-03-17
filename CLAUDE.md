# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

Nyx Command Center is a **Wails v2** desktop application — Go backend + Svelte 3 frontend. It serves as a personal command center integrating chat (via OpenClaw), Google Workspace (Gmail, Calendar), MongoDB-backed client/project/task management, and a health monitoring system.

## Commands

```bash
# Development (hot-reload for both Go and Svelte)
wails dev

# Production build
wails build

# Frontend only (runs Vite dev server on port 5173)
cd frontend && npm run dev

# Install frontend deps
cd frontend && npm install
```

There are no tests or linting configured yet.

## Architecture

### Go Backend (root `.go` files)

All Go files are in `package main`. The `App` struct is the single Wails-bound object — every public method on it becomes callable from the frontend.

- **main.go** — Wails app bootstrap. Embeds `frontend/dist` via `//go:embed`. Binds `App` and starts health check on DOM ready.
- **app.go** — Core `App` struct, multi-session chat system. Streams responses from OpenClaw (localhost:18789) via SSE. Wails events: `chat:chunk`, `chat:done`, `chat:error`. Chat sessions persist in MongoDB (`chat_sessions` collection, upsert by session key as `_id`). History trimmed to last 50 messages on save. Loads from DB on startup, creates defaults (General, Ideas) if empty.
- **clients.go** — MongoDB CRUD for Clients, BusinessUnits, Projects, Tasks. Singleton DB connection via `sync.Once`. Database name: `nyx`. Cascade deletes (client → business_units/projects, project → tasks).
- **tickets.go** — MongoDB CRUD for Tickets and Epics. Collections: `tickets`, `epics`. Auto-generated ticket codes (e.g. NYX-001). Kanban statuses: draft → ready → in_progress → review → done. Bulk status updates, reordering, stats by status. `MoveTicket` validates agent project assignment when moving to `in_progress`. `AssignTicket` assigns an agent (must be on project team) and auto-moves ready→in_progress.
- **agents.go** — Agent registry and project assignment system. Collections: `agents`, `project_assignments`. Agents are upserted by `agentId` (unique key). CRUD for agents, project team assignments with role (developer/reviewer/lead), and validation helpers (`IsAgentAssigned`). Auto-registers Nyx instance on handshake.
- **ticket_generator.go** — AI-powered ticket generation via OpenClaw. Takes a project ID + feature description, calls the LLM to produce structured tickets (with epic, acceptance criteria, estimates, story points), saves to MongoDB. Emits Wails events: `tickets:generating`, `tickets:generated`, `tickets:generate-error`.
- **google_auth.go** — Google OAuth2 flow with local callback server on port 8099. Token stored at `~/.openclaw/workspace/.credentials/google_token.json`. Scopes: Gmail, Calendar, Drive, user info.
- **google_services.go** — Gmail (list/read/send/mark-read) and Calendar (today/upcoming events, create/delete). Timezone hardcoded to `America/Mexico_City`.
- **health.go** — Checks OpenClaw Gateway, MongoDB, Google OAuth. Auto-repair attempts `openclaw gateway start` for the gateway. Emits `health:*` events.
- **config.go** — `init()` loads `.env` file, sets `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `MONGODB_URI`, `OPENCLAW_TOKEN` from environment.

### Frontend (Svelte 3 + Vite)

- **frontend/src/App.svelte** — Root. Shows `LoginScreen` → `HealthOverlay` → `Layout` with `svelte-spa-router`.
- **frontend/src/components/** — Layout, Header, Sidebar, HealthOverlay, LoginScreen.
- **frontend/src/pages/** — Dashboard, Chat, Clients, Project (`:id` route), Ideas, Email, Calendar, Settings, TicketBoard (kanban component embedded in Project page), TeamPanel (agent team management embedded in Project page "Team" tab).
- **frontend/wailsjs/** — Auto-generated Wails bindings. **Do not edit** `wailsjs/go/` files directly; they regenerate from Go method signatures on `wails dev`/`wails build`.

### Go ↔ Frontend Communication

Two patterns:
1. **Direct calls**: Frontend calls `window['go']['main']['App']['MethodName'](args)` (or imports from `wailsjs/go/main/App`).
2. **Events**: Go emits via `runtime.EventsEmit(ctx, "event:name", data)`, frontend listens with `window.runtime.EventsOn("event:name", callback)`. Used for streaming chat and health updates.

## Environment Variables (.env)

Required in `.env` at project root:
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` — Google OAuth credentials
- `MONGODB_URI` — MongoDB Atlas connection string
- `OPENCLAW_TOKEN` — Token for OpenClaw gateway API

## External Services

- **OpenClaw Gateway** — localhost:18789, OpenAI-compatible chat completions API
- **MongoDB Atlas** — database `nyx`, collections: `clients`, `business_units`, `projects`, `tasks`, `tickets`, `epics`, `agents`, `project_assignments`, `chat_sessions`
- **Google APIs** — Gmail, Calendar, Drive, UserInfo via OAuth2
