package db

import (
	"testing"

	"github.com/andrometry/server/internal/models"
)

func TestGetOutcomesByInterventionIDs_filtersCorrectly(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Insert interventions for two different agents
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-a1", UserID: "primary", Timestamp: "2026-01-01T10:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-a2", UserID: "primary", Timestamp: "2026-01-01T10:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-b1", UserID: "primary", Timestamp: "2026-01-01T10:00:00Z",
		AgentID: "other", InterventionType: "alert", Status: "delivered", Payload: "{}",
	})

	// Insert outcomes for each intervention
	scoreA := 0.8
	scoreB := 0.5
	db.InsertOutcome(&models.Outcome{
		OutcomeID: "out-a1", UserID: "primary", InterventionID: "int-a1",
		Timestamp: "2026-01-01T10:05:00Z", OutcomeType: "accepted", EffectivenessScore: &scoreA, Payload: "{}",
	})
	db.InsertOutcome(&models.Outcome{
		OutcomeID: "out-a2", UserID: "primary", InterventionID: "int-a2",
		Timestamp: "2026-01-01T10:05:00Z", OutcomeType: "behavior_change", EffectivenessScore: &scoreA, Payload: "{}",
	})
	db.InsertOutcome(&models.Outcome{
		OutcomeID: "out-b1", UserID: "primary", InterventionID: "int-b1",
		Timestamp: "2026-01-01T10:05:00Z", OutcomeType: "rejected", EffectivenessScore: &scoreB, Payload: "{}",
	})

	// Query only outcomes for coach's interventions
	outcomes, err := db.GetOutcomesByInterventionIDs("primary", []string{"int-a1", "int-a2"}, 100)
	if err != nil {
		t.Fatalf("GetOutcomesByInterventionIDs failed: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("outcomes count = %d, want 2", len(outcomes))
	}

	// Verify no outcomes from other agent's interventions
	for _, o := range outcomes {
		if o.InterventionID == "int-b1" {
			t.Errorf("should not include outcome for int-b1 (other agent)")
		}
	}
}

func TestGetOutcomesByInterventionIDs_limitRespected(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Insert 5 outcomes for one intervention
	for i := 0; i < 5; i++ {
		score := 0.5
		db.InsertOutcome(&models.Outcome{
			OutcomeID: "out-" + string(rune('a'+i)), UserID: "primary", InterventionID: "int-1",
			Timestamp: "2026-01-01T10:00:0Z", OutcomeType: "accepted", EffectivenessScore: &score, Payload: "{}",
		})
	}

	// Query with limit 2
	outcomes, err := db.GetOutcomesByInterventionIDs("primary", []string{"int-1"}, 2)
	if err != nil {
		t.Fatalf("GetOutcomesByInterventionIDs failed: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("outcomes count = %d, want 2 (limit)", len(outcomes))
	}
}

func TestGetOutcomesByInterventionIDs_emptyIDs(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	outcomes, err := db.GetOutcomesByInterventionIDs("primary", []string{}, 100)
	if err != nil {
		t.Fatalf("GetOutcomesByInterventionIDs failed: %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("outcomes count = %d, want 0 for empty IDs", len(outcomes))
	}
}

func TestGetInterventionsByAgent(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-c1", UserID: "primary", Timestamp: "2026-01-01T10:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-c2", UserID: "primary", Timestamp: "2026-01-01T11:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "approved_by_policy", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-o1", UserID: "primary", Timestamp: "2026-01-01T12:00:00Z",
		AgentID: "other", InterventionType: "alert", Status: "delivered", Payload: "{}",
	})

	interventions, err := db.GetInterventionsByAgent("primary", "coach", 100)
	if err != nil {
		t.Fatalf("GetInterventionsByAgent failed: %v", err)
	}
	if len(interventions) != 2 {
		t.Fatalf("interventions count = %d, want 2", len(interventions))
	}
	for _, i := range interventions {
		if i.AgentID != "coach" {
			t.Errorf("expected agent_id=coach, got %s", i.AgentID)
		}
	}
}

func TestGetInterventionsByDateRange(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-d1", UserID: "primary", Timestamp: "2026-01-01T08:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-d2", UserID: "primary", Timestamp: "2026-01-01T12:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})
	db.InsertIntervention(&models.Intervention{
		InterventionID: "int-d3", UserID: "primary", Timestamp: "2026-01-02T12:00:00Z",
		AgentID: "coach", InterventionType: "nudge", Status: "delivered", Payload: "{}",
	})

	// Query only Jan 1
	interventions, err := db.GetInterventionsByDateRange("primary", "2026-01-01T00:00:00Z", "2026-01-01T23:59:59Z", 100)
	if err != nil {
		t.Fatalf("GetInterventionsByDateRange failed: %v", err)
	}
	if len(interventions) != 2 {
		t.Fatalf("interventions count = %d, want 2", len(interventions))
	}
}
