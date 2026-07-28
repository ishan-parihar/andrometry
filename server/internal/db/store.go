package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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

// ValidateToken checks if a token hash is valid and returns its info.
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
	return &t, nil
}

// HasScope checks if a token's scopes string contains the required scope.
func HasScope(scopes, required string) bool {
	if scopes == "admin" || scopes == "*" {
		return true
	}
	for _, s := range splitScopes(scopes) {
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
func (db *DB) InsertUserCorrection(userID, correctionJSON string) error {
	_, err := db.Exec(`
		INSERT INTO audit_log (audit_id, user_id, actor_type, actor_id, action, target_type, metadata)
		VALUES (?, ?, 'user', ?, 'user_correction', 'segment', ?)
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
		WHERE user_id = ? AND action = 'user_correction'
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
		WHERE user_id = ? AND audit_id = ? AND action = 'user_correction'
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
