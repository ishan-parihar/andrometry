package briefing

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/andrometry/server/internal/models"
)

// Briefing represents a high-level agent briefing.
type Briefing struct {
	BriefingID     string               `json:"briefing_id"`
	GeneratedAt    string               `json:"generated_at"`
	State          BriefingState        `json:"state"`
	DeltaSinceLast *Delta               `json:"delta_since_last,omitempty"`
	Risks          []Risk               `json:"risks"`
	Opportunities  []Opportunity        `json:"opportunities"`
	SuggestedQueries []string           `json:"suggested_queries"`
}

type BriefingState struct {
	PrimaryActivity         string  `json:"primary_activity"`
	Place                   string  `json:"place"`
	FocusActive             bool    `json:"focus_active"`
	DistractionProbability  float64 `json:"distraction_probability"`
	FatigueProbability      float64 `json:"fatigue_probability"`
	AlignmentProbability    float64 `json:"alignment_probability"`
	ReadinessScore          float64 `json:"readiness_score"`
}

type Delta struct {
	DistractionProbabilityChange float64 `json:"distraction_probability_change"`
	FocusMinutesElapsed          int     `json:"focus_minutes_elapsed"`
}

type Risk struct {
	Type       string  `json:"type"`
	Severity   string  `json:"severity"`
	Confidence float64 `json:"confidence"`
	Message    string  `json:"message"`
}

type Opportunity struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Message    string  `json:"message"`
}

// Generate creates a briefing from the current context.
func Generate(contextJSON string, briefingID string) (*Briefing, error) {
	var ctx models.CurrentContext
	if err := json.Unmarshal([]byte(contextJSON), &ctx); err != nil {
		// If context is just the inner context object, try parsing it differently
		var inner map[string]interface{}
		if err2 := json.Unmarshal([]byte(contextJSON), &inner); err2 != nil {
			return nil, fmt.Errorf("parse context: %w", err)
		}
		// Build briefing from raw map
		return generateFromMap(inner, briefingID), nil
	}

	b := &Briefing{
		BriefingID:  briefingID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Risks:       []Risk{},
		Opportunities: []Opportunity{},
		SuggestedQueries: []string{},
	}

	// Extract context fields - handle both nested and flat structures
	contextMap := extractContextMap(contextJSON)

	b.State = BriefingState{
		PrimaryActivity:         getString(contextMap, "primary_activity", "unknown"),
		Place:                   getString(contextMap, "place", "unknown"),
		FocusActive:             getBool(contextMap, "focus_active"),
		DistractionProbability:  getFloat64(contextMap, "distraction_probability"),
		FatigueProbability:      getFloat64(contextMap, "fatigue_probability"),
		AlignmentProbability:    getFloat64(contextMap, "alignment_probability"),
		ReadinessScore:          getFloat64(contextMap, "readiness_score"),
	}

	// Detect risks
	b.Risks = detectRisks(contextMap)

	// Detect opportunities
	b.Opportunities = detectOpportunities(contextMap)

	// Generate suggested queries
	b.SuggestedQueries = generateSuggestedQueries(b.State)

	return b, nil
}

func extractContextMap(contextJSON string) map[string]interface{} {
	var result map[string]interface{}

	// Try parsing as full context object first
	if err := json.Unmarshal([]byte(contextJSON), &result); err != nil {
		return map[string]interface{}{}
	}

	// If there's a "context" key, use that
	if ctx, ok := result["context"]; ok {
		if ctxMap, ok := ctx.(map[string]interface{}); ok {
			return ctxMap
		}
	}

	return result
}

func detectRisks(context map[string]interface{}) []Risk {
	var risks []Risk

	distraction := getFloat64(context, "distraction_probability")
	if distraction > 0.4 {
		severity := "low"
		if distraction > 0.6 {
			severity = "medium"
		}
		if distraction > 0.8 {
			severity = "high"
		}
		risks = append(risks, Risk{
			Type:       "focus_drift",
			Severity:   severity,
			Confidence: distraction,
			Message:    fmt.Sprintf("Distraction probability is %.0f%%", distraction*100),
		})
	}

	fatigue := getFloat64(context, "fatigue_probability")
	if fatigue > 0.5 {
		severity := "low"
		if fatigue > 0.7 {
			severity = "medium"
		}
		if fatigue > 0.85 {
			severity = "high"
		}
		risks = append(risks, Risk{
			Type:       "fatigue",
			Severity:   severity,
			Confidence: fatigue,
			Message:    fmt.Sprintf("Fatigue probability is %.0f%%", fatigue*100),
		})
	}

	if interruptionRate, ok := context["interruption_rate"].(string); ok && interruptionRate == "high" {
		risks = append(risks, Risk{
			Type:       "high_interruption",
			Severity:   "medium",
			Confidence: 0.8,
			Message:    "High interruption rate detected",
		})
	}

	return risks
}

func detectOpportunities(context map[string]interface{}) []Opportunity {
	var opps []Opportunity

	if getBool(context, "focus_active") {
		opps = append(opps, Opportunity{
			Type:       "extend_focus_block",
			Confidence: 0.7,
			Message:    "Focus session is active — consider extending",
		})
	}

	alignment := getFloat64(context, "alignment_probability")
	if alignment > 0.7 {
		opps = append(opps, Opportunity{
			Type:       "maintain_momentum",
			Confidence: alignment,
			Message:    "High alignment — good time for challenging work",
		})
	}

	if getFloat64(context, "fatigue_probability") < 0.2 && !getBool(context, "focus_active") {
		opps = append(opps, Opportunity{
			Type:       "start_focus",
			Confidence: 0.6,
			Message:    "Low fatigue, no active focus — good time to start a focus session",
		})
	}

	return opps
}

func generateSuggestedQueries(state BriefingState) []string {
	queries := []string{"get_current_context"}

	if state.FocusActive {
		queries = append(queries, "get_focus_sessions_today")
	}

	if state.DistractionProbability > 0.3 {
		queries = append(queries, "get_distraction_segments", "get_timeline_last_2h")
	}

	if state.FatigueProbability > 0.5 {
		queries = append(queries, "get_sleep_summary", "get_daily_summary")
	}

	queries = append(queries, "get_daily_summary", "get_habit_metrics")
	return queries
}

func generateFromMap(m map[string]interface{}, briefingID string) *Briefing {
	b := &Briefing{
		BriefingID:  briefingID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		State: BriefingState{
			PrimaryActivity:         getString(m, "primary_activity", "unknown"),
			Place:                   getString(m, "place", "unknown"),
			FocusActive:             getBool(m, "focus_active"),
			DistractionProbability:  getFloat64(m, "distraction_probability"),
			FatigueProbability:      getFloat64(m, "fatigue_probability"),
			AlignmentProbability:    getFloat64(m, "alignment_probability"),
			ReadinessScore:          getFloat64(m, "readiness_score"),
		},
		Risks:            detectRisks(m),
		Opportunities:    detectOpportunities(m),
		SuggestedQueries: []string{"get_current_context", "get_daily_summary"},
	}
	return b
}

func getString(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case json.Number:
			f, _ := val.Float64()
			return f
		}
	}
	return 0
}

// BriefingToJSON serializes a briefing to a pretty JSON string.
func BriefingToJSON(b *Briefing) string {
	// Clean up empty arrays
	if b.Risks == nil {
		b.Risks = []Risk{}
	}
	if b.Opportunities == nil {
		b.Opportunities = []Opportunity{}
	}
	if b.SuggestedQueries == nil {
		b.SuggestedQueries = []string{}
	}
	// Remove trailing empty string from suggested queries
	for i := len(b.SuggestedQueries) - 1; i >= 0; i-- {
		if strings.TrimSpace(b.SuggestedQueries[i]) == "" {
			b.SuggestedQueries = append(b.SuggestedQueries[:i], b.SuggestedQueries[i+1:]...)
		}
	}

	data, _ := json.MarshalIndent(b, "", "  ")
	return string(data)
}
