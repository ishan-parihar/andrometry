package db

import (
	"log"
	"time"
)

// Cleanup handles retention policies and data pruning.
type Cleanup struct {
	DB *DB
}

// NewCleanup creates a new cleanup manager.
func NewCleanup(database *DB) *Cleanup {
	return &Cleanup{DB: database}
}

// RunCleanup executes all retention cleanup jobs.
func (c *Cleanup) RunCleanup() error {
	log.Println("[cleanup] starting retention cleanup")

	// Prune old raw events (30 days on server)
	if err := c.PruneOldEvents(30); err != nil {
		log.Printf("[cleanup] warning: prune events failed: %v", err)
	}

	// Prune old timeline segments (365 days)
	if err := c.PruneOldSegments(365); err != nil {
		log.Printf("[cleanup] warning: prune segments failed: %v", err)
	}

	// Prune old agent observations (90 days)
	if err := c.PruneOldObservations(90); err != nil {
		log.Printf("[cleanup] warning: prune observations failed: %v", err)
	}

	// Prune old interventions (90 days)
	if err := c.PruneOldInterventions(90); err != nil {
		log.Printf("[cleanup] warning: prune interventions failed: %v", err)
	}

	// Prune old audit log (365 days)
	if err := c.PruneOldAuditLog(365); err != nil {
		log.Printf("[cleanup] warning: prune audit log failed: %v", err)
	}

	log.Println("[cleanup] retention cleanup complete")
	return nil
}

// PruneOldEvents deletes events older than the specified number of days.
func (c *Cleanup) PruneOldEvents(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
	result, err := c.DB.Exec("DELETE FROM events WHERE received_at < ?", cutoff)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[cleanup] pruned %d old events (older than %d days)", rows, days)
	}
	return nil
}

// PruneOldSegments deletes timeline segments older than the specified number of days.
func (c *Cleanup) PruneOldSegments(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
	result, err := c.DB.Exec("DELETE FROM timeline_segments WHERE received_at < ?", cutoff)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[cleanup] pruned %d old segments (older than %d days)", rows, days)
	}
	return nil
}

// PruneOldObservations deletes agent observations older than the specified number of days.
func (c *Cleanup) PruneOldObservations(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
	result, err := c.DB.Exec("DELETE FROM agent_observations WHERE created_at < ?", cutoff)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[cleanup] pruned %d old observations (older than %d days)", rows, days)
	}
	return nil
}

// PruneOldInterventions deletes interventions older than the specified number of days.
func (c *Cleanup) PruneOldInterventions(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
	result, err := c.DB.Exec("DELETE FROM interventions WHERE created_at < ?", cutoff)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[cleanup] pruned %d old interventions (older than %d days)", rows, days)
	}
	return nil
}

// PruneOldAuditLog deletes audit log entries older than the specified number of days.
func (c *Cleanup) PruneOldAuditLog(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
	result, err := c.DB.Exec("DELETE FROM audit_log WHERE created_at < ?", cutoff)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[cleanup] pruned %d old audit entries (older than %d days)", rows, days)
	}
	return nil
}

// WipeUserData deletes all data for a user (kill switch).
func (c *Cleanup) WipeUserData(userID string) error {
	log.Printf("[cleanup] wiping all data for user %s", userID)

	tables := []string{
		"events", "timeline_segments", "daily_summary", "weekly_summary",
		"agent_observations", "interventions", "user_feedback",
		"goals", "habits", "baselines", "outcomes", "audit_log",
		"current_context", "five_minute_summary", "hourly_summary",
	}

	for _, table := range tables {
		if _, err := c.DB.Exec("DELETE FROM "+table+" WHERE user_id = ?", userID); err != nil {
			log.Printf("[cleanup] warning: failed to wipe %s: %v", table, err)
		}
	}

	log.Printf("[cleanup] data wipe complete for user %s", userID)
	return nil
}
