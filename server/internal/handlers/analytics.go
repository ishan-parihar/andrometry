package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/andrometry/server/internal/audit"
)

// GetInterventionEffectiveness handles GET /v1/interventions/{id}/effectiveness
// §35 Outcome Attribution Engine — computes weighted effectiveness score from outcomes and feedback.
// Scoring formula: 0.25 * accepted + 0.35 * behavior_change + 0.25 * metric_improvement + 0.15 * user_helpfulness
func (h *Handler) GetInterventionEffectiveness(w http.ResponseWriter, r *http.Request) {
	interventionID := r.PathValue("id")
	if interventionID == "" {
		respondError(w, http.StatusBadRequest, "missing intervention ID")
		return
	}

	uid := userID(r)

	// Fetch outcomes for this intervention
	outcomes, err := h.DB.GetOutcomes(uid, interventionID, 100)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query outcomes")
		return
	}

	// Fetch feedback targeting this intervention
	feedbacks, err := h.DB.GetFeedback(uid, "intervention", interventionID, 100)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query feedback")
		return
	}

	// Compute effectiveness score per §35 formula
	accepted := 0.0
	behaviorChange := 0.0
	metricImprovement := 0.0
	userHelpfulness := 0.0
	totalOutcomes := len(outcomes)
	totalFeedback := len(feedbacks)

	for _, o := range outcomes {
		switch o.OutcomeType {
		case "accepted", "completed":
			accepted = 1.0
		case "behavior_change":
			behaviorChange = 1.0
		case "metric_improved":
			metricImprovement = 1.0
		}
		if o.EffectivenessScore != nil {
			score := *o.EffectivenessScore
			behaviorChange = math.Max(behaviorChange, score)
			metricImprovement = math.Max(metricImprovement, score)
		}
	}

	// Aggregate feedback helpfulness
	helpfulCount := 0
	for _, f := range feedbacks {
		if f.FeedbackType == "helpful" || f.FeedbackType == "accepted" {
			helpfulCount++
		}
	}
	if totalFeedback > 0 {
		userHelpfulness = float64(helpfulCount) / float64(totalFeedback)
	}

	// Weighted score per §35
	effectivenessScore := 0.25*accepted + 0.35*behaviorChange + 0.25*metricImprovement + 0.15*userHelpfulness

	// Determine overall rating
	rating := "no_effect"
	switch {
	case effectivenessScore >= 0.7:
		rating = "highly_effective"
	case effectivenessScore >= 0.4:
		rating = "moderately_effective"
	case effectivenessScore > 0:
		rating = "marginally_effective"
	}

	result := map[string]interface{}{
		"intervention_id":     interventionID,
		"effectiveness_score": math.Round(effectivenessScore*1000) / 1000,
		"rating":              rating,
		"components": map[string]interface{}{
			"accepted":          math.Round(accepted*1000) / 1000,
			"behavior_change":   math.Round(behaviorChange*1000) / 1000,
			"metric_improvement": math.Round(metricImprovement*1000) / 1000,
			"user_helpfulness":  math.Round(userHelpfulness*1000) / 1000,
		},
		"total_outcomes":  totalOutcomes,
		"total_feedback":  totalFeedback,
	}

	audit.Log(h.DB, uid, "system", "", "compute_effectiveness", "intervention", interventionID, "")
	respondJSON(w, http.StatusOK, result)
}

// GetAnalyticsInsights handles GET /v1/analytics/insights
// §20 Insight and Analytics Layer — generates cross-domain insights from stored data.
func (h *Handler) GetAnalyticsInsights(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	period := queryParam(r, "period", "7d") // 1d, 7d, 30d

	// Determine time range
	var from string
	now := time.Now().UTC()
	switch period {
	case "1d":
		from = now.Add(-24 * time.Hour).Format(time.RFC3339)
	case "30d":
		from = now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	default: // 7d
		from = now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	}
	to := now.Format(time.RFC3339)

	// Collect data
	segments, _ := h.DB.GetTimeline(uid, from, to, 5000, "")
	events, _ := h.DB.GetEvents(uid, from, to, "", 5000)
	goals, _ := h.DB.GetGoals(uid)
	habits, _ := h.DB.GetHabits(uid)
	baselines, _ := h.DB.GetBaselines(uid)
	interventions, _ := h.DB.GetInterventionsByDateRange(uid, from, to, 500)

	// Compute focus metrics
	totalFocusMinutes := 0.0
	totalMeetingMinutes := 0.0
	totalDistractionMinutes := 0.0
	focusSegments := 0
	segmentTypes := map[string]int{}

	for _, s := range segments {
		segmentTypes[s.SegmentType]++
		start, _ := time.Parse(time.RFC3339, s.StartTime)
		end, _ := time.Parse(time.RFC3339, s.EndTime)
		duration := end.Sub(start).Minutes()
		if duration < 0 {
			duration = 0
		}
		switch s.SegmentType {
		case "focus_segment":
			totalFocusMinutes += duration
			focusSegments++
		case "meeting_segment":
			totalMeetingMinutes += duration
		case "distraction_segment":
			totalDistractionMinutes += duration
		}
	}

	// Compute habit stats
	activeHabits := 0
	completedHabits := 0
	avgStreak := 0.0
	for _, hab := range habits {
		if hab.Status == "active" {
			activeHabits++
		}
		if hab.Status == "completed" {
			completedHabits++
		}
		avgStreak += float64(hab.StreakDays)
	}
	if len(habits) > 0 {
		avgStreak /= float64(len(habits))
	}

	// Compute goal stats
	activeGoals := 0
	completedGoals := 0
	for _, g := range goals {
		if g.Status == "active" {
			activeGoals++
		}
		if g.Status == "completed" {
			completedGoals++
		}
	}

	// Compute intervention effectiveness
	totalInterventions := len(interventions)
	acceptedInterventions := 0
	for _, i := range interventions {
		if i.Status == "accepted" || i.Status == "completed" {
			acceptedInterventions++
		}
	}
	effectivenessRate := 0.0
	if totalInterventions > 0 {
		effectivenessRate = float64(acceptedInterventions) / float64(totalInterventions)
	}

	// Build baseline deviations
	baselineDeviations := map[string]interface{}{}
	for _, b := range baselines {
		switch b.MetricName {
		case "deep_work_minutes_daily":
			dailyFocus := totalFocusMinutes / math.Max(1, float64(7)) // rough daily average
			deviation := (dailyFocus - b.Value) / math.Max(1, b.Value)
			baselineDeviations[b.MetricName] = map[string]interface{}{
				"baseline":  b.Value,
				"current":   math.Round(dailyFocus*10) / 10,
				"deviation": math.Round(deviation*1000) / 1000,
			}
		}
	}

	insights := map[string]interface{}{
		"period":     period,
		"date_range": map[string]string{"from": from, "to": to},
		"focus": map[string]interface{}{
			"total_focus_minutes":    math.Round(totalFocusMinutes),
			"total_meeting_minutes":  math.Round(totalMeetingMinutes),
			"total_distraction_minutes": math.Round(totalDistractionMinutes),
			"focus_session_count":    focusSegments,
		},
		"segment_distribution": segmentTypes,
		"habits": map[string]interface{}{
			"active":        activeHabits,
			"completed":     completedHabits,
			"total":         len(habits),
			"avg_streak":    math.Round(avgStreak*10) / 10,
		},
		"goals": map[string]interface{}{
			"active":    activeGoals,
			"completed": completedGoals,
			"total":     len(goals),
		},
		"interventions": map[string]interface{}{
			"total":            totalInterventions,
			"accepted":         acceptedInterventions,
			"effectiveness_rate": math.Round(effectivenessRate*1000) / 1000,
		},
		"baseline_deviations": baselineDeviations,
		"event_count":         len(events),
		"segment_count":       len(segments),
	}

	audit.Log(h.DB, uid, "system", "", "analytics_insights", "analytics", period, "")
	respondJSON(w, http.StatusOK, insights)
}
