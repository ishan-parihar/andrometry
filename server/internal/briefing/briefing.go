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
	// Extract nested context if present
	contextMap := m
	if ctx, ok := m["context"]; ok {
		if ctxMap, ok := ctx.(map[string]interface{}); ok {
			contextMap = ctxMap
		}
	}

	// Compute probabilities from signals if not provided
	distractionProb := getFloat64(contextMap, "distraction_probability")
	fatigueProb := getFloat64(contextMap, "fatigue_probability")
	alignmentProb := getFloat64(contextMap, "alignment_probability")
	readinessScore := getFloat64(contextMap, "readiness_score")

	if distractionProb == 0 {
		distractionProb = computeDistractionProb(contextMap)
	}
	if fatigueProb == 0 {
		fatigueProb = computeFatigueProb(contextMap)
	}
	if alignmentProb == 0 {
		alignmentProb = computeAlignmentProb(contextMap)
	}
	if readinessScore == 0 {
		readinessScore = computeReadinessScore(fatigueProb, contextMap)
	}

	b := &Briefing{
		BriefingID:  briefingID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		State: BriefingState{
			PrimaryActivity:         getString(contextMap, "primary_activity", "unknown"),
			Place:                   getString(contextMap, "place", "unknown"),
			FocusActive:             getBool(contextMap, "focus_active"),
			DistractionProbability:  distractionProb,
			FatigueProbability:      fatigueProb,
			AlignmentProbability:    alignmentProb,
			ReadinessScore:          readinessScore,
		},
		Risks:            detectRisks(contextMap),
		Opportunities:    detectOpportunities(contextMap),
		SuggestedQueries: []string{"get_current_context", "get_daily_summary"},
	}
	return b
}

// computeDistractionProb estimates distraction from app switches, notifications, and category.
func computeDistractionProb(ctx map[string]interface{}) float64 {
	prob := 0.1
	appCategory := getString(ctx, "foreground_app_category", "unknown")
	if appCategory == "social" {
		prob += 0.3
	} else if appCategory == "entertainment" {
		prob += 0.2
	}
	// Notifications increase distraction
	notifRate := getString(ctx, "interruption_rate", "none")
	switch notifRate {
	case "high":
		prob += 0.2
	case "medium":
		prob += 0.1
	}
	// Focus active reduces distraction
	if getBool(ctx, "focus_active") {
		prob *= 0.7
	}
	return clamp01(prob)
}

// computeFatigueProb estimates fatigue from time of day and sleep info.
func computeFatigueProb(ctx map[string]interface{}) float64 {
	tz := getString(ctx, "timezone", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	hour := time.Now().In(loc).Hour()
	// Base fatigue from circadian rhythm
	var baseFatigue float64
	switch {
	case hour >= 0 && hour <= 5:
		baseFatigue = 0.7
	case hour >= 6 && hour <= 7:
		baseFatigue = 0.4
	case hour >= 8 && hour <= 11:
		baseFatigue = 0.2
	case hour >= 12 && hour <= 13:
		baseFatigue = 0.35 // post-lunch dip
	case hour >= 14 && hour <= 17:
		baseFatigue = 0.25
	case hour >= 18 && hour <= 20:
		baseFatigue = 0.35
	case hour >= 21 && hour <= 22:
		baseFatigue = 0.5
	default:
		baseFatigue = 0.65
	}
	return clamp01(baseFatigue)
}

// computeAlignmentProb estimates alignment from focus and app category.
func computeAlignmentProb(ctx map[string]interface{}) float64 {
	alignment := 0.5
	if getBool(ctx, "focus_active") {
		alignment += 0.3
	}
	appCategory := getString(ctx, "foreground_app_category", "unknown")
	switch appCategory {
	case "productivity", "education", "email", "calendar":
		alignment += 0.15
	case "social", "entertainment":
		alignment -= 0.2
	}
	return clamp01(alignment)
}

// computeReadinessScore estimates readiness from fatigue and other signals.
func computeReadinessScore(fatigueProb float64, ctx map[string]interface{}) float64 {
	fatigueComponent := (1.0 - fatigueProb) * 0.5
	readiness := 0.3 + fatigueComponent // base 0.3 + fatigue contribution
	return clamp01(readiness)
}

// clamp01 clamps a float64 to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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
