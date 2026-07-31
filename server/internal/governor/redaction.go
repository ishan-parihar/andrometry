package governor

import (
	"strings"
)

// RedactContext applies sensitive mode redaction rules to a context map.
// This is the server-side enforcement of §22.3 redaction rules.
// The Android app should also redact before sync, but the server must not leak sensitive data to agents.
func RedactContext(contextData map[string]interface{}, sm SensitiveMode) map[string]interface{} {
	if !sm.Active {
		return contextData
	}

	result := make(map[string]interface{})
	for k, v := range contextData {
		result[k] = v
	}

	// Redact location data
	if sm.RedactLocation {
		redactLocation(result)
	}

	// Redact app usage
	if sm.RedactAppUsage {
		redactAppUsage(result)
	}

	// Redact calendar
	if sm.RedactCalendar {
		redactCalendar(result)
	}

	// Mark context as private if configured
	if sm.MarkContextPrivate {
		result["sensitive_mode"] = true
		result["private"] = true
	}

	return result
}

// redactLocation removes or replaces exact coordinates with place labels.
func redactLocation(data map[string]interface{}) {
	// Remove exact coordinates
	delete(data, "latitude")
	delete(data, "longitude")
	delete(data, "lat")
	delete(data, "lng")
	delete(data, "coordinates")
	delete(data, "location_coords")

	// Remove raw location data
	delete(data, "raw_location")
	delete(data, "gps_data")

	// Replace semantic location with generic labels if present
	if loc, ok := data["location"].(map[string]interface{}); ok {
		delete(loc, "latitude")
		delete(loc, "longitude")
		delete(loc, "accuracy")
	}

	// Remove location from nested context objects
	if activity, ok := data["activity"].(map[string]interface{}); ok {
		delete(activity, "location")
		delete(activity, "location_coords")
	}
}

// redactAppUsage removes app-specific sensitive content.
func redactAppUsage(data map[string]interface{}) {
	// Remove app-specific data
	delete(data, "app_package")
	delete(data, "package_name")
	delete(data, "app_name")

	// Remove notification content
	delete(data, "notification_text")
	delete(data, "notification_title")
	delete(data, "notification_content")
	delete(data, "notification_big_text")

	// Remove message bodies
	delete(data, "message_body")
	delete(data, "sms_body")
	delete(data, "call_log")

	// Remove sender identities
	delete(data, "sender")
	delete(data, "sender_name")
	delete(data, "contact_name")
	delete(data, "phone_number")

	// Remove free-text notes
	delete(data, "notes")
	delete(data, "free_text")
	delete(data, "text_content")

	// Redact app usage in context
	if usage, ok := data["app_usage"].(map[string]interface{}); ok {
		delete(usage, "package_name")
		delete(usage, "app_name")
		delete(usage, "foreground_package")
	}

	// Redact current app
	if app, ok := data["current_app"].(map[string]interface{}); ok {
		delete(app, "package_name")
		delete(app, "app_name")
	}
}

// redactCalendar removes calendar titles if configured.
func redactCalendar(data map[string]interface{}) {
	// Remove calendar event details
	delete(data, "calendar_title")
	delete(data, "event_title")
	delete(data, "event_description")
	delete(data, "event_location")
	delete(data, "event_attendees")

	// Remove from nested context
	if cal, ok := data["calendar"].(map[string]interface{}); ok {
		delete(cal, "title")
		delete(cal, "event_title")
		delete(cal, "description")
		delete(cal, "location")
		delete(cal, "attendees")
	}

	// Remove calendar info
	if info, ok := data["calendar_info"].(map[string]interface{}); ok {
		delete(info, "current_event_title")
		delete(info, "event_title")
	}
}

// locationLabels is a set of labels that represent locations (immutable after init).
var locationLabels = map[string]bool{
	"home": true, "office": true, "gym": true, "work": true, "school": true,
	"commute": true, "coffee_shop": true, "restaurant": true, "park": true,
	"store": true, "mall": true, "hospital": true, "airport": true,
	"hotel": true, "library": true,
}

// appLabels is a set of labels that represent apps (immutable after init).
var appLabels = map[string]bool{
	"social_media": true, "messaging": true, "email": true, "browser": true,
	"video": true, "music": true, "gaming": true, "shopping": true,
	"news": true, "finance": true, "fitness": true, "weather": true,
	"maps": true, "camera": true, "photo": true,
}

// IsLocationLabel checks if a label represents a location.
func IsLocationLabel(label string) bool {
	return locationLabels[strings.ToLower(label)]
}

// IsAppLabel checks if a label represents an app.
func IsAppLabel(label string) bool {
	return appLabels[strings.ToLower(label)]
}


