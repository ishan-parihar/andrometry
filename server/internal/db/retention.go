package db

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// RetentionPolicy defines data lifecycle rules.
// Days = -1 means indefinite (never prune).
type RetentionPolicy struct {
	RawEventsDays          int `json:"raw_events_days"`
	TimelineSegmentsDays   int `json:"timeline_segments_days"`
	DailySummariesDays     int `json:"daily_summaries_days"` // -1 = indefinite
	WeeklySummariesDays    int `json:"weekly_summaries_days"` // -1 = indefinite
	FiveMinuteSummariesDays int `json:"five_minute_summaries_days"`
	HourlySummariesDays    int `json:"hourly_summaries_days"`
	AgentObservationsDays  int `json:"agent_observations_days"`
	InterventionsDays      int `json:"interventions_days"`
	OutcomesDays           int `json:"outcomes_days"`
	UserFeedbackDays       int `json:"user_feedback_days"` // -1 = indefinite
	AuditLogDays           int `json:"audit_log_days"`
}

// DefaultRetentionPolicy returns the recommended retention policy per §15.3.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RawEventsDays:           30,
		TimelineSegmentsDays:    365,
		DailySummariesDays:      -1, // indefinite
		WeeklySummariesDays:     -1, // indefinite
		FiveMinuteSummariesDays: 90,
		HourlySummariesDays:     365,
		AgentObservationsDays:   90,
		InterventionsDays:       90,
		OutcomesDays:            90,
		UserFeedbackDays:        -1, // indefinite per §15.3
		AuditLogDays:            365,
	}
}

// MarshalRetentionPolicy serializes to JSON.
func MarshalRetentionPolicy(rp RetentionPolicy) string {
	b, _ := json.Marshal(rp)
	return string(b)
}

// UnmarshalRetentionPolicy deserializes from JSON.
func UnmarshalRetentionPolicy(data string) RetentionPolicy {
	var rp RetentionPolicy
	if data == "" {
		return DefaultRetentionPolicy()
	}
	json.Unmarshal([]byte(data), &rp)
	return rp
}

// EnforceRetentionPolicy runs all retention cleanup jobs.
// Days = -1 or 0 means indefinite (never prune).
func (db *DB) EnforceRetentionPolicy(policy RetentionPolicy) error {
	log.Printf("[retention] enforcing policy: raw=%dd, segments=%dd, 5min=%dd, hourly=%dd, daily=%dd, weekly=%dd, observations=%dd, interventions=%dd, outcomes=%dd, feedback=%dd, audit=%dd",
		policy.RawEventsDays, policy.TimelineSegmentsDays,
		policy.FiveMinuteSummariesDays, policy.HourlySummariesDays,
		policy.DailySummariesDays, policy.WeeklySummariesDays,
		policy.AgentObservationsDays, policy.InterventionsDays,
		policy.OutcomesDays, policy.UserFeedbackDays, policy.AuditLogDays)

	pruned := 0

	prune := func(table, column string, days int) {
		if days <= 0 {
			return // indefinite or unset
		}
		cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s < ?", table, column), cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d from %s (>%dd)", rows, table, days)
				pruned += int(rows)
			}
		}
	}

	prune("events", "received_at", policy.RawEventsDays)
	prune("timeline_segments", "received_at", policy.TimelineSegmentsDays)
	prune("five_minute_summary", "start_time", policy.FiveMinuteSummariesDays)
	prune("hourly_summary", "start_time", policy.HourlySummariesDays)
	prune("agent_observations", "created_at", policy.AgentObservationsDays)
	prune("interventions", "created_at", policy.InterventionsDays)
	prune("outcomes", "created_at", policy.OutcomesDays)
	prune("user_feedback", "created_at", policy.UserFeedbackDays)
	prune("audit_log", "created_at", policy.AuditLogDays)

	if pruned > 0 {
		log.Printf("[retention] total pruned: %d rows", pruned)
		log.Printf("[retention:audit] action=retention_enforcement pruned_rows=%d", pruned)
	}
	log.Println("[retention] policy enforcement complete")
	return nil
}

// WipeUserData deletes all data for a user (kill switch).
func (db *DB) WipeUserData(userID string) error {
	log.Printf("[retention] wiping all data for user %s", userID)

	tables := []string{
		"events", "timeline_segments", "daily_summary", "weekly_summary",
		"agent_observations", "interventions", "user_feedback",
		"goals", "habits", "baselines", "outcomes", "audit_log",
		"current_context", "five_minute_summary", "hourly_summary",
	}

	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM "+table+" WHERE user_id = ?", userID); err != nil {
			log.Printf("[retention] warning: failed to wipe %s: %v", table, err)
		}
	}

	log.Printf("[retention] data wipe complete for user %s", userID)
	return nil
}
