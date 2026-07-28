package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrometry/server/internal/audit"
	"github.com/andrometry/server/internal/briefing"
	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/governor"
	"github.com/andrometry/server/internal/middleware"
	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	DB     *db.DB
	Policy governor.Policy
}

// NewHandler creates a new Handler with the given database and default policy.
func NewHandler(database *db.DB) *Handler {
	return &Handler{
		DB:     database,
		Policy: governor.DefaultPolicy(),
	}
}

func userID(r *http.Request) string {
	if v, ok := r.Context().Value(middleware.UserIDKey).(string); ok {
		return v
	}
	return "primary"
}

func actorID(r *http.Request) string {
	if v, ok := r.Context().Value(middleware.ActorIDKey).(string); ok {
		return v
	}
	return "unknown"
}

func actorType(r *http.Request) string {
	if v, ok := r.Context().Value(middleware.ActorTypeKey).(string); ok {
		return v
	}
	return "unknown"
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func queryParam(r *http.Request, key, fallback string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	return v
}

func queryParamInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func generateID() string {
	return uuid.New().String()
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// --- Device Ingestion ---

// BatchIngestion handles POST /v1/batch — accepts device event batches.
func (h *Handler) BatchIngestion(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var batch struct {
		DeviceID      string                   `json:"device_id"`
		BatchID       string                   `json:"batch_id"`
		ClientVersion string                   `json:"client_version"`
		SchemaVersion int                      `json:"schema_version"`
		SequenceStart int64                    `json:"sequence_start"`
		SequenceEnd   int64                    `json:"sequence_end"`
		Events        []map[string]interface{} `json:"events"`
		Segments      []map[string]interface{} `json:"segments"`
		Summaries     []map[string]interface{} `json:"summaries"`
		Context       map[string]interface{}   `json:"context"`
		DeviceState   map[string]interface{}   `json:"device_state"`
	}

	if err := json.Unmarshal(body, &batch); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)

	// Upsert device
	_ = h.DB.UpsertDevice(&models.Device{
		DeviceID:      batch.DeviceID,
		UserID:        uid,
		ClientVersion: batch.ClientVersion,
	})

	// Insert events
	var events []*models.Event
	for _, e := range batch.Events {
		payloadBytes, _ := json.Marshal(e["payload"])
		events = append(events, &models.Event{
			EventID:       getStringFromMap(e, "event_id", generateID()),
			DeviceID:      batch.DeviceID,
			UserID:        uid,
			Sequence:      getInt64FromMap(e, "sequence"),
			Timestamp:     getStringFromMap(e, "timestamp", nowUTC()),
			Timezone:      getStringFromMap(e, "timezone", "UTC"),
			EventType:     getStringFromMap(e, "event_type", "unknown"),
			Collector:     getStringFromMap(e, "collector", ""),
			Source:        getStringFromMap(e, "source", ""),
			SchemaVersion: getIntFromMap(e, "schema_version", 1),
			Payload:       string(payloadBytes),
		})
	}
	acceptedEvents, _ := h.DB.InsertEventsBatch(events)

	// Upsert segments
	var segments []*models.TimelineSegment
	for _, s := range batch.Segments {
		payloadBytes, _ := json.Marshal(s["payload"])
		evidenceBytes, _ := json.Marshal(s["evidence"])
		segments = append(segments, &models.TimelineSegment{
			SegmentID:   getStringFromMap(s, "segment_id", generateID()),
			DeviceID:    batch.DeviceID,
			UserID:      uid,
			SegmentType: getStringFromMap(s, "segment_type", "activity_segment"),
			Layer:       getStringFromMap(s, "layer", "activity"),
			StartTime:   getStringFromMap(s, "start_time", nowUTC()),
			EndTime:     getStringFromMap(s, "end_time", nowUTC()),
			Label:       getStringFromMap(s, "label", "unknown"),
			Confidence:  getFloat64FromMap(s, "confidence"),
			Version:     getIntFromMap(s, "version", 1),
			ValidFrom:   getStringFromMap(s, "valid_from", nowUTC()),
			Evidence:    string(evidenceBytes),
			Payload:     string(payloadBytes),
		})
	}
	acceptedSegments, _ := h.DB.UpsertSegmentsBatch(segments)

	// Upsert context
	if len(batch.Context) > 0 {
		ctxBytes, _ := json.Marshal(batch.Context)
		etag := fmt.Sprintf("%x", md5Sum(ctxBytes))
		_ = h.DB.UpsertContext(&models.CurrentContext{
			DeviceID:  batch.DeviceID,
			UserID:    uid,
			Timestamp: nowUTC(),
			Timezone:  getStringFromMap(batch.Context, "timezone", "UTC"),
			Context:   string(ctxBytes),
			ETag:      etag,
		})
	}

	// Upsert summaries
	acceptedSummaries := 0
	for _, s := range batch.Summaries {
		summaryBytes, _ := json.Marshal(s)
		date := getStringFromMap(s, "date", "")
		tz := getStringFromMap(s, "timezone", "UTC")
		if date != "" {
			_ = h.DB.UpsertDailySummary(&models.DailySummary{
				UserID:   uid,
				Date:     date,
				Timezone: tz,
				Summary:  string(summaryBytes),
				Version:  1,
			})
			acceptedSummaries++
		}
	}

	// Count pending interventions
	pendingInterventions, _ := h.DB.CountInterventionsToday(uid)
	_ = audit.Log(h.DB, uid, "device", batch.DeviceID, "batch_ingest", "batch", batch.BatchID,
		fmt.Sprintf(`{"events":%d,"segments":%d}`, acceptedEvents, acceptedSegments))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"accepted_events":       acceptedEvents,
		"rejected_events":       len(batch.Events) - acceptedEvents,
		"accepted_segments":     acceptedSegments,
		"accepted_summaries":    acceptedSummaries,
		"missing_sequences":     []int64{},
		"pending_interventions": pendingInterventions,
		"server_time":           nowUTC(),
	})
}

// --- Context ---

// GetCurrentContext handles GET /v1/context/current.
func (h *Handler) GetCurrentContext(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondError(w, http.StatusNotFound, "no context available")
		return
	}

	// ETag support
	ifMatch := r.Header.Get("If-None-Match")
	if ifMatch != "" && ifMatch == ctx.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", ctx.ETag)
	w.Header().Set("Content-Type", "application/json")

	// Return the context object with metadata
	var contextData map[string]interface{}
	json.Unmarshal([]byte(ctx.Context), &contextData)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timestamp": ctx.Timestamp,
		"timezone":  ctx.Timezone,
		"device_id": ctx.DeviceID,
		"etag":      ctx.ETag,
		"context":   contextData,
	})
}

// --- Briefing ---

// GetCurrentBriefing handles GET /v1/briefing/current.
func (h *Handler) GetCurrentBriefing(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	ctx, err := h.DB.GetContextByUser(uid)
	if err != nil {
		respondError(w, http.StatusNotFound, "no context available for briefing")
		return
	}

	briefingID := generateID()
	b, err := briefing.Generate(ctx.Context, briefingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate briefing")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, briefing.BriefingToJSON(b))
}

// --- Timeline ---

// GetTimeline handles GET /v1/timeline.
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	from := queryParam(r, "from", time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339))
	to := queryParam(r, "to", time.Now().UTC().Format(time.RFC3339))
	limit := queryParamInt(r, "limit", 200)
	segmentType := queryParam(r, "segment_type", "")

	segments, err := h.DB.GetTimeline(uid, from, to, limit, segmentType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query timeline")
		return
	}

	// Convert to API response format
	type segmentResp struct {
		SegmentID   string      `json:"segment_id"`
		SegmentType string      `json:"segment_type"`
		Layer       string      `json:"layer"`
		Start       string      `json:"start"`
		End         string      `json:"end"`
		Label       string      `json:"label"`
		Confidence  float64     `json:"confidence"`
		Version     int         `json:"version"`
		Payload     interface{} `json:"payload"`
	}

	var resp []segmentResp
	for _, s := range segments {
		var payload interface{}
		json.Unmarshal([]byte(s.Payload), &payload)
		resp = append(resp, segmentResp{
			SegmentID:   s.SegmentID,
			SegmentType: s.SegmentType,
			Layer:       s.Layer,
			Start:       s.StartTime,
			End:         s.EndTime,
			Label:       s.Label,
			Confidence:  s.Confidence,
			Version:     s.Version,
			Payload:     payload,
		})
	}

	if resp == nil {
		resp = []segmentResp{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timeline": resp,
	})
}

// --- Summaries ---

// GetDailySummary handles GET /v1/summary/daily.
func (h *Handler) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	date := queryParam(r, "date", time.Now().UTC().Format("2006-01-02"))

	summary, err := h.DB.GetDailySummary(uid, date)
	if err != nil {
		respondError(w, http.StatusNotFound, "no summary for this date")
		return
	}

	var summaryData interface{}
	json.Unmarshal([]byte(summary.Summary), &summaryData)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"date":     summary.Date,
		"timezone": summary.Timezone,
		"version":  summary.Version,
		"summary":  summaryData,
	})
}

// GetWeeklySummary handles GET /v1/summary/weekly.
func (h *Handler) GetWeeklySummary(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	week := queryParam(r, "week", currentWeek())

	summary, err := h.DB.GetWeeklySummary(uid, week)
	if err != nil {
		respondError(w, http.StatusNotFound, "no summary for this week")
		return
	}

	var summaryData interface{}
	json.Unmarshal([]byte(summary.Summary), &summaryData)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"week":     summary.Week,
		"timezone": summary.Timezone,
		"version":  summary.Version,
		"summary":  summaryData,
	})
}

// --- Events (drill-down) ---

// GetEvents handles GET /v1/events.
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	from := queryParam(r, "from", time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339))
	to := queryParam(r, "to", time.Now().UTC().Format(time.RFC3339))
	eventType := queryParam(r, "event_type", "")
	limit := queryParamInt(r, "limit", 200)

	events, err := h.DB.GetEvents(uid, from, to, eventType, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query events")
		return
	}

	if events == nil {
		events = []models.Event{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
	})
}

// --- Agent Observations ---

// PostObservation handles POST /v1/agent/observations.
func (h *Handler) PostObservation(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var obs struct {
		ObservationID   string                 `json:"observation_id"`
		AgentID         string                 `json:"agent_id"`
		ObservationType string                 `json:"observation_type"`
		Severity        string                 `json:"severity"`
		Message         string                 `json:"message"`
		Evidence        map[string]interface{} `json:"evidence"`
		Timestamp       string                 `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &obs); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if obs.ObservationID == "" {
		obs.ObservationID = generateID()
	}
	if obs.Timestamp == "" {
		obs.Timestamp = nowUTC()
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"message":  obs.Message,
		"evidence": obs.Evidence,
	})

	err := h.DB.InsertObservation(&models.AgentObservation{
		ObservationID:   obs.ObservationID,
		UserID:          uid,
		Timestamp:       obs.Timestamp,
		AgentID:         obs.AgentID,
		ObservationType: obs.ObservationType,
		Severity:        obs.Severity,
		Payload:         string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert observation")
		return
	}

	audit.Log(h.DB, uid, "agent", obs.AgentID, "create_observation", "observation", obs.ObservationID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"observation_id": obs.ObservationID, "status": "created"})
}

// GetObservations handles GET /v1/agent/observations.
func (h *Handler) GetObservations(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	limit := queryParamInt(r, "limit", 50)

	observations, err := h.DB.GetObservations(uid, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query observations")
		return
	}

	if observations == nil {
		observations = []models.AgentObservation{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"observations": observations,
	})
}

// --- Interventions ---

// PostIntervention handles POST /v1/agent/interventions.
func (h *Handler) PostIntervention(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var intervention struct {
		InterventionID   string                 `json:"intervention_id"`
		AgentID          string                 `json:"agent_id"`
		InterventionType string                 `json:"intervention_type"`
		Title            string                 `json:"title"`
		Message          string                 `json:"message"`
		SuggestedAction  string                 `json:"suggested_action"`
		Severity         string                 `json:"severity"`
		Evidence         map[string]interface{} `json:"evidence"`
		Timestamp        string                 `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &intervention); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if intervention.InterventionID == "" {
		intervention.InterventionID = generateID()
	}
	if intervention.Timestamp == "" {
		intervention.Timestamp = nowUTC()
	}
	if intervention.Severity == "" {
		intervention.Severity = "medium"
	}

	// Evaluate against policy
	approved, reason := governor.Evaluate(h.DB, uid, h.Policy, intervention.Severity)
	status := "proposed"
	if approved {
		status = "approved_by_policy"
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"title":            intervention.Title,
		"message":          intervention.Message,
		"suggested_action": intervention.SuggestedAction,
		"severity":         intervention.Severity,
		"evidence":         intervention.Evidence,
	})

	err := h.DB.InsertIntervention(&models.Intervention{
		InterventionID:   intervention.InterventionID,
		UserID:           uid,
		Timestamp:        intervention.Timestamp,
		AgentID:          intervention.AgentID,
		InterventionType: intervention.InterventionType,
		Status:           status,
		Payload:          string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert intervention")
		return
	}

	audit.Log(h.DB, uid, "agent", intervention.AgentID, "create_intervention", "intervention", intervention.InterventionID,
		fmt.Sprintf(`{"status":"%s","rejected_reason":"%s"}`, status, reason))

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"intervention_id": intervention.InterventionID,
		"status":          status,
		"rejected_reason": reason,
	})
}

// UpdateInterventionStatus handles PATCH /v1/agent/interventions/:id.
func (h *Handler) UpdateInterventionStatus(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		respondError(w, http.StatusBadRequest, "missing intervention ID")
		return
	}
	interventionID := parts[4]

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var update struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &update); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.DB.UpdateInterventionStatus(interventionID, update.Status); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update intervention")
		return
	}

	uid := userID(r)
	audit.Log(h.DB, uid, actorType(r), actorID(r), "update_intervention_status", "intervention", interventionID,
		fmt.Sprintf(`{"status":"%s"}`, update.Status))

	respondJSON(w, http.StatusOK, map[string]string{"intervention_id": interventionID, "status": update.Status})
}

// GetPendingInterventions handles GET /v1/device/interventions.
func (h *Handler) GetPendingInterventions(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	interventions, err := h.DB.GetPendingInterventions(uid, 10)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query interventions")
		return
	}
	if interventions == nil {
		interventions = []models.Intervention{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"interventions": interventions,
	})
}

// --- Feedback ---

// PostFeedback handles POST /v1/feedback.
func (h *Handler) PostFeedback(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var fb struct {
		FeedbackID   string                 `json:"feedback_id"`
		TargetType   string                 `json:"target_type"`
		TargetID     string                 `json:"target_id"`
		FeedbackType string                 `json:"feedback_type"`
		Rating       int                    `json:"rating"`
		Note         string                 `json:"note"`
		Timestamp    string                 `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &fb); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if fb.FeedbackID == "" {
		fb.FeedbackID = generateID()
	}
	if fb.Timestamp == "" {
		fb.Timestamp = nowUTC()
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"rating": fb.Rating,
		"note":   fb.Note,
	})

	err := h.DB.InsertFeedback(&models.UserFeedback{
		FeedbackID:   fb.FeedbackID,
		UserID:       uid,
		Timestamp:    fb.Timestamp,
		TargetType:   fb.TargetType,
		TargetID:     fb.TargetID,
		FeedbackType: fb.FeedbackType,
		Payload:      string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert feedback")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_feedback", fb.TargetType, fb.TargetID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"feedback_id": fb.FeedbackID, "status": "created"})
}

// --- Goals ---

// GetGoals handles GET /v1/goals.
func (h *Handler) GetGoals(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	goals, err := h.DB.GetGoals(uid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query goals")
		return
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"goals": goals})
}

// PostGoal handles POST /v1/goals.
func (h *Handler) PostGoal(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var goal struct {
		GoalID   string                 `json:"goal_id"`
		GoalType string                 `json:"goal_type"`
		Title    string                 `json:"title"`
		Status   string                 `json:"status"`
		Target   map[string]interface{} `json:"target"`
	}

	if err := json.Unmarshal(body, &goal); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if goal.GoalID == "" {
		goal.GoalID = generateID()
	}
	if goal.Status == "" {
		goal.Status = "active"
	}

	targetBytes, _ := json.Marshal(goal.Target)
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"title":  goal.Title,
		"target": goal.Target,
	})

	err := h.DB.InsertGoal(&models.Goal{
		GoalID:   goal.GoalID,
		UserID:   uid,
		GoalType: goal.GoalType,
		Status:   goal.Status,
		Payload:  string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert goal")
		return
	}
	_ = targetBytes

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_goal", "goal", goal.GoalID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"goal_id": goal.GoalID, "status": "created"})
}

// --- Habits ---

// GetHabits handles GET /v1/habits.
func (h *Handler) GetHabits(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	habits, err := h.DB.GetHabits(uid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query habits")
		return
	}
	if habits == nil {
		habits = []models.Habit{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"habits": habits})
}

// PostHabit handles POST /v1/habits.
func (h *Handler) PostHabit(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var habit struct {
		HabitID         string   `json:"habit_id"`
		HabitType       string   `json:"habit_type"`
		Title           string   `json:"title"`
		Target          struct {
			Metric  string  `json:"metric"`
			Minimum float64 `json:"minimum"`
			Unit    string  `json:"unit"`
		} `json:"target"`
		Schedule struct {
			Days   []string `json:"days"`
			Window string   `json:"window"`
		} `json:"schedule"`
		EvidenceSources []string `json:"evidence_sources"`
	}

	if err := json.Unmarshal(body, &habit); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if habit.HabitID == "" {
		habit.HabitID = generateID()
	}

	targetBytes, _ := json.Marshal(habit.Target)
	scheduleBytes, _ := json.Marshal(habit.Schedule)
	evidenceBytes, _ := json.Marshal(habit.EvidenceSources)

	err := h.DB.InsertHabit(&models.Habit{
		HabitID:         habit.HabitID,
		UserID:          uid,
		HabitType:       habit.HabitType,
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

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_habit", "habit", habit.HabitID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"habit_id": habit.HabitID, "status": "created"})
}

// --- Outcomes ---

// PostOutcome handles POST /v1/outcomes.
func (h *Handler) PostOutcome(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var outcome struct {
		OutcomeID          string                 `json:"outcome_id"`
		InterventionID     string                 `json:"intervention_id"`
		OutcomeType        string                 `json:"outcome_type"`
		EffectivenessScore *float64               `json:"effectiveness_score"`
		Evidence           map[string]interface{} `json:"evidence"`
		SelfReport         map[string]interface{} `json:"self_report"`
		Timestamp          string                 `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &outcome); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	uid := userID(r)
	if outcome.OutcomeID == "" {
		outcome.OutcomeID = generateID()
	}
	if outcome.Timestamp == "" {
		outcome.Timestamp = nowUTC()
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"evidence":   outcome.Evidence,
		"self_report": outcome.SelfReport,
	})

	err := h.DB.InsertOutcome(&models.Outcome{
		OutcomeID:          outcome.OutcomeID,
		UserID:             uid,
		InterventionID:     outcome.InterventionID,
		Timestamp:          outcome.Timestamp,
		OutcomeType:        outcome.OutcomeType,
		EffectivenessScore: outcome.EffectivenessScore,
		Payload:            string(payloadBytes),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert outcome")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "create_outcome", "outcome", outcome.OutcomeID, "")
	respondJSON(w, http.StatusCreated, map[string]string{"outcome_id": outcome.OutcomeID, "status": "created"})
}

// GetOutcomes handles GET /v1/outcomes.
func (h *Handler) GetOutcomes(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	interventionID := queryParam(r, "intervention_id", "")
	limit := queryParamInt(r, "limit", 50)

	outcomes, err := h.DB.GetOutcomes(uid, interventionID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query outcomes")
		return
	}
	if outcomes == nil {
		outcomes = []models.Outcome{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"outcomes": outcomes})
}

// --- Audit Log ---

// GetAuditLog handles GET /v1/audit.
func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	limit := queryParamInt(r, "limit", 100)

	logs, err := h.DB.GetAuditLog(uid, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query audit log")
		return
	}
	if logs == nil {
		logs = []models.AuditLog{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"audit_log": logs})
}

// --- Health ---

// HealthCheck handles GET /v1/health.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"server":    "andrometry-relay",
		"version":   "0.1.0",
		"timestamp": nowUTC(),
	})
}

// --- Helpers ---

func getStringFromMap(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return 1
}

func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case int64:
			return val
		case int:
			return int64(val)
		}
	}
	return 0
}

func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func currentWeek() string {
	_, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", time.Now().Year(), w)
}

func md5Sum(data []byte) [16]byte {
	// Simple non-cryptographic hash for ETag generation
	var result [16]byte
	hash := uint64(0xcbf29ce484222325) // FNV-1a offset basis
	for _, b := range data {
		hash ^= uint64(b)
		hash *= 0x100000001b3 // FNV-1a prime
	}
	result[0] = byte(hash)
	result[1] = byte(hash >> 8)
	result[2] = byte(hash >> 16)
	result[3] = byte(hash >> 24)
	result[4] = byte(hash >> 32)
	result[5] = byte(hash >> 40)
	result[6] = byte(hash >> 48)
	result[7] = byte(hash >> 56)
	// Second 8 bytes from a second pass
	hash2 := uint64(0x517cc1b727220a95) // FNV-1a prime as offset
	for i := len(data) - 1; i >= 0; i-- {
		hash2 ^= uint64(data[i])
		hash2 *= 0x100000001b3
	}
	result[8] = byte(hash2)
	result[9] = byte(hash2 >> 8)
	result[10] = byte(hash2 >> 16)
	result[11] = byte(hash2 >> 24)
	result[12] = byte(hash2 >> 32)
	result[13] = byte(hash2 >> 40)
	result[14] = byte(hash2 >> 48)
	result[15] = byte(hash2 >> 56)
	return result
}
