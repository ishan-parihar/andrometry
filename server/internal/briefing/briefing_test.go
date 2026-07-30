package briefing

import (
	"encoding/json"
	"testing"
)

func TestGenerate_SimpleContext(t *testing.T) {
	contextJSON := `{
		"primary_activity": "deep_work",
		"place": "office",
		"focus_active": true,
		"distraction_probability": 0.18,
		"fatigue_probability": 0.31,
		"alignment_probability": 0.74,
		"readiness_score": 0.68
	}`

	b, err := Generate(contextJSON, "briefing-1")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if b.BriefingID != "briefing-1" {
		t.Errorf("expected briefing_id 'briefing-1', got '%s'", b.BriefingID)
	}
	if b.GeneratedAt == "" {
		t.Error("expected non-empty generated_at")
	}
	if b.State.PrimaryActivity != "deep_work" {
		t.Errorf("expected primary_activity 'deep_work', got '%s'", b.State.PrimaryActivity)
	}
	if b.State.Place != "office" {
		t.Errorf("expected place 'office', got '%s'", b.State.Place)
	}
	if !b.State.FocusActive {
		t.Error("expected focus_active to be true")
	}
	if b.State.DistractionProbability != 0.18 {
		t.Errorf("expected distraction_probability 0.18, got %f", b.State.DistractionProbability)
	}
}

func TestGenerate_NestedContext(t *testing.T) {
	contextJSON := `{
		"timestamp": "2026-07-28T10:14:00Z",
		"timezone": "Asia/Kolkata",
		"context": {
			"primary_activity": "meeting",
			"place": "office",
			"focus_active": false,
			"distraction_probability": 0.45,
			"fatigue_probability": 0.6,
			"alignment_probability": 0.5,
			"readiness_score": 0.4
		}
	}`

	b, err := Generate(contextJSON, "briefing-2")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if b.State.PrimaryActivity != "meeting" {
		t.Errorf("expected primary_activity 'meeting', got '%s'", b.State.PrimaryActivity)
	}
	if b.State.FocusActive {
		t.Error("expected focus_active to be false")
	}
}

func TestGenerate_DetectsRisks(t *testing.T) {
	contextJSON := `{
		"primary_activity": "distraction",
		"distraction_probability": 0.85,
		"fatigue_probability": 0.75,
		"interruption_rate": "high"
	}`

	b, err := Generate(contextJSON, "briefing-3")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(b.Risks) < 2 {
		t.Fatalf("expected at least 2 risks, got %d", len(b.Risks))
	}

	// Check for focus drift risk
	foundFocusDrift := false
	for _, r := range b.Risks {
		if r.Type == "focus_drift" {
			foundFocusDrift = true
			if r.Severity != "high" {
				t.Errorf("expected focus_drift severity 'high', got '%s'", r.Severity)
			}
		}
	}
	if !foundFocusDrift {
		t.Error("expected focus_drift risk to be detected")
	}

	// Check for fatigue risk
	foundFatigue := false
	for _, r := range b.Risks {
		if r.Type == "fatigue" {
			foundFatigue = true
		}
	}
	if !foundFatigue {
		t.Error("expected fatigue risk to be detected")
	}
}

func TestGenerate_DetectsOpportunities(t *testing.T) {
	contextJSON := `{
		"primary_activity": "deep_work",
		"focus_active": true,
		"alignment_probability": 0.82,
		"fatigue_probability": 0.15
	}`

	b, err := Generate(contextJSON, "briefing-4")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(b.Opportunities) < 1 {
		t.Fatalf("expected at least 1 opportunity, got %d", len(b.Opportunities))
	}

	foundExtendFocus := false
	for _, o := range b.Opportunities {
		if o.Type == "extend_focus_block" {
			foundExtendFocus = true
		}
	}
	if !foundExtendFocus {
		t.Error("expected extend_focus_block opportunity when focus is active")
	}
}

func TestGenerate_SuggestedQueries(t *testing.T) {
	contextJSON := `{
		"primary_activity": "deep_work",
		"focus_active": true,
		"distraction_probability": 0.5,
		"fatigue_probability": 0.6
	}`

	b, err := Generate(contextJSON, "briefing-5")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(b.SuggestedQueries) < 3 {
		t.Errorf("expected at least 3 suggested queries, got %d", len(b.SuggestedQueries))
	}

	// Should always include get_current_context
	found := false
	for _, q := range b.SuggestedQueries {
		if q == "get_current_context" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'get_current_context' in suggested queries")
	}
}

func TestGenerate_LowDistraction(t *testing.T) {
	contextJSON := `{
		"primary_activity": "deep_work",
		"distraction_probability": 0.1,
		"fatigue_probability": 0.2,
		"alignment_probability": 0.9
	}`

	b, err := Generate(contextJSON, "briefing-6")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Low distraction should not generate focus_drift risk
	for _, r := range b.Risks {
		if r.Type == "focus_drift" {
			t.Error("low distraction probability should not generate focus_drift risk")
		}
	}
}

func TestGenerate_InvalidJSON(t *testing.T) {
	_, err := Generate("not json", "briefing-7")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGenerate_EmptyJSON(t *testing.T) {
	_, err := Generate("{}", "briefing-8")
	if err != nil {
		t.Fatalf("Generate failed for empty JSON: %v", err)
	}
}

func TestBriefingToJSON(t *testing.T) {
	b := &Briefing{
		BriefingID:  "test-id",
		GeneratedAt: "2026-07-28T10:00:00Z",
		State: BriefingState{
			PrimaryActivity: "deep_work",
			Place:           "office",
			FocusActive:     true,
		},
		Risks:            []Risk{},
		Opportunities:    []Opportunity{},
		SuggestedQueries: []string{"get_current_context"},
	}

	jsonStr := BriefingToJSON(b)
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("BriefingToJSON produced invalid JSON: %v", err)
	}

	if parsed["briefing_id"] != "test-id" {
		t.Errorf("expected briefing_id 'test-id', got '%v'", parsed["briefing_id"])
	}
}

func TestGenerate_MinimalContext(t *testing.T) {
	contextJSON := `{"primary_activity": "unknown"}`

	b, err := Generate(contextJSON, "briefing-9")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if b.State.PrimaryActivity != "unknown" {
		t.Errorf("expected primary_activity 'unknown', got '%s'", b.State.PrimaryActivity)
	}
	// Should have default suggested queries
	if len(b.SuggestedQueries) < 1 {
		t.Error("expected at least 1 suggested query")
	}
}

func TestGenerate_HighFatigueNoFocus(t *testing.T) {
	contextJSON := `{
		"primary_activity": "leisure",
		"focus_active": false,
		"fatigue_probability": 0.15,
		"alignment_probability": 0.3
	}`

	b, err := Generate(contextJSON, "briefing-10")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Low fatigue + no focus should suggest starting focus
	foundStartFocus := false
	for _, o := range b.Opportunities {
		if o.Type == "start_focus" {
			foundStartFocus = true
		}
	}
	if !foundStartFocus {
		t.Error("expected start_focus opportunity when fatigue is low and focus is not active")
	}
}
