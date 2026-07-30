package audit

import (
	"testing"

	"github.com/andrometry/server/internal/db"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return database
}

func TestLog_CreatesAuditEntry(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	Log(database, "user1", "agent", "agent-1", "create_observation", "observation", "obs-123", `{"note":"test"}`)

	logs, err := database.GetAuditLog("user1", 10)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].ActorType != "agent" {
		t.Errorf("expected actor_type 'agent', got '%s'", logs[0].ActorType)
	}
	if logs[0].Action != "create_observation" {
		t.Errorf("expected action 'create_observation', got '%s'", logs[0].Action)
	}
	if logs[0].TargetType != "observation" {
		t.Errorf("expected target_type 'observation', got '%s'", logs[0].TargetType)
	}
	if logs[0].TargetID != "obs-123" {
		t.Errorf("expected target_id 'obs-123', got '%s'", logs[0].TargetID)
	}
}

func TestLog_MultipleEntries(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	Log(database, "user1", "agent", "agent-1", "create_observation", "observation", "obs-1", "")
	Log(database, "user1", "agent", "agent-1", "create_intervention", "intervention", "int-1", "")
	Log(database, "user1", "device", "device-1", "batch_ingest", "batch", "batch-1", "")

	logs, err := database.GetAuditLog("user1", 10)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 audit logs, got %d", len(logs))
	}
}

func TestLog_DifferentUsers(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	Log(database, "user1", "agent", "agent-1", "action1", "target", "t1", "")
	Log(database, "user2", "agent", "agent-2", "action2", "target", "t2", "")

	logs1, _ := database.GetAuditLog("user1", 10)
	logs2, _ := database.GetAuditLog("user2", 10)

	if len(logs1) != 1 {
		t.Errorf("expected 1 log for user1, got %d", len(logs1))
	}
	if len(logs2) != 1 {
		t.Errorf("expected 1 log for user2, got %d", len(logs2))
	}
}

func TestLog_EmptyMetadata(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	Log(database, "user1", "system", "", "cleanup", "", "", "")

	logs, err := database.GetAuditLog("user1", 10)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].ActorType != "system" {
		t.Errorf("expected actor_type 'system', got '%s'", logs[0].ActorType)
	}
}

func TestLog_BestEffort_DoesNotPanic(t *testing.T) {
	// Log should not panic even if database operations fail
	database := setupTestDB(t)
	database.Close()

	// This should not panic — best-effort logging
	Log(database, "user1", "agent", "agent-1", "test", "", "", "")
}
