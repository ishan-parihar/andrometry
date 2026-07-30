package handlers

import (
	"net/http"

	"github.com/andrometry/server/internal/models"
)

// GetAgentEffectiveness handles GET /v1/agents/:agent_id/effectiveness — computes agent effectiveness score from outcomes and feedback.
// §35 Outcome Attribution Engine — agents need to evaluate their own performance.
func (h *Handler) GetAgentEffectiveness(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		respondError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	// Get all interventions for this agent
	interventions, err := h.DB.GetInterventionsByAgent(uid, agentID, 100)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query interventions")
		return
	}

	// Build set of this agent's intervention IDs for batch outcomes query
	interventionIDs := make([]string, 0, len(interventions))
	for _, i := range interventions {
		interventionIDs = append(interventionIDs, i.InterventionID)
	}

	// Batch query outcomes for this agent's interventions (avoids N+1)
	var outcomes []models.Outcome
	if len(interventionIDs) > 0 {
		outcomes, err = h.DB.GetOutcomesByInterventionIDs(uid, interventionIDs, 200)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to query outcomes")
			return
		}
	}

	// Count intervention statuses
	totalInterventions := len(interventions)
	approvedCount := 0
	deliveredCount := 0
	for _, i := range interventions {
		switch i.Status {
		case "approved_by_policy":
			approvedCount++
		case "delivered", "accepted", "completed":
			deliveredCount++
		}
	}

	// Compute effectiveness from agent-filtered outcomes
	totalOutcomes := 0
	acceptedCount := 0
	behaviorChangeCount := 0
	totalEffectiveness := 0.0
	effectivenessSamples := 0

	for _, o := range outcomes {
		totalOutcomes++
		switch o.OutcomeType {
		case "accepted", "completed":
			acceptedCount++
		case "behavior_change", "metric_improved":
			behaviorChangeCount++
		}
		if o.EffectivenessScore != nil {
			totalEffectiveness += *o.EffectivenessScore
			effectivenessSamples++
		}
	}

	avgEffectiveness := 0.0
	if effectivenessSamples > 0 {
		avgEffectiveness = totalEffectiveness / float64(effectivenessSamples)
	}

	deliveryRate := 0.0
	if totalInterventions > 0 {
		deliveryRate = float64(deliveredCount) / float64(totalInterventions)
	}

	acceptanceRate := 0.0
	if totalOutcomes > 0 {
		acceptanceRate = float64(acceptedCount) / float64(totalOutcomes)
	}

	// effectiveness_score = 0.30 * delivery_rate + 0.35 * acceptance_rate + 0.35 * avg_effectiveness
	agentEffectiveness := 0.30*deliveryRate + 0.35*acceptanceRate + 0.35*avgEffectiveness

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agent_id":              agentID,
		"effectiveness_score":   agentEffectiveness,
		"total_interventions":   totalInterventions,
		"approved_count":        approvedCount,
		"delivered_count":       deliveredCount,
		"delivery_rate":         deliveryRate,
		"total_outcomes":        totalOutcomes,
		"accepted_count":        acceptedCount,
		"behavior_change_count": behaviorChangeCount,
		"acceptance_rate":       acceptanceRate,
		"avg_effectiveness":     avgEffectiveness,
	})
}
