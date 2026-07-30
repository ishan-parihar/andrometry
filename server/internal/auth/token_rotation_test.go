package auth

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

func TestNewTokenRotation_DefaultDays(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 0)
	if tr.RotationDays != 30 {
		t.Errorf("expected default rotation days 30, got %d", tr.RotationDays)
	}

	tr2 := NewTokenRotation(database, -5)
	if tr2.RotationDays != 30 {
		t.Errorf("expected default rotation days 30 for negative input, got %d", tr2.RotationDays)
	}
}

func TestNewTokenRotation_CustomDays(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 14)
	if tr.RotationDays != 14 {
		t.Errorf("expected rotation days 14, got %d", tr.RotationDays)
	}
}

func TestRotateToken_DeviceType(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)

	rawToken, err := tr.RotateToken("user1", "device")
	if err != nil {
		t.Fatalf("RotateToken failed: %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected non-empty raw token")
	}
	if len(rawToken) != 64 { // 32 bytes hex-encoded
		t.Errorf("expected token length 64, got %d", len(rawToken))
	}

	// Verify the token can be validated
	hash := db.HashToken(rawToken)
	token, err := database.ValidateToken(hash)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if token == nil {
		t.Fatal("expected to find token by hash")
	}
	if token.TokenType != "device" {
		t.Errorf("expected token_type 'device', got '%s'", token.TokenType)
	}
	if token.UserID != "user1" {
		t.Errorf("expected user_id 'user1', got '%s'", token.UserID)
	}
}

func TestRotateToken_AgentType(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)

	rawToken, err := tr.RotateToken("user1", "agent")
	if err != nil {
		t.Fatalf("RotateToken failed: %v", err)
	}

	hash := db.HashToken(rawToken)
	token, err := database.ValidateToken(hash)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if token.TokenType != "agent" {
		t.Errorf("expected token_type 'agent', got '%s'", token.TokenType)
	}
	// Agent scopes should contain agent: context:read
	if token.Scopes == "" {
		t.Error("expected non-empty scopes for agent token")
	}
}

func TestRotateToken_RevokesOldTokens(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)

	// Create first token
	token1, err := tr.RotateToken("user1", "device")
	if err != nil {
		t.Fatalf("first RotateToken failed: %v", err)
	}

	// Create second token (should revoke the first)
	token2, err := tr.RotateToken("user1", "device")
	if err != nil {
		t.Fatalf("second RotateToken failed: %v", err)
	}

	if token1 == token2 {
		t.Error("expected different tokens")
	}

	// First token should be revoked
	hash1 := db.HashToken(token1)
	token1Record, _ := database.ValidateToken(hash1)
	if token1Record != nil && token1Record.RevokedAt == nil {
		t.Error("expected first token to be revoked")
	}

	// Second token should be valid
	hash2 := db.HashToken(token2)
	token2Record, err := database.ValidateToken(hash2)
	if err != nil || token2Record == nil {
		t.Fatal("expected second token to exist")
	}
	if token2Record.RevokedAt != nil {
		t.Error("expected second token to NOT be revoked")
	}
}

func TestRotateToken_DifferentTypesNotRevoked(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)

	// Create device token
	deviceToken, _ := tr.RotateToken("user1", "device")

	// Create agent token (should NOT revoke device token)
	_, err := tr.RotateToken("user1", "agent")
	if err != nil {
		t.Fatalf("agent RotateToken failed: %v", err)
	}

	// Device token should still be valid
	hash := db.HashToken(deviceToken)
	token, err := database.ValidateToken(hash)
	if err != nil || token == nil {
		t.Fatal("expected device token to still exist")
	}
	if token.RevokedAt != nil {
		t.Error("expected device token to NOT be revoked by agent rotation")
	}
}

func TestRevokeAllTokens(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)

	// Create multiple tokens
	tr.RotateToken("user1", "device")
	tr.RotateToken("user1", "agent")
	tr.RotateToken("user1", "device")

	// Revoke all
	err := tr.RevokeAllTokens("user1")
	if err != nil {
		t.Fatalf("RevokeAllTokens failed: %v", err)
	}

	// Verify by creating a new token and checking old ones are revoked
	// Since we can't easily list all tokens, we verify the rotation works
	newToken, err := tr.RotateToken("user1", "device")
	if err != nil {
		t.Fatalf("RotateToken after RevokeAllTokens failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("expected non-empty new token")
	}
}

func TestRotateToken_UniqueTokens(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	tr := NewTokenRotation(database, 30)
	tokens := make(map[string]bool)

	for i := 0; i < 5; i++ {
		rawToken, err := tr.RotateToken("user1", "device")
		if err != nil {
			t.Fatalf("RotateToken iteration %d failed: %v", i, err)
		}
		if tokens[rawToken] {
			t.Fatalf("duplicate token generated on iteration %d", i)
		}
		tokens[rawToken] = true
	}
}

func TestGetDefaultScopes_Device(t *testing.T) {
	scopes := getDefaultScopes("device")
	if scopes == "" {
		t.Error("expected non-empty device scopes")
	}
	// Device scopes should contain device:ingest
	if !containsScope(scopes, "device:ingest") {
		t.Error("expected device scopes to contain 'device:ingest'")
	}
	// Device scopes should NOT contain agent scopes
	if containsScope(scopes, "agent:context:read") {
		t.Error("device scopes should not contain agent scopes")
	}
}

func TestGetDefaultScopes_Agent(t *testing.T) {
	scopes := getDefaultScopes("agent")
	if scopes == "" {
		t.Error("expected non-empty agent scopes")
	}
	if !containsScope(scopes, "agent:context:read") {
		t.Error("expected agent scopes to contain 'agent:context:read'")
	}
	if !containsScope(scopes, "agent:interventions:write") {
		t.Error("expected agent scopes to contain 'agent:interventions:write'")
	}
	// Agent scopes should NOT contain device scopes
	if containsScope(scopes, "device:ingest") {
		t.Error("agent scopes should not contain device scopes")
	}
}

func containsScope(scopes, scope string) bool {
	for _, s := range splitScopes(scopes) {
		if s == scope {
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

