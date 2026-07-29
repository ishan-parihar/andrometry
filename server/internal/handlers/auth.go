package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/andrometry/server/internal/audit"
)

// PostTokenRotation handles POST /v1/auth/rotate — rotates the caller's token and returns a new one.
// §28 Token Rotation — generates a new token and invalidates the old one for security.
func (h *Handler) PostTokenRotation(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req struct {
		TokenType string `json:"token_type"` // "device" or "agent"
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.TokenType == "" {
		respondError(w, http.StatusBadRequest, "token_type is required")
		return
	}
	if req.TokenType != "device" && req.TokenType != "agent" {
		respondError(w, http.StatusBadRequest, "token_type must be 'device' or 'agent'")
		return
	}

	uid := userID(r)

	newToken, err := h.Rotator.RotateToken(uid, req.TokenType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to rotate token")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "rotate_token", "token", "",
		`{"token_type":"`+req.TokenType+`"}`)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"new_token":   newToken,
		"token_type":  req.TokenType,
		"old_token":   "revoked",
		"message":     "Store the new token securely. The old token has been revoked.",
	})
}

// PostTokenRevoke handles POST /v1/auth/revoke — revokes all tokens for the caller (kill switch).
// §28 Token Rotation — emergency token revocation.
func (h *Handler) PostTokenRevoke(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	if err := h.Rotator.RevokeAllTokens(uid); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to revoke tokens")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "revoke_all_tokens", "user", uid, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "all_tokens_revoked",
		"user_id": uid,
		"message": "All tokens for this user have been revoked. Re-authenticate with new credentials.",
	})
}
