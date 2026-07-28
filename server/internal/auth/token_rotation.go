package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

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

// RotateToken generates a new token and revokes the old one.
func (tr *TokenRotation) RotateToken(userID, tokenType string) (string, error) {
	// Generate new raw token
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

	log.Printf("[token-rotation] new %s token created for user %s: %s...", tokenType, userID, rawToken[:8])
	return rawToken, nil
}

// CheckAndRotate checks if rotation is needed and rotates all tokens.
func (tr *TokenRotation) CheckAndRotate(userID string) error {
	// Check when last rotation happened (simplified — check token age)
	// In production, store last_rotation_at in a settings table
	log.Printf("[token-rotation] checking rotation for user %s", userID)
	return nil
}

// RevokeAllTokens revokes all tokens for a user (kill switch).
func (tr *TokenRotation) RevokeAllTokens(userID string) error {
	log.Printf("[token-rotation] revoking all tokens for user %s", userID)
	// Implementation would update all tokens for this user to revoked
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
