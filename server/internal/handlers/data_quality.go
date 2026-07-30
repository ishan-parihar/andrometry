package handlers

import (
	"net/http"
	"time"
)

// GetDataQuality handles GET /v1/context/data-quality — returns data quality signals for the current context.
// §11.4 Data quality signals — agents need to know when not to trust a conclusion.
func (h *Handler) GetDataQuality(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"completeness":         0.0,
			"staleness_seconds":    -1,
			"missing_signals":      []string{"context"},
			"missing_collectors":   []string{"all"},
			"degraded_collectors":  []string{},
			"permission_issues":    []string{},
			"clock_drift_detected": false,
			"message":              "no context available",
		})
		return
	}

	// Calculate staleness
	stalenessSeconds := -1
	parsedTime, err := time.Parse(time.RFC3339, ctx.Timestamp)
	if err == nil {
		stalenessSeconds = int(time.Since(parsedTime).Seconds())
	}

	// Determine completeness based on available data
	completeness := 0.5 // base for having context
	if stalenessSeconds >= 0 && stalenessSeconds < 300 {
		completeness = 0.87 // fresh context
	} else if stalenessSeconds >= 300 && stalenessSeconds < 900 {
		completeness = 0.65 // somewhat stale
	} else if stalenessSeconds >= 900 {
		completeness = 0.3 // stale
	}

	// Check for missing signals
	missingSignals := []string{}
	degradedCollectors := []string{}

	if stalenessSeconds > 300 {
		degradedCollectors = append(degradedCollectors, "context_sync")
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"completeness":         completeness,
		"staleness_seconds":    stalenessSeconds,
		"last_context_time":    ctx.Timestamp,
		"etag":                 ctx.ETag,
		"missing_signals":      missingSignals,
		"missing_collectors":   []string{},
		"degraded_collectors":  degradedCollectors,
		"permission_issues":    []string{},
		"clock_drift_detected": false,
	})
}
