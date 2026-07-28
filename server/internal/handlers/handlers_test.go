package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/governor"
	"github.com/andrometry/server/internal/middleware"
	"github.com/andrometry/server/internal/models"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func setupTestHandler(t *testing.T) (*Handler, *db.DB) {
	t.Helper()
	database := setupTestDB(t)
	h := NewHandler(database)
	return h, database
}

func withUserContext(userID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.ActorTypeKey, "device")
	ctx = context.WithValue(ctx, middleware.ActorIDKey, "test-device")
	return ctx
}

func TestHealthCheck(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["server"] != "andrometry-relay" {
		t.Errorf("server = %v, want andrometry-relay", resp["server"])
	}
}

func TestBatchIngestion_validBatch(t *testing.T) {
	h, _ := setupTestHandler(t)

	batch := map[string]interface{}{
		"device_id":      "pixel-9",
		"batch_id":       "batch-1",
		"client_version": "0.1.0",
		"schema_version": 1,
		"sequence_start": 100,
		"sequence_end":   102,
		"events": []map[string]interface{}{
			{"event_id": "e1", "sequence": 100, "timestamp": "2026-01-01T00:00:00Z", "timezone": "UTC", "event_type": "screen_on", "schema_version": 1, "payload": map[string]interface{}{"state": "on"}},
			{"event_id": "e2", "sequence": 101, "timestamp": "2026-01-01T00:01:00Z", "timezone": "UTC", "event_type": "app_usage_summary", "schema_version": 1, "payload": map[string]interface{}{}},
			{"event_id": "e3", "sequence": 102, "timestamp": "2026-01-01T00:02:00Z", "timezone": "UTC", "event_type": "screen_off", "schema_version": 1, "payload": map[string]interface{}{}},
		},
		"segments": []map[string]interface{}{
			{"segment_id": "seg-1", "segment_type": "focus_segment", "layer": "focus", "start_time": "2026-01-01T09:00:00Z", "end_time": "2026-01-01T10:00:00Z", "label": "deep_work", "confidence": 0.85, "version": 1, "valid_from": "2026-01-01T09:00:00Z", "payload": map[string]interface{}{"interruptions": 2}},
		},
		"context": map[string]interface{}{
			"primary_activity": "deep_work",
			"timezone":         "Asia/Kolkata",
		},
	}

	body, _ := json.Marshal(batch)
	req := httptest.NewRequest("POST", "/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.BatchIngestion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["accepted_events"].(float64) != 3 {
		t.Errorf("accepted_events = %v, want 3", resp["accepted_events"])
	}
	if resp["accepted_segments"].(float64) != 1 {
		t.Errorf("accepted_segments = %v, want 1", resp["accepted_segments"])
	}
}

func TestBatchIngestion_emptyBatch(t *testing.T) {
	h, _ := setupTestHandler(t)

	batch := map[string]interface{}{
		"device_id":      "pixel-9",
		"batch_id":       "batch-empty",
		"client_version": "0.1.0",
		"schema_version": 1,
		"sequence_start": 100,
		"sequence_end":   100,
		"events":         []map[string]interface{}{},
		"segments":       []map[string]interface{}{},
	}

	body, _ := json.Marshal(batch)
	req := httptest.NewRequest("POST", "/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.BatchIngestion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["accepted_events"].(float64) != 0 {
		t.Errorf("accepted_events = %v, want 0", resp["accepted_events"])
	}
}

func TestBatchIngestion_invalidJSON(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/v1/batch", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.BatchIngestion(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetCurrentContext_noContext(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/context/current", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetCurrentContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCurrentContext_withContext(t *testing.T) {
	h, _ := setupTestHandler(t)

	batch := map[string]interface{}{
		"device_id":      "pixel-9",
		"batch_id":       "b1",
		"client_version": "0.1.0",
		"schema_version": 1,
		"sequence_start": 1,
		"sequence_end":   1,
		"events":         []map[string]interface{}{},
		"segments":       []map[string]interface{}{},
		"context": map[string]interface{}{
			"primary_activity": "deep_work",
			"timezone":         "Asia/Kolkata",
		},
	}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest("POST", "/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))
	w := httptest.NewRecorder()
	h.BatchIngestion(w, req)

	req2 := httptest.NewRequest("GET", "/v1/context/current", nil)
	req2 = req2.WithContext(withUserContext("primary"))
	w2 := httptest.NewRecorder()
	h.GetCurrentContext(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["etag"] == nil || resp["etag"] == "" {
		t.Error("expected etag in response")
	}
}

func TestGetCurrentContext_etagConditional(t *testing.T) {
	h, _ := setupTestHandler(t)

	h.DB.UpsertContext(&models.CurrentContext{
		DeviceID:  "pixel-9",
		UserID:    "primary",
		Timestamp: "2026-01-01T10:00:00Z",
		Timezone:  "UTC",
		Context:   `{"primary_activity":"deep_work"}`,
		ETag:      "abc123",
	})

	req := httptest.NewRequest("GET", "/v1/context/current", nil)
	req.Header.Set("If-None-Match", "abc123")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetCurrentContext(w, req)

	if w.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d (304)", w.Code, http.StatusNotModified)
	}
}

func TestPostObservation(t *testing.T) {
	h, _ := setupTestHandler(t)

	obs := map[string]interface{}{
		"agent_id":         "coach",
		"observation_type": "risk",
		"severity":         "medium",
		"message":          "Focus drifting",
		"evidence":         map[string]interface{}{"distraction_prob": 0.4},
	}

	body, _ := json.Marshal(obs)
	req := httptest.NewRequest("POST", "/v1/agent/observations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withUserContext("primary")
	ctx = context.WithValue(ctx, middleware.ActorTypeKey, "agent")
	ctx = context.WithValue(ctx, middleware.ActorIDKey, "coach")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.PostObservation(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["observation_id"] == "" {
		t.Error("expected observation_id in response")
	}
}

func TestPostIntervention(t *testing.T) {
	h, _ := setupTestHandler(t)

	intervention := map[string]interface{}{
		"agent_id":          "coach",
		"intervention_type": "nudge",
		"title":             "Take a break",
		"message":           "You've been working for 2 hours",
		"suggested_action":  "stand_up",
		"severity":          "low",
	}

	body, _ := json.Marshal(intervention)
	req := httptest.NewRequest("POST", "/v1/agent/interventions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withUserContext("primary")
	ctx = context.WithValue(ctx, middleware.ActorTypeKey, "agent")
	ctx = context.WithValue(ctx, middleware.ActorIDKey, "coach")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.PostIntervention(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["intervention_id"] == nil {
		t.Error("expected intervention_id in response")
	}
	// Status depends on quiet hours — approved outside quiet hours, proposed inside
	status := resp["status"].(string)
	if status != "approved_by_policy" && status != "proposed" {
		t.Errorf("status = %v, want approved_by_policy or proposed", status)
	}
}

func TestPostFeedback(t *testing.T) {
	h, _ := setupTestHandler(t)

	fb := map[string]interface{}{
		"target_type":   "intervention",
		"target_id":     "int-1",
		"feedback_type": "helpful",
		"rating":        4,
		"note":          "This helped me get back to work",
	}

	body, _ := json.Marshal(fb)
	req := httptest.NewRequest("POST", "/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.PostFeedback(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestPostGoal(t *testing.T) {
	h, _ := setupTestHandler(t)

	goal := map[string]interface{}{
		"goal_type": "deep_work_minutes",
		"title":     "Deep work 2 hours daily",
		"target":    map[string]interface{}{"minutes": 120},
	}

	body, _ := json.Marshal(goal)
	req := httptest.NewRequest("POST", "/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.PostGoal(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	req2 := httptest.NewRequest("GET", "/v1/goals", nil)
	req2 = req2.WithContext(withUserContext("primary"))
	w2 := httptest.NewRecorder()
	h.GetGoals(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("get goals status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	goals, ok := resp["goals"].([]interface{})
	if !ok || goals == nil {
		t.Fatalf("expected goals array in response, got: %v", resp)
	}
	if len(goals) != 1 {
		t.Errorf("goals count = %d, want 1", len(goals))
	}
}

func TestPostHabit(t *testing.T) {
	h, _ := setupTestHandler(t)

	habit := map[string]interface{}{
		"habit_type":       "morning_routine",
		"title":            "Wake before 7am",
		"target":           map[string]interface{}{"metric": "wake_before_7am"},
		"schedule":         map[string]interface{}{"days": []string{"mon", "tue", "wed", "thu", "fri"}},
		"evidence_sources": []string{"screen_state"},
	}

	body, _ := json.Marshal(habit)
	req := httptest.NewRequest("POST", "/v1/habits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.PostHabit(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestGetTimeline_empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/timeline?from=2026-01-01T00:00:00Z&to=2026-01-01T23:59:59Z", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetTimeline(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	timeline := resp["timeline"].([]interface{})
	if len(timeline) != 0 {
		t.Errorf("timeline count = %d, want 0", len(timeline))
	}
}

func TestGetDailySummary_notFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/summary/daily?date=2026-01-01", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetDailySummary(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetEvents_empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/events?from=2026-01-01T00:00:00Z&to=2026-01-01T23:59:59Z", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	events := resp["events"].([]interface{})
	if len(events) != 0 {
		t.Errorf("events count = %d, want 0", len(events))
	}
}

func TestPolicyGovernor_defaultPolicy(t *testing.T) {
	p := governor.DefaultPolicy()
	if p.MaxInterventionsPerDay != 6 {
		t.Errorf("max_interventions = %d, want 6", p.MaxInterventionsPerDay)
	}
	if p.CooldownMinutes != 45 {
		t.Errorf("cooldown = %d, want 45", p.CooldownMinutes)
	}
}

func TestPolicyGovernor_lowSeverityApproved(t *testing.T) {
	database := setupTestDB(t)
	p := governor.DefaultPolicy()
	approved, reason := governor.Evaluate(database, "primary", p, "low")
	// Approved outside quiet hours, or quiet_hours blocked — both are valid
	if !approved && reason != "quiet_hours" {
		t.Error("low severity should be approved or blocked by quiet hours")
	}
}

func TestPolicyGovernor_highSeverityApproved(t *testing.T) {
	database := setupTestDB(t)
	p := governor.DefaultPolicy()
	approved, _ := governor.Evaluate(database, "primary", p, "high")
	if !approved {
		t.Error("high severity should be approved")
	}
}

// --- New endpoint tests ---

func TestGetFiveMinuteSummaries_empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/summary/five_minute?from=2026-01-01T00:00:00Z&to=2026-01-01T23:59:59Z", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetFiveMinuteSummaries(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	summaries := resp["summaries"].([]interface{})
	if len(summaries) != 0 {
		t.Errorf("summaries count = %d, want 0", len(summaries))
	}
}

func TestGetHourlySummaries_empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/summary/hourly?from=2026-01-01T00:00:00Z&to=2026-01-01T23:59:59Z", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetHourlySummaries(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	summaries := resp["summaries"].([]interface{})
	if len(summaries) != 0 {
		t.Errorf("summaries count = %d, want 0", len(summaries))
	}
}

func TestGetBaselines_empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/v1/baselines", nil)
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.GetBaselines(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	baselines := resp["baselines"].([]interface{})
	if len(baselines) != 0 {
		t.Errorf("baselines count = %d, want 0", len(baselines))
	}
}

func TestPostBaseline_andGet(t *testing.T) {
	h, _ := setupTestHandler(t)

	baseline := map[string]interface{}{
		"metric_name": "deep_work_minutes",
		"window_days": 7,
		"value":       120.5,
	}

	body, _ := json.Marshal(baseline)
	req := httptest.NewRequest("POST", "/v1/baselines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.PostBaseline(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	req2 := httptest.NewRequest("GET", "/v1/baselines", nil)
	req2 = req2.WithContext(withUserContext("primary"))
	w2 := httptest.NewRecorder()
	h.GetBaselines(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("get baselines status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	baselines := resp["baselines"].([]interface{})
	if len(baselines) != 1 {
		t.Fatalf("baselines count = %d, want 1", len(baselines))
	}
}

func TestPostUserCorrection(t *testing.T) {
	h, _ := setupTestHandler(t)

	correction := map[string]interface{}{
		"correction_type": "label",
		"segment_id":      "seg-999",
		"new_label":       "deep_work",
		"reason":          "user_corrected",
	}

	body, _ := json.Marshal(correction)
	req := httptest.NewRequest("POST", "/v1/corrections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))

	w := httptest.NewRecorder()
	h.PostUserCorrection(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "created" {
		t.Errorf("status = %v, want created", resp["status"])
	}
	// segment_updated should be false since seg-999 doesn't exist
	if resp["segment_updated"] != false {
		t.Errorf("segment_updated = %v, want false for non-existent segment", resp["segment_updated"])
	}
}

// --- Round-trip tests (insert then query back) ---

func TestBaseline_roundTrip(t *testing.T) {
	h, _ := setupTestHandler(t)

	// Insert baseline
	baseline := map[string]interface{}{
		"metric_name": "screen_time_minutes",
		"window_days": 14,
		"value":       180.0,
	}
	body, _ := json.Marshal(baseline)
	req := httptest.NewRequest("POST", "/v1/baselines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext("primary"))
	w := httptest.NewRecorder()
	h.PostBaseline(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("post baseline status = %d, want %d", w.Code, http.StatusCreated)
	}

	var postResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &postResp)
	baselineID := postResp["baseline_id"]

	// Query back
	req2 := httptest.NewRequest("GET", "/v1/baselines", nil)
	req2 = req2.WithContext(withUserContext("primary"))
	w2 := httptest.NewRecorder()
	h.GetBaselines(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("get baselines status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	baselines := resp["baselines"].([]interface{})
	if len(baselines) != 1 {
		t.Fatalf("baselines count = %d, want 1", len(baselines))
	}

	b := baselines[0].(map[string]interface{})
	if b["baseline_id"] != baselineID {
		t.Errorf("baseline_id = %v, want %v", b["baseline_id"], baselineID)
	}
	if b["metric_name"] != "screen_time_minutes" {
		t.Errorf("metric_name = %v, want screen_time_minutes", b["metric_name"])
	}
	if b["value"] != 180.0 {
		t.Errorf("value = %v, want 180.0", b["value"])
	}

	// Upsert same metric — should update, not duplicate
	baseline2 := map[string]interface{}{
		"metric_name": "screen_time_minutes",
		"window_days": 7,
		"value":       150.0,
	}
	body2, _ := json.Marshal(baseline2)
	req3 := httptest.NewRequest("POST", "/v1/baselines", bytes.NewReader(body2))
	req3.Header.Set("Content-Type", "application/json")
	req3 = req3.WithContext(withUserContext("primary"))
	w3 := httptest.NewRecorder()
	h.PostBaseline(w3, req3)

	if w3.Code != http.StatusCreated {
		t.Fatalf("upsert baseline status = %d, want %d", w3.Code, http.StatusCreated)
	}

	// Verify still only one baseline
	req4 := httptest.NewRequest("GET", "/v1/baselines", nil)
	req4 = req4.WithContext(withUserContext("primary"))
	w4 := httptest.NewRecorder()
	h.GetBaselines(w4, req4)

	var resp2 map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &resp2)
	baselines2 := resp2["baselines"].([]interface{})
	if len(baselines2) != 1 {
		t.Fatalf("baselines count after upsert = %d, want 1", len(baselines2))
	}

	b2 := baselines2[0].(map[string]interface{})
	if b2["value"] != 150.0 {
		t.Errorf("value after upsert = %v, want 150.0", b2["value"])
	}

	// Delete baseline
	deleteReq := httptest.NewRequest("DELETE", "/v1/baselines/"+baselineID, nil)
	deleteReq = deleteReq.WithContext(withUserContext("primary"))
	w5 := httptest.NewRecorder()
	h.DeleteBaseline(w5, deleteReq)

	if w5.Code != http.StatusOK {
		t.Errorf("delete baseline status = %d, want %d", w5.Code, http.StatusOK)
	}

	// Verify deleted
	req5 := httptest.NewRequest("GET", "/v1/baselines", nil)
	req5 = req5.WithContext(withUserContext("primary"))
	w6 := httptest.NewRecorder()
	h.GetBaselines(w6, req5)

	var resp3 map[string]interface{}
	json.Unmarshal(w6.Body.Bytes(), &resp3)
	baselines3 := resp3["baselines"].([]interface{})
	if len(baselines3) != 0 {
		t.Errorf("baselines count after delete = %d, want 0", len(baselines3))
	}
}

func TestFiveMinuteSummary_roundTrip(t *testing.T) {
	h, db := setupTestHandler(t)

	// Insert five-minute summary directly via store
	err := db.UpsertFiveMinuteSummary("primary", 
		"2026-01-01T10:00:00Z",
		"2026-01-01T10:05:00Z",
		`{"activity":"deep_work","minutes":5}`,
	)
	if err != nil {
		t.Fatalf("upsert five_minute summary: %v", err)
	}

	// Query via API
	req := httptest.NewRequest("GET", "/v1/summary/five_minute?from=2026-01-01T09:00:00Z&to=2026-01-01T11:00:00Z", nil)
	req = req.WithContext(withUserContext("primary"))
	w := httptest.NewRecorder()
	h.GetFiveMinuteSummaries(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	summaries := resp["summaries"].([]interface{})
	if len(summaries) != 1 {
		t.Fatalf("summaries count = %d, want 1", len(summaries))
	}

	s := summaries[0].(map[string]interface{})
	if s["start_time"] != "2026-01-01T10:00:00Z" {
		t.Errorf("start_time = %v, want 2026-01-01T10:00:00Z", s["start_time"])
	}
	if s["summary"] != `{"activity":"deep_work","minutes":5}` {
		t.Errorf("summary = %v, want correct JSON", s["summary"])
	}
}

func TestHourlySummary_roundTrip(t *testing.T) {
	h, db := setupTestHandler(t)

	// Insert hourly summary directly via store
	err := db.UpsertHourlySummary("primary", 
		"2026-01-01T10:00:00Z",
		"2026-01-01T11:00:00Z",
		`{"focus_minutes":42,"meeting_minutes":15}`,
	)
	if err != nil {
		t.Fatalf("upsert hourly summary: %v", err)
	}

	// Query via API
	req := httptest.NewRequest("GET", "/v1/summary/hourly?from=2026-01-01T09:00:00Z&to=2026-01-01T12:00:00Z", nil)
	req = req.WithContext(withUserContext("primary"))
	w := httptest.NewRecorder()
	h.GetHourlySummaries(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	summaries := resp["summaries"].([]interface{})
	if len(summaries) != 1 {
		t.Fatalf("summaries count = %d, want 1", len(summaries))
	}

	s := summaries[0].(map[string]interface{})
	if s["start_time"] != "2026-01-01T10:00:00Z" {
		t.Errorf("start_time = %v, want 2026-01-01T10:00:00Z", s["start_time"])
	}
	if s["summary"] != `{"focus_minutes":42,"meeting_minutes":15}` {
		t.Errorf("summary = %v, want correct JSON", s["summary"])
	}
}
