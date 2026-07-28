package governor

import (
	"encoding/json"
	"log"
)

// SensitiveMode controls privacy-preserving behavior.
type SensitiveMode struct {
	Active           bool     `json:"active"`
	PauseSync        bool     `json:"pause_sync"`
	RedactLocation   bool     `json:"redact_location"`
	RedactAppUsage   bool     `json:"redact_app_usage"`
	RedactCalendar   bool     `json:"redact_calendar"`
	MarkContextPrivate bool   `json:"mark_context_private"`
	MaxRetentionDays int      `json:"max_retention_days"`
	AllowedEventTypes []string `json:"allowed_event_types"`
}

// DefaultSensitiveMode returns a restrictive sensitive mode configuration.
func DefaultSensitiveMode() SensitiveMode {
	return SensitiveMode{
		Active:             false,
		PauseSync:          true,
		RedactLocation:     true,
		RedactAppUsage:     true,
		RedactCalendar:     true,
		MarkContextPrivate: true,
		MaxRetentionDays:   7,
		AllowedEventTypes: []string{
			"screen_state_changed",
			"device_unlocked",
			"battery_state_changed",
			"permission_state_changed",
		},
	}
}

// IsEventAllowed checks if an event type is allowed in sensitive mode.
func (sm *SensitiveMode) IsEventAllowed(eventType string) bool {
	if !sm.Active {
		return true
	}

	if len(sm.AllowedEventTypes) == 0 {
		return true
	}

	for _, allowed := range sm.AllowedEventTypes {
		if allowed == eventType {
			return true
		}
	}

	return false
}

// MarshalSensitiveMode serializes sensitive mode to JSON.
func MarshalSensitiveMode(sm SensitiveMode) string {
	b, _ := json.Marshal(sm)
	return string(b)
}

// UnmarshalSensitiveMode deserializes sensitive mode from JSON.
func UnmarshalSensitiveMode(data string) SensitiveMode {
	var sm SensitiveMode
	if data == "" {
		return DefaultSensitiveMode()
	}
	json.Unmarshal([]byte(data), &sm)
	return sm
}

// Activate enables sensitive mode with restrictive defaults.
func (sm *SensitiveMode) Activate() {
	sm.Active = true
	sm.PauseSync = true
	sm.RedactLocation = true
	sm.RedactAppUsage = true
	sm.RedactCalendar = true
	sm.MarkContextPrivate = true
	log.Println("[sensitive-mode] activated — sync paused, location/app/calendar redacted")
}

// Deactivate disables sensitive mode.
func (sm *SensitiveMode) Deactivate() {
	sm.Active = false
	sm.PauseSync = false
	sm.RedactLocation = false
	sm.RedactAppUsage = false
	sm.RedactCalendar = false
	sm.MarkContextPrivate = false
	log.Println("[sensitive-mode] deactivated — normal operation resumed")
}
