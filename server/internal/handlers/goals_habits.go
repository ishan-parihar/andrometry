package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/andrometry/server/internal/audit"
	"github.com/andrometry/server/internal/models"
)

// PostGoalV2 handles POST /v1/goals — creates a goal.
// Accepts both:
// 1. {"goal_type":"x", "title":"y", "status":"z", "target":{...}}
// 2. {"goal_type":"x", "status":"z", "target":"y"} (title from target string)
func (h *Handler) PostGoalV2(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	goalType, _ := raw["goal_type"].(string)
	status, _ := raw["status"].(string)
	if goalType == "" {
		goalType = "general"
	}
	if status == "" {
		status = "active"
	}

	// Build payload from available fields
	payload := make(map[string]interface{})
	for k, v := range raw {
		if k == "goal_id" || k == "goal_type" || k == "status" {
			continue
		}
		payload[k] = v
	}

	// Ensure title exists (from title field or target field)
	title, _ := raw["title"].(string)
	if title == "" {
		if target, ok := raw["target"]; ok {
			if targetStr, ok := target.(string); ok {
				title = targetStr
			} else {
				payload["target"] = target
			}
		}
	}
	if title != "" {
		payload["title"] = title
	}

	uid := userID(r)
	goalID, _ := raw["goal_id"].(string)
	if goalID == "" {
		goalID = generateID()
	}

	payloadBytes, _ := json.Marshal(payload)

	err := h.DB.InsertGoal(&models.Goal{
		GoalID:   goalID,
		UserID:   uid,
		GoalType: goalType,
		Status:   status,
		Payload:  string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert goal")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_goal", "goal", goalID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"goal_id": goalID, "status": "created"})
}

// PostHabitV2 handles POST /v1/habits — creates a habit.
// Accepts both:
// 1. {"habit_type":"x", "title":"y", "target":{...}, "schedule":{...}, "evidence_sources":[...]}
// 2. {"habit_type":"x", "target":"y", "schedule":"z", "evidence_sources":[...]} (flat strings)
func (h *Handler) PostHabitV2(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	habitType, _ := raw["habit_type"].(string)
	if habitType == "" {
		respondError(w, http.StatusBadRequest, "habit_type is required")
		return
	}

	uid := userID(r)
	habitID, _ := raw["habit_id"].(string)
	if habitID == "" {
		habitID = generateID()
	}

	// Normalize target
	target := raw["target"]
	if target == nil {
		target = ""
	}
	targetBytes, _ := json.Marshal(target)

	// Normalize schedule
	schedule := raw["schedule"]
	if schedule == nil {
		schedule = ""
	}
	scheduleBytes, _ := json.Marshal(schedule)

	// Normalize evidence_sources
	evidence := raw["evidence_sources"]
	if evidence == nil {
		evidence = []string{}
	}
	evidenceBytes, _ := json.Marshal(evidence)

	err := h.DB.InsertHabit(&models.Habit{
		HabitID:         habitID,
		UserID:          uid,
		HabitType:       habitType,
		Target:          string(targetBytes),
		Schedule:        string(scheduleBytes),
		EvidenceSources: string(evidenceBytes),
		Status:          "active",
		StreakDays:      0,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert habit")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_habit", "habit", habitID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"habit_id": habitID, "status": "created"})
}
