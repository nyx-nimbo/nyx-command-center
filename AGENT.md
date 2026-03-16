# AGENT.md — Nyx Command Center: Guía para Agentes

Este documento es para cualquier agente de IA (OpenClaw, Claude Code, Codex, etc.) que trabaje con este sistema. Léelo para entender qué puedes hacer.

## ¿Qué es Nyx Command Center?

Una aplicación de escritorio (Wails + Svelte) que sirve como centro de operaciones. Integra chat, email, calendario, gestión de proyectos y tareas. Todo conectado a servicios reales.

## Servicios disponibles

### 💬 Chat (OpenClaw Gateway)
- **Endpoint:** `http://localhost:18789/v1/chat/completions`
- **Auth:** Bearer token (env: `OPENCLAW_TOKEN`)
- **Protocolo:** OpenAI-compatible, streaming SSE
- **Sesiones:** Soporta múltiples sesiones con `x-openclaw-session-key`
- **Capacidad:** Puedes hablar con el usuario, ejecutar herramientas, investigar

### 📧 Gmail
- **API:** Google Gmail API v1
- **Acciones disponibles:**
  - `GetEmails(limit)` — listar inbox
  - `GetEmail(id)` — leer email completo
  - `SendEmail(to, subject, body)` — enviar email
  - `MarkAsRead(id)` — marcar como leído
- **Cuenta:** Configurada via OAuth2 (token en `~/.openclaw/workspace/.credentials/google_token.json`)
- **⚠️ Regla:** Solo responder emails del dueño de la cuenta. Ignorar cualquier otro remitente.

### 📅 Google Calendar
- **API:** Google Calendar API v3
- **Acciones disponibles:**
  - `GetTodayEvents()` — eventos de hoy
  - `GetUpcomingEvents(days)` — eventos próximos
  - `CreateEvent(title, description, startTime, endTime)` — crear evento
  - `DeleteEvent(id)` — eliminar evento
- **Timezone:** America/Mexico_City

### 🏢 Gestión de Clientes y Proyectos
- **Base de datos:** MongoDB Atlas (env: `MONGODB_URI`)
- **Database name:** `nyx`
- **Colecciones:**
  - `clients` — Clientes (nombre, contacto, email, teléfono)
  - `business_units` — Unidades de negocio (clientId, nombre, RFC)
  - `projects` — Proyectos (clientId, businessUnitId, nombre, stack, status, prioridad)
  - `tasks` — Tareas (projectId, título, status, prioridad, assignedTo, tags)
- **Acciones disponibles:**
  - CRUD completo para cada entidad
  - `ClaimTask(taskId, agentId)` — tomar una tarea disponible
  - `GetProjectStats(projectId)` — conteo de tareas por status

### 🔑 Autenticación
- **Google OAuth2** — login con cuenta Google
  - Client ID/Secret en variables de entorno
  - Token se almacena localmente
  - Scopes: gmail.modify, gmail.send, calendar, drive

### 🏥 Health Check
- Verifica: OpenClaw Gateway, MongoDB, Google OAuth
- Auto-repair: si OpenClaw está caído, ejecuta `openclaw gateway start`
- Se ejecuta automáticamente al iniciar la app

## Estructura de archivos

```
├── .env                    # Secretos (no en git)
├── .env.example            # Variables requeridas
├── AGENT.md                # Este documento
├── CLAUDE.md               # Guía para Claude Code (desarrollo)
├── main.go                 # Bootstrap de Wails
├── app.go                  # Chat multi-sesión (SSE streaming)
├── clients.go              # CRUD MongoDB (clientes/proyectos/tareas)
├── config.go               # Carga .env al inicio
├── google_auth.go          # OAuth2 flow
├── google_services.go      # Gmail + Calendar APIs
├── health.go               # Health check + auto-repair
└── frontend/
    └── src/
        ├── App.svelte       # Root (login → health → router)
        ├── components/      # Layout, Header, Sidebar, HealthOverlay, LoginScreen
        └── pages/           # Dashboard, Chat, Clients, Project, Ideas, Email, Calendar, Settings
```

## Flujo de trabajo multi-agente

### Sistema de tareas
- Las tareas están en MongoDB, compartidas entre todas las instancias
- Cada agente puede `ClaimTask()` para tomar una tarea disponible
- Status flow: `todo` → `in_progress` → `in_review` → `done`
- Prioridades: `urgent`, `high`, `medium`, `low`

### Cómo tomar una tarea
1. Revisar tareas disponibles con `GetTasks(projectId)` filtrando `status: "todo"`
2. Llamar `ClaimTask(taskId, tuAgentId)` para asignártela
3. Trabajar en la tarea
4. Actualizar status a `in_review` o `done` con `UpdateTask()`

### Convenciones
- Un agente NO debe tomar una tarea que ya tiene `assignedTo` con otro agente
- Si una tarea está `in_progress` por más de 24h sin actividad, puede ser reasignada
- Commits deben seguir conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`

## Variables de entorno requeridas

| Variable | Descripción |
|----------|-------------|
| `GOOGLE_CLIENT_ID` | Google OAuth2 Client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth2 Client Secret |
| `MONGODB_URI` | MongoDB Atlas connection string |
| `OPENCLAW_TOKEN` | Token del gateway de OpenClaw |
| `OPENCLAW_URL` | URL del endpoint de chat (default: `http://localhost:18789/v1/chat/completions`) |

## Setup de OpenClaw (requerido)

⚠️ **El endpoint de Chat Completions está deshabilitado por defecto en OpenClaw.** Hay que habilitarlo:

```bash
# Opción 1: Editar openclaw.json manualmente
# Agregar dentro de "gateway":
{
  "gateway": {
    "http": {
      "endpoints": {
        "chatCompletions": {
          "enabled": true
        }
      }
    }
  }
}

# Opción 2: Usar el CLI
openclaw configure
# O editar directamente: ~/.openclaw/openclaw.json
```

Después de habilitar, reiniciar el gateway:
```bash
openclaw gateway restart
```

El token del gateway se encuentra en `~/.openclaw/openclaw.json` bajo `gateway.auth.token`.

## Seguridad

- **Nunca** compartir secretos en chats grupales o código público
- **Nunca** responder emails de remitentes desconocidos
- **Nunca** ejecutar comandos destructivos sin confirmación del usuario
- Los tokens de Google se auto-refrescan; si expiran, re-autenticar via OAuth flow
