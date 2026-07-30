# Changelog

All notable changes to the Andrometry project are documented here.

## [0.1.0] — 2026-07-30

### Server — Full Architecture Implementation (§1–§42)

#### Endpoints (53 total)

**Health & Ingestion (§23)**
- `GET /v1/health` — Health check (no auth)
- `GET /v1/health/detailed` — Detailed health with DB stats
- `POST /v1/batch` — Batch ingestion from device (`device:ingest`)

**Context & Briefing (§17.3–§17.4)**
- `GET /v1/context/current` — Current context with ETag conditional (`agent:context:read`)
- `GET /v1/context/data-quality` — Data quality signals per §11.4 (`agent:context:read`)
- `GET /v1/briefing/current` — Agent briefing (`agent:briefing:read`)

**Timeline & Events (§17.5–§17.6)**
- `GET /v1/timeline` — Timeline segments with type filtering (`agent:timeline:read`)
- `GET /v1/events` — Event drill-down (`agent:events:read`)

**Summaries (§17.7–§17.8)**
- `GET /v1/summary/daily` — Daily summary (`agent:summaries:read`)
- `GET /v1/summary/weekly` — Weekly summary (`agent:summaries:read`)
- `GET /v1/summary/five_minute` — Five-minute summaries (`agent:summaries:read`)
- `GET /v1/summary/hourly` — Hourly summaries (`agent:summaries:read`)

**Agent Observations & Interventions (§17.9–§17.10)**
- `POST /v1/agent/observations` — Create observation (`agent:observations:write`)
- `GET /v1/agent/observations` — List observations (`agent:observations:write`)
- `POST /v1/agent/interventions` — Propose intervention (`agent:interventions:write`)
- `PATCH /v1/agent/interventions/{id}` — Update status (`agent:interventions:write`)
- `GET /v1/device/interventions` — Pending interventions (`device:read_interventions`)

**Feedback & Outcomes (§17.11, §35)**
- `POST /v1/feedback` — Submit feedback (`agent:feedback:read`)
- `GET /v1/feedback` — List feedback (`agent:feedback:read`)
- `POST /v1/outcomes` — Record outcome (`agent:outcomes:write`)
- `GET /v1/outcomes` — List outcomes (`agent:feedback:read`)
- `GET /v1/interventions/{id}/effectiveness` — Effectiveness score (`agent:interventions:write`)
- `GET /v1/agents/{agent_id}/effectiveness` — Agent effectiveness score (`agent:interventions:read`)

**Goals & Habits (§17.12–§17.13, §36–§39)**
- `GET /v1/goals` — List goals (`agent:goals:read`)
- `POST /v1/goals` — Create goal (`agent:goals:write`)
- `PATCH /v1/goals/{id}` — Update goal (`agent:goals:write`)
- `DELETE /v1/goals/{id}` — Delete goal (`agent:goals:write`)
- `GET /v1/goals/{id}/progress` — Evaluate progress (`agent:goals:read`)
- `GET /v1/habits` — List habits (`agent:habits:read`)
- `POST /v1/habits` — Create habit (`agent:habits:write`)
- `PATCH /v1/habits/{id}` — Update habit (`agent:habits:write`)
- `DELETE /v1/habits/{id}` — Delete habit (`agent:habits:write`)
- `POST /v1/habits/{id}/streak` — Record streak (`agent:habits:write`)
- `GET /v1/habits/{id}/evaluation` — Evaluate completion (`agent:habits:read`)
- `GET /v1/baselines` — List baselines (`agent:goals:read`)
- `POST /v1/baselines` — Create/upsert baseline (`agent:goals:write`)
- `DELETE /v1/baselines/{id}` — Delete baseline (`agent:goals:write`)

**User Corrections (§37)**
- `GET /v1/corrections` — List corrections (`device:write_feedback`)
- `POST /v1/corrections` — Create correction (`device:write_feedback`)
- `DELETE /v1/corrections/{id}` — Delete correction (`device:write_feedback`)

**Metrics & Analytics (§20, §39–§40)**
- `GET /v1/metrics/work-patterns` — Work pattern metrics (`agent:summaries:read`)
- `GET /v1/metrics/readiness` — Readiness/fatigue score (`agent:context:read`)
- `GET /v1/analytics/insights` — Cross-domain analytics (`agent:summaries:read`)

**Collector Health (§23.5)**
- `GET /v1/health/collectors` — Collector health & staleness (`agent:context:read`)

**SSE Stream (§21.6)**
- `GET /v1/stream` — Real-time context updates (`agent:context:read`)

**Data Export (§27)**
- `GET /v1/export/daily` — Daily data export (`admin`)
- `GET /v1/export/weekly` — Weekly data export (`admin`)

**Privacy & Security (§22, §28)**
- `GET /v1/privacy/sensitive-mode` — Get sensitive mode (`device:read_config`)
- `POST /v1/privacy/sensitive-mode` — Set sensitive mode (`device:read_config`)
- `POST /v1/privacy/wipe` — Wipe all data kill switch (`admin`)
- `POST /v1/auth/rotate` — Rotate token (`device:ingest`)
- `POST /v1/auth/revoke` — Revoke all tokens (`device:ingest`)

**Audit (§22.5)**
- `GET /v1/audit` — Audit log (`admin`)

#### MCP Tools (59 total)

Core tools (47): `get_current_context`, `get_current_briefing`, `get_timeline`, `get_focus_sessions`, `get_place_visits`, `get_distraction_segments`, `get_daily_summary`, `get_weekly_summary`, `search_events`, `add_agent_observation`, `get_agent_observations`, `propose_intervention`, `update_intervention_status`, `get_intervention_effectiveness`, `submit_feedback`, `get_feedback`, `record_outcome`, `get_intervention_outcomes`, `get_goals`, `create_goal`, `update_goal`, `delete_goal`, `get_habits`, `create_habit`, `update_habit`, `delete_habit`, `update_habit_streak`, `get_baselines`, `create_baseline`, `delete_baseline`, `get_corrections`, `submit_correction`, `get_sensitive_mode`, `set_sensitive_mode`, `get_server_health`, `get_audit_log`, `get_work_pattern_metrics`, `get_readiness_score`, `get_collector_health`, `evaluate_goal_progress`, `evaluate_habit_completion`, `get_analytics_insights`, `export_daily_data`, `export_weekly_data`, `wipe_all_data`, `get_five_minute_summaries`, `get_hourly_summaries`

Convenience tools (12, §18.1): `get_activity_segments`, `get_communication_segments`, `get_sleep_segments`, `get_app_usage`, `get_notification_load`, `get_health_summary_events`, `get_sleep_summary_events`, `get_calendar_context_events`, `get_data_quality`, `get_agent_effectiveness`, `get_habit_metrics`, `get_baseline_comparison`

#### JSON Schemas (19)

`event_envelope.json`, `timeline_segment.json`, `daily_summary.json`, `observation.json`, `intervention.json`, `feedback.json`, `outcome.json`, `goal.json`, `habit.json`, `briefing.json`, `current_context.json`, `data_quality_response.json`, `agent_effectiveness_response.json`, + 5 more

#### Database

- 18 tables: `devices`, `auth_tokens`, `events`, `current_context`, `timeline_segments`, `daily_summary`, `weekly_summary`, `five_minute_summary`, `hourly_summary`, `agent_observations`, `interventions`, `user_feedback`, `goals`, `habits`, `baselines`, `outcomes`, `audit_log`, `policy_settings`
- 62 DB methods with idempotent upserts, batch operations, and retention policy

#### Testing

- **50 Go unit tests** (45 handler tests + 5 DB tests)
- **63 integration tests** (server integration test suite)
- **59 MCP tool tests** (all tools validated with mock responses)
- **19 schema contract tests** (JSON schema validation)
- **18 engine tests** (inference engine integration)
- **Golden tests** (deterministic output validation)
- **OpenAPI contract tests** (spec validation)

#### Security

- Bearer token authentication with scope-based access control
- Token rotation with safe revocation (preserves new token)
- Sensitive mode with data redaction
- Kill switch for complete data wipe
- Clock drift detection
- Input validation on all POST endpoints

#### Deployment

- SQLite database with WAL mode
- Configurable retention policy (events: 90d, summaries: 365d)
- Caddy reverse proxy configuration
- Systemd service file
- Bootstrap token generation for device and agent tokens
