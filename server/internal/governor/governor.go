package governor

import (
	"encoding/json"
	"time"

	"github.com/andrometry/server/internal/db"
)

// Policy defines the rules for agent intervention rate limiting.
type Policy struct {
	MaxInterventionsPerDay     int       `json:"max_interventions_per_day"`
	CooldownMinutes            int       `json:"cooldown_minutes"`
	QuietHoursStart            string    `json:"quiet_hours_start"` // HH:MM
	QuietHoursEnd              string    `json:"quiet_hours_end"`   // HH:MM
	MinConfidenceForNudge      float64   `json:"min_confidence_for_nudge"`
	SuppressAfterDismissals    bool      `json:"suppress_after_repeated_dismissals"`
	AllowHighSeverityOverride  bool      `json:"allow_high_severity_override"`
	MaxObservationsPerDay      int       `json:"max_observations_per_day"`
}

// DefaultPolicy returns the default policy settings.
func DefaultPolicy() Policy {
	return Policy{
		MaxInterventionsPerDay:    6,
		CooldownMinutes:           45,
		QuietHoursStart:           "22:30",
		QuietHoursEnd:             "07:30",
		MinConfidenceForNudge:     0.65,
		SuppressAfterDismissals:   true,
		AllowHighSeverityOverride: true,
		MaxObservationsPerDay:     20,
	}
}// Evaluate checks whether an intervention should be approved by policy.
func Evaluate(database *db.DB, userID string, policy Policy, severity string) (approved bool, reason string) {
	// Check quiet hours
	if inQuietHours(policy) {
		if severity != "high" || !policy.AllowHighSeverityOverride {
			return false, "quiet_hours"
	}
	}

	// Check daily intervention limit
	count, err := database.CountInterventionsToday(userID)
	if err == nil && count >= policy.MaxInterventionsPerDay {
		return false, "daily_limit_reached"
	}

	// Check cooldown between interventions
	if policy.CooldownMinutes > 0 {
		lastTs, err := database.GetLastInterventionTime(userID)
		if err == nil && lastTs != "" {
			lastTime, parseErr := time.Parse(time.RFC3339, lastTs)
			if parseErr == nil {
				elapsed := time.Since(lastTime)
				cooldown := time.Duration(policy.CooldownMinutes) * time.Minute
				if elapsed < cooldown {
					return false, "cooldown"
				}
			}
		}
	}

	return true, ""
}

// inQuietHours checks if the current time is within quiet hours.
func inQuietHours(p Policy) bool {
	now := time.Now()
	currentTime := now.Format("15:04")

	if p.QuietHoursStart > p.QuietHoursEnd {
		// Crosses midnight, e.g. 22:30 to 07:30
		return currentTime >= p.QuietHoursStart || currentTime <= p.QuietHoursEnd
	}
	return currentTime >= p.QuietHoursStart && currentTime <= p.QuietHoursEnd
}

// MarshalPolicy serializes a policy to JSON.
func MarshalPolicy(p Policy) string {
	b, _ := json.Marshal(p)
	return string(b)
}

// UnmarshalPolicy deserializes a policy from JSON.
func UnmarshalPolicy(data string) Policy {
	var p Policy
	if data == "" {
		return DefaultPolicy()
	}
	json.Unmarshal([]byte(data), &p)
	return p
}
