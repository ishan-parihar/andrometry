package models

import (
	"encoding/json"
	"testing"
)

func TestDevice_Fields(t *testing.T) {
	d := Device{
		DeviceID:      "device-1",
		UserID:        "user-1",
		ClientVersion: "0.1.0",
		Status:        "active",
	}

	if d.DeviceID != "device-1" {
		t.Errorf("expected DeviceID 'device-1', got '%s'", d.DeviceID)
	}
	if d.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got '%s'", d.UserID)
	}
	if d.ClientVersion != "0.1.0" {
		t.Errorf("expected ClientVersion '0.1.0', got '%s'", d.ClientVersion)
	}
}

func TestDevice_JSON(t *testing.T) {
	d := Device{
		DeviceID: "device-1",
		UserID:   "user-1",
		Status:   "active",
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var d2 Device
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if d2.DeviceID != d.DeviceID {
		t.Errorf("round-trip DeviceID mismatch: %s != %s", d2.DeviceID, d.DeviceID)
	}
}

func TestEvent_Fields(t *testing.T) {
	e := Event{
		EventID:       "event-1",
		DeviceID:      "device-1",
		UserID:        "user-1",
		Sequence:      100,
		Timestamp:     "2026-07-28T10:00:00Z",
		Timezone:      "Asia/Kolkata",
		EventType:     "app_usage_summary",
		SchemaVersion: 1,
		Payload:       `{"key":"value"}`,
	}

	if e.Sequence != 100 {
		t.Errorf("expected Sequence 100, got %d", e.Sequence)
	}
	if e.EventType != "app_usage_summary" {
		t.Errorf("expected EventType 'app_usage_summary', got '%s'", e.EventType)
	}
}

func TestTimelineSegment_Fields(t *testing.T) {
	s := TimelineSegment{
		SegmentID:   "seg-1",
		DeviceID:    "device-1",
		UserID:      "user-1",
		SegmentType: "focus_segment",
		Layer:       "focus",
		StartTime:   "2026-07-28T09:00:00Z",
		EndTime:     "2026-07-28T11:00:00Z",
		Label:       "deep_work",
		Confidence:  0.85,
		Version:     1,
		Payload:     `{"interruptions":3}`,
	}

	if s.SegmentType != "focus_segment" {
		t.Errorf("expected SegmentType 'focus_segment', got '%s'", s.SegmentType)
	}
	if s.Confidence != 0.85 {
		t.Errorf("expected Confidence 0.85, got %f", s.Confidence)
	}
}

func testTimelineSegmentJSON(t *testing.T) {
	s := TimelineSegment{
		SegmentID:   "seg-1",
		SegmentType: "focus_segment",
		StartTime:   "2026-07-28T09:00:00Z",
		EndTime:     "2026-07-28T11:00:00Z",
		Label:       "deep_work",
		Confidence:  0.85,
		Version:     1,
		Payload:     `{"interruptions":3}`,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var s2 TimelineSegment
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if s2.SegmentID != s.SegmentID {
		t.Errorf("round-trip SegmentID mismatch")
	}
}

func TestDailySummary_Fields(t *testing.T) {
	ds := DailySummary{
		UserID:   "user-1",
		Date:     "2026-07-28",
		Timezone: "Asia/Kolkata",
		Summary:  `{"deep_work_minutes":182}`,
		Version:  1,
	}

	if ds.Date != "2026-07-28" {
		t.Errorf("expected Date '2026-07-28', got '%s'", ds.Date)
	}
}

func TestWeeklySummary_Fields(t *testing.T) {
	ws := WeeklySummary{
		UserID:   "user-1",
		Week:     "2026-W31",
		Timezone: "Asia/Kolkata",
		Summary:  `{"total_focus_minutes":1200}`,
		Version:  1,
	}

	if ws.Week != "2026-W31" {
		t.Errorf("expected Week '2026-W31', got '%s'", ws.Week)
	}
}

func TestAgentObservation_Fields(t *testing.T) {
	obs := AgentObservation{
		ObservationID:   "obs-1",
		UserID:          "user-1",
		Timestamp:       "2026-07-28T10:00:00Z",
		AgentID:         "agent-1",
		ObservationType: "focus_drift",
		Severity:        "medium",
		Payload:         `{"message":"distraction detected"}`,
	}

	if obs.ObservationType != "focus_drift" {
		t.Errorf("expected ObservationType 'focus_drift', got '%s'", obs.ObservationType)
	}
	if obs.Severity != "medium" {
		t.Errorf("expected Severity 'medium', got '%s'", obs.Severity)
	}
}

func TestIntervention_Fields(t *testing.T) {
	intv := Intervention{
		InterventionID:   "int-1",
		UserID:           "user-1",
		Timestamp:        "2026-07-28T10:00:00Z",
		AgentID:          "agent-1",
		InterventionType: "nudge",
		Status:           "approved_by_policy",
		Payload:          `{"title":"Take a break"}`,
	}

	if intv.Status != "approved_by_policy" {
		t.Errorf("expected Status 'approved_by_policy', got '%s'", intv.Status)
	}
}

func TestGoal_Fields(t *testing.T) {
	g := Goal{
		GoalID:   "goal-1",
		UserID:   "user-1",
		GoalType: "productivity",
		Status:   "active",
		Payload:  `{"title":"Deep work 3 hours daily"}`,
	}

	if g.GoalType != "productivity" {
		t.Errorf("expected GoalType 'productivity', got '%s'", g.GoalType)
	}
	if g.Status != "active" {
		t.Errorf("expected Status 'active', got '%s'", g.Status)
	}
}

func TestHabit_Fields(t *testing.T) {
	h := Habit{
		HabitID:   "habit-1",
		UserID:    "user-1",
		HabitType: "exercise",
		Status:    "active",
		StreakDays: 7,
	}

	if h.StreakDays != 7 {
		t.Errorf("expected StreakDays 7, got %d", h.StreakDays)
	}
	if h.Status != "active" {
		t.Errorf("expected Status 'active', got '%s'", h.Status)
	}
}

func TestHabitStrength_Pointer(t *testing.T) {
	strength := 0.75
	h := Habit{
		HabitID:       "habit-1",
		HabitStrength: &strength,
	}

	if h.HabitStrength == nil {
		t.Fatal("expected non-nil HabitStrength")
	}
	if *h.HabitStrength != 0.75 {
		t.Errorf("expected HabitStrength 0.75, got %f", *h.HabitStrength)
	}

	// Test nil case
	h2 := Habit{HabitID: "habit-2"}
	if h2.HabitStrength != nil {
		t.Error("expected nil HabitStrength when not set")
	}
}

func TestBaseline_Fields(t *testing.T) {
	b := Baseline{
		BaselineID: "bl-1",
		UserID:     "user-1",
		MetricName: "deep_work_minutes",
		WindowDays: 7,
		Value:      180.5,
	}

	if b.MetricName != "deep_work_minutes" {
		t.Errorf("expected MetricName 'deep_work_minutes', got '%s'", b.MetricName)
	}
	if b.Value != 180.5 {
		t.Errorf("expected Value 180.5, got %f", b.Value)
	}
}

func TestOutcome_Fields(t *testing.T) {
	score := 0.82
	o := Outcome{
		OutcomeID:          "out-1",
		UserID:             "user-1",
		InterventionID:     "int-1",
		OutcomeType:        "behavior_change",
		EffectivenessScore: &score,
	}

	if o.OutcomeType != "behavior_change" {
		t.Errorf("expected OutcomeType 'behavior_change', got '%s'", o.OutcomeType)
	}
	if o.EffectivenessScore == nil || *o.EffectivenessScore != 0.82 {
		t.Error("expected EffectivenessScore 0.82")
	}
}

func TestAuditLog_Fields(t *testing.T) {
	al := AuditLog{
		AuditID:    "audit-1",
		UserID:     "user-1",
		ActorType:  "agent",
		ActorID:    "agent-1",
		Action:     "create_observation",
		TargetType: "observation",
		TargetID:   "obs-1",
	}

	if al.ActorType != "agent" {
		t.Errorf("expected ActorType 'agent', got '%s'", al.ActorType)
	}
	if al.Action != "create_observation" {
		t.Errorf("expected Action 'create_observation', got '%s'", al.Action)
	}
}

func TestUserFeedback_Fields(t *testing.T) {
	fb := UserFeedback{
		FeedbackID:   "fb-1",
		UserID:       "user-1",
		TargetType:   "intervention",
		TargetID:     "int-1",
		FeedbackType: "accepted",
	}

	if fb.FeedbackType != "accepted" {
		t.Errorf("expected FeedbackType 'accepted', got '%s'", fb.FeedbackType)
	}
}

func TestAuthToken_Fields(t *testing.T) {
	at := AuthToken{
		TokenID:   "tok-1",
		UserID:    "user-1",
		TokenType: "device",
		Scopes:    "device:ingest,device:read_config",
	}

	if at.TokenType != "device" {
		t.Errorf("expected TokenType 'device', got '%s'", at.TokenType)
	}
	// TokenHash should be excluded from JSON
	data, _ := json.Marshal(at)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if _, exists := m["token_hash"]; exists {
		t.Error("expected token_hash to be excluded from JSON output")
	}
}
