# Andrometry

**Personal Context Engine for AI-Agent Alignment**

A local-first, agent-native personal observability system that enables AI agents to know your current state, understand your behavior timeline, detect misalignment, and provide timely coaching.

## Architecture

```text
┌──────────────────────────────────────────────┐
│           Android Collector App              │
│  Kotlin · Room · WorkManager · Hilt         │
│  Screen · Usage · Notifications · Calendar   │
│  Location · Health · Focus · Device State    │
│                                              │
│  Inference: Context · Segments · Confidence  │
│  Storage: Events · Segments · Summaries      │
└──────────────────┬───────────────────────────┘
                   │ HTTPS POST, gzip, token auth
                   │ batched, idempotent, retry-safe
                   ▼
┌──────────────────────────────────────────────┐
│           Minimal Context Relay              │
│  Go · SQLite · Caddy · systemd              │
│  Auth · Batch Ingestion · Timeline API       │
│  Briefing · Summaries · Observations         │
│  Interventions · Feedback · Goals · Habits   │
│  Policy Governor · Audit Log                 │
└──────────────────┬───────────────────────────┘
                   │ HTTPS / MCP tools
                   ▼
┌──────────────────────────────────────────────┐
│              AI Agents                       │
│  Context Polling · Briefing · Timeline       │
│  Observations · Interventions · Outcomes     │
│  Habits · Goals · Alignment Scoring          │
└──────────────────────────────────────────────┘
```

## Project Structure

```text
andrometry/
├── docs/architecture/       # Architecture & alignment docs
├── schemas/                 # JSON Schema definitions
│   ├── events/              # Event envelope schema
│   ├── segments/            # Timeline segment schema
│   ├── context/             # Current context schema
│   ├── agent/               # Observations, interventions, feedback, goals, habits
│   └── summaries/           # Daily summary schema
├── server/                  # Go relay server
│   ├── cmd/andrometry/      # Main entry point
│   ├── internal/            # Auth, DB, handlers, middleware, models, governor, briefing, audit
│   ├── api/                 # OpenAPI spec
│   └── migrations/          # SQL migrations
├── android/                 # Android collector app
│   └── app/src/main/java/com/andrometry/
│       ├── core/            # Database, network, security, config
│       ├── collectors/      # Screen, usage, notifications, calendar, location, health
│       ├── inference/       # Classifier, segments, context, confidence, baselines
│       ├── summaries/       # Five-minute, hourly, daily, weekly
│       ├── sync/            # Queue, uploader, idempotency, retry
│       ├── agents/          # Observations, interventions, goals, habits, feedback
│       └── ui/              # Permissions, status, debug, inbox, privacy
├── agents/mcp/              # MCP server wrapping relay API as tools
├── scripts/                 # Synthetic event generator, utilities
└── test/                    # Synthetic data, golden tests
```

## Build Prerequisites

- **Server**: Go 1.22+
- **Android**: Android SDK (compileSdk 35), **Java 17** (Kotlin 2.1.0 compiler cannot parse Java 26 version strings)
  ```bash
  export JAVA_HOME=/usr/lib/jvm/java-17-openjdk
  ```
- **MCP Server**: Node.js 18+

## Quick Start

### 1. Start the Relay Server

```bash
cd server
# Bootstrap a device and agent token
go run ./cmd/andrometry --bootstrap-token=dev-token-123

# Or without bootstrap (create tokens manually)
go run ./cmd/andrometry --port=8420
```

### 2. Generate Synthetic Data

```bash
cd scripts
node generate.js --token=dev-token-123 --relay=http://localhost:8420 --date=2026-07-28

# Or output to file
node generate.js --output=test_day.json
```

### 3. Query Context (as an AI agent)

```bash
# Get current context
curl -H "Authorization: Bearer agent-dev-token-123" http://localhost:8420/v1/context/current

# Get briefing
curl -H "Authorization: Bearer agent-dev-token-123" http://localhost:8420/v1/briefing/current

# Get today's timeline
curl -H "Authorization: Bearer agent-dev-token-123" "http://localhost:8420/v1/timeline?from=2026-07-28T00:00:00Z&to=2026-07-28T23:59:59Z"
```

### 4. Run MCP Server (for AI agents)

```bash
cd agents/mcp
npm install
export ANDROMETRY_RELAY_URL=http://localhost:8420
export ANDROMETRY_AGENT_TOKEN=agent-dev-token-123
npm start
```

## API Endpoints

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/v1/health` | — | Health check |
| `POST` | `/v1/batch` | `device:ingest` | Ingest event batch from device |
| `GET` | `/v1/context/current` | `agent:context:read` | Current context (ETag support) |
| `GET` | `/v1/briefing/current` | `agent:briefing:read` | Agent briefing with risks/opportunities |
| `GET` | `/v1/timeline` | `agent:timeline:read` | Semantic timeline segments |
| `GET` | `/v1/events` | `agent:events:read` | Raw events (drill-down) |
| `GET` | `/v1/summary/daily` | `agent:summaries:read` | Daily summary |
| `GET` | `/v1/summary/weekly` | `agent:summaries:read` | Weekly summary |
| `POST` | `/v1/agent/observations` | `agent:observations:write` | Write agent observation |
| `POST` | `/v1/agent/interventions` | `agent:interventions:write` | Propose intervention |
| `POST` | `/v1/feedback` | `agent:feedback:read` | Submit user feedback |
| `POST` | `/v1/goals` | `agent:goals:write` | Create goal |
| `POST` | `/v1/habits` | `agent:habits:write` | Create habit |
| `POST` | `/v1/outcomes` | `agent:outcomes:write` | Record intervention outcome |
| `GET` | `/v1/audit` | `admin` | Audit log |

## MCP Tools (for AI agents)

| Tool | Description |
|------|-------------|
| `get_current_context` | Current state with confidence & data quality |
| `get_current_briefing` | High-level briefing with risks & opportunities |
| `get_timeline` | Semantic timeline segments |
| `get_focus_sessions` | Today's focus sessions |
| `get_place_visits` | Location place visits |
| `get_distraction_segments` | Distraction intervals |
| `get_daily_summary` | Rich daily summary with scores |
| `get_weekly_summary` | Weekly trends |
| `search_events` | Raw event drill-down |
| `add_agent_observation` | Write observation |
| `propose_intervention` | Propose nudge/suggestion |
| `record_outcome` | Record intervention outcome |
| `get_goals` / `create_goal` | Goal management |
| `get_habits` / `create_habit` | Habit tracking |

## Key Design Principles

1. **Local-first** — Phone is the primary sensing, storage, and inference node
2. **Semantic over raw** — Agents consume summaries, not raw telemetry
3. **Event-driven** — BroadcastReceivers + WorkManager, not continuous polling
4. **Batched sync** — Compressed, idempotent, retry-safe uploads
5. **Agent-native** — Context, briefing, timeline, and observation APIs
6. **Closed-loop** — Sense → Infer → Expose → Agent → Intervene → Feedback → Recalibrate
7. **Privacy by default** — Metadata only, no content, sensitive mode, kill switch

## Schema Versioning

All events use a stable envelope with `schema_version`, `event_version`, and `client_version`. Schemas are defined in `schemas/` and the API contract is documented in `server/api/openapi.yaml`.

## License

Private — Personal project.
