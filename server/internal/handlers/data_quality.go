package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// GetDataQuality handles GET /v1/context/data-quality — returns data quality signals for the current context.
// §11.4 Data quality signals — agents need to know when not to trust a conclusion.
// Checks actual collector availability signals: usage_stats, notifications, health_connect, location, calendar.
func (h *Handler) GetDataQuality(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"completeness":         0.0,
			"staleness_seconds":    -1,
			"missing_collectors":   []string{"usage_stats", "notifications", "health_connect", "location", "calendar", "calls_sms"},
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

	// Parse context JSON to extract collector-specific signals
	var ctxMap map[string]interface{}
	if err := json.Unmarshal([]byte(ctx.Context), &ctxMap); err != nil {
		ctxMap = make(map[string]interface{})
	}

	// Check collector availability per §11.4
	// A collector is "available" if we have recent data from it in the context
	missingCollectors := []string{}
	degradedCollectors := []string{}

	// Usage stats: available if app_usage or screen data present
	if _, ok := ctxMap["app_usage_summary"]; !ok {
		if _, ok := ctxMap["screen_state"]; !ok {
			missingCollectors = append(missingCollectors, "usage_stats")
		}
	}

	// Notifications: available if notification events present
	if _, ok := ctxMap["notification_summary"]; !ok {
		missingCollectors = append(missingCollectors, "notifications")
	}

	// Health Connect: available if health data present
	if _, ok := ctxMap["health_data"]; !ok {
		missingCollectors = append(missingCollectors, "health_connect")
	}

	// Location: available if place or location data present
	if _, ok := ctxMap["place"]; !ok {
		if _, ok := ctxMap["location"]; !ok {
			missingCollectors = append(missingCollectors, "location")
		}
	}

	// Calendar: available if calendar events present
	if _, ok := ctxMap["calendar_events"]; !ok {
		missingCollectors = append(missingCollectors, "calendar")
	}

	// Calls/SMS: available if communication data present
	if _, ok := ctxMap["communication_summary"]; !ok {
		missingCollectors = append(missingCollectors, "calls_sms")
	}

	// Staleness-based degradation
	if stalenessSeconds > 300 {
		degradedCollectors = append(degradedCollectors, "context_sync")
	}
	if stalenessSeconds > 900 {
		degradedCollectors = append(degradedCollectors, "all_collectors")
	}

	// Compute completeness: 1.0 minus fraction of missing collectors
	totalCollectors := 6 // usage_stats, notifications, health_connect, location, calendar, calls_sms
	availableCount := totalCollectors - len(missingCollectors)
	completeness := float64(availableCount) / float64(totalCollectors)

	// Penalize for staleness
	if stalenessSeconds >= 900 {
		completeness *= 0.5
	} else if stalenessSeconds >= 300 {
		completeness *= 0.75
	}

	// Check for clock drift: context timestamp more than 5 minutes in the future
	clockDrift := false
	if err == nil && parsedTime.After(time.Now().Add(5 * time.Minute)) {
		clockDrift = true
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"completeness":         completeness,
		"staleness_seconds":    stalenessSeconds,
		"last_context_time":    ctx.Timestamp,
		"etag":                 ctx.ETag,
		"missing_collectors":   missingCollectors,
		"degraded_collectors":  degradedCollectors,
		"permission_issues":    []string{},
		"clock_drift_detected": clockDrift,
	})
}
