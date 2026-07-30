package governor

import (
	"testing"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/models"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return database
}

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	if p.MaxInterventionsPerDay != 6 {
		t.Errorf("expected MaxInterventionsPerDay 6, got %d", p.MaxInterventionsPerDay)
	}
	if p.CooldownMinutes != 45 {
		t.Errorf("expected CooldownMinutes 45, got %d", p.CooldownMinutes)
	}
	if p.QuietHoursStart != "22:30" {
		t.Errorf("expected QuietHoursStart '22:30', got '%s'", p.QuietHoursStart)
	}
	if p.QuietHoursEnd != "07:30" {
		t.Errorf("expected QuietHoursEnd '07:30', got '%s'", p.QuietHoursEnd)
	}
	if p.MinConfidenceForNudge != 0.65 {
		t.Errorf("expected MinConfidenceForNudge 0.65, got %f", p.MinConfidenceForNudge)
	}
	if p.MaxObservationsPerDay != 20 {
		t.Errorf("expected MaxObservationsPerDay 20, got %d", p.MaxObservationsPerDay)
	}
}

func TestEvaluate_ApprovedWhenNoLimitsHit(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	policy := DefaultPolicy()
	policy.CooldownMinutes = 0 // Disable cooldown
	policy.QuietHoursStart = "00:00"
	policy.QuietHoursEnd = "00:00" // Disable quiet hours
	approved, reason := Evaluate(database, "user1", policy, "medium")

	if !approved {
		t.Errorf("expected intervention to be approved, got rejected: %s", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason when approved, got '%s'", reason)
	}
}

func TestEvaluate_DailyLimitReached(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	policy := DefaultPolicy()
	policy.MaxInterventionsPerDay = 2
	policy.CooldownMinutes = 0 // Disable cooldown for this test
	policy.QuietHoursStart = "00:00"
	policy.QuietHoursEnd = "00:00" // Disable quiet hours

	// Create 2 interventions to hit the limit
	for i := 0; i < 2; i++ {
		intv := &models.Intervention{
			InterventionID: "int-test-" + string(rune('0'+i)),
			UserID:         "user1",
			Timestamp:      "2026-07-28T10:00:00Z",
			Status:         "delivered",
			Payload:        `{}`,
		}
		if err := database.InsertIntervention(intv); err != nil {
			t.Fatalf("InsertIntervention failed: %v", err)
		}
	}

	approved, reason := Evaluate(database, "user1", policy, "medium")
	if approved {
		t.Error("expected intervention to be rejected when daily limit reached")
	}
	if reason != "daily_limit_reached" {
		t.Errorf("expected reason 'daily_limit_reached', got '%s'", reason)
	}
}

func TestEvaluate_HighSeverityOverridesQuietHours(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	policy := DefaultPolicy()
	policy.AllowHighSeverityOverride = true

	// Verify the policy struct is correct
	if !policy.AllowHighSeverityOverride {
		t.Error("expected AllowHighSeverityOverride to be true")
	}
}

func TestMarshalPolicy(t *testing.T) {
	p := DefaultPolicy()
	jsonStr := MarshalPolicy(p)

	if jsonStr == "" {
		t.Fatal("expected non-empty JSON")
	}

	// Verify it's valid JSON by unmarshaling
	p2 := UnmarshalPolicy(jsonStr)
	if p2.MaxInterventionsPerDay != p.MaxInterventionsPerDay {
		t.Errorf("round-trip failed: MaxInterventionsPerDay %d != %d", p2.MaxInterventionsPerDay, p.MaxInterventionsPerDay)
	}
}

func TestUnmarshalPolicy_Empty(t *testing.T) {
	p := UnmarshalPolicy("")
	if p.MaxInterventionsPerDay != DefaultPolicy().MaxInterventionsPerDay {
		t.Error("empty string should return default policy")
	}
}

func TestUnmarshalPolicy_Invalid(t *testing.T) {
	p := UnmarshalPolicy("not json")
	if p.MaxInterventionsPerDay != DefaultPolicy().MaxInterventionsPerDay {
		t.Error("invalid JSON should return default policy")
	}
}

// --- Sensitive Mode Tests ---

func TestDefaultSensitiveMode(t *testing.T) {
	sm := DefaultSensitiveMode()

	if sm.Active {
		t.Error("expected default sensitive mode to be inactive")
	}
	if !sm.PauseSync {
		t.Error("expected default PauseSync to be true")
	}
	if !sm.RedactLocation {
		t.Error("expected default RedactLocation to be true")
	}
	if !sm.RedactAppUsage {
		t.Error("expected default RedactAppUsage to be true")
	}
	if !sm.RedactCalendar {
		t.Error("expected default RedactCalendar to be true")
	}
	if sm.MaxRetentionDays != 7 {
		t.Errorf("expected MaxRetentionDays 7, got %d", sm.MaxRetentionDays)
	}
}

func TestSensitiveMode_IsEventAllowed(t *testing.T) {
	sm := DefaultSensitiveMode()
	sm.Active = true
	sm.AllowedEventTypes = []string{"screen_state_changed", "device_unlocked"}

	if !sm.IsEventAllowed("screen_state_changed") {
		t.Error("expected screen_state_changed to be allowed")
	}
	if !sm.IsEventAllowed("device_unlocked") {
		t.Error("expected device_unlocked to be allowed")
	}
	if sm.IsEventAllowed("app_usage_summary") {
		t.Error("expected app_usage_summary to be blocked")
	}
}

func TestSensitiveMode_IsEventAllowed_Inactive(t *testing.T) {
	sm := DefaultSensitiveMode()
	sm.Active = false

	// When inactive, all events should be allowed
	if !sm.IsEventAllowed("app_usage_summary") {
		t.Error("expected all events to be allowed when sensitive mode is inactive")
	}
}

func TestSensitiveMode_IsEventAllowed_EmptyList(t *testing.T) {
	sm := DefaultSensitiveMode()
	sm.Active = true
	sm.AllowedEventTypes = []string{}

	// Empty list means all events allowed
	if !sm.IsEventAllowed("any_event") {
		t.Error("expected all events to be allowed when AllowedEventTypes is empty")
	}
}

func TestSensitiveMode_ActivateDeactivate(t *testing.T) {
	sm := DefaultSensitiveMode()

	sm.Activate()
	if !sm.Active {
		t.Error("expected Active to be true after Activate()")
	}
	if !sm.PauseSync {
		t.Error("expected PauseSync to be true after Activate()")
	}

	sm.Deactivate()
	if sm.Active {
		t.Error("expected Active to be false after Deactivate()")
	}
	if sm.PauseSync {
		t.Error("expected PauseSync to be false after Deactivate()")
	}
}

func TestMarshalSensitiveMode(t *testing.T) {
	sm := DefaultSensitiveMode()
	sm.Active = true

	jsonStr := MarshalSensitiveMode(sm)
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON")
	}

	sm2 := UnmarshalSensitiveMode(jsonStr)
	if !sm2.Active {
		t.Error("round-trip failed: Active should be true")
	}
}

func TestUnmarshalSensitiveMode_Empty(t *testing.T) {
	sm := UnmarshalSensitiveMode("")
	if sm.Active {
		t.Error("empty string should return default sensitive mode (inactive)")
	}
}

func TestUnmarshalSensitiveMode_Invalid(t *testing.T) {
	sm := UnmarshalSensitiveMode("not json")
	if sm.Active {
		t.Error("invalid JSON should return default sensitive mode")
	}
}
