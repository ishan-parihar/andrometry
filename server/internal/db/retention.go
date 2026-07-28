package db

import (
	"encoding/json"
	"log"
	"time"
)

// RetentionPolicy defines data lifecycle rules.
type RetentionPolicy struct {
	RawEventsDays        int `json:"raw_events_days"`
	TimelineSegmentsDays int `json:"timeline_segments_days"`
	DailySummariesDays   int `json:"daily_summaries_days"`
	AgentObservationsDays int `json:"agent_observations_days"`
	InterventionsDays    int `json:"interventions_days"`
	AuditLogDays         int `json:"audit_log_days"`
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

// EnforceRetentionPolicy runs the retention cleanup based on policy.
func (db *DB) EnforceRetentionPolicy(policy RetentionPolicy) error {
	log.Printf("[retention] enforcing retention policy: raw=%dd, segments=%dd, observations=%dd, interventions=%dd, audit=%dd",
		policy.RawEventsDays, policy.TimelineSegmentsDays,
		policy.AgentObservationsDays, policy.InterventionsDays, policy.AuditLogDays)

	if policy.RawEventsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.RawEventsDays).Format("2006-01-02T15:04:05Z")
		result, _ := db.Exec("DELETE FROM events WHERE received_at < ?", cutoff)
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d raw events", rows)
		}
	}

	if policy.TimelineSegmentsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.TimelineSegmentsDays).Format("2006-01-02T15:04:05Z")
		result, _ := db.Exec("DELETE FROM timeline_segments WHERE received_at < ?", cutoff)
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d timeline segments", rows)
		}
	}

	if policy.AgentObservationsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.AgentObservationsDays).Format("2006-01-02T15:04:05Z")
		result, _ := db.Exec("DELETE FROM agent_observations WHERE created_at < ?", cutoff)
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d agent observations", rows)
		}
	}

	if policy.InterventionsDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.InterventionsDays).Format("2006-01-02T15:04:05Z")
		result, _ := db.Exec("DELETE FROM interventions WHERE created_at < ?", cutoff)
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d interventions", rows)
		}
	}

	if policy.AuditLogDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -policy.AuditLogDays).Format("2006-01-02T15:04:05Z")
		result, _ := db.Exec("DELETE FROM audit_log WHERE created_at < ?", cutoff)
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[retention] pruned %d audit log entries", rows)
		}
	}

	log.Println("[retention] policy enforcement complete")
	return nil
}
