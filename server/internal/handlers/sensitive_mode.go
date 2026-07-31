package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/andrometry/server/internal/audit"
	"github.com/andrometry/server/internal/governor"
)

// GetSensitiveMode handles GET /v1/privacy/sensitive-mode.
func (h *Handler) GetSensitiveMode(w http.ResponseWriter, r *http.Request) {
	settings, err := h.DB.GetSensitiveModeSettings()
	if err != nil {
		// No persisted settings — return defaults
		mode := governor.DefaultSensitiveMode()
		respondJSON(w, http.StatusOK, mode)
		return
	}
	mode := governor.UnmarshalSensitiveMode(settings)
	respondJSON(w, http.StatusOK, mode)
}

// SetSensitiveMode handles POST /v1/privacy/sensitive-mode.
func (h *Handler) SetSensitiveMode(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req struct {
		Active bool `json:"active"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)

	mode := governor.DefaultSensitiveMode()
	if req.Active {
		mode.Activate()
	} else {
		mode.Deactivate()
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "set_sensitive_mode", "privacy", "",
		`{"active":`+boolToString(req.Active)+`}`)

	// Persist to DB
	_ = h.DB.SaveSensitiveModeSettings(governor.MarshalSensitiveMode(mode))

	// Invalidate cache so next loadSensitiveMode() picks up the change
	h.invalidateSensitiveModeCache()

	respondJSON(w, http.StatusOK, mode)
}

// WipeData handles POST /v1/privacy/wipe — deletes all user data.
func (h *Handler) WipeData(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	if err := h.DB.WipeUserData(uid); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to wipe data")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "wipe_data", "user", uid, "")
	respondJSON(w, http.StatusOK, map[string]string{"status": "wiped", "user_id": uid})
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
