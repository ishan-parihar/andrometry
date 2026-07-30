package db

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

// UpsertDevice inserts or updates a device.
func (db *DB) UpsertDevice(d *models.Device) error {
	_, err := db.Exec(`
		INSERT INTO devices (device_id, user_id, client_version, last_seen_at, status)
		VALUES (?, ?, ?, datetime('now'), 'active')
		ON CONFLICT(device_id) DO UPDATE SET
			last_seen_at = datetime('now'),
			client_version = COALESCE(excluded.client_version, devices.client_version)
	`, d.DeviceID, d.UserID, d.ClientVersion)
	return err
}

// HashToken produces a SHA-256 hash of a bearer token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateToken inserts a new auth token.
func (db *DB) CreateToken(t *models.AuthToken) error {
	_, err := db.Exec(`
		INSERT INTO auth_tokens (token_id, user_id, token_hash, token_type, scopes, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, t.TokenID, t.UserID, t.TokenHash, t.TokenType, t.Scopes, t.ExpiresAt)
	return err
}

// ValidateToken checks if a token hash is valid, not revoked, and not expired.
func (db *DB) ValidateToken(tokenHash string) (*models.AuthToken, error) {
	var t models.AuthToken
	err := db.QueryRow(`
		SELECT token_id, user_id, token_type, scopes, created_at, expires_at
		FROM auth_tokens
		WHERE token_hash = ? AND revoked_at IS NULL
	`, tokenHash).Scan(&t.TokenID, &t.UserID, &t.TokenType, &t.Scopes, &t.CreatedAt, &t.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("token not found or revoked")
	}
	// Check expiry if set
	if t.ExpiresAt != nil && *t.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *t.ExpiresAt)
		if err == nil && time.Now().UTC().After(parsed) {
			return nil, fmt.Errorf("token expired")
		}
	}
	return &t, nil
}

// HasScope checks if a token's scopes string contains the required scope.
// Individual scope tokens are checked, and admin/wildcard on any token grants all access.
func HasScope(scopes, required string) bool {
	for _, s := range splitScopes(scopes) {
		if s == "admin" || s == "*" {
			return true
		}
		if s == required {
			return true
		}
	}
	return false
}

func splitScopes(scopes string) []string {
	var result []string
	current := ""
	for _, c := range scopes {
		if c == ',' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// InsertEvent inserts an event with idempotency (unique on device_id + sequence).
func (db *DB) InsertEvent(e *models.Event) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO events (event_id, device_id, user_id, sequence, timestamp, timezone, event_type, collector, source, schema_version, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.EventID, e.DeviceID, e.UserID, e.Sequence, e.Timestamp, e.Timezone, e.EventType, e.Collector, e.Source, e.SchemaVersion, e.Payload)
	return err
}

// InsertEventsBatch inserts multiple events in a transaction.
func (db *DB) InsertEventsBatch(events []*models.Event) (accepted int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO events (event_id, device_id, user_id, sequence, timestamp, timezone, event_type, collector, source, schema_version, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, e := range events {
		result, err := stmt.Exec(e.EventID, e.DeviceID, e.UserID, e.Sequence, e.Timestamp, e.Timezone, e.EventType, e.Collector, e.Source, e.SchemaVersion, e.Payload)
		if err != nil {
			return accepted, err
		}
		if n, _ := result.RowsAffected(); n > 0 {
			accepted++
		}
	}

	return accepted, tx.Commit()
}

// UpsertContext upserts the current context for a device.
func (db *DB) UpsertContext(c *models.CurrentContext) error {
	_, err := db.Exec(`
		INSERT INTO current_context (device_id, user_id, timestamp, timezone, context, etag, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(device_id) DO UPDATE SET
			timestamp = excluded.timestamp,
			timezone = excluded.timezone,
			context = excluded.context,
			etag = excluded.etag,
			updated_at = datetime('now')
	`, c.DeviceID, c.UserID, c.Timestamp, c.Timezone, c.Context, c.ETag)
	return err
}

// GetContext retrieves the current context for a device.
func (db *DB) GetContext(deviceID string) (*models.CurrentContext, error) {
	var c models.CurrentContext
	err := db.QueryRow(`
		SELECT device_id, user_id, timestamp, timezone, context, etag, updated_at
		FROM current_context
		WHERE device_id = ?
	`, deviceID).Scan(&c.DeviceID, &c.UserID, &c.Timestamp, &c.Timezone, &c.Context, &c.ETag, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetContextByUser retrieves the latest context across all devices for a user.
func (db *DB) GetContextByUser(userID string) (*models.CurrentContext, error) {
	var c models.CurrentContext
	err := db.QueryRow(`
		SELECT device_id, user_id, timestamp, timezone, context, etag, updated_at
		FROM current_context
		WHERE user_id = ?
		ORDER BY updated_at DESC LIMIT 1
	`, userID).Scan(&c.DeviceID, &c.UserID, &c.Timestamp, &c.Timezone, &c.Context, &c.ETag, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetSegmentByID retrieves a single segment by ID.
func (db *DB) GetSegmentByID(segmentID string) (*models.TimelineSegment, error) {
	var s models.TimelineSegment
	err := db.QueryRow(`
		SELECT segment_id, device_id, user_id, segment_type, layer,
			start_time, end_time, label, confidence, version,
			valid_from, valid_to, ingested_at, revised_at, revision_reason,
			evidence, payload, received_at
		FROM timeline_segments
		WHERE segment_id = ?
	`, segmentID).Scan(
		&s.SegmentID, &s.DeviceID, &s.UserID, &s.SegmentType, &s.Layer,
		&s.StartTime, &s.EndTime, &s.Label, &s.Confidence, &s.Version,
		&s.ValidFrom, &s.ValidTo, &s.IngestedAt, &s.RevisedAt, &s.RevisionReason,
		&s.Evidence, &s.Payload, &s.ReceivedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSegment upserts a timeline segment.
func (db *DB) UpsertSegment(s *models.TimelineSegment) error {
	_, err := db.Exec(`
		INSERT INTO timeline_segments (
			segment_id, device_id, user_id, segment_type, layer,
			start_time, end_time, label, confidence, version,
			valid_from, valid_to, ingested_at, evidence, payload
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
		ON CONFLICT(segment_id) DO UPDATE SET
			version = excluded.version,
			valid_to = excluded.valid_to,
			revised_at = datetime('now'),
			revision_reason = CASE WHEN timeline_segments.label != excluded.label THEN 'data_backfill' ELSE NULL END,
			label = excluded.label,
			confidence = excluded.confidence,
			evidence = excluded.evidence,
			payload = excluded.payload
	`, s.SegmentID, s.DeviceID, s.UserID, s.SegmentType, s.Layer,
		s.StartTime, s.EndTime, s.Label, s.Confidence, s.Version,
		s.ValidFrom, s.ValidTo, s.Evidence, s.Payload)
	return err
}

// UpsertSegmentsBatch inserts/upserts multiple segments in a transaction.
func (db *DB) UpsertSegmentsBatch(segments []*models.TimelineSegment) (accepted int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO timeline_segments (
			segment_id, device_id, user_id, segment_type, layer,
			start_time, end_time, label, confidence, version,
			valid_from, valid_to, ingested_at, evidence, payload
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
		ON CONFLICT(segment_id) DO UPDATE SET
			version = excluded.version,
			valid_to = excluded.valid_to,
			revised_at = datetime('now'),
			label = excluded.label,
			confidence = excluded.confidence,
			evidence = excluded.evidence,
			payload = excluded.payload
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, s := range segments {
		_, err := stmt.Exec(
			s.SegmentID, s.DeviceID, s.UserID, s.SegmentType, s.Layer,
			s.StartTime, s.EndTime, s.Label, s.Confidence, s.Version,
			s.ValidFrom, s.ValidTo, s.Evidence, s.Payload,
		)
		if err != nil {
			return accepted, err
		}
		accepted++
	}

	return accepted, tx.Commit()
}

// GetTimeline retrieves timeline segments for a user within a time range.
func (db *DB) GetTimeline(userID, from, to string, limit int, segmentType string) ([]models.TimelineSegment, error) {
	query := `
		SELECT segment_id, device_id, user_id, segment_type, layer,
			start_time, end_time, label, confidence, version,
			valid_from, valid_to, ingested_at, revised_at, revision_reason,
			evidence, payload, received_at
		FROM timeline_segments
		WHERE user_id = ? AND start_time >= ? AND start_time <= ?
	`
	args := []interface{}{userID, from, to}

	if segmentType != "" {
		query += " AND segment_type = ?"
		args = append(args, segmentType)
	}

	query += " ORDER BY start_time ASC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []models.TimelineSegment
	for rows.Next() {
		var s models.TimelineSegment
		err := rows.Scan(
			&s.SegmentID, &s.DeviceID, &s.UserID, &s.SegmentType, &s.Layer,
			&s.StartTime, &s.EndTime, &s.Label, &s.Confidence, &s.Version,
			&s.ValidFrom, &s.ValidTo, &s.IngestedAt, &s.RevisedAt, &s.RevisionReason,
			&s.Evidence, &s.Payload, &s.ReceivedAt,
		)
		if err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, nil
}

// UpsertDailySummary upserts a daily summary.
func (db *DB) UpsertDailySummary(s *models.DailySummary) error {
	_, err := db.Exec(`
		INSERT INTO daily_summary (user_id, date, timezone, summary, version, updated_at)
		VALUES (?, ?, ?, ?, 1, datetime('now'))
		ON CONFLICT(user_id, date) DO UPDATE SET
			summary = excluded.summary,
			version = daily_summary.version + 1,
			updated_at = datetime('now')
	`, s.UserID, s.Date, s.Timezone, s.Summary)
	return err
}

// GetDailySummary retrieves a daily summary.
func (db *DB) GetDailySummary(userID, date string) (*models.DailySummary, error) {
	var s models.DailySummary
	err := db.QueryRow(`
		SELECT user_id, date, timezone, summary, version, updated_at
		FROM daily_summary
		WHERE user_id = ? AND date = ?
	`, userID, date).Scan(&s.UserID, &s.Date, &s.Timezone, &s.Summary, &s.Version, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertWeeklySummary upserts a weekly summary.
func (db *DB) UpsertWeeklySummary(s *models.WeeklySummary) error {
	_, err := db.Exec(`
		INSERT INTO weekly_summary (user_id, week, timezone, summary, version, updated_at)
		VALUES (?, ?, ?, ?, 1, datetime('now'))
		ON CONFLICT(user_id, week) DO UPDATE SET
			summary = excluded.summary,
			version = weekly_summary.version + 1,
			updated_at = datetime('now')
	`, s.UserID, s.Week, s.Timezone, s.Summary)
	return err
}

// GetWeeklySummary retrieves a weekly summary.
func (db *DB) GetWeeklySummary(userID, week string) (*models.WeeklySummary, error) {
	var s models.WeeklySummary
	err := db.QueryRow(`
		SELECT user_id, week, timezone, summary, version, updated_at
		FROM weekly_summary
		WHERE user_id = ? AND week = ?
	`, userID, week).Scan(&s.UserID, &s.Week, &s.Timezone, &s.Summary, &s.Version, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetEvents retrieves events for drill-down.
func (db *DB) GetEvents(userID, from, to, eventType string, limit int) ([]models.Event, error) {
	query := `
		SELECT event_id, device_id, user_id, sequence, timestamp, timezone,
			event_type, collector, source, schema_version, payload, received_at
		FROM events
		WHERE user_id = ? AND timestamp >= ? AND timestamp <= ?
	`
	args := []interface{}{userID, from, to}

	if eventType != "" {
		query += " AND event_type = ?"
		args = append(args, eventType)
	}

	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		err := rows.Scan(
			&e.EventID, &e.DeviceID, &e.UserID, &e.Sequence, &e.Timestamp,
			&e.Timezone, &e.EventType, &e.Collector, &e.Source,
			&e.SchemaVersion, &e.Payload, &e.ReceivedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// InsertObservation inserts an agent observation.
func (db *DB) InsertObservation(o *models.AgentObservation) error {
	_, err := db.Exec(`
		INSERT INTO agent_observations (observation_id, user_id, timestamp, agent_id, observation_type, severity, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, o.ObservationID, o.UserID, o.Timestamp, o.AgentID, o.ObservationType, o.Severity, o.Payload)
	return err
}

// GetObservations retrieves observations for a user.
func (db *DB) GetObservations(userID string, limit int) ([]models.AgentObservation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT observation_id, user_id, timestamp, agent_id, observation_type, severity, payload, created_at
		FROM agent_observations
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []models.AgentObservation
	for rows.Next() {
		var o models.AgentObservation
		err := rows.Scan(&o.ObservationID, &o.UserID, &o.Timestamp, &o.AgentID, &o.ObservationType, &o.Severity, &o.Payload, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		observations = append(observations, o)
	}
	return observations, nil
}

// InsertIntervention inserts an intervention.
func (db *DB) InsertIntervention(i *models.Intervention) error {
	_, err := db.Exec(`
		INSERT INTO interventions (intervention_id, user_id, timestamp, agent_id, intervention_type, status, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, i.InterventionID, i.UserID, i.Timestamp, i.AgentID, i.InterventionType, i.Status, i.Payload)
	return err
}

// GetPendingInterventions retrieves pending interventions for a device to deliver.
func (db *DB) GetPendingInterventions(userID string, limit int) ([]models.Intervention, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := db.Query(`
		SELECT intervention_id, user_id, timestamp, agent_id, intervention_type, status, payload, created_at, updated_at
		FROM interventions
		WHERE user_id = ? AND status = 'approved_by_policy'
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interventions []models.Intervention
	for rows.Next() {
		var i models.Intervention
		err := rows.Scan(&i.InterventionID, &i.UserID, &i.Timestamp, &i.AgentID, &i.InterventionType, &i.Status, &i.Payload, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			return nil, err
		}
		interventions = append(interventions, i)
	}
	return interventions, nil
}

// UpdateInterventionStatus updates the status of an intervention.
func (db *DB) UpdateInterventionStatus(interventionID, status string) error {
	_, err := db.Exec(`
		UPDATE interventions SET status = ?, updated_at = datetime('now')
		WHERE intervention_id = ?
	`, status, interventionID)
	return err
}

// InsertFeedback inserts user feedback.
func (db *DB) InsertFeedback(f *models.UserFeedback) error {
	_, err := db.Exec(`
		INSERT INTO user_feedback (feedback_id, user_id, timestamp, target_type, target_id, feedback_type, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, f.FeedbackID, f.UserID, f.Timestamp, f.TargetType, f.TargetID, f.FeedbackType, f.Payload)
	return err
}

// InsertGoal inserts a goal.
func (db *DB) InsertGoal(g *models.Goal) error {
	_, err := db.Exec(`
		INSERT INTO goals (goal_id, user_id, goal_type, status, payload)
		VALUES (?, ?, ?, ?, ?)
	`, g.GoalID, g.UserID, g.GoalType, g.Status, g.Payload)
	return err
}

// GetGoals retrieves goals for a user.
func (db *DB) GetGoals(userID string) ([]models.Goal, error) {
	rows, err := db.Query(`
		SELECT goal_id, user_id, updated_at, goal_type, status, payload
		FROM goals
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		err := rows.Scan(&g.GoalID, &g.UserID, &g.UpdatedAt, &g.GoalType, &g.Status, &g.Payload)
		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// UpdateGoalStatus updates the status of a goal.
func (db *DB) UpdateGoalStatus(goalID, status string) error {
	_, err := db.Exec(`
		UPDATE goals SET status = ?, updated_at = datetime('now')
		WHERE goal_id = ?
	`, status, goalID)
	return err
}

// InsertHabit inserts a habit.
func (db *DB) InsertHabit(h *models.Habit) error {
	_, err := db.Exec(`
		INSERT INTO habits (habit_id, user_id, habit_type, target, schedule, evidence_sources, status, streak_days, habit_strength)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.HabitID, h.UserID, h.HabitType, h.Target, h.Schedule, h.EvidenceSources, h.Status, h.StreakDays, h.HabitStrength)
	return err
}

// GetHabits retrieves habits for a user.
func (db *DB) GetHabits(userID string) ([]models.Habit, error) {
	rows, err := db.Query(`
		SELECT habit_id, user_id, habit_type, target, schedule, evidence_sources, status, streak_days, habit_strength, created_at, updated_at
		FROM habits
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		err := rows.Scan(&h.HabitID, &h.UserID, &h.HabitType, &h.Target, &h.Schedule, &h.EvidenceSources, &h.Status, &h.StreakDays, &h.HabitStrength, &h.CreatedAt, &h.UpdatedAt)
		if err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

// InsertOutcome inserts an intervention outcome.
func (db *DB) InsertOutcome(o *models.Outcome) error {
	_, err := db.Exec(`
		INSERT INTO outcomes (outcome_id, user_id, intervention_id, timestamp, outcome_type, effectiveness_score, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, o.OutcomeID, o.UserID, o.InterventionID, o.Timestamp, o.OutcomeType, o.EffectivenessScore, o.Payload)
	return err
}

// GetOutcomes retrieves outcomes for a user or intervention.
func (db *DB) GetOutcomes(userID, interventionID string, limit int) ([]models.Outcome, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT outcome_id, user_id, intervention_id, timestamp, outcome_type, effectiveness_score, payload, created_at
		FROM outcomes
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if interventionID != "" {
		query += " AND intervention_id = ?"
		args = append(args, interventionID)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []models.Outcome
	for rows.Next() {
		var o models.Outcome
		err := rows.Scan(&o.OutcomeID, &o.UserID, &o.InterventionID, &o.Timestamp, &o.OutcomeType, &o.EffectivenessScore, &o.Payload, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, o)
	}
	return outcomes, nil
}

// InsertAuditLog inserts an audit log entry.
func (db *DB) InsertAuditLog(a *models.AuditLog) error {
	_, err := db.Exec(`
		INSERT INTO audit_log (audit_id, user_id, actor_type, actor_id, action, target_type, target_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, a.AuditID, a.UserID, a.ActorType, a.ActorID, a.Action, a.TargetType, a.TargetID, a.Metadata)
	return err
}

// GetAuditLog retrieves audit log entries.
func (db *DB) GetAuditLog(userID string, limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT audit_id, user_id, actor_type, actor_id, action, target_type, target_id, metadata, created_at
		FROM audit_log
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var a models.AuditLog
		err := rows.Scan(&a.AuditID, &a.UserID, &a.ActorType, &a.ActorID, &a.Action, &a.TargetType, &a.TargetID, &a.Metadata, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, a)
	}
	return logs, nil
}

// CountInterventionsToday counts interventions created today for rate limiting.
func (db *DB) CountInterventionsToday(userID string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM interventions
		WHERE user_id = ? AND date(created_at) = date('now')
	`, userID).Scan(&count)
	return count, err
}

// GetInterventionsByAgent retrieves interventions for a specific agent within a user.
func (db *DB) GetInterventionsByAgent(userID, agentID string, limit int) ([]models.Intervention, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT intervention_id, user_id, timestamp, agent_id, intervention_type, status, payload, created_at, updated_at
		FROM interventions
		WHERE user_id = ? AND agent_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interventions []models.Intervention
	for rows.Next() {
		var i models.Intervention
		err := rows.Scan(&i.InterventionID, &i.UserID, &i.Timestamp, &i.AgentID, &i.InterventionType, &i.Status, &i.Payload, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			return nil, err
		}
		interventions = append(interventions, i)
	}
	return interventions, nil
}

// GetInterventionsByDateRange retrieves interventions for a user within a time range.
func (db *DB) GetInterventionsByDateRange(userID, from, to string, limit int) ([]models.Intervention, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := db.Query(`
		SELECT intervention_id, user_id, timestamp, agent_id, intervention_type, status, payload, created_at, updated_at
		FROM interventions
		WHERE user_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
		LIMIT ?
	`, userID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interventions []models.Intervention
	for rows.Next() {
		var i models.Intervention
		err := rows.Scan(&i.InterventionID, &i.UserID, &i.Timestamp, &i.AgentID, &i.InterventionType, &i.Status, &i.Payload, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			return nil, err
		}
		interventions = append(interventions, i)
	}
	return interventions, nil
}

// GetOutcomesByInterventionIDs retrieves outcomes for multiple intervention IDs in a single query (avoids N+1).
func (db *DB) GetOutcomesByInterventionIDs(userID string, interventionIDs []string, limit int) ([]models.Outcome, error) {
	if len(interventionIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}

	// Build IN clause safely with numbered placeholders
	query := `
		SELECT outcome_id, user_id, intervention_id, timestamp, outcome_type, effectiveness_score, payload, created_at
		FROM outcomes
		WHERE user_id = ? AND intervention_id IN (`
	args := []interface{}{userID}
	for i, id := range interventionIDs {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args = append(args, id)
	}
	query += `)
		ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []models.Outcome
	for rows.Next() {
		var o models.Outcome
		err := rows.Scan(&o.OutcomeID, &o.UserID, &o.InterventionID, &o.Timestamp, &o.OutcomeType, &o.EffectivenessScore, &o.Payload, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, o)
	}
	return outcomes, nil
}

// GetDevice retrieves a device by ID.
func (db *DB) GetDevice(deviceID string) (*models.Device, error) {
	var d models.Device
	err := db.QueryRow(`
		SELECT device_id, user_id, created_at, last_seen_at, client_version, status
		FROM devices
		WHERE device_id = ?
	`, deviceID).Scan(&d.DeviceID, &d.UserID, &d.CreatedAt, &d.LastSeenAt, &d.ClientVersion, &d.Status)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// DetectMissingSequences finds gaps in the sequence range for a device.
// Safety limit of 10000 to prevent OOM on large ranges.
func (db *DB) DetectMissingSequences(deviceID string, from, to int64) []int64 {
	if from == 0 && to == 0 {
		return []int64{}
	}
	rows, err := db.Query(`
		SELECT sequence FROM events
		WHERE device_id = ? AND sequence >= ? AND sequence <= ?
		ORDER BY sequence ASC LIMIT 10000
	`, deviceID, from, to)
	if err != nil {
		return []int64{}
	}
	defer rows.Close()

	var existing []int64
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err == nil {
			existing = append(existing, seq)
		}
	}

	var missing []int64
	expected := from
	for _, seq := range existing {
		for expected < seq {
			missing = append(missing, expected)
			expected++
		}
		expected = seq + 1
	}
	for expected <= to {
		missing = append(missing, expected)
		expected++
	}

	return missing
}

// UpsertFiveMinuteSummary upserts a five-minute summary.
func (db *DB) UpsertFiveMinuteSummary(userID, startTime, endTime, summaryJSON string) error {
	_, err := db.Exec(`
		INSERT INTO five_minute_summary (user_id, start_time, end_time, summary, sync_state)
		VALUES (?, ?, ?, ?, 'pending')
		ON CONFLICT(user_id, start_time) DO UPDATE SET
			end_time = excluded.end_time,
			summary = excluded.summary,
			sync_state = 'pending'
	`, userID, startTime, endTime, summaryJSON)
	return err
}

// GetFiveMinuteSummaries retrieves five-minute summaries in a time range for a user.
func (db *DB) GetFiveMinuteSummaries(userID, from, to string) ([]map[string]string, error) {
	rows, err := db.Query(`
		SELECT start_time, end_time, summary
		FROM five_minute_summary
		WHERE user_id = ? AND start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var start, end, summary string
		if err := rows.Scan(&start, &end, &summary); err == nil {
			results = append(results, map[string]string{
				"start_time": start,
				"end_time":   end,
				"summary":    summary,
			})
		}
	}
	return results, nil
}

// UpsertHourlySummary upserts an hourly summary.
func (db *DB) UpsertHourlySummary(userID, startTime, endTime, summaryJSON string) error {
	_, err := db.Exec(`
		INSERT INTO hourly_summary (user_id, start_time, end_time, summary, sync_state)
		VALUES (?, ?, ?, ?, 'pending')
		ON CONFLICT(user_id, start_time) DO UPDATE SET
			end_time = excluded.end_time,
			summary = excluded.summary,
			sync_state = 'pending'
	`, userID, startTime, endTime, summaryJSON)
	return err
}

// GetHourlySummaries retrieves hourly summaries in a time range for a user.
func (db *DB) GetHourlySummaries(userID, from, to string) ([]map[string]string, error) {
	rows, err := db.Query(`
		SELECT start_time, end_time, summary
		FROM hourly_summary
		WHERE user_id = ? AND start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var start, end, summary string
		if err := rows.Scan(&start, &end, &summary); err == nil {
			results = append(results, map[string]string{
				"start_time": start,
				"end_time":   end,
				"summary":    summary,
			})
		}
	}
	return results, nil
}

// UpsertBaseline upserts a baseline metric, ensuring one baseline per (user_id, metric_name).
// Uses ON CONFLICT for atomic upsert (no TOCTOU race).
func (db *DB) UpsertBaseline(b *models.Baseline) error {
	_, err := db.Exec(`
		INSERT INTO baselines (baseline_id, user_id, metric_name, window_days, value, computed_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id, metric_name) DO UPDATE SET
			value = excluded.value,
			window_days = excluded.window_days,
			computed_at = datetime('now')
	`, b.BaselineID, b.UserID, b.MetricName, b.WindowDays, b.Value)
	return err
}

// GetBaselines retrieves all baselines for a user.
func (db *DB) GetBaselines(userID string) ([]models.Baseline, error) {
	rows, err := db.Query(`
		SELECT baseline_id, user_id, metric_name, window_days, value, computed_at
		FROM baselines
		WHERE user_id = ?
		ORDER BY metric_name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baselines []models.Baseline
	for rows.Next() {
		var b models.Baseline
		err := rows.Scan(&b.BaselineID, &b.UserID, &b.MetricName, &b.WindowDays, &b.Value, &b.ComputedAt)
		if err != nil {
			return nil, err
		}
		baselines = append(baselines, b)
	}
	return baselines, nil
}

// InsertUserCorrection inserts a user correction record.
// Uses action 'correction' to distinguish from the audit.Log entry which uses 'user_correction'.
func (db *DB) InsertUserCorrection(userID, correctionJSON string) error {
	_, err := db.Exec(`
		INSERT INTO audit_log (audit_id, user_id, actor_type, actor_id, action, target_type, metadata)
		VALUES (?, ?, 'user', ?, 'correction', 'segment', ?)
	`, uuid.New().String(), userID, userID, correctionJSON)
	return err
}

// GetUserCorrections retrieves user corrections for a user.
func (db *DB) GetUserCorrections(userID string, limit int) ([]map[string]string, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT audit_id, metadata, created_at
		FROM audit_log
		WHERE user_id = ? AND action = 'correction'
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var id, metadata, createdAt string
		if err := rows.Scan(&id, &metadata, &createdAt); err == nil {
			results = append(results, map[string]string{
				"correction_id": id,
				"metadata":      metadata,
				"created_at":    createdAt,
			})
		}
	}
	return results, nil
}

// DeleteUserCorrection deletes a user correction by ID.
func (db *DB) DeleteUserCorrection(userID, correctionID string) error {
	_, err := db.Exec(`
		DELETE FROM audit_log
		WHERE user_id = ? AND audit_id = ? AND action = 'correction'
	`, userID, correctionID)
	return err
}

// DeleteBaseline deletes a baseline metric by ID for a user.
func (db *DB) DeleteBaseline(userID, baselineID string) error {
	_, err := db.Exec(`
		DELETE FROM baselines
		WHERE user_id = ? AND baseline_id = ?
	`, userID, baselineID)
	return err
}

// --- Goal CRUD (update/delete) ---

// GetGoalByID retrieves a single goal by ID.
func (db *DB) GetGoalByID(userID, goalID string) (*models.Goal, error) {
	var g models.Goal
	err := db.QueryRow(`
		SELECT goal_id, user_id, updated_at, goal_type, status, payload
		FROM goals
		WHERE user_id = ? AND goal_id = ?
	`, userID, goalID).Scan(&g.GoalID, &g.UserID, &g.UpdatedAt, &g.GoalType, &g.Status, &g.Payload)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateGoal updates a goal's fields.
func (db *DB) UpdateGoal(g *models.Goal) error {
	_, err := db.Exec(`
		UPDATE goals SET goal_type = ?, status = ?, payload = ?, updated_at = datetime('now')
		WHERE goal_id = ? AND user_id = ?
	`, g.GoalType, g.Status, g.Payload, g.GoalID, g.UserID)
	return err
}

// DeleteGoal deletes a goal by ID for a user.
func (db *DB) DeleteGoal(userID, goalID string) error {
	_, err := db.Exec(`
		DELETE FROM goals
		WHERE user_id = ? AND goal_id = ?
	`, userID, goalID)
	return err
}

// --- Habit CRUD (update/delete/streak) ---

// GetHabitByID retrieves a single habit by ID.
func (db *DB) GetHabitByID(userID, habitID string) (*models.Habit, error) {
	var h models.Habit
	err := db.QueryRow(`
		SELECT habit_id, user_id, habit_type, target, schedule, evidence_sources, status, streak_days, habit_strength, created_at, updated_at
		FROM habits
		WHERE user_id = ? AND habit_id = ?
	`, userID, habitID).Scan(&h.HabitID, &h.UserID, &h.HabitType, &h.Target, &h.Schedule, &h.EvidenceSources, &h.Status, &h.StreakDays, &h.HabitStrength, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// UpdateHabit updates a habit's fields.
func (db *DB) UpdateHabit(h *models.Habit) error {
	_, err := db.Exec(`
		UPDATE habits SET habit_type = ?, target = ?, schedule = ?, evidence_sources = ?, status = ?,
			streak_days = ?, habit_strength = ?, updated_at = datetime('now')
		WHERE habit_id = ? AND user_id = ?
	`, h.HabitType, h.Target, h.Schedule, h.EvidenceSources, h.Status, h.StreakDays, h.HabitStrength, h.HabitID, h.UserID)
	return err
}

// UpdateHabitStreak updates streak days and habit strength for a habit.
func (db *DB) UpdateHabitStreak(userID, habitID string, streakDays int, habitStrength float64) error {
	_, err := db.Exec(`
		UPDATE habits SET streak_days = ?, habit_strength = ?, updated_at = datetime('now')
		WHERE habit_id = ? AND user_id = ?
	`, streakDays, habitStrength, habitID, userID)
	return err
}

// DeleteHabit deletes a habit by ID for a user.
func (db *DB) DeleteHabit(userID, habitID string) error {
	_, err := db.Exec(`
		DELETE FROM habits
		WHERE user_id = ? AND habit_id = ?
	`, userID, habitID)
	return err
}

// --- Feedback GET ---

// GetFeedback retrieves feedback for a user, optionally filtered by target.
func (db *DB) GetFeedback(userID, targetType, targetID string, limit int) ([]models.UserFeedback, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT feedback_id, user_id, timestamp, target_type, target_id, feedback_type, payload, created_at
		FROM user_feedback
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if targetType != "" {
		query += " AND target_type = ?"
		args = append(args, targetType)
	}
	if targetID != "" {
		query += " AND target_id = ?"
		args = append(args, targetID)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedbacks []models.UserFeedback
	for rows.Next() {
		var f models.UserFeedback
		err := rows.Scan(&f.FeedbackID, &f.UserID, &f.Timestamp, &f.TargetType, &f.TargetID, &f.FeedbackType, &f.Payload, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, f)
	}
	return feedbacks, nil
}

// --- Token Management ---

// RevokeAllTokensForUser revokes all tokens for a user.
func (db *DB) RevokeAllTokensForUser(userID string) error {
	_, err := db.Exec(`
		UPDATE auth_tokens SET revoked_at = datetime('now')
		WHERE user_id = ? AND revoked_at IS NULL
	`, userID)
	return err
}

// RevokeTokensByType revokes only active (non-revoked) tokens of a given type for a user.
// Used during token rotation to invalidate old tokens after creating a new one.
// If excludeTokenID is non-empty, that token is preserved (not revoked).
// Returns the number of tokens revoked.
func (db *DB) RevokeTokensByType(userID, tokenType, excludeTokenID string) (int64, error) {
	if excludeTokenID != "" {
		result, err := db.Exec(`
			UPDATE auth_tokens SET revoked_at = datetime('now')
			WHERE user_id = ? AND token_type = ? AND revoked_at IS NULL AND token_id != ?
		`, userID, tokenType, excludeTokenID)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}
	result, err := db.Exec(`
		UPDATE auth_tokens SET revoked_at = datetime('now')
		WHERE user_id = ? AND token_type = ? AND revoked_at IS NULL
	`, userID, tokenType)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLastInterventionTime returns the timestamp of the most recent intervention for a user.
// Uses the `timestamp` column (RFC3339 from the device) rather than `created_at` (SQLite datetime format).
func (db *DB) GetLastInterventionTime(userID string) (string, error) {
	var ts string
	err := db.QueryRow(`
		SELECT timestamp FROM interventions
		WHERE user_id = ?
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&ts)
	if err != nil {
		return "", err
	}
	return ts, nil
}

// --- Sensitive Mode Persistence ---

// GetSensitiveModeSettings retrieves the persisted sensitive mode settings.
func (db *DB) GetSensitiveModeSettings() (string, error) {
	var settings string
	err := db.QueryRow(`
		SELECT settings FROM policy_settings
		WHERE id = 1
	`).Scan(&settings)
	if err != nil {
		return "", err
	}
	return settings, nil
}

// SaveSensitiveModeSettings persists sensitive mode settings.
func (db *DB) SaveSensitiveModeSettings(settingsJSON string) error {
	_, err := db.Exec(`
		INSERT INTO policy_settings (id, settings, updated_at)
		VALUES (1, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			settings = excluded.settings,
			updated_at = datetime('now')
	`, settingsJSON)
	return err
}

// --- User Preferences (alignment weights, classification overrides) ---

// UserPreference represents a single user preference key-value pair.
type UserPreference struct {
	PreferenceID string `json:"preference_id"`
	UserID       string `json:"user_id"`
	Category     string `json:"category"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	UpdatedAt    string `json:"updated_at"`
}

// UpsertPreference inserts or updates a user preference (atomic upsert).
func (db *DB) UpsertPreference(userID, category, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO user_preferences (preference_id, user_id, category, key, value, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id, category, key) DO UPDATE SET
			value = excluded.value,
			updated_at = datetime('now')
	`, uuid.New().String(), userID, category, key, value)
	return err
}

// GetPreferences retrieves all preferences for a user, optionally filtered by category.
func (db *DB) GetPreferences(userID, category string) ([]UserPreference, error) {
	query := `SELECT preference_id, user_id, category, key, value, updated_at FROM user_preferences WHERE user_id = ?`
	args := []interface{}{userID}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	query += " ORDER BY category, key"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []UserPreference
	for rows.Next() {
		var p UserPreference
		if err := rows.Scan(&p.PreferenceID, &p.UserID, &p.Category, &p.Key, &p.Value, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, nil
}

// GetPreference retrieves a single preference by category and key.
func (db *DB) GetPreference(userID, category, key string) (*UserPreference, error) {
	var p UserPreference
	err := db.QueryRow(`
		SELECT preference_id, user_id, category, key, value, updated_at
		FROM user_preferences
		WHERE user_id = ? AND category = ? AND key = ?
	`, userID, category, key).Scan(&p.PreferenceID, &p.UserID, &p.Category, &p.Key, &p.Value, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePreference deletes a user preference by category and key.
func (db *DB) DeletePreference(userID, category, key string) error {
	_, err := db.Exec(`DELETE FROM user_preferences WHERE user_id = ? AND category = ? AND key = ?`, userID, category, key)
	return err
}

// DeletePreferencesByCategory deletes all preferences in a category for a user.
func (db *DB) DeletePreferencesByCategory(userID, category string) error {
	_, err := db.Exec(`DELETE FROM user_preferences WHERE user_id = ? AND category = ?`, userID, category)
	return err
}

// CountSegments returns the total number of timeline segments for a user.
func (db *DB) CountSegments(userID string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM timeline_segments WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// GetCorrectionStats returns correction counts by corrected_label for feedback loop.
func (db *DB) GetCorrectionStats(userID string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT metadata FROM audit_log
		WHERE user_id = ? AND action = 'correction'
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelCounts := make(map[string]int)
	for rows.Next() {
		var metadata string
		if err := rows.Scan(&metadata); err != nil {
			continue
		}
		// Parse metadata to extract new_label
		var correction map[string]interface{}
		if json.Unmarshal([]byte(metadata), &correction) == nil {
			if newLabel, ok := correction["new_label"].(string); ok && newLabel != "" {
				labelCounts[newLabel]++
			}
			if oldLabel, ok := correction["old_label"].(string); ok && oldLabel != "" {
				labelCounts[oldLabel]--
			}
		}
	}

	var results []map[string]interface{}
	for label, count := range labelCounts {
		if count > 0 {
			results = append(results, map[string]interface{}{
				"label":          label,
				"correction_count": count,
			})
		}
	}
	return results, nil
}

// --- DB Stats for Health Endpoint ---

// DBStats holds database statistics for the health endpoint.
type DBStats struct {
	EventsCount       int64 `json:"events_count"`
	SegmentsCount     int64 `json:"segments_count"`
	DailySummariesCount int64 `json:"daily_summaries_count"`
	ObservationsCount int64 `json:"observations_count"`
	InterventionsCount int64 `json:"interventions_count"`
	FeedbackCount     int64 `json:"feedback_count"`
	GoalsCount        int64 `json:"goals_count"`
	HabitsCount       int64 `json:"habits_count"`
	BaselinesCount    int64 `json:"baselines_count"`
	DevicesCount      int64 `json:"devices_count"`
	AuditLogCount     int64 `json:"audit_log_count"`
}

// GetDBStats returns counts for all major tables.
func (db *DB) GetDBStats() (DBStats, error) {
	var stats DBStats
	tables := []struct {
		query string
		field *int64
	}{
		{"SELECT COUNT(*) FROM events", &stats.EventsCount},
		{"SELECT COUNT(*) FROM timeline_segments", &stats.SegmentsCount},
		{"SELECT COUNT(*) FROM daily_summary", &stats.DailySummariesCount},
		{"SELECT COUNT(*) FROM agent_observations", &stats.ObservationsCount},
		{"SELECT COUNT(*) FROM interventions", &stats.InterventionsCount},
		{"SELECT COUNT(*) FROM user_feedback", &stats.FeedbackCount},
		{"SELECT COUNT(*) FROM goals", &stats.GoalsCount},
		{"SELECT COUNT(*) FROM habits", &stats.HabitsCount},
		{"SELECT COUNT(*) FROM baselines", &stats.BaselinesCount},
		{"SELECT COUNT(*) FROM devices", &stats.DevicesCount},
		{"SELECT COUNT(*) FROM audit_log", &stats.AuditLogCount},
	}
	for _, t := range tables {
		_ = db.QueryRow(t.query).Scan(t.field)
	}
	return stats, nil
}
