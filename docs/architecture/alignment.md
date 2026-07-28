Below is a refined architectural plan for your Android-first personal tracking system. I treated your uploaded document as **v1** and refined it into **v2**, with explicit decisions about what to **keep**, **expand**, **contract**, **remove**, and **add**.

The refined goal is not merely:

> “Track app usage, location, health, notifications, etc.”

The refined goal is:

> Build a local-first, agent-native personal observability and alignment engine that produces trustworthy semantic context, reconstructs a rich timeline of behavior, exposes secure endpoints for AI agents, and supports a closed-loop agentic system for observation, intervention, feedback, outcome tracking, and recalibration.

---

# ANDROMETRY v2  
## Refined Architecture for Agent-Aligned Personal Observability

## 1. Refined North Star

Your system should be understood as three coupled planes:

1. **Sensing Plane** — Android app collects behavioral, device, health, location, calendar, notification, and communication metadata.
2. **Context Plane** — phone and/or server transform raw signals into semantic objects: activities, places, focus sessions, distraction intervals, habits, summaries, confidence scores.
3. **Agentic Plane** — AI agents query context, write observations, propose interventions, receive feedback, evaluate outcomes, and improve over time.

The most important architectural principle remains:

> The phone should produce trustworthy semantic context.  
> The server should expose that context securely and minimally.  
> Agents should act on context, timeline, summaries, and outcomes — not raw telemetry alone.

---

# 2. Major Refinements to the Original Architecture

## 2.1 Keep

These parts of the original architecture are correct and should remain central.

| Area | Reason to Keep |
|---|---|
| Android app as primary sensing node | Best for privacy, battery, offline reliability |
| Local-first SQLite/Room storage | Required for offline resilience |
| Batched HTTPS sync | Prevents battery-draining real-time telemetry |
| Thin relay server | Avoids infrastructure bloat |
| SQLite on server | Enough for single-user personal system |
| Semantic summaries over raw streams | Agents need meaning, not noise |
| Geofence-based location | Battery-efficient and semantically useful |
| Health Connect summaries | Privacy-preserving and low-frequency |
| Current context object | Primary agent polling target |
| Daily/weekly summaries | Essential for reflection and analysis |
| ETag-based polling | Makes frequent agent polling cheap |
| MCP/tool exposure | Makes the system agent-native |
| Minimal Go/Rust/Python server | Keeps operations simple |
| Caddy reverse proxy | Simple automatic TLS |
| Optional Litestream backup | Good minimal disaster recovery |

---

## 2.2 Expand

These areas exist in the original document but need to become first-class architectural components.

### 1. Timeline Reconstruction

The original document mentions timeline queries, but the timeline must become the primary data product.

Agents should not ask:

> “Give me raw events from 9:00 to 12:00.”

They should ask:

> “What was I doing from 9:00 to 12:00?”

And receive a clean semantic timeline:

- Sleep segment
- Commute segment
- Office place visit
- Focus session
- Meeting segment
- Distraction segment
- Break segment
- Communication segment
- Exercise segment

The timeline should be layered, versioned, and correctable.

---

### 2. Agent Feedback Loop

The original architecture includes observations and interventions, but the loop must be expanded into a full control plane.

Add:

- Agent identity
- Agent scopes
- Observation ledger
- Intervention ledger
- User feedback
- Outcome attribution
- Intervention effectiveness scoring
- Agent rate limits
- Quiet hours
- Confidence thresholds
- Recalibration loops

Without this, agents can comment but cannot improve.

---

### 3. Confidence and Data Quality

Every inference must include confidence and evidence.

Bad:

```json
{
  "primary_activity": "deep_work"
}
```

Good:

```json
{
  "primary_activity": "deep_work",
  "confidence": 0.81,
  "evidence": {
    "focus_active": true,
    "productivity_app_share": 0.74,
    "app_switch_rate": "low",
    "notification_load": "low"
  },
  "missing_signals": [
    "health_connect"
  ]
}
```

Agents must know when not to trust a conclusion.

---

### 4. Schema Stability and Versioning

The system will evolve. You need explicit schema governance.

Add:

- `schema_version`
- `event_version`
- `client_version`
- `classifier_version`
- `segment_version`
- `summary_version`
- `device_id`
- `collector_id`
- `source`
- `ingested_at`
- `valid_from`
- `valid_to`

This prevents future breakage.

---

### 5. Privacy and Governance

The original document has privacy controls, but for a system this sensitive, privacy must be architectural.

Expand:

- Redaction before sync
- Sensitive mode
- Permission state tracking
- Token scopes
- Audit log
- Retention policies
- Deletion/wipe controls
- Local-only fields
- Contact hashing
- Calendar title redaction
- Notification content prohibition by default

---

### 6. System Health Telemetry

The system must observe itself.

Add:

- Collector health
- Sync queue depth
- Oldest pending event age
- Worker status
- Permission status
- Battery state
- Storage availability
- Clock drift detection
- Stale signal detection

Agents need to know when the observability system itself is degraded.

---

## 2.3 Contract or Remove

These should not be default features.

| Remove or Defer | Reason |
|---|---|
| Continuous high-accuracy GPS | Battery drain, privacy risk |
| Real-time notification content streaming | Sensitive, noisy, unnecessary |
| Raw SMS bodies | High privacy risk |
| Raw call logs | High privacy risk |
| Continuous sensor streaming | Battery drain |
| Permanent raw event storage | Storage bloat |
| Full real-time event streaming from phone | Battery and network cost |
| Heavy Postgres/Kafka/Redis stack | Unnecessary for single-user system |
| Docker-heavy microservices | Operational complexity |
| Real-time analytics pipeline | Not needed initially |
| Always-on WebSocket from phone | Battery and OS constraint issues |
| Immediate upload of every event | Battery and network inefficiency |
| Polling `UsageStatsManager` every second | Battery drain, Android hostility |
| Letting agents read raw events first | Token waste and poor abstraction |

The correct default is:

> Rich semantic summaries + selective raw events + stable agent APIs.

---

## 2.4 Add Missing Components

These were missing or under-specified in the original document and should now be added.

### 1. Schema Registry / Event Catalog

A canonical catalog defining:

- Event types
- Payload schemas
- Segment types
- Summary schemas
- Agent object schemas
- Versioning rules
- Compatibility rules

This should be contract-first.

---

### 2. Bitemporal Timeline Engine

The timeline must support correction over time.

Example:

At 10:00, the system infers:

```text
09:30–11:00 = deep_work
```

At 12:00, calendar sync reveals:

```text
09:30–10:15 = meeting
```

The system must preserve:

- What it believed at the time
- What the corrected truth is now
- When the correction happened
- Why the correction happened

Add temporal fields:

```json
{
  "event_time": "2026-07-28T09:37:00Z",
  "valid_from": "2026-07-28T09:37:00Z",
  "valid_to": "2026-07-28T11:02:00Z",
  "ingested_at": "2026-07-28T10:05:00Z",
  "revised_at": "2026-07-28T12:10:00Z",
  "revision_reason": "calendar_sync"
}
```

---

### 3. Agent Control Plane

A dedicated layer for agent interaction.

It should handle:

- Agent authentication
- Agent scopes
- Observation writes
- Intervention proposals
- Feedback reads
- Outcome evaluation
- Rate limiting
- Quiet hours
- Severity thresholds
- Audit logging
- Dry-run mode

---

### 4. Intervention Delivery Layer

The original architecture stores interventions but does not fully specify delivery.

Add:

- Phone-side intervention inbox
- Local notification delivery
- Optional chat/email/webhook delivery
- Optional Tasker/MacroDroid integration
- Intervention expiry
- Intervention dismissal
- Acceptance tracking
- Response latency tracking

Agents should not merely write interventions into a database. The system must deliver them and track responses.

---

### 5. Outcome Attribution Engine

Agents need to know whether their advice worked.

Add outcome types:

- Accepted
- Dismissed
- Ignored
- Completed
- Behavior changed
- Metric improved
- Self-reported helpfulness
- No measurable effect
- Negative effect

Example:

```json
{
  "intervention_id": "uuid",
  "outcome_type": "behavior_change",
  "evidence": {
    "distraction_minutes_before": 12,
    "distraction_minutes_after": 3,
    "focus_resumed": true
  },
  "effectiveness_score": 0.72
}
```

---

### 6. Baseline and Calibration Store

The system must learn normal behavior.

Store baselines for:

- Deep work minutes
- Distraction minutes
- Sleep duration
- Bedtime regularity
- Steps
- Notification load
- Unlock count
- Meeting load
- Focus session success rate
- Response latency
- Fatigue indicators

Agents should compare current state against baselines, not absolute thresholds.

---

### 7. Policy Governor

Agents should not be allowed to spam the user.

Add rules:

- Maximum interventions per day
- Cooldown between nudges
- Quiet hours
- Severity-based override
- User fatigue protection
- Low-confidence suppression
- Repeated-dismissal suppression

Example:

```json
{
  "max_interventions_per_day": 6,
  "cooldown_minutes": 45,
  "quiet_hours": {
    "start": "22:30",
    "end": "07:30"
  },
  "min_confidence_for_nudge": 0.65
}
```

---

### 8. Agent Briefing API

Agents should not always assemble context from many endpoints.

Add a high-level briefing endpoint:

```text
GET /v1/briefing/current
```

Example response:

```json
{
  "briefing_id": "uuid",
  "generated_at": "2026-07-28T10:15:00Z",
  "state": {
    "primary_activity": "deep_work",
    "place": "office",
    "focus_active": true,
    "distraction_probability": 0.18,
    "fatigue_probability": 0.31,
    "alignment_probability": 0.74
  },
  "delta_since_last": {
    "distraction_probability_change": -0.04,
    "focus_minutes_elapsed": 37
  },
  "risks": [
    {
      "type": "focus_drift",
      "severity": "low",
      "confidence": 0.62
    }
  ],
  "opportunities": [
    {
      "type": "extend_focus_block",
      "confidence": 0.71
    }
  ],
  "suggested_queries": [
    "get_timeline_last_2h",
    "get_focus_sessions_today",
    "get_daily_summary"
  ]
}
```

This reduces agent friction and token usage.

---

### 9. User Correction Loop

The user should be able to correct inferred data.

Examples:

- “This was not distraction; it was research.”
- “This meeting was personal, not work.”
- “I was not at the office; I was at a cafe.”
- “This focus session failed because of an emergency.”

Corrections should feed:

- Segment revision
- Classifier calibration
- App classification overrides
- Habit evaluation
- Agent confidence

---

### 10. Habit Engine

The original document mentions habits, but habit tracking should be explicit.

Add:

- Habit definitions
- Cue detection
- Completion evidence
- Streak logic
- Exception rules
- Travel mode
- Recovery rules
- Habit strength score

Example:

```json
{
  "habit_id": "morning_deep_work",
  "target": {
    "metric": "deep_work_minutes_before_noon",
    "minimum": 90
  },
  "schedule": {
    "days": ["mon", "tue", "wed", "thu", "fri"],
    "window": "08:00-12:00"
  },
  "evidence_sources": [
    "focus_segment",
    "activity_segment",
    "app_usage_segment"
  ]
}
```

---

### 11. Work Pattern Metrics

Add explicit work-pattern analytics.

Examples:

- Average focus bout length
- Context switching rate
- App switch bursts
- Meeting fragmentation
- Maker-time blocks
- Interruption density
- Response latency
- Communication load during focus
- Late-night work index
- Energy-adjusted productivity
- Fragmentation score

These are essential for alignment coaching.

---

### 12. Readiness / Fatigue Model

Add a lightweight readiness model.

Inputs:

- Sleep duration
- Sleep regularity
- Time awake
- Late-night screen usage
- Heart rate trends
- Steps/activity
- Workload yesterday
- Meeting load
- Distraction load
- Self-reported energy, optional

Outputs:

```json
{
  "readiness_score": 0.68,
  "fatigue_probability": 0.41,
  "recommended_focus_intensity": "moderate",
  "recommended_break": false
}
```

---

### 13. Analytics Archive

For long-term analysis, add an export/archive layer.

Recommended:

- Nightly export from SQLite
- JSONL for raw-ish events
- Parquet for analytics
- Optional object storage
- Optional DuckDB queries

This keeps the relay thin while enabling serious analysis later.

---

### 14. Testing and Simulation Harness

Add from the beginning:

- Synthetic event generator
- Replayable event sequences
- Golden test cases
- Schema contract tests
- Timeline reconstruction tests
- Sync idempotency tests
- Battery profile tests
- Agent query simulations

Without this, the system will become fragile as it evolves.

---

# 3. Revised High-Level Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                    Android Collector App                     │
│                                                              │
│  Collectors                                                  │
│  - Screen/unlock                                             │
│  - App usage                                                 │
│  - Notifications                                             │
│  - Calendar                                                  │
│  - Location/geofence                                         │
│  - Health Connect                                            │
│  - Communication metadata                                    │
│  - Focus sessions                                            │
│  - Device state                                              │
│  - Permission state                                          │
│                                                              │
│  Processing                                                  │
│  - Event normalization                                       │
│  - App classification                                        │
│  - Timeline segmentation                                     │
│  - Context inference                                         │
│  - Confidence scoring                                        │
│  - Data quality scoring                                      │
│  - Baseline comparison                                       │
│  - Privacy redaction                                         │
│  - Summary generation                                        │
│                                                              │
│  Storage                                                     │
│  - Room/SQLite                                               │
│  - Event queue                                               │
│  - Timeline segments                                         │
│  - Context cache                                             │
│  - Summary cache                                             │
│  - Intervention inbox                                        │
│  - Goals/habits cache                                        │
│  - User corrections                                          │
└───────────────────────────┬──────────────────────────────────┘
                            │ HTTPS POST, gzip, token auth
                            │ batched, idempotent, retry-safe
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                    Minimal Context Relay                     │
│                                                              │
│  - Single Go binary                                          │
│  - SQLite                                                    │
│  - Device auth                                               │
│  - Agent auth                                                │
│  - Batch ingestion                                           │
│  - Current context cache                                     │
│  - Timeline API                                              │
│  - Summary API                                               │
│  - Briefing API                                              │
│  - Event drill-down API                                      │
│  - Agent observation API                                     │
│  - Intervention API                                          │
│  - Feedback API                                              │
│  - Goal/habit API                                            │
│  - Outcome API                                               │
│  - Audit log                                                 │
│  - Optional SSE/webhook                                      │
│  - Optional MCP server                                       │
│  - Optional Litestream backup                                │
└───────────────────────────┬──────────────────────────────────┘
                            │ HTTPS poll / SSE / MCP tools
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                        AI Agents                             │
│                                                              │
│  - Current context polling                                   │
│  - Briefing consumption                                      │
│  - Timeline drill-down                                       │
│  - Daily/weekly analysis                                     │
│  - Habit tracking                                            │
│  - Alignment scoring                                         │
│  - Risk detection                                            │
│  - Intervention generation                                   │
│  - Outcome evaluation                                        │
│  - Reflection generation                                     │
│  - Calibration feedback                                      │
└──────────────────────────────────────────────────────────────┘
```

---

# 4. Refined Android App Architecture

## 4.1 Recommended Stack

Use:

- Kotlin
- Room/SQLite
- WorkManager
- Coroutines/Flow
- Hilt or Koin
- DataStore
- Retrofit or Ktor
- OkHttp with gzip
- Health Connect SDK
- Fused Location Provider
- Geofencing API
- NotificationListenerService
- UsageStatsManager
- Calendar ContentObserver
- Optional foreground service for active modes

---

## 4.2 Recommended Module Structure

```text
app/
  core/
    database/
    network/
    security/
    time/
    logs/
    config/
  collectors/
    screen/
    usage/
    notifications/
    calendar/
    location/
    health/
    communication/
    device/
    focus/
    permissions/
  inference/
    classifier/
    segments/
    context/
    confidence/
    data_quality/
    baselines/
  summaries/
    five_minute/
    hourly/
    daily/
    weekly/
  sync/
    queue/
    uploader/
    idempotency/
    retry/
    backfill/
  agents/
    observations/
    interventions/
    goals/
    habits/
    feedback/
    outcomes/
  ui/
    permissions/
    status/
    debug_timeline/
    intervention_inbox/
    privacy/
```

---

## 4.3 Collector Matrix

This refines the original collection strategy and corrects Android-specific constraints.

| Signal | Source | Method | Default Cadence | Semantic Product |
|---|---|---|---|---|
| Screen on/off | Android broadcasts | BroadcastReceiver | Event-driven | Screen sessions |
| Unlock | `ACTION_USER_PRESENT` | BroadcastReceiver | Event-driven | Unlock count, session starts |
| App usage | `UsageStatsManager` | Periodic worker or active-mode service | Balanced: 15 min; Active: 1–5 min | App/category time, switching rate |
| Notifications | `NotificationListenerService` | Callback + periodic summary | Event-driven + 5–15 min summary | Notification load, interruption level |
| Calendar | ContentObserver + periodic sync | Observer + worker | On change + 15–30 min | Busy/free, next event, meeting load |
| Location | Geofence + passive location | GeofencingClient + passive provider | Event-driven + 15 min reconciliation | Place visits, movement state |
| Health | Health Connect | Periodic worker | 30–60 min | Steps, sleep, heart rate, exercise |
| Communication metadata | Notification metadata | Notification listener | Event-driven + summary | Calls/SMS counts, response latency |
| Focus sessions | Manual/calendar/inferred | App logic | Event-driven | Focus segments |
| Device state | Battery/connectivity broadcasts | BroadcastReceiver | Event-driven | Battery, charging, network |
| Permission state | Permission checks | Worker + UI events | 15 min + on change | Data quality signals |
| App health | Internal telemetry | Worker | 15 min | Queue depth, collector health |

Important Android refinement:

> Periodic WorkManager jobs cannot reliably run every 5 minutes. The minimum periodic interval is normally 15 minutes.  
> If you need 1–5 minute granularity, use an active mode foreground service during work hours, focus sessions, or charging. Otherwise default to 15-minute batch collection.

---

## 4.4 Permission Strategy

### Core Permissions

Required:

```text
PACKAGE_USAGE_STATS
POST_NOTIFICATIONS
RECEIVE_BOOT_COMPLETED
INTERNET
ACCESS_NETWORK_STATE
FOREGROUND_SERVICE
```

Depending on Android version and foreground service use:

```text
FOREGROUND_SERVICE_SPECIAL_USE
```

or another appropriate type. Use foreground services selectively.

---

### Notification Listener

Requires user to grant notification access in system settings.

Track:

```json
{
  "notification_listener_enabled": true,
  "notification_listener_restricted": false
}
```

Android may restrict notification access for some apps. The app should detect and surface this.

---

### Location

Default:

```text
ACCESS_COARSE_LOCATION
ACCESS_FINE_LOCATION
ACCESS_BACKGROUND_LOCATION
```

But only use background location for geofencing/place inference. Do not continuously track GPS.

---

### Health Connect

Request only what you need:

- Steps
- Sleep
- Heart rate
- Exercise sessions
- Active calories, optional

Track availability:

```json
{
  "health_connect_available": true,
  "steps_permission": true,
  "sleep_permission": true,
  "heart_rate_permission": false
}
```

---

### Calendar

Use:

```text
READ_CALENDAR
```

Default sync policy:

- Store titles locally only
- Sync categories and busy/free status
- Redact titles before server sync unless explicitly enabled

---

### Calls/SMS

Avoid direct sensitive permissions initially.

Prefer:

- Communication notification metadata
- Counts only
- Hashed contact identifiers if needed
- No message bodies
- No phone numbers
- No call recordings

Only later, if absolutely necessary:

```text
READ_CALL_LOG
READ_SMS
```

But this should be treated as high-risk and optional.

---

## 4.5 Battery Profiles

### Profile A: Eco

Best for long-term passive tracking.

- No persistent foreground service
- App usage summary every 15–30 minutes
- Location via geofence/passive only
- Health sync every 60 minutes
- Context snapshot every 10–15 minutes
- Upload every 30 minutes or on Wi-Fi/charging
- Raw events kept 3–7 days

Expected result:

- Very low battery impact
- Good daily insights
- Limited minute-level precision

---

### Profile B: Balanced

Recommended default.

- Foreground service only during work hours or focus sessions
- App usage summary every 15 minutes
- Active-mode 5-minute summaries when enabled
- Location passive + geofence
- Health sync every 30–60 minutes
- Context snapshot every 5–15 minutes
- Upload every 10–15 minutes
- Raw events kept 14 days

Expected result:

- Strong agent usefulness
- Moderate battery impact
- Good real-time context

---

### Profile C: High Fidelity

Use temporarily.

- Foreground service all day
- App usage every 1–2 minutes
- Context snapshot every 1 minute
- More frequent passive location
- Upload every 5 minutes
- Raw events kept 30 days

Expected result:

- High precision
- Higher battery cost
- Useful for debugging or special projects

---

### Adaptive Profile

Add automatic switching:

```text
If charging and work hours:
  switch to Balanced or High Fidelity

If battery < 20%:
  switch to Eco

If focus session starts:
  temporarily increase fidelity

If device idle overnight:
  reduce polling
```

---

## 4.6 Minimal UI

Even if UI/UX is not a priority, you need a few screens for reliability.

Build only:

1. Permission status screen
2. Collector status screen
3. Sync status screen
4. Debug timeline screen
5. Current context screen
6. Intervention inbox screen
7. Privacy/sensitive mode switch
8. Export/backup controls

The intervention inbox is important. Agents should not only send data to the server; the user should also be able to see proposed interventions locally.

---

# 5. Refined Data Model

## 5.1 Event Envelope

Every event should use a stable envelope.

```json
{
  "event_id": "uuid",
  "device_id": "pixel-9-pro",
  "user_id": "primary",
  "schema_version": 1,
  "event_version": 1,
  "client_version": "0.1.0",
  "collector": "usage_stats",
  "source": "workmanager",
  "sequence": 10231,
  "timestamp": "2026-07-28T10:05:00Z",
  "timezone": "Asia/Kolkata",
  "utc_offset_seconds": 19800,
  "event_type": "app_usage_summary",
  "payload": {}
}
```

Important fields:

- `event_id`: globally unique
- `sequence`: monotonic per device
- `schema_version`: compatibility control
- `timezone`: essential for sleep/circadian analysis
- `collector`: helps debug missing data
- `source`: observed vs inferred vs user-provided

---

## 5.2 Event Types

### Device/System

```text
boot_completed
screen_state_changed
device_unlocked
battery_state_changed
charging_state_changed
timezone_changed
connectivity_changed
permission_state_changed
app_health
sync_heartbeat
```

### App Usage

```text
app_usage_summary
app_category_summary
app_switch_burst
app_classification_updated
```

### Notifications

```text
notification_event
notification_summary
notification_interruption_summary
```

### Focus

```text
focus_session
focus_state_changed
focus_interruption
```

### Calendar

```text
calendar_context
calendar_event_changed
```

### Location

```text
location_context
place_visit_started
place_visit_ended
geofence_transition
movement_context
```

### Health

```text
health_summary
sleep_summary
activity_recognition
exercise_session
```

### Communication

```text
communication_summary
communication_notification_summary
```

### Context

```text
context_snapshot
activity_segment
daily_summary
weekly_summary
```

### Agent Loop

```text
agent_observation
agent_intervention
user_feedback
intervention_outcome
goal_updated
habit_updated
baseline_updated
user_correction
```

---

## 5.3 Semantic Timeline Layers

The timeline should not be a single flat list. It should be layered.

Recommended layers:

1. **Activity Layer**
   - deep_work
   - shallow_work
   - meeting
   - communication
   - learning
   - admin
   - planning
   - break
   - leisure
   - social
   - distraction
   - exercise
   - commute
   - sleep
   - unknown

2. **Place Layer**
   - home
   - office
   - gym
   - cafe
   - library
   - commute
   - unknown

3. **Focus Layer**
   - focus_session
   - inferred_focus
   - interrupted_focus
   - failed_focus

4. **Communication Layer**
   - calls
   - messages
   - email
   - chat
   - notification-heavy intervals

5. **Health Layer**
   - sleep
   - exercise
   - steps
   - inactivity
   - elevated heart rate

6. **Calendar Layer**
   - meeting
   - personal
   - health
   - travel
   - focus block
   - errand

7. **Device Layer**
   - screen on/off
   - unlock bursts
   - charging
   - battery state

Agents can query layers independently or request a merged timeline.

---

## 5.4 Timeline Segment Object

```json
{
  "segment_id": "uuid",
  "device_id": "pixel-9-pro",
  "segment_type": "focus_segment",
  "layer": "focus",
  "start_time": "2026-07-28T09:37:00+05:30",
  "end_time": "2026-07-28T11:02:00+05:30",
  "label": "deep_work",
  "confidence": 0.86,
  "version": 1,
  "valid_from": "2026-07-28T09:37:00+05:30",
  "valid_to": "2026-07-28T11:02:00+05:30",
  "ingested_at": "2026-07-28T10:05:00Z",
  "revised_at": null,
  "revision_reason": null,
  "evidence": {
    "focus_active": true,
    "productivity_app_share": 0.74,
    "app_switch_rate": "low",
    "notification_load": "low"
  },
  "payload": {
    "interruptions": 3,
    "distraction_minutes": 6,
    "success_score": 0.81
  }
}
```

---

## 5.5 Current Context Object

This is the primary object agents should poll.

```json
{
  "timestamp": "2026-07-28T10:14:00Z",
  "timezone": "Asia/Kolkata",
  "device_id": "pixel-9-pro",
  "schema_version": 1,
  "context": {
    "primary_activity": "deep_work",
    "activity_confidence": 0.81,
    "place": "office",
    "place_confidence": 0.92,
    "movement": "still",
    "foreground_app_category": "productivity",
    "screen_state": "on",
    "minutes_since_unlock": 41,
    "focus_active": true,
    "focus_minutes_elapsed": 37,
    "focus_source": "manual",
    "interruption_rate": "low",
    "notifications_last_15m": 3,
    "distraction_probability": 0.18,
    "fatigue_probability": 0.31,
    "alignment_probability": 0.74,
    "readiness_score": 0.68,
    "calendar": {
      "current_event": "none",
      "next_event_minutes": 46,
      "busy_until": "2026-07-28T12:30:00Z"
    },
    "health": {
      "steps_last_hour": 210,
      "activity": "still",
      "heart_rate_avg": 74
    },
    "sleep": {
      "last_sleep_minutes": 460,
      "sleep_debt_estimate_minutes": 35
    },
    "battery": {
      "level": 0.74,
      "charging": false
    },
    "data_quality": {
      "completeness": 0.87,
      "staleness_seconds": 42,
      "usage_stats_available": true,
      "notification_listener_available": true,
      "calendar_available": true,
      "location_fresh": true,
      "health_connect_available": true,
      "last_sync_minutes_ago": 7,
      "missing_signals": []
    }
  }
}
```

---

## 5.6 Daily Summary Object

The daily summary should be rich enough for reflection and coaching.

```json
{
  "date": "2026-07-28",
  "timezone": "Asia/Kolkata",
  "sleep": {
    "minutes": 460,
    "regularity_score": 0.82,
    "bedtime": "2026-07-27T23:41:00+05:30",
    "wake_time": "2026-07-28T07:21:00+05:30"
  },
  "health": {
    "steps": 8123,
    "exercise_minutes": 35,
    "active_minutes": 58
  },
  "work": {
    "deep_work_minutes": 182,
    "shallow_work_minutes": 96,
    "meeting_minutes": 95,
    "admin_minutes": 41
  },
  "distraction": {
    "total_minutes": 64,
    "social_minutes": 38,
    "browser_hopping_minutes": 14,
    "during_focus_minutes": 11
  },
  "interruptions": {
    "notification_count": 143,
    "unlock_count": 58,
    "app_switch_count": 312
  },
  "location": {
    "home_minutes": 620,
    "office_minutes": 510,
    "gym_minutes": 55,
    "commute_minutes": 70
  },
  "focus": {
    "sessions": 3,
    "successful_sessions": 2,
    "average_interruptions": 2.3
  },
  "work_patterns": {
    "average_focus_bout_minutes": 41,
    "switching_rate_per_hour": 18,
    "meeting_fragmentation_score": 0.43,
    "late_night_work_minutes": 22
  },
  "scores": {
    "alignment": 0.74,
    "productivity": 0.69,
    "wellbeing": 0.71,
    "focus_control": 0.66,
    "sleep_quality": 0.78,
    "readiness": 0.68
  },
  "baselines": {
    "deep_work_minutes_7d_avg": 158,
    "distraction_minutes_7d_avg": 71,
    "sleep_minutes_7d_avg": 442
  }
}
```

---

# 6. Timeline Reconstruction Engine

This must become one of the core engines.

## 6.1 Segment Types

```text
activity_segment
focus_segment
place_visit
app_usage_segment
meeting_segment
communication_segment
sleep_segment
distraction_segment
break_segment
exercise_segment
commute_segment
```

---

## 6.2 Segment Generation Rules

Example: deep work segment.

```text
If:
  productivity app usage > 20 minutes
  app switches low
  no meeting
  notifications low
  focus session active or inferred
Then:
  create deep_work segment
```

Example: distraction segment.

```text
If:
  social app usage > 10 minutes
  during intended focus
  app switches high
Then:
  create distraction segment
```

Example: place visit.

```text
If:
  geofence enter office
  passive location confirms office
  duration > 10 minutes
Then:
  create office place_visit segment
```

---

## 6.3 Segment Correction

Segments must be versioned.

Example:

```json
{
  "segment_id": "uuid",
  "version": 3,
  "label": "meeting",
  "confidence": 0.84,
  "revision_reason": "calendar_sync"
}
```

The server should upsert by `segment_id`.

---

## 6.4 Overlap and Gap Handling

The timeline engine must handle:

- Overlapping segments
- Gaps between segments
- Late-arriving data
- Corrected calendar events
- Missing location data
- Missing health data
- Timezone changes
- Device clock drift

Recommended approach:

1. Store raw events.
2. Generate candidate segments.
3. Merge related segments.
4. Resolve overlaps by layer priority.
5. Mark gaps as `unknown` if too large.
6. Recompute summaries when segments change.

---

# 7. Context Inference Engine

## 7.1 Primary Activity Taxonomy

```text
deep_work
shallow_work
meeting
communication
learning
admin
planning
break
leisure
social
distraction
exercise
commute
sleep
unknown
```

---

## 7.2 Classification Inputs

Use:

- Foreground app category
- App usage duration
- App switch frequency
- Calendar state
- Focus session state
- Location
- Movement
- Time of day
- Notification load
- Health activity
- Sleep debt
- User-defined rules
- User corrections

---

## 7.3 Confidence Scoring

Confidence should depend on:

- Signal freshness
- Signal agreement
- Permission availability
- Historical accuracy
- Data completeness
- Timezone stability
- Location confidence
- Calendar availability
- Collector health

Example:

```json
{
  "primary_activity": "deep_work",
  "confidence": 0.81,
  "evidence": {
    "focus_active": true,
    "productivity_app_share": 0.74,
    "app_switch_rate": "low",
    "notification_load": "low",
    "calendar_clear": true
  },
  "missing": [
    "health_connect"
  ]
}
```

---

## 7.4 Data Quality Object

Attach this to every context snapshot.

```json
{
  "completeness": 0.87,
  "staleness_seconds": 42,
  "missing_collectors": [],
  "degraded_collectors": [
    "health_connect"
  ],
  "permission_issues": [],
  "clock_drift_detected": false
}
```

Agents should use this to decide whether to act.

---

# 8. Agentic Loop Architecture

This is the most important refinement.

Your system should not merely provide data. It should support this loop:

```text
Sense
→ Infer
→ Store
→ Expose
→ Agent interprets
→ Agent observes/intervenes
→ Intervention delivered
→ User responds
→ Outcome recorded
→ Model recalibrates
```

---

## 8.1 Agent Control Plane

Add a dedicated agent control plane.

Responsibilities:

- Authenticate agents
- Scope agent access
- Rate limit interventions
- Enforce quiet hours
- Audit agent actions
- Provide briefings
- Track outcomes
- Recalibrate confidence

---

## 8.2 Agent Query Ladder

Agents should not read raw events first.

Use this ladder:

1. Current context
2. Current briefing
3. Recent timeline
4. Daily summary
5. Segment history
6. Event drill-down
7. Raw export/analytics

This reduces cost and improves speed.

---

## 8.3 Agent Briefing Object

```json
{
  "briefing_id": "uuid",
  "generated_at": "2026-07-28T10:15:00Z",
  "state": {
    "primary_activity": "deep_work",
    "place": "office",
    "focus_active": true,
    "distraction_probability": 0.18,
    "fatigue_probability": 0.31,
    "alignment_probability": 0.74
  },
  "delta_since_last": {
    "distraction_probability_change": -0.04,
    "focus_minutes_elapsed": 37
  },
  "risks": [
    {
      "type": "focus_drift",
      "severity": "low",
      "confidence": 0.62
    }
  ],
  "opportunities": [
    {
      "type": "extend_focus_block",
      "confidence": 0.71
    }
  ],
  "suggested_queries": [
    "get_timeline_last_2h",
    "get_focus_sessions_today",
    "get_daily_summary"
  ]
}
```

This is a major friction reducer.

---

## 8.4 Policy Governor

Example policy:

```json
{
  "max_interventions_per_day": 6,
  "cooldown_minutes": 45,
  "quiet_hours": {
    "start": "22:30",
    "end": "07:30"
  },
  "min_confidence_for_nudge": 0.65,
  "suppress_after_repeated_dismissals": true,
  "allow_high_severity_override": true
}
```

---

# 9. Intervention Delivery and Outcome Attribution

This was under-specified in the original document.

## 9.1 Intervention Object

```json
{
  "intervention_id": "uuid",
  "agent_id": "alignment-coach",
  "intervention_type": "nudge",
  "status": "proposed",
  "title": "Return to focus",
  "message": "You have drifted into social apps during a focus block.",
  "suggested_action": "resume_focus_session",
  "severity": "medium",
  "expires_at": "2026-07-28T10:30:00Z",
  "evidence": {
    "distraction_minutes": 12,
    "focus_active": true
  }
}
```

---

## 9.2 Intervention Statuses

```text
proposed
approved_by_policy
delivered
accepted
dismissed
ignored
completed
failed
suppressed
expired
```

---

## 9.3 Delivery Channels

Recommended:

1. Local Android notification
2. Intervention inbox inside the app
3. Optional chat message through your agent framework
4. Optional email digest
5. Optional webhook
6. Optional Tasker/MacroDroid action

Do not rely only on the server-side intervention table. The phone should pull pending interventions during sync and deliver them locally.

---

## 9.4 Outcome Object

```json
{
  "outcome_id": "uuid",
  "intervention_id": "uuid",
  "timestamp": "2026-07-28T10:31:00Z",
  "outcome_type": "behavior_change",
  "evidence": {
    "distraction_minutes_before": 12,
    "distraction_minutes_after": 3,
    "focus_resumed": true
  },
  "self_report": {
    "helpful": true,
    "rating": 4
  },
  "effectiveness_score": 0.72
}
```

---

## 9.5 Effectiveness Scoring

Effectiveness should combine:

- Was it delivered?
- Was it accepted?
- Did behavior change?
- Did the target metric improve?
- Did the user report it as helpful?
- Did the effect persist?
- Did it cause negative side effects?

Example:

```text
effectiveness_score =
  0.25 * accepted +
  0.35 * behavior_change +
  0.25 * metric_improvement +
  0.15 * user_helpfulness
```

The exact formula can evolve.

---

# 10. Server Architecture

## 10.1 Recommended Production-Minimal Stack

Use:

- Ubuntu VPS or Raspberry Pi
- Single Go binary
- SQLite
- Caddy reverse proxy
- systemd
- Litestream optional
- Tailscale optional

Go is recommended for production-minimal infrastructure.

Python/FastAPI is acceptable for prototype.

---

## 10.2 Server Responsibilities

The server should:

- Authenticate devices
- Authenticate agents
- Accept event batches
- Store events
- Upsert timeline segments
- Update current context
- Store summaries
- Serve agent queries
- Accept agent observations
- Accept interventions
- Accept feedback
- Track goals and habits
- Track outcomes
- Enforce scopes
- Write audit logs
- Optionally stream updates via SSE/webhook
- Optionally expose MCP tools

It should not initially do:

- Heavy analytics
- Stream processing
- ML training
- Complex orchestration
- Microservices

---

# 11. API Design

Keep the API small, stable, and agent-friendly.

## 11.1 Authentication

Use bearer tokens.

Device token scopes:

```text
device:ingest
device:read_config
device:read_interventions
device:write_feedback
```

Agent token scopes:

```text
agent:context:read
agent:briefing:read
agent:timeline:read
agent:events:read
agent:summaries:read
agent:observations:write
agent:interventions:write
agent:feedback:read
agent:goals:read
agent:goals:write
agent:habits:read
agent:habits:write
agent:outcomes:write
```

Store only token hashes server-side.

---

## 11.2 Device Ingestion

```text
POST /v1/batch
Authorization: Bearer <device_token>
Content-Encoding: gzip
Content-Type: application/json
```

Body:

```json
{
  "device_id": "pixel-9-pro",
  "batch_id": "uuid",
  "sequence_start": 10231,
  "sequence_end": 10287,
  "events": [],
  "segments": [],
  "summaries": [],
  "context": {},
  "device_state": {
    "battery": 0.74,
    "charging": false,
    "storage_free_mb": 42000,
    "queue_size": 12
  }
}
```

Response:

```json
{
  "accepted_events": 42,
  "rejected_events": 0,
  "accepted_segments": 7,
  "accepted_summaries": 2,
  "missing_sequences": [],
  "pending_interventions": 1,
  "server_time": "2026-07-28T10:15:03Z"
}
```

The `pending_interventions` field allows the phone to fetch agent interventions during normal sync.

---

## 11.3 Current Context

```text
GET /v1/context/current
Authorization: Bearer <agent_token>
```

Support:

```text
If-None-Match
ETag
304 Not Modified
```

This makes frequent polling cheap.

---

## 11.4 Briefing

```text
GET /v1/briefing/current
Authorization: Bearer <agent_token>
```

Optional parameters:

```text
?horizon=2h
&agent_id=alignment-coach
&include_evidence=true
```

---

## 11.5 Timeline

```text
GET /v1/timeline?from=...&to=...&limit=200
Authorization: Bearer <agent_token>
```

Optional filters:

```text
segment_type=focus_segment
label=deep_work
layer=focus
min_confidence=0.7
```

---

## 11.6 Events

For drill-down only.

```text
GET /v1/events?from=...&to=...&event_type=app_usage_summary&limit=200
Authorization: Bearer <agent_token>
```

Agents should not use this as their primary interface.

---

## 11.7 Summaries

```text
GET /v1/summary/daily?date=2026-07-28
GET /v1/summary/weekly?week=2026-W31
```

---

## 11.8 Agent Observations

```text
POST /v1/agent/observations
Authorization: Bearer <agent_token>
```

Body:

```json
{
  "observation_id": "uuid",
  "agent_id": "alignment-coach",
  "observation_type": "risk",
  "severity": "medium",
  "message": "Distraction probability is increasing during a focus block.",
  "evidence": {
    "distraction_probability": 0.41,
    "recent_social_minutes": 12
  },
  "related_context_timestamp": "2026-07-28T10:14:00Z"
}
```

---

## 11.9 Interventions

```text
POST /v1/agent/interventions
Authorization: Bearer <agent_token>
```

---

## 11.10 Feedback

```text
POST /v1/feedback
Authorization: Bearer <agent_token_or_device_token>
```

Body:

```json
{
  "feedback_id": "uuid",
  "target_type": "intervention",
  "target_id": "uuid",
  "feedback_type": "helpful",
  "rating": 4,
  "note": "This helped me return to work."
}
```

---

## 11.11 Goals and Habits

```text
GET /v1/goals
POST /v1/goals
GET /v1/habits
POST /v1/habits
```

---

## 11.12 Outcomes

```text
POST /v1/outcomes
GET /v1/outcomes?intervention_id=...
```

---

## 11.13 Optional SSE Stream

```text
GET /v1/stream
Authorization: Bearer <agent_token>
Accept: text/event-stream
```

Events:

```text
context_changed
focus_session_started
focus_session_ended
distraction_spike
intervention_updated
daily_summary_ready
```

The phone should not maintain SSE. The server can stream to agents.

---

# 12. MCP Server / Agent Tool Layer

Wrap the API as an MCP server.

## 12.1 Context Tools

```text
get_current_context
get_current_briefing
get_context_history
get_data_quality
```

## 12.2 Timeline Tools

```text
get_timeline
get_focus_sessions
get_place_visits
get_activity_segments
get_distraction_segments
get_communication_segments
get_sleep_segments
```

## 12.3 Summary Tools

```text
get_daily_summary
get_weekly_summary
get_habit_metrics
get_baseline_comparison
get_work_pattern_metrics
```

## 12.4 Event Tools

```text
search_events
get_app_usage
get_notification_load
get_communication_summary
get_health_summary
get_sleep_summary
get_calendar_context
```

## 12.5 Agent Loop Tools

```text
add_agent_observation
propose_intervention
update_intervention_status
record_user_feedback
get_intervention_outcomes
evaluate_agent_effectiveness
```

## 12.6 Goal/Habit Tools

```text
get_goals
create_goal
update_goal
evaluate_goal_progress
get_habits
create_habit
evaluate_habit_completion
```

---

# 13. Privacy, Security, and Consent

This system is highly sensitive. Privacy must be architectural.

## 13.1 Default Privacy Posture

Default should be:

- No notification content
- No message bodies
- No exact coordinates
- No contact names
- No phone numbers
- No raw SMS
- No raw call logs
- No calendar titles unless needed
- No permanent raw events
- Local-first storage
- Encrypted transport
- Token authentication
- Minimal server retention

---

## 13.2 Redaction Rules

Before sync:

- Replace exact coordinates with place labels
- Hash contacts
- Remove notification text
- Remove calendar titles if configured
- Remove app-specific sensitive content
- Strip free-text notes
- Remove message bodies
- Remove sender identities unless explicitly enabled

---

## 13.3 Sensitive Mode

User-controlled.

When enabled:

- Pause notification metadata
- Pause location
- Pause calendar sync
- Redact app usage
- Mark context as private
- Stop sync temporarily

Example:

```json
{
  "event_type": "privacy_mode_changed",
  "timestamp": "2026-07-28T21:00:00Z",
  "payload": {
    "sensitive_mode": true,
    "pause_sync": true,
    "redact_location": true
  }
}
```

---

## 13.4 Kill Switch

The app should support:

- Pause tracking
- Pause sync
- Delete local data
- Delete server data
- Revoke tokens
- Wipe agent observations
- Wipe interventions

This is essential for trust.

---

# 14. Reliability and Failure Handling

## 14.1 Offline-First Queue

The app must handle:

- No network
- Server downtime
- Token expiry
- Large backlogs
- Device reboot
- Permission revocation
- Low battery
- Low storage

Queue states:

```text
pending
uploading
uploaded
failed
dropped
```

---

## 14.2 Retry Policy

Use exponential backoff:

```text
1 minute
5 minutes
15 minutes
30 minutes
60 minutes
```

Retry faster on Wi-Fi/charging.

---

## 14.3 Idempotency

Server should reject duplicates using:

```text
device_id + event_id
```

or:

```text
device_id + sequence
```

This allows safe retries.

---

## 14.4 Gap Detection

Server should detect missing sequences.

Example:

```text
Received sequence 1000, then 1005.
Missing: 1001–1004.
```

Server response:

```json
{
  "action": "request_backfill",
  "sequence_from": 1001,
  "sequence_to": 1004
}
```

---

## 14.5 Clock Correctness

Every event must include:

- UTC timestamp
- Local timezone
- UTC offset

Detect timezone changes and store them as events.

---

# 15. Analytics and Retention

## 15.1 Local Retention

Recommended:

| Data Type | Retention |
|---|---:|
| Raw events | 7–14 days |
| Five-minute summaries | 30–90 days |
| Hourly summaries | 90–365 days |
| Daily summaries | indefinite |
| Timeline segments | 90–365 days |
| Current context | latest only |
| Agent observations | 90 days |
| Interventions | 90 days |
| User feedback | indefinite |
| Goals/habits | indefinite |

---

## 15.2 Server Retention

Recommended:

| Data Type | Retention |
|---|---:|
| Raw events | 30 days |
| Timeline segments | 365 days |
| Daily summaries | indefinite |
| Weekly summaries | indefinite |
| Agent observations | 90 days |
| Interventions | 90 days |
| Feedback | indefinite |
| Audit log | 365 days |

---

## 15.3 Analytics Archive

Add nightly export:

```text
SQLite
  ↓
JSONL export
  ↓
Parquet archive
  ↓
DuckDB queries
```

This allows serious analysis without bloating the relay.

---

# 16. Implementation Plan

## Phase 0: Contracts and Repository Setup

Goal: make the system schema-stable from day one.

Build:

- Monorepo layout
- JSON Schema definitions
- OpenAPI spec
- Event catalog
- Segment catalog
- Summary catalog
- Agent object catalog
- Versioning policy

Recommended repo layout:

```text
andrometry/
  android/
  server/
  schemas/
    events/
    segments/
    summaries/
    agent/
  agents/
    mcp/
    tools/
  docs/
  scripts/
  test/
    synthetic/
    golden/
```

Exit criteria:

- Schemas are versioned.
- API contract exists.
- Event types are enumerated.
- Compatibility rules are documented.

---

## Phase 1: Android Foundation

Goal: reliable local event kernel.

Build:

- App skeleton
- Permission screen
- Room schema
- Event envelope
- Sequence generator
- Collector status screen
- Event debug screen

Collect:

- Screen on/off
- Unlock
- Battery/charging
- Timezone
- Boot completed
- Permission state

Exit criteria:

- Events persist across reboot.
- Debug screen shows recent events.
- No excessive battery use.
- Sequence numbers are monotonic.

---

## Phase 2: App Usage and Notifications

Goal: understand digital behavior.

Build:

- UsageStatsManager worker
- App classification engine
- Notification listener service
- Notification metadata storage
- Five-minute or fifteen-minute summaries

Collect:

- App usage summary
- Category summary
- Notification summary
- Unlock count
- Screen-on minutes

Exit criteria:

App can answer:

- How much time in productivity apps?
- How much time in distracting apps?
- How many unlocks?
- How many notifications?
- Which categories dominated last hour?

---

## Phase 3: Context Engine

Goal: infer current state.

Build:

- Current context object
- Confidence scoring
- Data quality object
- Context snapshot worker

Infer:

- Primary activity
- Distraction probability
- Focus likelihood
- Interruption rate
- Fatigue probability
- Alignment probability

Exit criteria:

- Current context updates regularly.
- Context is readable in debug screen.
- Confidence and missing signals are visible.

---

## Phase 4: Timeline Segmentation

Goal: produce a semantic timeline.

Build:

- Segment engine
- Activity segments
- Focus segments
- Distraction segments
- Communication segments
- Sleep segments later

Store:

- Timeline segments locally
- Segment versions
- Segment confidence
- Segment evidence

Exit criteria:

App can show:

- What happened from 9:00 to 12:00?
- When was focus?
- When was distraction?
- When was communication?
- When was break?

---

## Phase 5: Batch Sync

Goal: reliable upload.

Build:

- Sync queue
- Batch uploader
- gzip compression
- Idempotency
- Retry/backoff
- Heartbeat
- Backfill handling

Upload:

- Events
- Segments
- Summaries
- Current context
- Permission state
- App health

Exit criteria:

- Phone can lose network for hours and sync later.
- Server rejects duplicate event IDs.
- Missing sequences are detectable.
- Sync survives reboot.

---

## Phase 6: Thin Relay Server

Goal: provide reliable agent endpoint.

Build:

- Go HTTP server
- SQLite schema
- Token auth
- Batch ingestion
- Current context endpoint
- Briefing endpoint
- Timeline endpoint
- Summary endpoint
- Event endpoint
- Observation endpoint
- Intervention endpoint

Deploy:

- systemd
- Caddy
- TLS
- backups

Exit criteria:

- Agents can query current context remotely.
- Agents can read timeline and daily summary.
- Duplicate batches are safe.
- API is stable and cheap to poll.

---

## Phase 7: Agent Loop

Goal: close the agentic loop.

Build:

- Agent observations
- Interventions
- Feedback
- Goals
- Habits
- Outcome tracking
- Intervention inbox
- Policy governor

Add:

- MCP server
- SSE optional
- Agent tool catalog

Exit criteria:

Agent can:

- Read context
- Read briefing
- Write observation
- Propose intervention
- Deliver intervention
- Receive feedback
- Evaluate outcome

---

## Phase 8: Location and Health

Add only after core pipeline is stable.

Build:

- Geofences
- Passive location
- Place inference
- Health Connect summaries
- Sleep summaries
- Activity recognition

Exit criteria:

- Location is semantic and low-power.
- Health summaries appear in daily summary.
- Battery impact remains acceptable.

---

## Phase 9: Analytics and Reflection

Goal: enable insight generation.

Build:

- Daily summary generator
- Weekly summary generator
- Baseline calculations
- Habit metrics
- Work pattern metrics
- Intervention effectiveness metrics

Optional:

- DuckDB export
- Parquet archive
- Scheduled insight generation

Exit criteria:

Agents can answer:

- How is today compared to baseline?
- Which habits are trending?
- Which interventions worked?
- What patterns cause distraction?
- How does sleep affect focus?

---

## Phase 10: Hardening

Goal: make it safe for long-term use.

Build:

- Token rotation
- Audit log
- Sensitive mode
- Delete/wipe controls
- Export controls
- Error reporting
- Battery profile switching
- Retention jobs
- Contract tests
- Replay tests

Exit criteria:

- System is safe for long-term personal use.
- User can pause or delete data easily.
- Agents operate within scopes.
- Schema changes do not break old clients.

---

# 17. Acceptance Criteria

The system is successful if agents can reliably answer these questions.

## Current State

- What am I doing right now?
- Where am I?
- Am I focused?
- Am I distracted?
- Am I in a meeting?
- Am I commuting?
- Am I exercising?
- Am I likely fatigued?
- Is my current behavior aligned with my goals?

---

## Recent Past

- What did I do in the last two hours?
- When did my focus session start?
- How many interruptions did I have?
- Which apps distracted me?
- Did I respond to many messages?
- Was I at the office?
- Did I take a break?

---

## Daily Analysis

- How much deep work did I get today?
- How much distraction did I have?
- How many meetings did I have?
- Did I meet my habits?
- How was my sleep?
- How many steps did I get?
- How was my notification load?
- How was my focus control?
- How does today compare to my baseline?

---

## Weekly Analysis

- Is my deep work increasing or decreasing?
- Is my distraction trending up or down?
- Am I sleeping regularly?
- Am I exercising enough?
- Which interventions helped?
- Which habits are strong?
- Which habits are failing?
- What patterns are causing misalignment?

---

## Agent Effectiveness

- Did the last intervention help?
- Did the user accept it?
- Did behavior change afterward?
- Did the relevant metric improve?
- Should this agent reduce its intervention frequency?
- Should this agent’s confidence be adjusted?

---

# 18. Risks and Mitigations

## 18.1 Android Background Restrictions

Risk: collectors killed or delayed.

Mitigation:

- Use WorkManager correctly.
- Use event-driven receivers.
- Use foreground service selectively.
- Request battery optimization exemption if needed.
- Show permission health screen.
- Detect degraded collectors.

---

## 18.2 Battery Drain

Risk: tracking becomes expensive.

Mitigation:

- Tiered collection.
- No continuous GPS.
- Batched writes.
- Batched sync.
- Adaptive profiles.
- Default Balanced/Eco mode.

---

## 18.3 Privacy Overcollection

Risk: system becomes creepy or unsafe.

Mitigation:

- Metadata by default.
- Redaction before sync.
- Sensitive mode.
- Local-first storage.
- Retention limits.
- Audit log.
- Kill switch.

---

## 18.4 Agent Noise

Risk: agents over-notify or give bad advice.

Mitigation:

- Confidence thresholds.
- Intervention rate limits.
- Feedback tracking.
- Outcome evaluation.
- Agent severity levels.
- Quiet hours.
- Repeated-dismissal suppression.

---

## 18.5 Data Gaps

Risk: missing data leads to wrong conclusions.

Mitigation:

- Data quality object.
- Permission state events.
- Collector health telemetry.
- Confidence penalties.
- Gap detection.
- Backfill requests.

---

## 18.6 Schema Drift

Risk: future changes break agents.

Mitigation:

- Schema versioning.
- Stable IDs.
- Additive fields.
- Event catalog.
- API versioning.
- Contract tests.

---

# 19. Final Recommended Canonical Architecture

Use this as the canonical build plan.

## Android App

Build:

- Kotlin
- Room/SQLite
- WorkManager
- BroadcastReceivers
- NotificationListenerService
- UsageStatsManager
- Health Connect
- Geofencing/passive location
- Calendar ContentObserver
- On-device event normalization
- On-device app classification
- On-device timeline segmentation
- On-device context snapshots
- On-device confidence scoring
- On-device privacy redaction
- Batched compressed HTTPS sync
- Intervention inbox
- Permission and collector health screens

---

## Thin Relay Server

Build:

- Single Go binary
- SQLite
- Caddy HTTPS
- Token auth with scopes
- Batch ingestion
- Current context API
- Briefing API
- Timeline API
- Summary API
- Event drill-down API
- Agent observation API
- Intervention API
- Feedback API
- Goal/habit API
- Outcome API
- Audit log
- Optional SSE
- Optional MCP server
- Optional Litestream backup

---

## AI Agents

Agents should:

- Poll current context with ETag
- Use briefings when available
- Read timeline when needed
- Read daily/weekly summaries
- Query segments instead of raw events
- Write observations
- Propose interventions
- Respect policy limits
- Track outcomes
- Adjust recommendations based on feedback

---

# 20. Most Important Refinement

The original architecture is already strong. The biggest refinement is this:

> Do not build merely a tracking pipeline.  
> Build a closed-loop personal alignment engine.

That means the system must support:

```text
Sensing
→ Semantic inference
→ Timeline reconstruction
→ Secure exposure
→ Agent interpretation
→ Intervention
→ Delivery
→ User response
→ Outcome attribution
→ Recalibration
```

If you build the system this way, your AI agents will not just know what you are doing.

They will be able to:

- Detect misalignment early.
- Understand why it happened.
- Intervene at the right moment.
- Learn which interventions help.
- Improve over time.
- Support genuine personal transformation.