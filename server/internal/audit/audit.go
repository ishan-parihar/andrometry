package audit

import (
	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

// Log writes an audit entry.
func Log(database *db.DB, userID, actorType, actorID, action, targetType, targetID, metadata string) {
	entry := &models.AuditLog{
		AuditID:    uuid.New().String(),
		UserID:     userID,
		ActorType:  actorType,
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   metadata,
	}
	// Best effort — don't fail the request if audit write fails
	_ = database.InsertAuditLog(entry)
}
