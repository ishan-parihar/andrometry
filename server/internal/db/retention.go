package db

import (
	"encoding/json"
	"log"
	"time"
)

// RetentionPolicy defines data lifecycle rules.
type RetentionPolicy struct {
	RawEventsDays         int `json:"raw_events_days"`
	TimelineSegmentsDays  int `json:"timeline_segments_days"`
	DailySummariesDays    int `json:"daily_summaries_days"` // -1 = indefinite
	AgentObservationsDays int `json:"agent_observations_days"`
	InterventionsDays     int `json:"interventions_days"`
	AuditLogDays          int `json:"audit_log_days"`
}

// DefaultRetentionPolicy returns the recommended retention policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RawEventsDays:         30,
		TimelineSegmentsDays:  365,
		DailySummariesDays:    -1, // indefinite
		AgentObservationsDays: 90,
		InterventionsDays:     90,
		AuditLogDays:          365,
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
func (db *DB) EnforceRetentionPolicy(policy RetentionPolicy) error {
	log.Printf("[retention] enforcing policy: raw=%dd, segments=%dd, observations=%dd, interventions=%dd, audit=%dd",
		policy.RawEventsDays, policy.TimelineSegmentsDays,
		policy.AgentObservationsDays, policy.InterventionsDays, policy.AuditLogDays)

	pruned := 0

	if policy.RawEventsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.RawEventsDays).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec("DELETE FROM events WHERE received_at < ?", cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d raw events", rows)
				pruned += int(rows)
			}
		}
	}

	if policy.TimelineSegmentsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.TimelineSegmentsDays).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec("DELETE FROM timeline_segments WHERE received_at < ?", cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d timeline segments", rows)
				pruned += int(rows)
			}
		}
	}

	if policy.AgentObservationsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.AgentObservationsDays).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec("DELETE FROM agent_observations WHERE created_at < ?", cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d agent observations", rows)
				pruned += int(rows)
			}
		}
	}

	if policy.InterventionsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.InterventionsDays).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec("DELETE FROM interventions WHERE created_at < ?", cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d interventions", rows)
				pruned += int(rows)
			}
		}
	}

	// Prune five_minute_summary (>90 days), hourly_summary (>365 days), outcomes (>90 days)
	fiveMinCutoff := time.Now().AddDate(0, 0, -90).Format("2006-01-02T15:04:05Z")
	if result, err := db.Exec("DELETE FROM five_minute_summary WHERE start_time < ?", fiveMinCutoff); err == nil {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d five-minute summaries", rows)
			pruned += int(rows)
		}
	}

	hourlyCutoff := time.Now().AddDate(0, 0, -365).Format("2006-01-02T15:04:05Z")
	if result, err := db.Exec("DELETE FROM hourly_summary WHERE start_time < ?", hourlyCutoff); err == nil {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d hourly summaries", rows)
			pruned += int(rows)
		}
	}

	if result, err := db.Exec("DELETE FROM outcomes WHERE created_at < ?", fiveMinCutoff); err == nil {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d outcomes", rows)
			pruned += int(rows)
		}
	}

	// Prune user_feedback (>180 days) — kept longer than observations
	feedbackCutoff := time.Now().AddDate(0, 0, -180).Format("2006-01-02T15:04:05Z")
	if result, err := db.Exec("DELETE FROM user_feedback WHERE created_at < ?", feedbackCutoff); err == nil {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d user feedback entries", rows)
			pruned += int(rows)
		}
	}

	if policy.AuditLogDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.AuditLogDays).Format("2006-01-02T15:04:05Z")
		if result, err := db.Exec("DELETE FROM audit_log WHERE created_at < ?", cutoff); err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[retention] pruned %d audit entries", rows)
				pruned += int(rows)
			}
		}
	}

	if pruned > 0 {
		log.Printf("[retention] total pruned: %d rows", pruned)
		// Audit log the retention enforcement action
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
