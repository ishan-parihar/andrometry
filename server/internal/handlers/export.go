package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andrometry/server/internal/audit"
	"github.com/andrometry/server/internal/models"
)

// ExportDaily handles GET /v1/export/daily?date=YYYY-MM-DD
// §27 Data Export — exports all data for a single day as a structured JSON bundle.
// Includes: events, segments, summaries, context, observations, interventions,
// outcomes, feedback, goals, habits, baselines, corrections.
func (h *Handler) ExportDaily(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	date := queryParam(r, "date", time.Now().UTC().Format("2006-01-02"))

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		respondError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	// Build the day's time range in UTC
	dayStart := date + "T00:00:00Z"
	dayEnd := date + "T23:59:59Z"

	// Collect all data for the day
	events, _ := h.DB.GetEvents(uid, dayStart, dayEnd, "", 10000)
	segments, _ := h.DB.GetTimeline(uid, dayStart, dayEnd, 10000, "")
	dailySummary, _ := h.DB.GetDailySummary(uid, date)
	contextData, _ := h.DB.GetContextByUser(uid)
	observations, _ := h.DB.GetObservations(uid, 10000)
	interventions, _ := h.DB.GetInterventionsByDateRange(uid, dayStart, dayEnd, 10000)
	outcomes, _ := h.DB.GetOutcomes(uid, "", 10000)
	feedbacks, _ := h.DB.GetFeedback(uid, "", "", 10000)
	goals, _ := h.DB.GetGoals(uid)
	habits, _ := h.DB.GetHabits(uid)
	baselines, _ := h.DB.GetBaselines(uid)
	corrections, _ := h.DB.GetUserCorrections(uid, 10000)
	fiveMinSummaries, _ := h.DB.GetFiveMinuteSummaries(uid, dayStart, dayEnd)
	hourlySummaries, _ := h.DB.GetHourlySummaries(uid, dayStart, dayEnd)
	auditLogs, _ := h.DB.GetAuditLog(uid, 10000)

	// Nil-safe slices
	if events == nil {
		events = []models.Event{}
	}
	if segments == nil {
		segments = []models.TimelineSegment{}
	}
	if observations == nil {
		observations = []models.AgentObservation{}
	}
	if interventions == nil {
		interventions = []models.Intervention{}
	}
	if outcomes == nil {
		outcomes = []models.Outcome{}
	}
	if feedbacks == nil {
		feedbacks = []models.UserFeedback{}
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	if habits == nil {
		habits = []models.Habit{}
	}
	if baselines == nil {
		baselines = []models.Baseline{}
	}
	if corrections == nil {
		corrections = []map[string]string{}
	}
	if fiveMinSummaries == nil {
		fiveMinSummaries = []map[string]string{}
	}
	if hourlySummaries == nil {
		hourlySummaries = []map[string]string{}
	}
	if auditLogs == nil {
		auditLogs = []models.AuditLog{}
	}

	// Parse context
	var contextObj map[string]interface{}
	if contextData != nil {
		json.Unmarshal([]byte(contextData.Context), &contextObj)
	}

	// Parse daily summary
	var summaryObj map[string]interface{}
	if dailySummary != nil {
		json.Unmarshal([]byte(dailySummary.Summary), &summaryObj)
	}

	export := map[string]interface{}{
		"export_version":  "1.0",
		"exported_at":     nowUTC(),
		"user_id":         uid,
		"date":            date,
		"date_range":      map[string]string{"from": dayStart, "to": dayEnd},
		"events":          events,
		"segments":        segments,
		"daily_summary":   summaryObj,
		"five_minute_summaries": fiveMinSummaries,
		"hourly_summaries":      hourlySummaries,
		"context":         contextObj,
		"observations":    observations,
		"interventions":   interventions,
		"outcomes":        outcomes,
		"feedback":        feedbacks,
		"goals":           goals,
		"habits":          habits,
		"baselines":       baselines,
		"corrections":     corrections,
		"audit_log":       auditLogs,
	}

	audit.Log(h.DB, uid, "user", uid, "export_daily", "export", date, fmt.Sprintf(`{"date":"%s"}`, date))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="andrometry-export-%s.json"`, date))
	json.NewEncoder(w).Encode(export)
}

// ExportWeekly handles GET /v1/export/weekly?week=YYYY-WXX
// §27 Data Export — exports all data for a single week.
func (h *Handler) ExportWeekly(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	week := queryParam(r, "week", currentWeek())

	// Parse the week string to get date range
	weekStart, err := time.Parse("2006-W02", week)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid week format, expected YYYY-WXX")
		return
	}
	// ISO week: Monday is day 1
	weekStart = weekStart.Add(-time.Duration(weekStart.Weekday()-time.Monday) * 24 * time.Hour)
	weekEnd := weekStart.AddDate(0, 0, 7)
	dayStart := weekStart.Format(time.RFC3339)
	dayEnd := weekEnd.Format(time.RFC3339)

	// Collect all data for the week
	events, _ := h.DB.GetEvents(uid, dayStart, dayEnd, "", 50000)
	segments, _ := h.DB.GetTimeline(uid, dayStart, dayEnd, 50000, "")
	contextData, _ := h.DB.GetContextByUser(uid)
	observations, _ := h.DB.GetObservations(uid, 10000)
	interventions, _ := h.DB.GetInterventionsByDateRange(uid, dayStart, dayEnd, 10000)
	outcomes, _ := h.DB.GetOutcomes(uid, "", 10000)
	feedbacks, _ := h.DB.GetFeedback(uid, "", "", 10000)
	goals, _ := h.DB.GetGoals(uid)
	habits, _ := h.DB.GetHabits(uid)
	baselines, _ := h.DB.GetBaselines(uid)
	corrections, _ := h.DB.GetUserCorrections(uid, 10000)
	fiveMinSummaries, _ := h.DB.GetFiveMinuteSummaries(uid, dayStart, dayEnd)
	hourlySummaries, _ := h.DB.GetHourlySummaries(uid, dayStart, dayEnd)
	weeklySummary, _ := h.DB.GetWeeklySummary(uid, week)
	auditLogs, _ := h.DB.GetAuditLog(uid, 10000)

	// Nil-safe slices
	if events == nil { events = []models.Event{} }
	if segments == nil { segments = []models.TimelineSegment{} }
	if observations == nil { observations = []models.AgentObservation{} }
	if interventions == nil { interventions = []models.Intervention{} }
	if outcomes == nil { outcomes = []models.Outcome{} }
	if feedbacks == nil { feedbacks = []models.UserFeedback{} }
	if goals == nil { goals = []models.Goal{} }
	if habits == nil { habits = []models.Habit{} }
	if baselines == nil { baselines = []models.Baseline{} }
	if corrections == nil { corrections = []map[string]string{} }
	if fiveMinSummaries == nil { fiveMinSummaries = []map[string]string{} }
	if hourlySummaries == nil { hourlySummaries = []map[string]string{} }
	if auditLogs == nil { auditLogs = []models.AuditLog{} }

	var contextObj map[string]interface{}
	if contextData != nil {
		json.Unmarshal([]byte(contextData.Context), &contextObj)
	}

	var weeklyObj map[string]interface{}
	if weeklySummary != nil {
		json.Unmarshal([]byte(weeklySummary.Summary), &weeklyObj)
	}

	export := map[string]interface{}{
		"export_version":  "1.0",
		"exported_at":     nowUTC(),
		"user_id":         uid,
		"week":            week,
		"date_range":      map[string]string{"from": dayStart, "to": dayEnd},
		"events":          events,
		"segments":        segments,
		"weekly_summary":  weeklyObj,
		"five_minute_summaries": fiveMinSummaries,
		"hourly_summaries":      hourlySummaries,
		"context":         contextObj,
		"observations":    observations,
		"interventions":   interventions,
		"outcomes":        outcomes,
		"feedback":        feedbacks,
		"goals":           goals,
		"habits":          habits,
		"baselines":       baselines,
		"corrections":     corrections,
		"audit_log":       auditLogs,
	}

	audit.Log(h.DB, uid, "user", uid, "export_weekly", "export", week, fmt.Sprintf(`{"week":"%s"}`, week))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="andrometry-export-%s.json"`, week))
	json.NewEncoder(w).Encode(export)
}
