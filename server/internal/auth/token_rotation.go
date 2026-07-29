package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

// TokenRotation handles automatic token rotation for security.
type TokenRotation struct {
	DB            *db.DB
	RotationDays  int // How often to rotate (default: 30 days)
}

// NewTokenRotation creates a new token rotation manager.
func NewTokenRotation(database *db.DB, rotationDays int) *TokenRotation {
	if rotationDays <= 0 {
		rotationDays = 30
	}
	return &TokenRotation{
		DB:           database,
		RotationDays: rotationDays,
	}
}

// RotateToken generates a new token, revokes all old tokens for the user+type, and returns the new raw token.
// This is a two-phase operation: create new first, then revoke old, so the user always has at least one valid token.
func (tr *TokenRotation) RotateToken(userID, tokenType string) (string, error) {
	// Phase 1: Generate new raw token
	rawToken, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	hash := db.HashToken(rawToken)

	// Create new token
	tokenID := uuid.New().String()
	token := &models.AuthToken{
		TokenID:   tokenID,
		UserID:    userID,
		TokenHash: hash,
		TokenType: tokenType,
		Scopes:    getDefaultScopes(tokenType),
	}

	if err := tr.DB.CreateToken(token); err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	// Phase 2: Revoke old tokens of the same type for this user
	// The new token is already persisted, so revoking old ones is safe.
	revoked, err := tr.DB.RevokeTokensByType(userID, tokenType, tokenID)
	if err != nil {
		log.Printf("[token-rotation] warning: failed to revoke old tokens: %v", err)
		// Don't fail the rotation — new token is valid
	} else if revoked > 0 {
		log.Printf("[token-rotation] rotated %s token for user %s: %s... (%d old tokens revoked)", tokenType, userID, rawToken[:8], revoked)
	} else {
		log.Printf("[token-rotation] rotated %s token for user %s: %s... (no old tokens to revoke)", tokenType, userID, rawToken[:8])
	}

	return rawToken, nil
}

// RevokeAllTokens revokes all tokens for a user (kill switch).
func (tr *TokenRotation) RevokeAllTokens(userID string) error {
	log.Printf("[token-rotation] revoking all tokens for user %s", userID)
	if err := tr.DB.RevokeAllTokensForUser(userID); err != nil {
		return fmt.Errorf("revoke all tokens: %w", err)
	}
	return nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func getDefaultScopes(tokenType string) string {
	if tokenType == "device" {
		return "device:ingest,device:read_config,device:read_interventions,device:write_feedback"
	}
	return "agent:context:read,agent:briefing:read,agent:timeline:read,agent:events:read,agent:summaries:read,agent:observations:write,agent:interventions:write,agent:feedback:read,agent:goals:read,agent:goals:write,agent:habits:read,agent:habits:write,agent:outcomes:write"
}
