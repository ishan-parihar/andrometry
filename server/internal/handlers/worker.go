package handlers

import (
	"log"
	"time"
)

// StartDailySummaryWorker runs a background goroutine that periodically
// generates daily summaries for all active users.
// This ensures summaries are created even if the device doesn't trigger
// a batch ingestion that day.
func (h *Handler) StartDailySummaryWorker(interval time.Duration) {
	go func() {
		h.generateDailySummaries()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			h.generateDailySummaries()
		}
	}()
	log.Printf("[worker] daily summary worker started (interval: %s)", interval)
}

// generateDailySummaries generates daily summaries for all active users.
func (h *Handler) generateDailySummaries() {
	// Get all active devices (seen in last 7 days)
	devices, err := h.DB.GetActiveDevices(7)
	if err != nil {
		log.Printf("[worker] failed to get active devices: %v", err)
		return
	}

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")

	for _, device := range devices {
		// Check if a daily summary already exists for yesterday
		existing, _ := h.DB.GetDailySummary(device.UserID, yesterday)
		if existing != nil {
			continue // Already generated
		}

		// Generate daily summary for yesterday
		if err := h.Summarizer.GenerateDailySummary(device.UserID, yesterday, "UTC"); err != nil {
			log.Printf("[worker] daily summary generation failed for user %s: %v", device.UserID, err)
		} else {
			log.Printf("[worker] generated daily summary for user %s, date %s", device.UserID, yesterday)
		}
	}
}
