package db

import (
	"testing"
	"time"

	"github.com/andrometry/server/internal/models"
)

func TestOpen_createsDatabase(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	tables := []string{
		"devices", "auth_tokens", "events", "current_context",
		"timeline_segments", "daily_summary", "weekly_summary",
		"five_minute_summary", "hourly_summary", "agent_observations",
		"interventions", "user_feedback", "goals", "habits",
		"baselines", "outcomes", "audit_log", "policy_settings",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestHashToken_deterministic(t *testing.T) {
	h1 := HashToken("test-token-123")
	h2 := HashToken("test-token-123")
	if h1 != h2 {
		t.Errorf("HashToken not deterministic: %s != %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("HashToken length = %d, want 64", len(h1))
	}
}

func TestHashToken_differentInputs(t *testing.T) {
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if h1 == h2 {
		t.Error("HashToken produced same hash for different inputs")
	}
}

func TestInsertEventsBatch_idempotent(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	events := []*models.Event{
		{
			EventID:       "evt-1",
			DeviceID:      "device-1",
			UserID:        "primary",
			Sequence:      1,
			Timestamp:     "2026-01-01T00:00:00Z",
			Timezone:      "UTC",
			EventType:     "screen_state_changed",
			SchemaVersion: 1,
			Payload:       `{"state":"on"}`,
		},
	}

	accepted, err := db.InsertEventsBatch(events)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}
	if accepted != 1 {
		t.Errorf("first insert accepted = %d, want 1", accepted)
	}

	accepted, err = db.InsertEventsBatch(events)
	if err != nil {
		t.Fatalf("second insert failed: %v", err)
	}
	if accepted != 0 {
		t.Errorf("second insert accepted = %d, want 0 (idempotent)", accepted)
	}
}

func TestInsertEventsBatch_multipleEvents(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	events := []*models.Event{
		{EventID: "evt-1", DeviceID: "d1", UserID: "primary", Sequence: 1, Timestamp: "2026-01-01T00:00:00Z", Timezone: "UTC", EventType: "screen_on", SchemaVersion: 1, Payload: "{}"},
		{EventID: "evt-2", DeviceID: "d1", UserID: "primary", Sequence: 2, Timestamp: "2026-01-01T00:01:00Z", Timezone: "UTC", EventType: "screen_off", SchemaVersion: 1, Payload: "{}"},
		{EventID: "evt-3", DeviceID: "d1", UserID: "primary", Sequence: 3, Timestamp: "2026-01-01T00:02:00Z", Timezone: "UTC", EventType: "app_usage_summary", SchemaVersion: 1, Payload: "{}"},
	}

	accepted, err := db.InsertEventsBatch(events)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if accepted != 3 {
		t.Errorf("accepted = %d, want 3", accepted)
	}
}

func TestUpsertContext_andGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := &models.CurrentContext{
		DeviceID:  "pixel-9",
		UserID:    "primary",
		Timestamp: "2026-01-01T10:00:00Z",
		Timezone:  "Asia/Kolkata",
		Context:   `{"primary_activity":"deep_work"}`,
		ETag:      "abc123",
	}

	if err := db.UpsertContext(ctx); err != nil {
		t.Fatalf("UpsertContext failed: %v", err)
	}

	got, err := db.GetContext("pixel-9")
	if err != nil {
		t.Fatalf("GetContext failed: %v", err)
	}
	if got.Context != ctx.Context {
		t.Errorf("context = %s, want %s", got.Context, ctx.Context)
	}
	if got.ETag != ctx.ETag {
		t.Errorf("etag = %s, want %s", got.ETag, ctx.ETag)
	}

	ctx.Context = `{"primary_activity":"meeting"}`
	ctx.ETag = "def456"
	if err := db.UpsertContext(ctx); err != nil {
		t.Fatalf("UpsertContext update failed: %v", err)
	}

	got, err = db.GetContext("pixel-9")
	if err != nil {
		t.Fatalf("GetContext after update failed: %v", err)
	}
	if got.Context != `{"primary_activity":"meeting"}` {
		t.Errorf("context after update = %s", got.Context)
	}
}

func TestUpsertSegment_andGetTimeline(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	seg := &models.TimelineSegment{
		SegmentID:   "seg-1",
		DeviceID:    "d1",
		UserID:      "primary",
		SegmentType: "focus_segment",
		Layer:       "focus",
		StartTime:   "2026-01-01T09:00:00Z",
		EndTime:     "2026-01-01T10:00:00Z",
		Label:       "deep_work",
		Confidence:  0.85,
		Version:     1,
		ValidFrom:   "2026-01-01T09:00:00Z",
		Evidence:    `{"focus_active":true}`,
		Payload:     `{"interruptions":2}`,
	}

	if err := db.UpsertSegment(seg); err != nil {
		t.Fatalf("UpsertSegment failed: %v", err)
	}

	segments, err := db.GetTimeline("primary", "2026-01-01T00:00:00Z", "2026-01-01T23:59:59Z", 100, "")
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("GetTimeline returned %d segments, want 1", len(segments))
	}
	if segments[0].Label != "deep_work" {
		t.Errorf("label = %s, want deep_work", segments[0].Label)
	}
}

func TestUpsertSegment_versionUpdate(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	seg := &models.TimelineSegment{
		SegmentID: "seg-1", DeviceID: "d1", UserID: "primary",
		SegmentType: "activity_segment", Layer: "activity",
		StartTime: "2026-01-01T09:00:00Z", EndTime: "2026-01-01T10:00:00Z",
		Label: "shallow_work", Confidence: 0.7, Version: 1,
		ValidFrom: "2026-01-01T09:00:00Z", Payload: "{}",
	}
	db.UpsertSegment(seg)

	seg.Version = 2
	seg.Label = "deep_work"
	seg.Confidence = 0.9
	db.UpsertSegment(seg)

	segments, _ := db.GetTimeline("primary", "2026-01-01T00:00:00Z", "2026-01-01T23:59:59Z", 100, "")
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	if segments[0].Label != "deep_work" {
		t.Errorf("label = %s, want deep_work after update", segments[0].Label)
	}
	if segments[0].Version != 2 {
		t.Errorf("version = %d, want 2", segments[0].Version)
	}
}

func TestGetTimeline_filterByType(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	segs := []*models.TimelineSegment{
		{SegmentID: "seg-1", DeviceID: "d1", UserID: "primary", SegmentType: "focus_segment", Layer: "focus", StartTime: "2026-01-01T09:00:00Z", EndTime: "2026-01-01T10:00:00Z", Label: "deep_work", Confidence: 0.85, Version: 1, ValidFrom: "2026-01-01T09:00:00Z", Payload: "{}"},
		{SegmentID: "seg-2", DeviceID: "d1", UserID: "primary", SegmentType: "place_visit", Layer: "place", StartTime: "2026-01-01T08:00:00Z", EndTime: "2026-01-01T12:00:00Z", Label: "office", Confidence: 0.93, Version: 1, ValidFrom: "2026-01-01T08:00:00Z", Payload: "{}"},
		{SegmentID: "seg-3", DeviceID: "d1", UserID: "primary", SegmentType: "focus_segment", Layer: "focus", StartTime: "2026-01-01T14:00:00Z", EndTime: "2026-01-01T15:00:00Z", Label: "shallow_work", Confidence: 0.7, Version: 1, ValidFrom: "2026-01-01T14:00:00Z", Payload: "{}"},
	}
	db.UpsertSegmentsBatch(segs)

	focusOnly, err := db.GetTimeline("primary", "2026-01-01T00:00:00Z", "2026-01-01T23:59:59Z", 100, "focus_segment")
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}
	if len(focusOnly) != 2 {
		t.Errorf("focus segments = %d, want 2", len(focusOnly))
	}
}

func TestDailySummary_upsertAndGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	summary := &models.DailySummary{
		UserID:   "primary",
		Date:     "2026-01-01",
		Timezone: "Asia/Kolkata",
		Summary:  `{"deep_work_minutes":120}`,
		Version:  1,
	}

	if err := db.UpsertDailySummary(summary); err != nil {
		t.Fatalf("UpsertDailySummary failed: %v", err)
	}

	got, err := db.GetDailySummary("primary", "2026-01-01")
	if err != nil {
		t.Fatalf("GetDailySummary failed: %v", err)
	}
	if got.Summary != summary.Summary {
		t.Errorf("summary = %s, want %s", got.Summary, summary.Summary)
	}

	summary.Summary = `{"deep_work_minutes":180}`
	summary.Version = 2
	db.UpsertDailySummary(summary)

	got, _ = db.GetDailySummary("primary", "2026-01-01")
	if got.Version != 2 {
		t.Errorf("version = %d, want 2", got.Version)
	}
}

func TestInsertObservation_andGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	obs := &models.AgentObservation{
		ObservationID:   "obs-1",
		UserID:          "primary",
		Timestamp:       "2026-01-01T10:00:00Z",
		AgentID:         "coach",
		ObservationType: "risk",
		Severity:        "medium",
		Payload:         `{"message":"focus drifting"}`,
	}

	if err := db.InsertObservation(obs); err != nil {
		t.Fatalf("InsertObservation failed: %v", err)
	}

	observations, err := db.GetObservations("primary", 10)
	if err != nil {
		t.Fatalf("GetObservations failed: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(observations))
	}
	if observations[0].AgentID != "coach" {
		t.Errorf("agent_id = %s, want coach", observations[0].AgentID)
	}
}

func TestInsertIntervention_andStatusUpdate(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	intervention := &models.Intervention{
		InterventionID:   "int-1",
		UserID:           "primary",
		Timestamp:        "2026-01-01T10:00:00Z",
		AgentID:          "coach",
		InterventionType: "nudge",
		Status:           "proposed",
		Payload:          `{"title":"Take a break"}`,
	}

	if err := db.InsertIntervention(intervention); err != nil {
		t.Fatalf("InsertIntervention failed: %v", err)
	}

	if err := db.UpdateInterventionStatus("int-1", "delivered"); err != nil {
		t.Fatalf("UpdateInterventionStatus failed: %v", err)
	}

	pending, err := db.GetPendingInterventions("primary", 10)
	if err != nil {
		t.Fatalf("GetPendingInterventions failed: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("pending = %d, want 0", len(pending))
	}
}

func TestInsertFeedback(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	fb := &models.UserFeedback{
		FeedbackID:   "fb-1",
		UserID:       "primary",
		Timestamp:    "2026-01-01T10:05:00Z",
		TargetType:   "intervention",
		TargetID:     "int-1",
		FeedbackType: "helpful",
		Payload:      `{"rating":4}`,
	}

	if err := db.InsertFeedback(fb); err != nil {
		t.Fatalf("InsertFeedback failed: %v", err)
	}
}

func TestInsertGoal_andGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	goal := &models.Goal{
		GoalID:   "goal-1",
		UserID:   "primary",
		GoalType: "deep_work_minutes",
		Status:   "active",
		Payload:  `{"target":120}`,
	}

	if err := db.InsertGoal(goal); err != nil {
		t.Fatalf("InsertGoal failed: %v", err)
	}

	goals, err := db.GetGoals("primary")
	if err != nil {
		t.Fatalf("GetGoals failed: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("goals = %d, want 1", len(goals))
	}
	if goals[0].GoalType != "deep_work_minutes" {
		t.Errorf("goal_type = %s, want deep_work_minutes", goals[0].GoalType)
	}
}

func TestInsertHabit_andGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	habit := &models.Habit{
		HabitID:         "habit-1",
		UserID:          "primary",
		HabitType:       "morning_routine",
		Target:          `{"metric":"wake_before_7am"}`,
		Schedule:        `{"days":["mon","tue","wed","thu","fri"]}`,
		EvidenceSources: `["screen_state"]`,
		Status:          "active",
		StreakDays:      5,
	}

	if err := db.InsertHabit(habit); err != nil {
		t.Fatalf("InsertHabit failed: %v", err)
	}

	habits, err := db.GetHabits("primary")
	if err != nil {
		t.Fatalf("GetHabits failed: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("habits = %d, want 1", len(habits))
	}
	if habits[0].StreakDays != 5 {
		t.Errorf("streak_days = %d, want 5", habits[0].StreakDays)
	}
}

func TestInsertOutcome(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	score := 0.75
	outcome := &models.Outcome{
		OutcomeID:          "out-1",
		UserID:             "primary",
		InterventionID:     "int-1",
		Timestamp:          "2026-01-01T10:30:00Z",
		OutcomeType:        "behavior_change",
		EffectivenessScore: &score,
		Payload:            `{"distraction_before":12,"distraction_after":3}`,
	}

	if err := db.InsertOutcome(outcome); err != nil {
		t.Fatalf("InsertOutcome failed: %v", err)
	}

	outcomes, err := db.GetOutcomes("primary", "", 10)
	if err != nil {
		t.Fatalf("GetOutcomes failed: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(outcomes))
	}
}

func TestInsertAuditLog(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	entry := &models.AuditLog{
		AuditID:    "audit-1",
		UserID:     "primary",
		ActorType:  "agent",
		ActorID:    "coach",
		Action:     "create_observation",
		TargetType: "observation",
		TargetID:   "obs-1",
		Metadata:   `{"severity":"medium"}`,
	}

	if err := db.InsertAuditLog(entry); err != nil {
		t.Fatalf("InsertAuditLog failed: %v", err)
	}

	logs, err := db.GetAuditLog("primary", 10)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(logs))
	}
	if logs[0].Action != "create_observation" {
		t.Errorf("action = %s, want create_observation", logs[0].Action)
	}
}

func TestCountInterventionsToday(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	count, err := db.CountInterventionsToday("primary")
	if err != nil {
		t.Fatalf("CountInterventionsToday failed: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-1", UserID: "primary",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		AgentID: "coach", InterventionType: "nudge", Status: "proposed", Payload: "{}",
	})

	count, err = db.CountInterventionsToday("primary")
	if err != nil {
		t.Fatalf("CountInterventionsToday after insert failed: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestGetEvents_filterByType(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	events := []*models.Event{
		{EventID: "e1", DeviceID: "d1", UserID: "primary", Sequence: 1, Timestamp: "2026-01-01T09:00:00Z", Timezone: "UTC", EventType: "screen_on", SchemaVersion: 1, Payload: "{}"},
		{EventID: "e2", DeviceID: "d1", UserID: "primary", Sequence: 2, Timestamp: "2026-01-01T09:01:00Z", Timezone: "UTC", EventType: "app_usage_summary", SchemaVersion: 1, Payload: "{}"},
		{EventID: "e3", DeviceID: "d1", UserID: "primary", Sequence: 3, Timestamp: "2026-01-01T09:02:00Z", Timezone: "UTC", EventType: "screen_off", SchemaVersion: 1, Payload: "{}"},
	}
	db.InsertEventsBatch(events)

	screenEvents, err := db.GetEvents("primary", "2026-01-01T00:00:00Z", "2026-01-01T23:59:59Z", "screen_on", 100)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}
	if len(screenEvents) != 1 {
		t.Errorf("screen events = %d, want 1", len(screenEvents))
	}
}
