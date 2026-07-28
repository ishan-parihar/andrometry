package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/andrometry/server/internal/audit"
	"github.com/andrometry/server/internal/models"
)

// --- Work Pattern Metrics (§39) ---

// GetWorkPatternMetrics handles GET /v1/metrics/work-patterns — computes focus bout lengths,
// context switching rates, interruption density, and fragmentation scores from timeline segments.
func (h *Handler) GetWorkPatternMetrics(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	from := queryParam(r, "from", time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339))
	to := queryParam(r, "to", time.Now().UTC().Format(time.RFC3339))

	segments, err := h.DB.GetTimeline(uid, from, to, 1000, "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query timeline")
		return
	}

	metrics := computeWorkPatterns(segments)
	respondJSON(w, http.StatusOK, metrics)
}

// workPatternMetrics holds computed work pattern analytics.
type workPatternMetrics struct {
	Period                    string  `json:"period"`
	TotalFocusMinutes         float64 `json:"total_focus_minutes"`
	AverageFocusBoutMinutes   float64 `json:"average_focus_bout_minutes"`
	MaxFocusBoutMinutes       float64 `json:"max_focus_bout_minutes"`
	FocusBoutCount            int     `json:"focus_bout_count"`
	ContextSwitchingRatePerHr float64 `json:"context_switching_rate_per_hour"`
	MeetingMinutes            float64 `json:"meeting_minutes"`
	MeetingFragmentationScore float64 `json:"meeting_fragmentation_score"`
	CommunicationMinutes      float64 `json:"communication_minutes"`
	DistractionMinutes        float64 `json:"distraction_minutes"`
	InterruptionDensityPerHr  float64 `json:"interruption_density_per_hour"`
	DeepWorkRatio             float64 `json:"deep_work_ratio"`
	ShallowWorkRatio          float64 `json:"shallow_work_ratio"`
	BreakRatio                float64 `json:"break_ratio"`
	LateNightWorkMinutes      float64 `json:"late_night_work_minutes"`
	FragmentationScore        float64 `json:"fragmentation_score"`
}

func computeWorkPatterns(segments []models.TimelineSegment) workPatternMetrics {
	var m workPatternMetrics
	if len(segments) == 0 {
		return m
	}

	m.Period = segments[0].StartTime + " → " + segments[len(segments)-1].EndTime

	var totalMinutes float64
	var focusBouts []float64
	segmentTypeCounts := make(map[string]int)

	var switches int
	var prevType string

	for _, s := range segments {
		start, _ := time.Parse(time.RFC3339, s.StartTime)
		end, _ := time.Parse(time.RFC3339, s.EndTime)
		dur := end.Sub(start).Minutes()
		if dur <= 0 {
			continue
		}
		totalMinutes += dur
		segmentTypeCounts[s.SegmentType]++

		if prevType != "" && prevType != s.SegmentType {
			switches++
		}
		prevType = s.SegmentType

		if s.SegmentType == "focus_segment" || (s.SegmentType == "activity_segment" && s.Label == "deep_work") {
			focusBouts = append(focusBouts, dur)
			m.TotalFocusMinutes += dur
		}

		switch s.SegmentType {
		case "meeting_segment":
			m.MeetingMinutes += dur
		case "communication_segment":
			m.CommunicationMinutes += dur
		case "distraction_segment":
			m.DistractionMinutes += dur
		}

		hour := start.Hour()
		if (hour >= 22 || hour < 6) && (s.SegmentType == "activity_segment" || s.SegmentType == "focus_segment") {
			m.LateNightWorkMinutes += dur
		}
	}

	if len(focusBouts) > 0 {
		m.FocusBoutCount = len(focusBouts)
		var sum float64
		for _, b := range focusBouts {
			sum += b
			if b > m.MaxFocusBoutMinutes {
				m.MaxFocusBoutMinutes = b
			}
		}
		m.AverageFocusBoutMinutes = math.Round(sum/float64(len(focusBouts))*100) / 100
	}

	if totalMinutes > 0 {
		hours := totalMinutes / 60.0
		m.ContextSwitchingRatePerHr = math.Round(float64(switches)/hours*100) / 100
		m.DeepWorkRatio = math.Round(m.TotalFocusMinutes/totalMinutes*10000) / 10000
		others := m.MeetingMinutes + m.CommunicationMinutes + m.DistractionMinutes
		m.ShallowWorkRatio = math.Round((totalMinutes-m.TotalFocusMinutes-others)/totalMinutes*10000) / 10000
	}

	if m.MeetingMinutes > 0 {
		meetingCount := segmentTypeCounts["meeting_segment"]
		if meetingCount > 0 {
			m.MeetingFragmentationScore = math.Round(float64(meetingCount)/m.MeetingMinutes*60*100) / 100
		}
	}

	if m.TotalFocusMinutes > 0 {
		focusHours := m.TotalFocusMinutes / 60.0
		m.InterruptionDensityPerHr = math.Round(float64(switches)/focusHours*100) / 100
	}

	if totalMinutes > 0 {
		m.FragmentationScore = math.Round(float64(len(segments))/totalMinutes*60*100) / 100
	}

	return m
}

// --- Readiness and Fatigue Model (§40) ---

// GetReadinessScore handles GET /v1/metrics/readiness — computes a readiness score
// from sleep, activity, workload, and screen time inputs.
func (h *Handler) GetReadinessScore(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"readiness_score": 0.5,
			"confidence":      0.2,
			"inputs":          map[string]interface{}{},
			"note":            "no context available, returning default",
		})
		return
	}

	var contextData map[string]interface{}
	json.Unmarshal([]byte(ctx.Context), &contextData)

	from := time.Now().UTC().Add(-6 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	segments, _ := h.DB.GetTimeline(uid, from, to, 200, "")

	readiness := computeReadiness(contextData, segments)
	respondJSON(w, http.StatusOK, readiness)
}

type readinessResult struct {
	ReadinessScore  float64                `json:"readiness_score"`
	FatigueProb     float64                `json:"fatigue_probability"`
	Confidence      float64                `json:"confidence"`
	Inputs          map[string]interface{} `json:"inputs"`
	Recommendations []string               `json:"recommendations"`
}

func computeReadiness(context map[string]interface{}, segments []models.TimelineSegment) readinessResult {
	var r readinessResult
	r.Inputs = make(map[string]interface{})
	r.Recommendations = []string{}

	sleepMinutes := 0.0
	if sleep, ok := context["sleep"].(map[string]interface{}); ok {
		if sm, ok := sleep["last_sleep_minutes"].(float64); ok {
			sleepMinutes = sm
		}
	}
	r.Inputs["sleep_minutes"] = sleepMinutes

	focusMinutesToday := 0.0
	distractionMinutesToday := 0.0
	meetingMinutesToday := 0.0
	for _, s := range segments {
		start, _ := time.Parse(time.RFC3339, s.StartTime)
		end, _ := time.Parse(time.RFC3339, s.EndTime)
		dur := end.Sub(start).Minutes()
		switch s.SegmentType {
		case "focus_segment":
			focusMinutesToday += dur
		case "distraction_segment":
			distractionMinutesToday += dur
		case "meeting_segment":
			meetingMinutesToday += dur
		}
	}
	r.Inputs["focus_minutes_today"] = focusMinutesToday
	r.Inputs["distraction_minutes_today"] = distractionMinutesToday
	r.Inputs["meeting_minutes_today"] = meetingMinutesToday

	fatigueFromContext := 0.0
	if fp, ok := context["fatigue_probability"].(float64); ok {
		fatigueFromContext = fp
	}
	r.Inputs["fatigue_probability_from_context"] = fatigueFromContext

	screenTimeMinutes := 0.0
	if sc, ok := context["screen_time_minutes"].(float64); ok {
		screenTimeMinutes = sc
	}
	r.Inputs["screen_time_minutes"] = screenTimeMinutes

	sleepScore := 0.5
	if sleepMinutes > 0 {
		sleepScore = math.Min(1.0, sleepMinutes/480.0)
	}

	focusBalance := 0.5
	if focusMinutesToday+distractionMinutesToday > 0 {
		focusBalance = focusMinutesToday / (focusMinutesToday + distractionMinutesToday)
	}

	screenPenalty := 0.0
	if screenTimeMinutes > 480 {
		screenPenalty = (screenTimeMinutes - 480) / 480.0 * 0.2
	}

	score := 0.35*sleepScore + 0.25*(1.0-fatigueFromContext) + 0.25*focusBalance + 0.15*(1.0-screenPenalty)
	score = math.Max(0.0, math.Min(1.0, score))
	r.ReadinessScore = math.Round(score*1000) / 1000
	r.FatigueProb = math.Round(fatigueFromContext*1000) / 1000
	r.Confidence = 0.6

	if sleepMinutes < 360 {
		r.Recommendations = append(r.Recommendations, "sleep_debt_detected: consider catching up on sleep")
	}
	if fatigueFromContext > 0.6 {
		r.Recommendations = append(r.Recommendations, "high_fatigue: consider taking a break or reducing cognitive load")
	}
	if distractionMinutesToday > focusMinutesToday && focusMinutesToday > 0 {
		r.Recommendations = append(r.Recommendations, "distraction_heavy: focus sessions are being interrupted frequently")
	}
	if screenTimeMinutes > 600 {
		r.Recommendations = append(r.Recommendations, "excessive_screen_time: consider a screen break")
	}
	if len(r.Recommendations) == 0 {
		r.Recommendations = append(r.Recommendations, "good_state: current readiness is adequate for focused work")
	}

	return r
}

// --- Collector Health Status (§23.5) ---

// GetCollectorStatus handles GET /v1/health/collectors — returns per-collector staleness and health.
func (h *Handler) GetCollectorStatus(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"collectors": []interface{}{},
			"note":       "no context available",
		})
		return
	}

	var contextData map[string]interface{}
	json.Unmarshal([]byte(ctx.Context), &contextData)

	ctxTime, err := time.Parse(time.RFC3339, ctx.Timestamp)
	if err != nil {
		ctxTime = time.Now().UTC()
	}
	stalenessSec := time.Since(ctxTime).Seconds()

	var collectors []map[string]interface{}
	collectors = append(collectors, map[string]interface{}{
		"name":              "usage_stats",
		"status":            collectorHealthStatus(stalenessSec, contextData, "usage_stats_available"),
		"staleness_seconds": stalenessSec,
	})
	collectors = append(collectors, map[string]interface{}{
		"name":              "notification_listener",
		"status":            collectorHealthStatus(stalenessSec, contextData, "notification_listener_available"),
		"staleness_seconds": stalenessSec,
	})
	collectors = append(collectors, map[string]interface{}{
		"name":              "calendar",
		"status":            collectorHealthStatus(stalenessSec, contextData, "calendar_available"),
		"staleness_seconds": stalenessSec,
	})
	collectors = append(collectors, map[string]interface{}{
		"name":              "location",
		"status":            collectorHealthStatus(stalenessSec, contextData, "location_fresh"),
		"staleness_seconds": stalenessSec,
	})
	collectors = append(collectors, map[string]interface{}{
		"name":              "health_connect",
		"status":            collectorHealthStatus(stalenessSec, contextData, "health_connect_available"),
		"staleness_seconds": stalenessSec,
	})

	stats, _ := h.DB.GetDBStats()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"last_context_at":   ctx.Timestamp,
		"staleness_seconds": int(math.Round(stalenessSec)),
		"collectors":        collectors,
		"db_stats":          stats,
	})
}

func collectorHealthStatus(stalenessSec float64, context map[string]interface{}, key string) string {
	if stalenessSec > 900 {
		return "stale"
	}
	if dq, ok := context["data_quality"].(map[string]interface{}); ok {
		if v, ok := dq[key].(bool); ok && v {
			return "healthy"
		}
		return "degraded"
	}
	return "unknown"
}

// --- Goal Progress Evaluation (§36) ---

// EvaluateGoalProgress handles GET /v1/goals/{id}/progress — evaluates progress toward a goal.
func (h *Handler) EvaluateGoalProgress(w http.ResponseWriter, r *http.Request) {
	goalID := r.PathValue("id")
	if goalID == "" {
		respondError(w, http.StatusBadRequest, "missing goal ID")
		return
	}
	uid := userID(r)

	goal, err := h.DB.GetGoalByID(uid, goalID)
	if err != nil {
		respondError(w, http.StatusNotFound, "goal not found")
		return
	}

	var payload map[string]interface{}
	json.Unmarshal([]byte(goal.Payload), &payload)

	baselines, _ := h.DB.GetBaselines(uid)
	baselineMap := make(map[string]float64)
	for _, b := range baselines {
		baselineMap[b.MetricName] = b.Value
	}

	result := map[string]interface{}{
		"goal_id":    goal.GoalID,
		"goal_type":  goal.GoalType,
		"status":     goal.Status,
		"payload":    payload,
		"baselines":  baselineMap,
		"assessment": "pending_evaluation",
	}

	if goal.Status == "completed" {
		result["assessment"] = "completed"
		result["progress"] = 1.0
	} else if goal.Status == "abandoned" {
		result["assessment"] = "abandoned"
		result["progress"] = 0.0
	}

	audit.Log(h.DB, uid, "system", "", "evaluate_goal_progress", "goal", goalID, "")
	respondJSON(w, http.StatusOK, result)
}

// --- Habit Completion Evaluation (§38) ---

// EvaluateHabitCompletion handles GET /v1/habits/{id}/evaluation — evaluates habit adherence.
func (h *Handler) EvaluateHabitCompletion(w http.ResponseWriter, r *http.Request) {
	habitID := r.PathValue("id")
	if habitID == "" {
		respondError(w, http.StatusBadRequest, "missing habit ID")
		return
	}
	uid := userID(r)

	habit, err := h.DB.GetHabitByID(uid, habitID)
	if err != nil {
		respondError(w, http.StatusNotFound, "habit not found")
		return
	}

	var target map[string]interface{}
	json.Unmarshal([]byte(habit.Target), &target)
	var schedule map[string]interface{}
	json.Unmarshal([]byte(habit.Schedule), &schedule)

	result := map[string]interface{}{
		"habit_id":       habit.HabitID,
		"habit_type":     habit.HabitType,
		"status":         habit.Status,
		"streak_days":    habit.StreakDays,
		"habit_strength": habit.HabitStrength,
		"target":         target,
		"schedule":       schedule,
		"assessment":     "pending_tracking",
	}

	if habit.StreakDays >= 21 {
		result["assessment"] = "habit_forming"
	} else if habit.StreakDays >= 7 {
		result["assessment"] = "building_streak"
	} else if habit.StreakDays > 0 {
		result["assessment"] = "early_stage"
	}

	if habit.HabitStrength != nil && *habit.HabitStrength > 0.7 {
		result["strength_assessment"] = "strong"
	} else {
		result["strength_assessment"] = "developing"
	}

	audit.Log(h.DB, uid, "system", "", "evaluate_habit_completion", "habit", habitID, "")
	respondJSON(w, http.StatusOK, result)
}
