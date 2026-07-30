package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/andrometry/server/internal/audit"
)

// GetPreferences handles GET /v1/preferences — retrieves user preferences (alignment weights, classification overrides).
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	category := queryParam(r, "category", "")

	prefs, err := h.DB.GetPreferences(uid, category)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query preferences")
		return
	}


	// Group by category for the response
	grouped := make(map[string]map[string]string)
	for _, p := range prefs {
		if grouped[p.Category] == nil {
			grouped[p.Category] = make(map[string]string)
		}
		grouped[p.Category][p.Key] = p.Value
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"preferences": grouped,
	})
}

// PutPreferences handles PUT /v1/preferences — upserts one or more preferences.
// Supports two formats:
// 1. Array format: {"preferences": [{"category":"x","key":"y","value":"z"}]}
// 2. Category+map format: {"category":"x", "preferences": {"key1":"val1", "key2":"val2"}}
func (h *Handler) PutPreferences(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Try array format first
	var reqArray struct {
		Preferences []struct {
			Category string `json:"category"`
			Key      string `json:"key"`
			Value    string `json:"value"`
		} `json:"preferences"`
	}
	if err := json.Unmarshal(body, &reqArray); err == nil && len(reqArray.Preferences) > 0 {
		updated := 0
		for _, p := range reqArray.Preferences {
			if p.Category == "" || p.Key == "" {
				continue
			}
			if err := h.DB.UpsertPreference(uid, p.Category, p.Key, p.Value); err != nil {
				continue
			}
			updated++
		}
		audit.Log(h.DB, uid, actorType(r), actorID(r), "update_preferences", "preferences", "",
			`{"updated":`+strconv.Itoa(updated)+`}`)
		respondJSON(w, http.StatusOK, map[string]interface{}{"status": "updated", "updated": updated})
		return
	}

	// Try category+map format
	var reqMap struct {
		Category    string                 `json:"category"`
		Preferences map[string]interface{} `json:"preferences"`
	}
	if err := json.Unmarshal(body, &reqMap); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if reqMap.Category == "" {
		respondError(w, http.StatusBadRequest, "category is required")
		return
	}

	updated := 0
	for key, val := range reqMap.Preferences {
		valueStr := fmt.Sprintf("%v", val)
		if err := h.DB.UpsertPreference(uid, reqMap.Category, key, valueStr); err != nil {
			continue
		}
		updated++
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "update_preferences", "preferences", reqMap.Category,
		`{"updated":`+strconv.Itoa(updated)+`}`)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "updated",
		"updated": updated,
	})
}

// DeletePreference handles DELETE /v1/preferences/{category}/{key} — deletes a single preference.
func (h *Handler) DeletePreference(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	category := r.PathValue("category")
	key := r.PathValue("key")

	if category == "" || key == "" {
		respondError(w, http.StatusBadRequest, "category and key are required")
		return
	}

	if err := h.DB.DeletePreference(uid, category, key); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete preference")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "delete_preference", "preferences", category+"/"+key, "")
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}


