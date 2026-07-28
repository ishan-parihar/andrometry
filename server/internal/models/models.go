package models

import "time"

// Device represents a registered device.
type Device struct {
	DeviceID     string     `json:"device_id"`
	UserID       string     `json:"user_id"`
	CreatedAt    time.Time  `json:"created_at"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	ClientVersion string    `json:"client_version,omitempty"`
	Status       string     `json:"status"` // active, revoked
}

// AuthToken represents an authentication token.
type AuthToken struct {
	TokenID    string     `json:"token_id"`
	UserID     string     `json:"user_id"`
	TokenHash  string     `json:"-"`
	TokenType  string     `json:"token_type"` // device, agent
	Scopes     string     `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// Event represents an ingested event.
type Event struct {
	EventID       string    `json:"event_id"`
	DeviceID      string    `json:"device_id"`
	UserID        string    `json:"user_id"`
	Sequence      int64     `json:"sequence"`
	Timestamp     string    `json:"timestamp"`
	Timezone      string    `json:"timezone"`
	EventType     string    `json:"event_type"`
	Collector     string    `json:"collector"`
	Source        string    `json:"source"`
	SchemaVersion int       `json:"schema_version"`
	Payload       string    `json:"payload"` // JSON string
	ReceivedAt    time.Time `json:"received_at"`
}

// CurrentContext represents the latest context snapshot for a device.
type CurrentContext struct {
	DeviceID  string    `json:"device_id"`
	UserID    string    `json:"user_id"`
	Timestamp string    `json:"timestamp"`
	Timezone  string    `json:"timezone"`
	Context   string    `json:"context"` // JSON string
	ETag      string    `json:"etag"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TimelineSegment represents a semantic timeline segment.
type TimelineSegment struct {
	SegmentID      string     `json:"segment_id"`
	DeviceID       string     `json:"device_id"`
	UserID         string     `json:"user_id"`
	SegmentType    string     `json:"segment_type"`
	Layer          string     `json:"layer"`
	StartTime      string     `json:"start_time"`
	EndTime        string     `json:"end_time"`
	Label          string     `json:"label"`
	Confidence     float64    `json:"confidence"`
	Version        int        `json:"version"`
	ValidFrom      string     `json:"valid_from"`
	ValidTo        *string    `json:"valid_to,omitempty"`
	IngestedAt     time.Time  `json:"ingested_at"`
	RevisedAt      *time.Time `json:"revised_at,omitempty"`
	RevisionReason *string    `json:"revision_reason,omitempty"`
	Evidence       string     `json:"evidence,omitempty"`
	Payload        string     `json:"payload"`
	ReceivedAt     time.Time  `json:"received_at"`
}

// DailySummary represents a daily summary.
type DailySummary struct {
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	Timezone  string    `json:"timezone"`
	Summary   string    `json:"summary"` // JSON string
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WeeklySummary represents a weekly summary.
type WeeklySummary struct {
	UserID    string    `json:"user_id"`
	Week      string    `json:"week"`
	Timezone  string    `json:"timezone"`
	Summary   string    `json:"summary"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentObservation represents an observation written by an AI agent.
type AgentObservation struct {
	ObservationID string    `json:"observation_id"`
	UserID        string    `json:"user_id"`
	Timestamp     string    `json:"timestamp"`
	AgentID       string    `json:"agent_id"`
	ObservationType string  `json:"observation_type"`
	Severity      string    `json:"severity"`
	Payload       string    `json:"payload"`
	CreatedAt     time.Time `json:"created_at"`
}

// Intervention represents an agent-proposed intervention.
type Intervention struct {
	InterventionID    string     `json:"intervention_id"`
	UserID            string     `json:"user_id"`
	Timestamp         string     `json:"timestamp"`
	AgentID           string     `json:"agent_id"`
	InterventionType  string     `json:"intervention_type"`
	Status            string     `json:"status"`
	Payload           string     `json:"payload"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// UserFeedback represents user feedback on interventions/observations.
type UserFeedback struct {
	FeedbackID   string    `json:"feedback_id"`
	UserID       string    `json:"user_id"`
	Timestamp    string    `json:"timestamp"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	FeedbackType string    `json:"feedback_type"`
	Payload      string    `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
}

// Goal represents a user goal.
type Goal struct {
	GoalID    string    `json:"goal_id"`
	UserID    string    `json:"user_id"`
	UpdatedAt time.Time `json:"updated_at"`
	GoalType  string    `json:"goal_type"`
	Status    string    `json:"status"`
	Payload   string    `json:"payload"`
}

// Habit represents a tracked habit.
type Habit struct {
	HabitID         string    `json:"habit_id"`
	UserID          string    `json:"user_id"`
	HabitType       string    `json:"habit_type"`
	Target          string    `json:"target"`
	Schedule        string    `json:"schedule"`
	EvidenceSources string    `json:"evidence_sources"`
	Status          string    `json:"status"`
	StreakDays      int       `json:"streak_days"`
	HabitStrength   *float64  `json:"habit_strength,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Baseline represents a computed baseline metric.
type Baseline struct {
	BaselineID  string    `json:"baseline_id"`
	UserID      string    `json:"user_id"`
	MetricName  string    `json:"metric_name"`
	WindowDays  int       `json:"window_days"`
	Value       float64   `json:"value"`
	ComputedAt  time.Time `json:"computed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Outcome represents an intervention outcome.
type Outcome struct {
	OutcomeID        string    `json:"outcome_id"`
	UserID           string    `json:"user_id"`
	InterventionID   string    `json:"intervention_id"`
	Timestamp        string    `json:"timestamp"`
	OutcomeType      string    `json:"outcome_type"`
	EffectivenessScore *float64 `json:"effectiveness_score,omitempty"`
	Payload          string    `json:"payload"`
	CreatedAt        time.Time `json:"created_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	AuditID    string    `json:"audit_id"`
	UserID     string    `json:"user_id"`
	ActorType  string    `json:"actor_type"` // device, agent, system
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type,omitempty"`
	TargetID   string    `json:"target_id,omitempty"`
	Metadata   string    `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
