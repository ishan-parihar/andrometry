# ANDROMETRY — Personal Context Engine for AI-Agent Alignment

This is an architectural plan for building an Android-first personal tracking system that gives your AI agents reliable, low-friction access to what you are doing, where you are, how focused you are, and how your day is unfolding.

The goal is not merely “logging data.”  
The goal is to create a **personal observability and alignment engine**:

- The Android app senses and normalizes your behavior.
- The phone stores and summarizes locally.
- A thin remote relay exposes secure endpoints.
- AI agents query current context, timeline, summaries, and patterns.
- Agents write observations, interventions, and feedback back into the system.
- The system improves its classification and usefulness over time.

This refined architecture keeps the strongest parts of your original document while adding the missing pieces required for a serious agentic loop: schema stability, timeline reconstruction, agent feedback, intervention tracking, data quality, privacy controls, and operational reliability.

---

# 1. Product Intent

## 1.1 Core Purpose

Build a self-contained personal digital observability system that enables AI agents to:

1. Know your current state with reasonable confidence.
2. Understand your recent behavior timeline.
3. Analyze daily and weekly patterns.
4. Detect misalignment, distraction, fatigue, or habit drift.
5. Provide timely coaching, reflection, and intervention.
6. Track whether their advice led to measurable improvement.

The system should track:

- App usage
- Productivity vs distraction time
- Screen on/off and unlocks
- Notification load
- Focus sessions
- Calendar context
- Calls/SMS metadata, preferably privacy-preserving
- Location semantics: home, office, gym, cafe, commute
- Sleep/wake patterns
- Steps and activity
- Device state: battery, charging, connectivity
- Agent observations and interventions
- User feedback and outcomes

## 1.2 Primary Design Constraint

The system must be:

- Battery-efficient
- Android-compliant
- Local-first
- Reliable offline
- Securely accessible remotely
- Agent-native
- Minimal infrastructure
- Privacy-conscious by default

You want sophisticated tracking, but it must not become a heavy telemetry platform. The correct design is:

> Rich semantic summaries + selective raw events + stable agent APIs.

Not:

> Continuous raw streaming of everything to the server.

---

# 2. Refined Design Principles

## 2.1 Local-first, remote-available

The phone is the primary sensing, classification, and storage node.

The server is a thin authenticated relay and query layer.

The app must function fully without the server. The server only becomes necessary when remote agents need access.

Flow:

```text
Collect
→ Normalize
→ Classify
→ Store locally
→ Aggregate
→ Sync selectively
→ Expose to agents
→ Receive agent observations
→ Track outcomes
```

## 2.2 Semantic over raw

Agents usually do not need:

- Every GPS coordinate
- Every notification body
- Every raw app transition
- Every sensor sample

Agents usually need:

- “You are at the office.”
- “You have been in deep work for 37 minutes.”
- “Your distraction probability is rising.”
- “You slept poorly and are likely fatigued.”
- “You have a meeting in 12 minutes.”
- “Your productivity alignment today is lower than baseline.”

Therefore, the phone should produce semantic objects:

```json
{
  "primary_activity": "deep_work",
  "place": "office",
  "focus_active": true,
  "distraction_probability": 0.18,
  "fatigue_probability": 0.31
}
```

Raw events are retained for drill-down, but agents should normally consume summaries and context objects.

## 2.3 Event-driven over polling

Prefer OS callbacks and scheduled batch workers over continuous polling.

Good:

- Screen on/off broadcasts
- Notification listener callbacks
- Geofence enter/exit
- Calendar ContentObserver
- Health Connect periodic reads
- WorkManager periodic summaries

Bad:

- Polling app usage every second
- Continuous high-accuracy GPS
- Persistent wake locks
- Always-on WebSocket from phone
- Immediate upload of every event

## 2.4 Batched, compressed, idempotent sync

The phone should upload in batches:

- Every 10–15 minutes
- On Wi-Fi/charging
- On significant context change
- When app opens
- Before device sleep if pending data exists

Every batch should be:

- Compressed
- Authenticated
- Idempotent
- Sequence-numbered
- Retry-safe

## 2.5 Agent-native data products

The system should expose not just events, but agent-ready products:

1. Current context
2. Recent timeline segments
3. Daily summary
4. Weekly summary
5. Habit metrics
6. Focus session history
7. Location visit history
8. App category history
9. Notification load history
10. Health summary
11. Agent observations
12. Intervention outcomes
13. User feedback

This is what makes the system useful for autonomous agents.

## 2.6 Closed-loop agentic architecture

The system must support this loop:

```text
Sense
→ Infer
→ Store
→ Expose
→ Agent interprets
→ Agent observes/intervenes
→ User responds
→ Outcome recorded
→ Model recalibrates
```

Without the feedback and outcome layer, agents can comment but cannot improve.

Therefore, this refined architecture adds:

- Goals
- Agent observations
- Interventions
- User feedback
- Outcome tracking
- Baseline metrics
- Confidence scores
- Data quality signals

---

# 3. What to Keep, Expand, Contract, and Add

## 3.1 Keep from original architecture

Keep these ideas:

- Android app as primary logger and classifier
- Thin relay server
- SQLite on device and server
- Batched HTTPS sync
- On-device summarization
- Low-power tiered collection
- Geofence-based location
- Health Connect summaries
- Current context object
- Daily summary object
- Minimal Go/Rust/Python server
- Caddy reverse proxy
- Optional Litestream backup
- ETag-based cheap agent polling
- MCP tool exposure

These are correct.

## 3.2 Expand

Expand these areas:

### Timeline reconstruction

The original document mentions timeline queries, but the architecture needs explicit timeline segment tables:

- Activity segments
- Focus segments
- Place visits
- App usage intervals
- Sleep intervals
- Meeting intervals
- Communication intervals
- Distraction intervals

Agents need to ask:

> “What was I doing from 9:00 to 11:30?”

The answer should be a clean semantic timeline, not a pile of raw events.

### Agent feedback loop

Add:

- Agent observations
- Agent interventions
- User feedback
- Outcome events
- Goal tracking
- Baseline comparison

This turns the system from passive logging into an alignment engine.

### Data quality and provenance

Every context inference should include:

- Confidence
- Source signals
- Missing signals
- Staleness
- Permission state
- Device health

Agents need to know when not to trust a conclusion.

Example:

```json
{
  "primary_activity": "deep_work",
  "confidence": 0.62,
  "data_quality": {
    "calendar_available": false,
    "health_available": true,
    "location_fresh": false,
    "usage_stats_available": true
  }
}
```

### Schema versioning

The system will evolve. Add:

- `schema_version`
- `event_version`
- `client_version`
- `device_id`
- `collector_id`

This prevents future pain.

### Security and governance

Add:

- Token scopes
- Device provisioning
- Token rotation
- Audit log
- Sensitive mode
- Redaction rules
- Retention policies
- Encryption at rest
- Permission state tracking

This is required for a personal surveillance-grade system.

## 3.3 Contract or remove

Do not build these as default features:

- Continuous high-accuracy GPS
- Real-time notification content streaming
- Raw SMS bodies
- Raw call logs
- Continuous sensor streaming
- Heavy Postgres/Kafka/Redis stack
- Docker-heavy microservices
- Real-time analytics pipeline
- Full event streaming from phone
- Permanent raw event storage

These increase battery drain, privacy risk, and infrastructure bloat.

## 3.4 Add missing components

Add these new components (see dedicated sections for each):

1. **Schema Registry / Event Catalog** — §10 Event Taxonomy
2. **Timeline Segment Engine** — §12 Timeline Reconstruction
3. **Context Classifier** — §11 Context Inference Engine
4. **Confidence and Data Quality Engine** — §11.3, §11.4
5. **Agent Observation Ledger** — §5 Agentic Loop Architecture
6. **Intervention and Outcome Tracker** — §34 Intervention Delivery, §35 Outcome Attribution
7. **Goal and Baseline Store** — §36 Baseline and Calibration Store
8. **Permission and Privacy State Tracker** — §7 Android Permissions
9. **System Health Telemetry** — §23.5 App Health Telemetry
10. **MCP Server / Agent SDK** — §18 MCP Server
11. **Optional Analytics Worker** — §20 Insight and Analytics Layer
12. **Export and Backup Layer** — §20.4 Analytics Archive
13. **Policy Governor** — §33 Policy Governor
14. **User Correction Loop** — §37 User Correction Loop
15. **Habit Engine** — §38 Habit Engine
16. **Work Pattern Metrics** — §39 Work Pattern Metrics
17. **Readiness/Fatigue Model** — §40 Readiness and Fatigue Model
18. **Testing and Simulation Harness** — §41 Testing and Simulation Harness

---

# 4. High-Level Architecture

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
│  - Calls/SMS metadata via notification metadata              │
│  - Focus sessions                                            │
│  - Device state                                              │
│                                                              │
│  Processing                                                  │
│  - Event normalization                                       │
│  - Category classification                                   │
│  - Timeline segmentation                                     │
│  - Context inference                                         │
│  - Confidence scoring                                        │
│  - Data quality scoring                                      │
│  - Baseline comparison                                       │
│  - Summarization                                             │
│  - Privacy redaction                                         │
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
│  - Baselines                                                 │
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
│  - Agent auth + control plane                                │
│  - Batch ingestion                                           │
│  - Current context cache                                     │
│  - Briefing API                                              │
│  - Timeline API                                              │
│  - Summary API                                               │
│  - Event drill-down API                                      │
│  - Agent observation API                                     │
│  - Intervention API                                          │
│  - Feedback API                                              │
│  - Goal/habit API                                            │
│  - Outcome API                                               │
│  - Policy governor                                           │
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
│  - Timeline drill-down                                       │
│  - Daily/weekly analysis                                     │
│  - Habit tracking                                            │
│  - Alignment scoring                                         │
│  - Risk detection                                            │
│  - Intervention generation                                   │
│  - Outcome evaluation                                        │
│  - Reflection generation                                     │
└──────────────────────────────────────────────────────────────┘
```

---

# 5. The Agentic Loop Architecture

This is the most important refinement.

Your system should not merely provide data. It should support an autonomous alignment loop.

## 5.1 Loop layers

### Layer 1: Sensing

The phone collects signals:

- App usage
- Screen state
- Unlocks
- Notifications
- Calendar
- Location
- Health
- Communication metadata
- Focus sessions
- Device state

### Layer 2: Inference

The phone infers semantic state:

- Primary activity
- Place
- Movement
- Focus state
- Distraction risk
- Fatigue risk
- Interruption load
- Work pattern
- Habit adherence

### Layer 3: Storage

The phone stores:

- Raw events
- Summaries
- Segments
- Current context
- Daily metrics
- Agent observations
- Interventions
- Feedback

### Layer 4: Exposure

The server exposes:

- Current context
- Timeline
- Summaries
- Events
- Habits
- Goals
- Observations
- Interventions
- Feedback

### Layer 5: Agent Interpretation

Agents query:

```text
What is the user doing now?
Is this aligned with their goals?
What happened in the last two hours?
What patterns are emerging?
Is distraction increasing?
Is fatigue likely?
Did the last intervention help?
```

### Layer 6: Agent Action

Agents write:

- Observations
- Risks
- Recommendations
- Interventions
- Reflections
- Goal updates
- Experiment proposals

### Layer 7: Outcome Tracking

The system records:

- Did the user accept the intervention?
- Did behavior change?
- Did the metric improve?
- Was the prediction correct?
- Was the advice useful?

### Layer 8: Recalibration

The system adjusts:

- Classification weights
- Risk thresholds
- Baselines
- Agent confidence
- Notification frequency
- Tracking profile

This creates a true agentic system, not just a dashboard backend.

## 5.2 Agent Control Plane

A dedicated layer for agent interaction should be built into the server.

Responsibilities:

- Authenticate agents
- Scope agent access
- Rate limit interventions
- Enforce quiet hours
- Audit agent actions
- Provide briefings
- Track outcomes
- Recalibrate confidence

The control plane ensures agents operate within boundaries and cannot spam the user or write arbitrary data.

## 5.3 Agent Query Ladder

Agents should not read raw events first. Use this escalation ladder:

```text
Level 1: current context
Level 2: current briefing
Level 3: recent timeline
Level 4: daily summary
Level 5: segment history
Level 6: event drill-down
Level 7: raw export/analytics
```

This reduces token usage and API load. Agents should always start at the highest abstraction that answers their question.

---

# 6. Android App Architecture

## 6.1 Recommended stack

Use:

- Kotlin
- Room/SQLite
- WorkManager
- Coroutines/Flow
- Hilt or Koin for dependency injection
- DataStore for preferences
- Retrofit/Ktor for networking
- OkHttp with gzip
- Health Connect SDK
- Fused Location Provider
- Geofencing API
- NotificationListenerService
- UsageStatsManager
- Calendar ContentObserver
- Optional foreground service for active modes

## 6.2 App modules

Structure the app into clear modules:

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

## 6.3 Collector Matrix

This refines the collection strategy and corrects Android-specific constraints.

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

> Periodic WorkManager jobs cannot reliably run every 5 minutes. The minimum periodic interval is normally 15 minutes. If you need 1–5 minute granularity, use an active mode foreground service during work hours, focus sessions, or charging. Otherwise default to 15-minute batch collection.

## 6.4 Minimal UI

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

# 7. Android Permissions and OS Constraints

This system must work within Android’s model. Trying to bypass Android restrictions will cause instability and battery drain.

## 7.1 Required permissions

Depending on features:

### Core

- `PACKAGE_USAGE_STATS`
- `POST_NOTIFICATIONS`
- `RECEIVE_BOOT_COMPLETED`
- `FOREGROUND_SERVICE`
- `FOREGROUND_SERVICE_DATA_SYNC` or appropriate type
- `INTERNET`
- `ACCESS_NETWORK_STATE`

### Notifications

- Notification listener permission granted by user in system settings

### Location

- `ACCESS_COARSE_LOCATION`
- `ACCESS_FINE_LOCATION`
- `ACCESS_BACKGROUND_LOCATION` if geofencing/place inference requires it

### Health

- Health Connect permissions:
  - steps
  - sleep
  - heart rate
  - exercise sessions
  - active calories if desired

### Calendar

- `READ_CALENDAR`

### Calls/SMS

Avoid direct sensitive permissions initially.

Prefer:

- Notification metadata from communication apps

If later required:

- `READ_CALL_LOG`
- `READ_SMS`

But only if absolutely necessary and with explicit consent.

## 7.2 Permission state tracking

The app should continuously track permission state:

```json
{
  "event_type": "permission_state",
  "timestamp": "2026-07-28T10:00:00Z",
  "payload": {
    "usage_stats": true,
    "notification_listener": true,
    "notification_listener_restricted": false,
    "calendar": true,
    "location": true,
    "background_location": false,
    "health_connect": true,
    "health_connect_steps": true,
    "health_connect_sleep": true,
    "health_connect_heart_rate": false,
    "battery_optimization_exempt": true
  }
}
```

Agents need this because missing permissions explain missing data.

## 7.3 Notification listener details

Requires user to grant notification access in system settings.

Track:

```json
{
  "notification_listener_enabled": true,
  "notification_listener_restricted": false
}
```

Android may restrict notification access for some apps. The app should detect and surface this.

## 7.4 Location details

Default:

```text
ACCESS_COARSE_LOCATION
ACCESS_FINE_LOCATION
ACCESS_BACKGROUND_LOCATION
```

But only use background location for geofencing/place inference. Do not continuously track GPS.

## 7.5 Health Connect details

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

## 7.6 Calendar details

Use:

```text
READ_CALENDAR
```

Default sync policy:

- Store titles locally only
- Sync categories and busy/free status
- Redact titles before server sync unless explicitly enabled

## 7.7 Calls/SMS details

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

# 8. Tiered Collection Strategy

Keep the tiered model, but refine it.

## 8.1 Tier 0: Always-on, extremely low power

Event-driven only.

Signals:

- Screen on
- Screen off
- User unlock
- Boot completed
- Battery state
- Charging state
- Timezone change
- Notification posted/removed
- Geofence enter/exit
- Permission changes

Implementation:

- BroadcastReceiver
- NotificationListenerService
- GeofencingClient

Battery impact: low.

## 8.2 Tier 1: Periodic low-power workers

Use WorkManager.

Recommended intervals:

| Signal | Interval |
|---|---:|
| App usage summary | 5 minutes |
| Notification summary | 5 minutes |
| Calendar sync | 15 minutes |
| Location reconciliation | 15 minutes |
| Health Connect sync | 30–60 minutes |
| Context snapshot | 5 minutes |
| Timeline segmentation | 5–15 minutes |
| Daily summary rebuild | hourly and nightly |
| Sync upload | 10–15 minutes |

Important Android refinement:

> Periodic WorkManager jobs cannot reliably run every 5 minutes. The minimum periodic interval is normally 15 minutes. If you need 1–5 minute granularity, use an active mode foreground service during work hours, focus sessions, or charging. Otherwise default to 15-minute batch collection.

Constraints:

```kotlin
Constraints.Builder()
    .setRequiredNetworkType(NetworkType.CONNECTED)
    .setRequiresBatteryNotLow(true)
    .setRequiresStorageNotLow(true)
    .build()
```

For heavy sync, optionally require charging or Wi-Fi.

## 8.3 Tier 2: Active mode

Enable only when useful:

- During work hours
- During focus sessions
- While charging
- When battery > 50%
- When user enables high-fidelity mode

Active mode may collect:

- App usage every 1–2 minutes
- Context snapshot every 1 minute
- More frequent passive location
- Notification interaction latency
- Focus session state changes

Do not run this all day by default.

## 8.4 Tier 3: Debug mode

For testing only.

May include:

- Raw event logging
- Frequent location samples
- Detailed app transitions
- Diagnostic logs
- Sync traces

Disabled in normal use.

---

# 9. Data Collection Methods

## 9.1 App usage

Do not continuously monitor app switches.

Every 5 minutes:

1. Query `UsageStatsManager`.
2. Compute foreground intervals.
3. Classify apps into categories.
4. Aggregate by category.
5. Store summary.
6. Optionally store top apps.

Example:

```json
{
  "event_type": "app_usage_summary",
  "schema_version": 1,
  "start": "2026-07-28T10:00:00Z",
  "end": "2026-07-28T10:05:00Z",
  "payload": {
    "by_category": {
      "productivity": 184,
      "communication": 42,
      "social": 38,
      "browser": 36
    },
    "top_apps": [
      {
        "package": "com.notion.id",
        "category": "productivity",
        "seconds": 154
      },
      {
        "package": "com.slack",
        "category": "communication",
        "seconds": 42
      }
    ],
    "app_switch_count": 7
  }
}
```

### App classification system

Create a local app category engine.

Categories:

- productivity
- communication
- calendar
- email
- browser
- social
- entertainment
- news
- finance
- health
- travel
- shopping
- maps
- system
- unknown

Classification sources:

1. Package allowlist/denylist
2. Android app category
3. User overrides
4. Remote category pack
5. Heuristic label matching

Store user overrides locally and sync to server.

Example:

```json
{
  "package": "com.notion.id",
  "category": "productivity",
  "subcategory": "notes",
  "alignment": "aligned",
  "source": "user_override"
}
```

This is important because “productivity” is personal. Notion may be work for one person and distraction for another.

## 9.2 Screen and unlocks

Use broadcasts:

- `ACTION_SCREEN_ON`
- `ACTION_SCREEN_OFF`
- `ACTION_USER_PRESENT`

Store:

```json
{
  "event_type": "screen_off",
  "timestamp": "2026-07-28T23:41:12Z"
}
```

Aggregate:

```json
{
  "unlock_count": 58,
  "first_unlock": "2026-07-28T07:12:00Z",
  "last_screen_off": "2026-07-28T23:58:00Z",
  "screen_on_minutes": 312
}
```

Also derive:

- Session length
- Late-night usage
- Morning first unlock
- Screen-free intervals
- Bedtime regularity

## 9.3 Notifications

Use `NotificationListenerService`.

Default: store metadata only.

For each notification event:

```json
{
  "event_type": "notification_event",
  "timestamp": "2026-07-28T10:01:12Z",
  "payload": {
    "package": "com.slack",
    "category": "communication",
    "posted": true,
    "removed": false,
    "dismissed": false,
    "clicked": false
  }
}
```

Avoid storing:

- Full title
- Full text
- Sender identity
- Message body

Unless explicitly enabled.

Summarize every 5 minutes:

```json
{
  "event_type": "notification_summary",
  "start": "2026-07-28T10:00:00Z",
  "end": "2026-07-28T10:05:00Z",
  "payload": {
    "total": 6,
    "by_category": {
      "communication": 3,
      "work": 2,
      "social": 1
    },
    "interruption_level": "medium",
    "response_latency_seconds_avg": 84
  }
}
```

Add notification load metrics:

- Notifications per hour
- Interruption density
- Communication interruptions during focus
- Late-night notifications
- Response latency

## 9.4 Location

Location is the biggest battery risk.

Use this hierarchy:

1. Geofences for known places
2. Passive location provider
3. Significant location changes
4. Low-frequency fused location updates
5. High-frequency GPS only in debug mode

Known places:

- home
- office
- gym
- cafe
- library
- commute
- unknown

Send semantic location:

```json
{
  "event_type": "location_context",
  "timestamp": "2026-07-28T10:05:00Z",
  "payload": {
    "place": "office",
    "confidence": 0.91,
    "movement": "still",
    "changed": false
  }
}
```

Do not send exact coordinates by default.

Store place visits as segments:

```json
{
  "segment_type": "place_visit",
  "start": "2026-07-28T09:12:00Z",
  "end": "2026-07-28T12:41:00Z",
  "payload": {
    "place": "office",
    "confidence": 0.93
  }
}
```

This is much more useful to agents than raw coordinates.

## 9.5 Health data

Use Health Connect periodically.

Do not stream sensors.

Recommended:

- Steps: every 30–60 minutes
- Sleep: morning read plus nightly reconciliation
- Heart rate: every 30–60 minutes if available
- Exercise sessions: event-based or hourly
- Active calories: optional

Example:

```json
{
  "event_type": "health_summary",
  "start": "2026-07-28T09:00:00Z",
  "end": "2026-07-28T10:00:00Z",
  "payload": {
    "steps": 842,
    "active_minutes": 7,
    "heart_rate_avg": 74,
    "heart_rate_min": 62,
    "heart_rate_max": 98
  }
}
```

Sleep summary:

```json
{
  "event_type": "sleep_summary",
  "date": "2026-07-27",
  "payload": {
    "sleep_minutes": 460,
    "bedtime": "2026-07-27T23:41:00+05:30",
    "wake_time": "2026-07-28T07:21:00+05:30",
    "regularity_score": 0.82,
    "in_bed_minutes": 480
  }
}
```

## 9.6 Calendar

Use `ContentObserver` plus periodic reconciliation.

On calendar change:

- Sync changed events

Every 15 minutes:

- Sync next 24 hours

Store metadata:

```json
{
  "event_type": "calendar_context",
  "timestamp": "2026-07-28T10:00:00Z",
  "payload": {
    "current_event": "none",
    "next_event_in_minutes": 45,
    "next_event_category": "meeting",
    "busy_until": "2026-07-28T12:30:00Z"
  }
}
```

Avoid storing full meeting titles by default.

Optional categories:

- meeting
- personal
- health
- travel
- focus
- errand
- unknown

If titles are needed for agent usefulness, store them only locally or redact/hash them before sync.

## 9.7 Calls/SMS metadata

This is sensitive.

Preferred approach:

- Use communication notification metadata.

Avoid:

- Message bodies
- Phone numbers
- Contact names
- Call recordings
- Raw SMS content

If direct call/SMS metadata is required, store summaries only:

```json
{
  "event_type": "communication_summary",
  "start": "2026-07-28T09:00:00Z",
  "end": "2026-07-28T10:00:00Z",
  "payload": {
    "calls_incoming": 1,
    "calls_outgoing": 0,
    "call_minutes": 4,
    "sms_received": 3,
    "sms_sent": 0,
    "contact_hashes": [
      "hash_1",
      "hash_2"
    ]
  }
}
```

Use salted hashes for contacts if contact identity is needed for pattern recognition.

## 9.8 Focus sessions

Focus sessions should be first-class objects.

They can come from:

- Manual start in app
- Android focus mode
- Calendar focus blocks
- Inferred deep work
- Agent-suggested sessions

Store:

```json
{
  "event_type": "focus_session",
  "timestamp": "2026-07-28T09:37:00Z",
  "payload": {
    "action": "start",
    "source": "manual",
    "intended_duration_minutes": 90,
    "goal": "deep work"
  }
}
```

End:

```json
{
  "event_type": "focus_session",
  "timestamp": "2026-07-28T11:02:00Z",
  "payload": {
    "action": "end",
    "reason": "manual_end",
    "duration_minutes": 85,
    "interruptions": 3,
    "distraction_minutes": 6,
    "success_score": 0.81
  }
}
```

Also create a focus segment:

```json
{
  "segment_type": "focus_session",
  "start": "2026-07-28T09:37:00Z",
  "end": "2026-07-28T11:02:00Z",
  "payload": {
    "source": "manual",
    "interruptions": 3,
    "distraction_minutes": 6,
    "success_score": 0.81
  }
}
```

Agents should be able to evaluate focus quality over time.

---

# 10. Event Taxonomy

Define a stable event catalog.

## 10.1 Event envelope

Every event should have a common envelope:

```json
{
  "event_id": "uuid",
  "device_id": "pixel-9-pro",
  "schema_version": 1,
  "event_version": 1,
  "client_version": "0.1.0",
  "timestamp": "2026-07-28T10:05:00Z",
  "timezone": "Asia/Kolkata",
  "utc_offset_seconds": 19800,
  "event_type": "app_usage_summary",
  "collector": "usage_stats",
  "source": "workmanager",
  "sequence": 10231,
  "payload": {}
}
```

Important fields:

- `event_id`: globally unique
- `sequence`: monotonic per device
- `schema_version`: for compatibility
- `timezone`: essential for sleep/circadian analysis
- `collector`: helps debug missing data
- `source`: helps distinguish inferred vs observed events

## 10.2 Recommended event types

### Device and system

- `boot_completed`
- `screen_state_changed`
- `device_unlocked`
- `battery_state_changed`
- `charging_state_changed`
- `timezone_changed`
- `connectivity_changed`
- `permission_state_changed`
- `app_health`
- `sync_heartbeat`

### App usage

- `app_usage_summary`
- `app_category_summary`
- `app_switch_burst`
- `app_classification_updated`

### Notifications

- `notification_event`
- `notification_summary`
- `notification_interruption_summary`

### Focus

- `focus_session`
- `focus_state_changed`
- `focus_interruption`

### Calendar

- `calendar_context`
- `calendar_event_changed`

### Location

- `location_context`
- `place_visit_started`
- `place_visit_ended`
- `geofence_transition`
- `movement_context`

### Health

- `health_summary`
- `sleep_summary`
- `activity_recognition`
- `exercise_session`

### Communication

- `communication_summary`
- `communication_notification_summary`

### Context

- `context_snapshot`
- `activity_segment`
- `daily_summary`
- `weekly_summary`

### Agent loop

- `agent_observation`
- `agent_intervention`
- `user_feedback`
- `intervention_outcome`
- `goal_updated`
- `habit_updated`
- `baseline_updated`
- `user_correction`

---

# 11. Context Inference Engine

This is the heart of the system.

The phone should maintain a continuously updated current-context object.

## 11.1 Current context object

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
      "missing_signals": [],
      "missing_collectors": [],
      "degraded_collectors": [],
      "permission_issues": [],
      "clock_drift_detected": false
    }
  }
}
```

This is the object agents should usually poll.

## 11.2 Activity classification

Possible primary activities:

- `deep_work`
- `shallow_work`
- `meeting`
- `communication`
- `learning`
- `admin`
- `planning`
- `break`
- `leisure`
- `social`
- `distraction`
- `exercise`
- `commute`
- `sleep`
- `unknown`

Classification inputs:

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
- User-defined rules

Example rule:

```text
If:
  focus_active = true
  foreground_app_category = productivity
  app_switch_count_last_5m < 4
  notification_interruption_level = low
Then:
  primary_activity = deep_work
  confidence += 0.15
```

## 11.3 Confidence scoring

Every inference should include confidence.

Confidence factors:

- Signal freshness
- Signal agreement
- Permission availability
- Historical accuracy
- Data completeness
- Timezone stability
- Location confidence
- Calendar availability

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

Agents should use confidence to decide whether to act.

## 11.4 Data quality signals

Add a data quality object to every context snapshot:

```json
{
  "completeness": 0.87,
  "staleness_seconds": 42,
  "missing_collectors": [],
  "degraded_collectors": ["health_connect"],
  "permission_issues": [],
  "clock_drift_detected": false
}
```

This prevents agents from making strong claims on weak data.

---

# 12. Timeline Reconstruction

This is a major refinement.

Agents need a semantic timeline, not only raw events.

## 12.1 Timeline segment types

Create materialized segments:

- `activity_segment`
- `focus_segment`
- `place_visit`
- `app_usage_segment`
- `meeting_segment`
- `communication_segment`
- `sleep_segment`
- `distraction_segment`
- `break_segment`
- `exercise_segment`
- `commute_segment`

## 12.2 Example timeline response

```json
{
  "timezone": "Asia/Kolkata",
  "timeline": [
    {
      "segment_id": "uuid",
      "segment_type": "sleep_segment",
      "start": "2026-07-27T23:41:00+05:30",
      "end": "2026-07-28T07:21:00+05:30",
      "label": "sleep",
      "confidence": 0.88
    },
    {
      "segment_id": "uuid",
      "segment_type": "place_visit",
      "start": "2026-07-28T08:35:00+05:30",
      "end": "2026-07-28T12:41:00+05:30",
      "label": "office",
      "confidence": 0.93
    },
    {
      "segment_id": "uuid",
      "segment_type": "focus_segment",
      "start": "2026-07-28T09:37:00+05:30",
      "end": "2026-07-28T11:02:00+05:30",
      "label": "deep_work",
      "confidence": 0.86,
      "payload": {
        "interruptions": 3,
        "distraction_minutes": 6
      }
    },
    {
      "segment_id": "uuid",
      "segment_type": "meeting_segment",
      "start": "2026-07-28T11:30:00+05:30",
      "end": "2026-07-28T12:15:00+05:30",
      "label": "meeting",
      "confidence": 0.79
    }
  ]
}
```

## 12.3 Segment generation rules

The app should run a segmentation worker every 5–15 minutes.

It should:

1. Read recent raw events.
2. Merge related events.
3. Infer segment boundaries.
4. Assign labels.
5. Assign confidence.
6. Store segments locally.
7. Sync changed segments.

Example:

```text
If:
  productivity app usage > 20 minutes
  app switches low
  no meeting
  notifications low
Then:
  create deep_work segment
```

Example:

```text
If:
  social app usage > 10 minutes
  during intended focus
  app switches high
Then:
  create distraction segment
```

## 12.4 Segment correction

Segments may be corrected later.

Examples:

- Calendar sync reveals a meeting.
- Sleep data arrives late.
- Location confidence improves.
- User manually corrects a segment.

Use versioned segments:

```json
{
  "segment_id": "uuid",
  "version": 3,
  "start": "...",
  "end": "...",
  "label": "meeting",
  "confidence": 0.84,
  "revision_reason": "calendar_sync"
}
```

The server should upsert by `segment_id`.

## 12.5 Semantic timeline layers

The timeline should not be a single flat list. It should be layered.

Recommended layers:

1. **Activity Layer** — deep_work, shallow_work, meeting, communication, learning, admin, planning, break, leisure, social, distraction, exercise, commute, sleep, unknown
2. **Place Layer** — home, office, gym, cafe, library, commute, unknown
3. **Focus Layer** — focus_session, inferred_focus, interrupted_focus, failed_focus
4. **Communication Layer** — calls, messages, email, chat, notification-heavy intervals
5. **Health Layer** — sleep, exercise, steps, inactivity, elevated heart rate
6. **Calendar Layer** — meeting, personal, health, travel, focus block, errand
7. **Device Layer** — screen on/off, unlock bursts, charging, battery state

Agents can query layers independently or request a merged timeline.

## 12.6 Bitemporal segment object

Every segment must support correction over time. Use bitemporal fields:

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

The timeline engine must preserve:

- What it believed at the time
- What the corrected truth is now
- When the correction happened
- Why the correction happened

## 12.7 Overlap and gap handling

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

# 13. Local Database Design

Use Room/SQLite.

## 13.1 Core tables

```sql
CREATE TABLE events (
  event_id TEXT PRIMARY KEY,
  sequence INTEGER NOT NULL,
  timestamp TEXT NOT NULL,
  timezone TEXT NOT NULL,
  event_type TEXT NOT NULL,
  collector TEXT,
  source TEXT,
  schema_version INTEGER NOT NULL,
  payload TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_sync_state ON events(sync_state);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_sequence ON events(sequence);
```

```sql
CREATE TABLE current_context (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  timestamp TEXT NOT NULL,
  timezone TEXT NOT NULL,
  context TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE timeline_segments (
  segment_id TEXT PRIMARY KEY,
  segment_type TEXT NOT NULL,
  layer TEXT,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  label TEXT NOT NULL,
  confidence REAL,
  version INTEGER NOT NULL DEFAULT 1,
  valid_from TEXT NOT NULL,
  valid_to TEXT,
  ingested_at TEXT NOT NULL,
  revised_at TEXT,
  revision_reason TEXT,
  evidence TEXT,
  payload TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);

CREATE INDEX idx_segments_time ON timeline_segments(start_time, end_time);
CREATE INDEX idx_segments_type ON timeline_segments(segment_type);
```

```sql
CREATE TABLE five_minute_summary (
  start_time TEXT PRIMARY KEY,
  end_time TEXT NOT NULL,
  summary TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE hourly_summary (
  start_time TEXT PRIMARY KEY,
  end_time TEXT NOT NULL,
  summary TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE daily_summary (
  date TEXT PRIMARY KEY,
  timezone TEXT NOT NULL,
  summary TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE agent_observations (
  observation_id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  agent_id TEXT,
  observation_type TEXT,
  severity TEXT,
  payload TEXT NOT NULL,
  delivered INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

```sql
CREATE TABLE interventions (
  intervention_id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  agent_id TEXT,
  intervention_type TEXT,
  status TEXT NOT NULL DEFAULT 'proposed',
  payload TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE user_feedback (
  feedback_id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  feedback_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE goals (
  goal_id TEXT PRIMARY KEY,
  updated_at TEXT NOT NULL,
  goal_type TEXT NOT NULL,
  status TEXT NOT NULL,
  payload TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE habits (
  habit_id TEXT PRIMARY KEY,
  habit_type TEXT NOT NULL,
  target TEXT NOT NULL,
  schedule TEXT NOT NULL,
  evidence_sources TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  streak_days INTEGER NOT NULL DEFAULT 0,
  habit_strength REAL,
  exception_rules TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE baselines (
  baseline_id TEXT PRIMARY KEY,
  metric_name TEXT NOT NULL,
  window_days INTEGER NOT NULL DEFAULT 7,
  value REAL NOT NULL,
  computed_at TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE app_classifications (
  package_name TEXT PRIMARY KEY,
  category TEXT NOT NULL,
  subcategory TEXT,
  alignment TEXT,
  source TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

```sql
CREATE TABLE permission_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  updated_at TEXT NOT NULL,
  state TEXT NOT NULL,
  sync_state TEXT NOT NULL DEFAULT 'pending'
);
```

## 13.2 SQLite performance settings

Use:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -8000;
PRAGMA mmap_size = 134217728;
```

Room can be configured with these pragmas.

## 13.3 Local retention policy

Recommended:

| Data type | Retention |
|---|---:|
| Raw events | 7–14 days |
| Five-minute summaries | 30–90 days |
| Hourly summaries | 90–365 days |
| Daily summaries | indefinite |
| Timeline segments | 90–365 days |
| Current context | latest only |
| Agent observations | 90 days |
| Interventions | 90 days |
| Outcomes | 90 days |
| User feedback | indefinite |
| Goals | indefinite |
| Habits | indefinite |
| Baselines | 90 days |

Run nightly compaction:

1. Aggregate raw events.
2. Rebuild daily summary.
3. Delete expired raw events.
4. Vacuum occasionally.

---

# 14. Sync Strategy

## 14.1 Upload triggers

Upload when:

- Every 10–15 minutes
- Context changes significantly
- Wi-Fi and charging are available
- App opens
- Before device sleep if pending batch exists
- Agent urgently requests fresh data, if allowed

Do not upload continuously.

## 14.2 Batch format

```json
{
  "device_id": "pixel-9-pro",
  "batch_id": "uuid",
  "client_version": "0.1.0",
  "schema_version": 1,
  "sequence_start": 10231,
  "sequence_end": 10287,
  "sent_at": "2026-07-28T10:15:00Z",
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

Use:

- gzip compression
- HTTPS
- idempotency keys
- sequence numbers
- retry with exponential backoff

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

## 14.4 Delta sync

Only send changes.

Examples:

- Location unchanged: send heartbeat only.
- Context unchanged: send minimal heartbeat.
- Segment unchanged: do not resend.
- Daily summary changed: send new version.

Minimal heartbeat:

```json
{
  "event_type": "sync_heartbeat",
  "timestamp": "2026-07-28T10:15:00Z",
  "payload": {
    "battery": 0.74,
    "charging": false,
    "storage_free_mb": 42000,
    "queue_size": 12,
    "last_context_change_minutes_ago": 3
  }
}
```

## 14.5 Gap detection

The server should detect missing sequences:

```text
Received sequence 1000, then 1005.
Missing: 1001–1004.
```

Server can request backfill:

```json
{
  "action": "request_backfill",
  "sequence_from": 1001,
  "sequence_to": 1004
}
```

The app can fetch this from server response and backfill on next sync.

---

# 15. Server Architecture

Keep it minimal.

## 15.1 Recommended production-minimal stack

Use:

- Ubuntu VPS or Raspberry Pi
- Single Go binary
- SQLite
- Caddy reverse proxy
- systemd
- Litestream optional
- Tailscale optional

Why Go:

- Small memory footprint
- Single static binary
- Easy systemd deployment
- Good HTTP server
- Good concurrency
- No heavy runtime

Python/FastAPI is acceptable for prototype.  
Go is better for minimal production infra.

## 15.2 Server responsibilities

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
- Track goals
- Optionally stream updates via SSE/webhook
- Optionally expose MCP tools

It should not initially do:

- Heavy analytics
- Stream processing
- ML training
- Complex orchestration
- Microservices

Analytics can be done later with:

- DuckDB
- SQLite views
- Python notebooks
- Scheduled scripts
- Agent-driven analysis

## 15.3 Server retention policy

Recommended:

| Data type | Retention |
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

# 16. Server Database Schema

SQLite is enough for a single-user system.

```sql
CREATE TABLE devices (
  device_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TEXT,
  client_version TEXT,
  status TEXT NOT NULL DEFAULT 'active'
);
```

```sql
CREATE TABLE auth_tokens (
  token_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  token_type TEXT NOT NULL,
  scopes TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TEXT,
  revoked_at TEXT
);
```

```sql
CREATE TABLE events (
  event_id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  timestamp TEXT NOT NULL,
  timezone TEXT NOT NULL,
  event_type TEXT NOT NULL,
  collector TEXT,
  source TEXT,
  schema_version INTEGER NOT NULL,
  payload TEXT NOT NULL,
  received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_device_time ON events(device_id, timestamp DESC);
CREATE INDEX idx_events_user_time ON events(user_id, timestamp DESC);
CREATE INDEX idx_events_type_time ON events(event_type, timestamp DESC);
CREATE UNIQUE INDEX idx_events_device_sequence ON events(device_id, sequence);
```

```sql
CREATE TABLE current_context (
  device_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  timezone TEXT NOT NULL,
  context TEXT NOT NULL,
  etag TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

```sql
CREATE TABLE timeline_segments (
  segment_id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  segment_type TEXT NOT NULL,
  layer TEXT,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  label TEXT NOT NULL,
  confidence REAL,
  version INTEGER NOT NULL,
  valid_from TEXT NOT NULL,
  valid_to TEXT,
  ingested_at TEXT NOT NULL,
  revised_at TEXT,
  revision_reason TEXT,
  evidence TEXT,
  payload TEXT NOT NULL,
  received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_segments_user_time ON timeline_segments(user_id, start_time DESC, end_time DESC);
CREATE INDEX idx_segments_user_type ON timeline_segments(user_id, segment_type, start_time DESC);
```

```sql
CREATE TABLE daily_summary (
  user_id TEXT NOT NULL,
  date TEXT NOT NULL,
  timezone TEXT NOT NULL,
  summary TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, date)
);
```

```sql
CREATE TABLE weekly_summary (
  user_id TEXT NOT NULL,
  week TEXT NOT NULL,
  timezone TEXT NOT NULL,
  summary TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, week)
);
```

```sql
CREATE TABLE agent_observations (
  observation_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  agent_id TEXT,
  observation_type TEXT,
  severity TEXT,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_agent_obs_time ON agent_observations(user_id, timestamp DESC);
```

```sql
CREATE TABLE interventions (
  intervention_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  agent_id TEXT,
  intervention_type TEXT,
  status TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_interventions_time ON interventions(user_id, timestamp DESC);
```

```sql
CREATE TABLE user_feedback (
  feedback_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  feedback_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feedback_target ON user_feedback(user_id, target_type, target_id);
```

```sql
CREATE TABLE goals (
  goal_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  goal_type TEXT NOT NULL,
  status TEXT NOT NULL,
  payload TEXT NOT NULL
);
```

```sql
CREATE TABLE habits (
  habit_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  habit_type TEXT NOT NULL,
  target TEXT NOT NULL,
  schedule TEXT NOT NULL,
  evidence_sources TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  streak_days INTEGER NOT NULL DEFAULT 0,
  habit_strength REAL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_habits_user ON habits(user_id);
```

```sql
CREATE TABLE baselines (
  baseline_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  metric_name TEXT NOT NULL,
  window_days INTEGER NOT NULL DEFAULT 7,
  value REAL NOT NULL,
  computed_at TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_baselines_user_metric ON baselines(user_id, metric_name);
```

```sql
CREATE TABLE outcomes (
  outcome_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  intervention_id TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  outcome_type TEXT NOT NULL,
  effectiveness_score REAL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_outcomes_intervention ON outcomes(intervention_id);
CREATE INDEX idx_outcomes_user_time ON outcomes(user_id, timestamp DESC);
```

```sql
CREATE TABLE audit_log (
  audit_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  actor_id TEXT,
  action TEXT NOT NULL,
  target_type TEXT,
  target_id TEXT,
  metadata TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Enable:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
```

---

# 17. API Design

Keep the API small, stable, and agent-friendly.

## 17.1 Authentication

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

## 17.2 Device ingestion

```http
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
  "context": {}
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

## 17.3 Current context

```http
GET /v1/context/current
Authorization: Bearer <agent_token>
```

Response:

```json
{
  "timestamp": "2026-07-28T10:14:00Z",
  "timezone": "Asia/Kolkata",
  "etag": "abc123",
  "context": {
    "primary_activity": "deep_work",
    "activity_confidence": 0.81,
    "place": "office",
    "place_confidence": 0.92,
    "focus_active": true,
    "focus_minutes_elapsed": 37,
    "distraction_probability": 0.18,
    "fatigue_probability": 0.31,
    "alignment_probability": 0.74,
    "readiness_score": 0.68,
    "data_quality": {
      "completeness": 0.87,
      "staleness_seconds": 42,
      "missing_signals": []
    }
  }
}
```

Support:

```http
If-None-Match: "abc123"
```

If unchanged:

```http
304 Not Modified
```

This makes frequent agent polling cheap.

## 17.4 Timeline

```http
GET /v1/timeline?from=2026-07-28T00:00:00Z&to=2026-07-28T23:59:59Z&limit=200
Authorization: Bearer <agent_token>
```

Optional filters:

```text
segment_type=focus_segment
label=deep_work
min_confidence=0.7
```

Response:

```json
{
  "timezone": "Asia/Kolkata",
  "timeline": [
    {
      "segment_id": "uuid",
      "segment_type": "focus_segment",
      "start": "2026-07-28T09:37:00+05:30",
      "end": "2026-07-28T11:02:00+05:30",
      "label": "deep_work",
      "confidence": 0.86,
      "payload": {
        "interruptions": 3,
        "distraction_minutes": 6
      }
    }
  ],
  "next_cursor": "..."
}
```

## 17.5 Events

For drill-down only.

```http
GET /v1/events?from=...&to=...&event_type=app_usage_summary&limit=200
Authorization: Bearer <agent_token>
```

Response:

```json
{
  "events": [],
  "next_cursor": "..."
}
```

Agents should not use this as their primary interface.

## 17.6 Daily summary

```http
GET /v1/summary/daily?date=2026-07-28
Authorization: Bearer <agent_token>
```

Response:

```json
{
  "date": "2026-07-28",
  "timezone": "Asia/Kolkata",
  "deep_work_minutes": 182,
  "meeting_minutes": 95,
  "distraction_minutes": 64,
  "sleep_minutes": 460,
  "steps": 8123,
  "alignment_score": 0.74,
  "work_patterns": {
    "average_focus_bout_minutes": 41,
    "switching_rate_per_hour": 18,
    "meeting_fragmentation_score": 0.43
  }
}
```

## 17.7 Weekly summary

```http
GET /v1/summary/weekly?week=2026-W31
Authorization: Bearer <agent_token>
```

## 17.8 Agent observations

```http
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

## 17.9 Interventions

```http
POST /v1/agent/interventions
Authorization: Bearer <agent_token>
```

Body:

```json
{
  "intervention_id": "uuid",
  "agent_id": "alignment-coach",
  "intervention_type": "nudge",
  "status": "proposed",
  "title": "Return to focus",
  "message": "You have drifted into social apps during a focus block.",
  "suggested_action": "resume_focus_session",
  "expires_at": "2026-07-28T10:30:00Z",
  "evidence": {
    "distraction_minutes": 12,
    "focus_active": true
  }
}
```

Intervention statuses:

- `proposed`
- `approved_by_policy`
- `delivered`
- `accepted`
- `dismissed`
- `ignored`
- `completed`
- `failed`
- `suppressed`
- `expired`

## 17.10 User feedback

```http
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

## 17.11 Goals

```http
GET /v1/goals
Authorization: Bearer <agent_token>
```

```http
POST /v1/goals
Authorization: Bearer <agent_token>
```

Example goal:

```json
{
  "goal_id": "uuid",
  "goal_type": "deep_work",
  "status": "active",
  "payload": {
    "target_minutes_per_day": 180,
    "target_window": "09:00-12:00",
    "timezone": "Asia/Kolkata",
    "priority": "high"
  }
}
```

## 17.12 Goals and Habits

```http
GET /v1/goals
Authorization: Bearer <agent_token>
```

```http
POST /v1/goals
Authorization: Bearer <agent_token>
```

```http
GET /v1/habits
Authorization: Bearer <agent_token>
```

```http
POST /v1/habits
Authorization: Bearer <agent_token>
```

## 17.13 Outcomes

```http
POST /v1/outcomes
Authorization: Bearer <agent_token>
```

```http
GET /v1/outcomes?intervention_id=...
Authorization: Bearer <agent_token>
```

## 17.14 Briefing

```http
GET /v1/briefing/current
Authorization: Bearer <agent_token>
```

Optional parameters:

```text
?horizon=2h
&agent_id=alignment-coach
&include_evidence=true
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

## 17.15 Optional SSE stream

For active agents:

```http
GET /v1/stream
Authorization: Bearer <agent_token>
Accept: text/event-stream
```

Events:

```text
event: context_changed
data: {"timestamp":"...","context":{...}}

event: focus_session_started
data: {"session_id":"..."}

event: focus_session_ended
data: {"session_id":"..."}

event: distraction_spike
data: {"distraction_probability":0.41}

event: observation_created
data: {"observation_id":"..."}

event: intervention_updated
data: {"intervention_id":"...","status":"accepted"}

event: daily_summary_ready
data: {"date":"2026-07-28"}
```

Do not make the phone maintain SSE. The server can stream to agents.

---

# 18. MCP Server / Agent Tool Layer

To make the system agent-native, wrap the API as an MCP server.

## 18.1 Recommended MCP tools

### Context tools

- `get_current_context`
- `get_current_briefing`
- `get_context_history`
- `get_data_quality`

### Timeline tools

- `get_timeline`
- `get_focus_sessions`
- `get_place_visits`
- `get_activity_segments`
- `get_distraction_segments`
- `get_communication_segments`
- `get_sleep_segments`

### Summary tools

- `get_daily_summary`
- `get_weekly_summary`
- `get_habit_metrics`
- `get_baseline_comparison`
- `get_work_pattern_metrics`

### Event tools

- `search_events`
- `get_app_usage`
- `get_notification_load`
- `get_communication_summary`
- `get_health_summary`
- `get_sleep_summary`
- `get_calendar_context`

### Agent loop tools

- `add_agent_observation`
- `propose_intervention`
- `update_intervention_status`
- `record_user_feedback`
- `get_intervention_outcomes`
- `evaluate_agent_effectiveness`

### Goal/Habit tools

- `get_goals`
- `create_goal`
- `update_goal`
- `evaluate_goal_progress`
- `get_habits`
- `create_habit`
- `evaluate_habit_completion`

## 18.2 Example MCP tool schema

```json
{
  "name": "get_current_context",
  "description": "Get the user's current inferred context, including activity, place, focus state, and risk probabilities.",
  "input_schema": {
    "type": "object",
    "properties": {
      "include_evidence": {
        "type": "boolean"
      },
      "include_data_quality": {
        "type": "boolean"
      }
    }
  }
}
```

## 18.3 Agent query pattern

Agents should follow this pattern:

```text
1. Get current context.
2. If confidence is low, get data quality.
3. If unclear, get last 2 hours timeline.
4. If analyzing day, get daily summary.
5. If investigating pattern, query segments/events.
6. Write observation.
7. If action needed, propose intervention.
8. Later, evaluate outcome.
```

This minimizes token usage and API load.

---

# 19. Data Products for Agents

The system should produce stable, high-level data products.

## 19.1 Daily summary object

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

## 19.2 Weekly summary object

```json
{
  "week": "2026-W31",
  "timezone": "Asia/Kolkata",
  "deep_work_minutes": 910,
  "meeting_minutes": 512,
  "distraction_minutes": 388,
  "sleep_avg_minutes": 451,
  "steps_avg": 7842,
  "focus_sessions": 16,
  "successful_focus_sessions": 11,
  "alignment_score_avg": 0.71,
  "trend": {
    "deep_work_delta_vs_prev_week": 0.12,
    "distraction_delta_vs_prev_week": -0.08,
    "sleep_regularity_delta": 0.04
  }
}
```

## 19.3 Habit metrics object

```json
{
  "date": "2026-07-28",
  "habits": [
    {
      "habit_id": "morning_deep_work",
      "status": "completed",
      "streak_days": 6,
      "evidence": {
        "deep_work_minutes_before_noon": 96
      }
    },
    {
      "habit_id": "gym",
      "status": "completed",
      "streak_days": 3,
      "evidence": {
        "place_visit": "gym",
        "minutes": 55
      }
    },
    {
      "habit_id": "sleep_by_2330",
      "status": "failed",
      "streak_days": 0,
      "evidence": {
        "bedtime": "2026-07-27T23:41:00+05:30"
      }
    }
  ]
}
```

---

# 20. Insight and Analytics Layer

Keep this optional and separate at first.

Do not build heavy analytics into the core relay immediately.

## 20.1 Minimal analytics approach

Use:

- SQLite views
- Scheduled SQL jobs
- DuckDB queries
- Python notebooks
- Agent-generated summaries

Example DuckDB query:

```sql
SELECT
  date,
  json_extract(summary, '$.work.deep_work_minutes') AS deep_work_minutes,
  json_extract(summary, '$.distraction.total_minutes') AS distraction_minutes
FROM daily_summary;
```

## 20.2 Later analytics worker

If needed, add a separate worker:

```text
contextd SQLite
  ↓ nightly export
DuckDB / Parquet
  ↓
analytics worker
  ↓
weekly summaries, patterns, anomalies
```

This keeps the relay thin.

## 20.3 Insight types

The system should eventually generate:

- Daily reflection
- Weekly review
- Habit streaks
- Focus quality trends
- Distraction triggers
- Sleep-performance correlation
- Location patterns
- Meeting load analysis
- Notification burden
- Intervention effectiveness
- Goal progress
- Baseline deviations

Examples:

```json
{
  "insight_type": "distraction_trigger",
  "message": "Social app usage spikes after 15 minutes of unfocused browser use.",
  "confidence": 0.72,
  "evidence": {
    "pattern_count": 14,
    "average_latency_minutes": 16
  }
}
```

## 20.4 Analytics archive

For long-term analysis, add an export/archive layer.

Recommended pipeline:

```text
SQLite
  ↓ nightly export
JSONL (raw-ish events)
  ↓
Parquet (analytics-optimized)
  ↓
DuckDB queries
```

This keeps the relay thin while enabling serious analysis later.

Optional additions:

- Object storage for Parquet files
- Scheduled DuckDB aggregation jobs
- Agent-driven analysis queries
- Weekly pattern detection
- Monthly trend reports

---

# 21. Agent Autonomy and Friction Reduction

To make agents effective with minimal friction, design the system around stable abstractions.

## 21.1 Stable IDs

Every object should have a stable ID:

- event_id
- segment_id
- summary_id
- observation_id
- intervention_id
- feedback_id
- goal_id

Agents need to reference objects reliably.

## 21.2 Confidence everywhere

Agents should never receive a strong label without confidence.

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
  "confidence": 0.81
}
```

## 21.3 Evidence packages

For important inferences, include evidence:

```json
{
  "label": "deep_work",
  "confidence": 0.81,
  "evidence": {
    "focus_active": true,
    "productivity_app_share": 0.74,
    "app_switch_rate": "low",
    "notification_load": "low"
  }
}
```

This allows agents to explain their reasoning.

## 21.4 Query escalation ladder

Agents should not read raw events first.

Use this ladder:

```text
Level 1: current context
Level 2: recent timeline
Level 3: daily summary
Level 4: segment history
Level 5: event drill-down
Level 6: raw export/analytics
```

This reduces cost and improves speed.

## 21.5 Cheap polling

Use:

- ETag
- Last-Modified
- 304 Not Modified
- Cursor pagination
- Minimal payloads

Active assistant polling:

```text
Every 30–60 seconds: GET /v1/context/current with ETag
```

Background advisor polling:

```text
Every 5 minutes: GET /v1/context/current
Every hour: GET /v1/summary/daily
```

## 21.6 Event-driven agent updates

For active agents, use server-side SSE:

```text
context_changed
focus_session_started
focus_session_ended
distraction_spike
intervention_updated
daily_summary_ready
```

The phone does not need to maintain persistent connections.

---

# 22. Privacy, Security, and Consent

This system is highly sensitive. Privacy must be architectural, not optional.

## 22.1 Default privacy posture

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

## 22.2 Sensitive mode

Add a user-controlled sensitive mode.

When enabled:

- Pause notification metadata
- Pause location
- Pause calendar sync
- Redact app usage
- Mark context as `private`
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

## 22.3 Redaction rules

Before sync, apply redaction:

- Replace exact coordinates with place labels.
- Hash contacts.
- Remove notification text.
- Remove calendar titles if configured.
- Remove app-specific sensitive content.
- Strip free-text notes.
- Remove message bodies.
- Remove sender identities unless explicitly enabled.

The app should have a redaction pipeline that runs before any data leaves the device. This is not optional; it is architectural.

## 22.4 Encryption

Use:

- TLS for transport
- Token hashing server-side
- Android Keystore for device secrets
- Optional encrypted SQLite or file-level encryption
- Encrypted backups if possible

## 22.5 Audit log

Server should log:

- Token creation
- Token revocation
- Agent observation writes
- Intervention writes
- Goal changes
- Data export
- Configuration changes

Example:

```json
{
  "actor_type": "agent",
  "actor_id": "alignment-coach",
  "action": "intervention.create",
  "target_type": "intervention",
  "target_id": "uuid"
}
```

## 22.6 Kill switch

The app should support:

- Pause tracking
- Pause sync
- Delete local data
- Delete server data
- Revoke tokens
- Wipe agent observations

This is essential for user trust.

---

# 23. Reliability and Failure Handling

## 23.1 Offline-first queue

The app must handle:

- No network
- Server downtime
- Token expiry
- Large backlogs
- Device reboot
- Permission revocation
- Low battery
- Low storage

Use a durable sync queue.

States:

```text
pending
uploading
uploaded
failed
dropped
```

## 23.2 Retry policy

Use exponential backoff:

```text
1 minute
5 minutes
15 minutes
30 minutes
60 minutes
```

Retry faster on Wi-Fi/charging.

## 23.3 Clock correctness

Every event must include:

- UTC timestamp
- local timezone
- UTC offset

This is essential for:

- Sleep analysis
- Circadian patterns
- Work-hour analysis
- Travel detection

Detect timezone changes and store them as events.

## 23.4 Stale data handling

If a collector has not produced data recently, mark it degraded.

Example:

```json
{
  "collector": "health_connect",
  "status": "stale",
  "last_update_minutes_ago": 180
}
```

## 23.5 App health telemetry

The app should report its own health:

```json
{
  "event_type": "app_health",
  "timestamp": "2026-07-28T10:15:00Z",
  "payload": {
    "queue_size": 12,
    "oldest_pending_event_minutes": 18,
    "last_sync_minutes_ago": 7,
    "battery": 0.74,
    "storage_free_mb": 42000,
    "workers_status": {
      "usage": "ok",
      "notifications": "ok",
      "location": "degraded",
      "health": "ok"
    }
  }
}
```

This lets agents know when the observability system itself is unhealthy.

---

# 24. Battery Profiles

Keep and refine the original profiles.

## 24.1 Profile A: Eco

Best for long-term background tracking.

- No persistent foreground service
- App usage summary every 10–15 minutes
- Location via geofence/passive only
- Health sync every 60 minutes
- Context snapshot every 5–10 minutes
- Upload every 30 minutes or on Wi-Fi/charging
- Raw events kept 3–7 days

Expected:

- Very low battery impact
- Good daily insights
- Limited second-by-second precision

## 24.2 Profile B: Balanced

Recommended default.

- Foreground service only during work hours
- App usage summary every 5 minutes
- Location passive + geofence
- Health sync every 30 minutes
- Context snapshot every 5 minutes or on change
- Upload every 10–15 minutes
- Raw events kept 14 days

Expected:

- Good real-time context
- Moderate battery impact
- Strong agent usefulness

## 24.3 Profile C: High Fidelity

Use temporarily.

- Foreground service all day
- App usage every 1–2 minutes
- Location every 2–5 minutes passive/fused
- Context snapshot every 1 minute
- Upload every 5 minutes
- Raw events kept 30 days

Expected:

- High precision
- Higher battery cost
- Useful for debugging or special projects

## 24.4 Adaptive profile

Add automatic profile switching:

```text
If charging and work hours:
  switch to Balanced or High Fidelity

If battery < 20%:
  switch to Eco

If user starts focus session:
  temporarily increase fidelity

If device idle overnight:
  reduce polling
```

---

# 25. What Not to Do

Avoid these:

- Polling `UsageStatsManager` every second
- Continuous high-accuracy GPS
- Streaming all notifications in real time
- Uploading every raw event immediately
- Running Postgres on the phone
- Keeping persistent WebSocket always active
- Holding wake locks
- Storing full notification content by default
- Retaining raw events forever
- Running continuous heavy analytics on the phone
- Building microservices too early
- Collecting SMS/call logs without strong reason
- Sending exact coordinates by default
- Letting agents read raw logs as their first step

---

# 26. Recommended Deployment Options

## 26.1 Option A: Ultra-minimal local-only

```text
Android app
local SQLite
export JSON/CSV
agents read exported files
```

Best if:

- Agents run on your laptop
- You do not need remote access
- You want zero server infra

## 26.2 Option B: File sync with Syncthing

```text
Android app
exports JSONL/current.json
Syncthing syncs to laptop/server
agents read files
```

Best if:

- You want no API server
- You want simple redundancy
- Near-real-time is acceptable

## 26.3 Option C: Thin relay VPS

Recommended for remote AI agents.

```text
Android app
HTTPS batch sync
single Caddy + SQLite service
agents poll REST API
optional MCP server
```

Best if:

- Agents are remote
- You want current context API
- You want minimal but proper infra

## 26.4 Option D: Serverless

```text
Android app
Cloudflare Worker
Cloudflare D1/KV/R2
agents poll Worker
```

Best if:

- You do not want to manage a server
- You accept vendor lock-in
- Your volume is personal-scale

---

# 27. Recommended Production-Minal Setup

For your stated goal, use this:

```text
Android app
  - Kotlin
  - Room/SQLite
  - WorkManager
  - Event-driven collectors
  - On-device summaries
  - On-device context classifier
  - Timeline segmentation
  - Batched compressed HTTPS sync

Thin relay server
  - Single Go binary
  - SQLite
  - Caddy HTTPS
  - Token auth
  - REST API
  - Optional SSE
  - Optional MCP server
  - Optional Litestream backup

AI agents
  - poll current context
  - query timeline when needed
  - read daily summaries
  - write observations
  - propose interventions
  - track outcomes
```

This is the sweet spot.

It is:

- Battery-friendly
- Minimal
- Self-contained
- Agent-observable
- Easy to operate
- Easy to extend

---

# 28. Server Deployment Example

Directory layout:

```text
/opt/context-engine/
  contextd
  context.db
  config.toml
```

Systemd service:

```ini
[Unit]
Description=Personal Context Engine Relay
After=network.target

[Service]
User=context
Group=context
WorkingDirectory=/opt/context-engine
ExecStart=/opt/context-engine/contextd --config /opt/context-engine/config.toml
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=full
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Caddyfile:

```text
context.yourdomain.com {
    reverse_proxy 127.0.0.1:8080
}
```

Firewall:

```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

Caddy handles TLS automatically.

---

# 29. Example Server Config

```toml
[server]
addr = "127.0.0.1:8080"
db_path = "/opt/context-engine/context.db"

[auth]
device_tokens = [
  "secret-device-token-1"
]
agent_tokens = [
  "secret-agent-token-1"
]

[sync]
max_batch_size = 500
max_request_bytes = 5242880

[retention]
raw_events_days = 30
daily_summaries_days = 0
timeline_segments_days = 365
agent_observations_days = 90

[streaming]
enable_sse = true

[mcp]
enable = true
addr = "127.0.0.1:8081"
```

In production, store token hashes, not raw tokens.

---

# 30. Implementation Plan

## Phase 1: Foundation

Goal: reliable local logger.

Build:

- App skeleton
- Permission screen
- SQLite/Room schema
- Event envelope
- Sequence generator
- Collector status screen

Collect:

- Screen on/off
- Unlock
- Battery/charging
- Timezone
- Boot completed
- Permission state

Do not build server yet.

Exit criteria:

- Events persist across reboot.
- Debug screen shows recent events.
- No excessive battery use.

## Phase 2: App Usage and Notifications

Build:

- UsageStatsManager worker
- App classification engine
- Notification listener service
- Notification metadata storage
- Five-minute summaries

Collect:

- App usage summary
- Category summary
- Notification summary
- Unlock count
- Screen-on minutes

Exit criteria:

- App can answer:
  - How much time in productivity apps?
  - How many unlocks?
  - How many notifications?
  - Which categories dominated last hour?

## Phase 3: Context Engine

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

Exit criteria:

- Current context updates every 5 minutes.
- Context is readable in debug screen.
- Confidence and missing signals are visible.

## Phase 4: Local Timeline Segmentation

Build:

- Segment engine
- Activity segments
- Focus segments
- Distraction segments
- Sleep segments later

Store:

- Timeline segments locally
- Segment versions
- Segment confidence

Exit criteria:

- App can show:
  - What happened from 9:00 to 12:00?
  - When was focus?
  - When was distraction?

## Phase 5: Batch Sync

Build:

- Sync queue
- Batch uploader
- gzip compression
- Idempotency
- Retry/backoff
- Heartbeat

Upload:

- Events
- Segments
- Summaries
- Current context
- Permission state
- App health

Exit criteria:

- Phone can lose network for hours and sync later without duplicates.
- Server rejects duplicate event IDs.
- Missing sequences are detectable.

## Phase 6: Thin Relay Server

Build:

- Go HTTP server
- SQLite schema
- Token auth
- Batch ingestion
- Current context endpoint
- Timeline endpoint
- Summary endpoint
- Event endpoint
- Observation endpoint

Deploy:

- systemd
- Caddy
- TLS
- backups

Exit criteria:

- Agents can query current context remotely.
- Agents can read timeline and daily summary.
- Duplicate batches are safe.

## Phase 7: Agent Loop

Build:

- Agent observations
- Interventions
- Intervention delivery layer
- Policy governor
- Feedback
- Goals
- Outcome tracking
- Effectiveness scoring

Add:

- MCP server
- SSE optional
- Agent tool catalog
- Briefing API

Exit criteria:

- Agent can:
  - Read context
  - Read briefing
  - Write observation
  - Propose intervention
  - Deliver intervention
  - Receive feedback
  - Evaluate outcome
  - Respect policy limits

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

## Phase 9: Analytics and Reflection

Build:

- Daily summary generator
- Weekly summary generator
- Baseline calculations
- Habit engine (cue detection, streak logic, habit strength)
- Work pattern metrics
- Readiness/fatigue model
- Intervention effectiveness metrics
- User correction loop

Optional:

- DuckDB export
- Parquet archive
- Scheduled insight generation

Exit criteria:

- Agents can answer:
  - How is this day compared to baseline?
  - Which habits are trending?
  - Which interventions worked?
  - What patterns cause distraction?
  - How does sleep affect focus?

## Phase 10: Hardening

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
- Synthetic event generator
- Golden test cases

Exit criteria:

- System is safe for long-term personal use.
- User can pause or delete data easily.
- Agents operate within scopes.
- Schema changes do not break old clients.

---

# 31. Acceptance Criteria

The system is successful if:

## 31.1 Tracking quality

- App usage summaries are accurate within 5-minute granularity.
- Unlock counts are accurate.
- Notification counts are accurate.
- Focus sessions are reliably recorded.
- Location places are correct for known places.
- Sleep/wake estimates are reasonable.
- Steps and activity sync reliably.
- Calendar context is fresh enough for agent advice.

## 31.2 Battery quality

In Balanced mode:

- Low single-digit percent battery drain per day.
- No continuous high-accuracy GPS.
- No persistent wake locks.
- Short periodic CPU bursts only.

## 31.3 Sync quality

- Offline queue survives reboot.
- Sync is idempotent.
- Duplicates are rejected.
- Missing sequences are detectable.
- Batch upload uses compression.
- Server downtime does not cause data loss.

## 31.4 Agent usefulness

Agents can reliably answer:

- What am I doing right now?
- How focused am I?
- Am I at risk of distraction?
- What did I do this morning?
- How much deep work did I get today?
- How is my sleep affecting my work?
- Did I go to the gym?
- Did I meet my habits?
- Was the last intervention helpful?

## 31.5 Operational simplicity

- One server binary.
- One SQLite database.
- One reverse proxy.
- Simple backups.
- Simple deployment.
- No heavy queue/stream/database infrastructure.

---

# 32. Risks and Mitigations

## 32.1 Android background restrictions

Risk: collectors killed or delayed.

Mitigation:

- Use WorkManager correctly.
- Use event-driven receivers.
- Use foreground service selectively.
- Request battery optimization exemption if needed.
- Show permission health screen.

## 32.2 Battery drain

Risk: tracking becomes expensive.

Mitigation:

- Tiered collection.
- No continuous GPS.
- Batched writes.
- Batched sync.
- Adaptive profiles.
- Default Balanced/Eco mode.

## 32.3 Privacy overcollection

Risk: system becomes creepy or unsafe.

Mitigation:

- Metadata by default.
- Redaction before sync.
- Sensitive mode.
- Local-first storage.
- Retention limits.
- Audit log.
- Kill switch.

## 32.4 Agent noise

Risk: agents over-notify or give bad advice.

Mitigation:

- Confidence thresholds.
- Intervention rate limits.
- Feedback tracking.
- Outcome evaluation.
- Agent severity levels.
- Quiet hours.
- Repeated-dismissal suppression.
- Policy governor enforcing cooldown and daily limits.

## 32.5 Data gaps

Risk: missing data leads to wrong conclusions.

Mitigation:

- Data quality object.
- Permission state events.
- Collector health telemetry.
- Confidence penalties.
- Gap detection.

## 32.6 Schema drift

Risk: future changes break agents.

Mitigation:

- Schema versioning.
- Stable IDs.
- Additive fields.
- Event catalog.
- API versioning.
- Contract tests.
- Replay tests.

---

# 33. Policy Governor

Agents should not be allowed to spam the user. Add a policy governor that enforces limits.

## 33.1 Policy rules

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

## 33.2 Suppression rules

The policy governor should suppress interventions when:

- Maximum daily interventions reached
- Cooldown period not elapsed
- Quiet hours active
- Confidence below threshold
- User dismissed last N interventions of same type
- User marked as fatigued
- Low-confidence signal from data quality

This prevents agent noise and protects user trust.

---

# 34. Intervention Delivery Layer

The original architecture stores interventions but does not fully specify delivery.

## 34.1 Delivery channels

Recommended delivery channels:

1. Local Android notification
2. Intervention inbox inside the app
3. Optional chat message through agent framework
4. Optional email digest
5. Optional webhook
6. Optional Tasker/MacroDroid action

Do not rely only on the server-side intervention table. The phone should pull pending interventions during sync and deliver them locally.

## 34.2 Intervention lifecycle

Track these states:

```text
proposed → approved_by_policy → delivered → accepted/dismissed/ignored → completed/failed
                                                                   → suppressed (by policy)
                                                                   → expired
```

## 34.3 Delivery tracking

Track for each intervention:

- Delivery timestamp
- Acceptance timestamp
- Response latency
- Dismissal count
- Expiry time

This data feeds into effectiveness scoring.

---

# 35. Outcome Attribution Engine

Agents need to know whether their advice worked.

## 35.1 Outcome types

```text
accepted
dismissed
ignored
completed
behavior_change
metric_improved
self_reported_helpful
no_measurable_effect
negative_effect
```

## 35.2 Outcome object

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

## 35.3 Effectiveness scoring

Effectiveness should combine:

- Was it delivered?
- Was it accepted?
- Did behavior change?
- Did the target metric improve?
- Did the user report it as helpful?
- Did the effect persist?
- Did it cause negative side effects?

Example formula:

```text
effectiveness_score =
  0.25 * accepted +
  0.35 * behavior_change +
  0.25 * metric_improvement +
  0.15 * user_helpfulness
```

The exact formula can evolve.

---

# 36. Baseline and Calibration Store

The system must learn normal behavior.

## 36.1 Baseline metrics

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

## 36.2 Baseline comparison

Agents should compare current state against baselines, not absolute thresholds.

Example:

```json
{
  "metric": "deep_work_minutes",
  "current": 182,
  "baseline_7d_avg": 158,
  "baseline_30d_avg": 145,
  "deviation": 0.15,
  "trend": "increasing"
}
```

## 36.3 Recalibration

Baselines should be recalculated:

- Weekly for short-term patterns
- Monthly for long-term trends
- After significant life changes (detected via schedule shifts)

---

# 37. User Correction Loop

The user should be able to correct inferred data.

## 37.1 Correction examples

- "This was not distraction; it was research."
- "This meeting was personal, not work."
- "I was not at the office; I was at a cafe."
- "This focus session failed because of an emergency."

## 37.2 Correction effects

Corrections should feed:

- Segment revision
- Classifier calibration
- App classification overrides
- Habit evaluation
- Agent confidence

## 37.3 Correction object

```json
{
  "correction_id": "uuid",
  "target_type": "segment",
  "target_id": "uuid",
  "original_label": "distraction",
  "corrected_label": "research",
  "reason": "user_correction",
  "timestamp": "2026-07-28T12:00:00Z"
}
```

---

# 38. Habit Engine

Habit tracking should be explicit and first-class.

## 38.1 Habit definition

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

## 38.2 Habit features

- Cue detection (time, location, context)
- Completion evidence (automatic from segments)
- Streak logic
- Exception rules (travel mode, sick days)
- Recovery rules (streak reset after N days)
- Habit strength score (increases with consistency)

## 38.3 Habit strength

```json
{
  "habit_id": "morning_deep_work",
  "streak_days": 12,
  "longest_streak": 21,
  "completion_rate_30d": 0.73,
  "habit_strength": 0.82
}
```

---

# 39. Work Pattern Metrics

Add explicit work-pattern analytics for alignment coaching.

## 39.1 Metrics

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

## 39.2 Work pattern object

```json
{
  "date": "2026-07-28",
  "average_focus_bout_minutes": 41,
  "switching_rate_per_hour": 18,
  "meeting_fragmentation_score": 0.43,
  "late_night_work_minutes": 22,
  "maker_time_block_minutes": 120,
  "interruption_density_per_hour": 4.2,
  "communication_during_focus_minutes": 8,
  "fragmentation_score": 0.52
}
```

These are essential for understanding work patterns and providing targeted coaching.

---

# 40. Readiness and Fatigue Model

Add a lightweight readiness model.

## 40.1 Inputs

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

## 40.2 Outputs

```json
{
  "readiness_score": 0.68,
  "fatigue_probability": 0.41,
  "recommended_focus_intensity": "moderate",
  "recommended_break": false
}
```

## 40.3 Usage

Agents should use readiness to:

- Adjust intervention timing
- Suggest appropriate task difficulty
- Recommend breaks
- Avoid heavy cognitive tasks when fatigued
- Tailor focus session recommendations

---

# 41. Testing and Simulation Harness

Add from the beginning to prevent fragility.

## 41.1 Test types

- Synthetic event generator
- Replayable event sequences
- Golden test cases
- Schema contract tests
- Timeline reconstruction tests
- Sync idempotency tests
- Battery profile tests
- Agent query simulations

## 41.2 Synthetic events

Generate synthetic events for:

- Unit testing collectors
- Integration testing sync
- Testing timeline reconstruction
- Validating agent queries
- Performance benchmarking

## 41.3 Golden test cases

Maintain golden test cases for:

- Typical workday patterns
- Weekend patterns
- Travel patterns
- Low-battery scenarios
- Permission denial scenarios
- Offline-to-online transitions

Without this, the system will become fragile as it evolves.

---

# 42. Final Recommended Architecture

Use this as the canonical plan:

```text
Android App
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
  - On-device classification
  - On-device timeline segmentation
  - On-device context snapshots
  - Batched compressed HTTPS sync

Thin Relay Server
  - Single Go binary
  - SQLite
  - Caddy HTTPS
  - Token auth with scopes
  - Batch ingestion
  - Current context API
  - Timeline API
  - Summary API
  - Event drill-down API
  - Agent observation API
  - Intervention API
  - Feedback API
  - Goal API
  - Optional SSE
  - Optional MCP server
  - Optional Litestream backup

AI Agents
  - Poll current context with ETag
  - Read timeline when needed
  - Read daily/weekly summaries
  - Query segments instead of raw events
  - Write observations
  - Propose interventions
  - Track outcomes
  - Adjust recommendations based on feedback
```

This gives you a comprehensive personal digital observability system without turning it into an unsustainable infrastructure project.

The most important architectural idea is this:

> The phone should produce trustworthy semantic context.  
> The server should expose that context securely and minimally.  
> The agents should act on context, timeline, summaries, and outcomes — not raw telemetry alone.

If you build it this way, the system becomes not just a tracker, but a foundation for personal transformation, habit alignment, and autonomous AI coaching.
