package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/andrometry/server/internal/audit"
)

// GetCorrectionFeedback handles GET /v1/corrections/feedback — returns aggregated correction statistics
// that can be used to adjust classification confidence scores (P1: feedback loop).
func (h *Handler) GetCorrectionFeedback(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	limit := queryParamInt(r, "limit", 100)

	// Get correction stats from the audit log
	stats, err := h.DB.GetCorrectionStats(uid, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query correction stats")
		return
	}

	// Build label correction map — positive = user corrected TO this label
	labelCorrections := make(map[string]int)
	totalCorrections := 0
	for _, s := range stats {
		if label, ok := s["label"].(string); ok {
			if count, ok := s["correction_count"].(int); ok {
				labelCorrections[label] = count
				totalCorrections += count
			}
		}
	}

	// Get actual segment count for correction rate
	totalSegments, _ := h.DB.CountSegments(uid)
	if totalSegments == 0 {
		totalSegments = 1 // avoid division by zero
	}

	// Generate suggested adjustments based on correction patterns
	adjustments := make(map[string]float64)
	for label, count := range labelCorrections {
		adjustment := float64(count) * 0.05 // 5% adjustment per correction
		if adjustment > 0.3 {
			adjustment = 0.3 // cap at 30%
		}
		if adjustment < -0.3 {
			adjustment = -0.3
		}
		adjustments[label] = adjustment
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"label_corrections":     labelCorrections,
		"total_corrections":     totalCorrections,
		"total_segments":        totalSegments,
		"correction_rate":       float64(totalCorrections) / float64(totalSegments),
		"suggested_adjustments": adjustments,
	})
}

// PostCorrectionFeedback handles POST /v1/corrections/feedback — applies correction feedback
// to adjust classification weights. This closes the feedback loop between user corrections
// and the classification system.
func (h *Handler) PostCorrectionFeedback(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req struct {
		SegmentID string `json:"segment_id"`
		OldLabel  string `json:"old_label"`
		NewLabel  string `json:"new_label"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.SegmentID == "" || req.NewLabel == "" {
		respondError(w, http.StatusBadRequest, "segment_id and new_label are required")
		return
	}

	uid := userID(r)

	// Fetch segment first to populate old_label if not provided
	segmentUpdated := false
	segment, _ := h.DB.GetSegmentByID(req.SegmentID)
	if segment != nil {
		if req.OldLabel == "" {
			req.OldLabel = segment.Label
		}
		segment.Label = req.NewLabel
		segment.Version++
		segment.RevisionReason = &req.Reason
		_ = h.DB.UpsertSegment(segment)
		segmentUpdated = true
	}

	// Store the correction with full context for feedback loop (now with correct old_label)
	correctionBytes, _ := json.Marshal(map[string]interface{}{
		"correction_type": "label",
		"segment_id":      req.SegmentID,
		"old_label":       req.OldLabel,
		"new_label":       req.NewLabel,
		"reason":          req.Reason,
	})

	err := h.DB.InsertUserCorrection(uid, string(correctionBytes))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to insert correction")
		return
	}

	audit.Log(h.DB, uid, actorType(r), actorID(r), "correction_feedback", "segment", req.SegmentID,
		`{"old_label":"`+req.OldLabel+`","new_label":"`+req.NewLabel+`","segment_updated":`+strconv.FormatBool(segmentUpdated)+`}`)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status":           "created",
		"segment_updated":  segmentUpdated,
		"feedback_applied": true,
	})
}
