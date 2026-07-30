package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// parseConfigFile reads a simple TOML config file and populates the Config struct.
// Supports [section] headers and key = "value" or key = value pairs.
func parseConfigFile(path string, cfg *Config) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[config] could not open %s: %v (using defaults)", path, err)
		return
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}

		// key = value
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		// Strip inline comments (unquoted #)
		inString := false
		for i, ch := range val {
			if (ch == '"' || ch == '\'') && (i == 0 || val[i-1] != '\\') {
				inString = !inString
			}
			if ch == '#' && !inString {
				val = strings.TrimSpace(val[:i])
				break
			}
		}

		// Strip quotes from string values
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}

		applyConfigValue(cfg, section, key, val)
	}

	log.Printf("[config] loaded config from %s", path)
}

// applyConfigValue maps a section.key to the Config struct.
func applyConfigValue(cfg *Config, section, key, val string) {
	switch section {
	case "server":
		switch key {
		case "address":
			cfg.Address = val
		case "database_path":
			cfg.DatabasePath = val
		}
	case "auth":
		switch key {
		case "bootstrap_token":
			cfg.BootstrapToken = val
		}
	case "sync":
		switch key {
		case "max_batch_events":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.MaxBatchEvents = v
			}
		case "max_batch_segments":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.MaxBatchSegments = v
			}
		case "max_batch_summaries":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.MaxBatchSummaries = v
			}
		}
	case "retention":
		switch key {
		case "raw_events_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.RawEventsDays = v
			}
		case "timeline_segments_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.TimelineSegmentsDays = v
			}
		case "daily_summaries_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.DailySummariesDays = v
			}
		case "agent_observations_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.AgentObservationsDays = v
			}
		case "interventions_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.InterventionsDays = v
			}
		case "audit_log_days":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Retention.AuditLogDays = v
			}
		}
	case "policy":
		switch key {
		case "max_interventions_per_day":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Policy.MaxInterventionsPerDay = v
			}
		case "cooldown_minutes":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.Policy.CooldownMinutes = v
			}
		case "quiet_hours_start":
			cfg.Policy.QuietHoursStart = val
		case "quiet_hours_end":
			cfg.Policy.QuietHoursEnd = val
		case "min_confidence_for_nudge":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.Policy.MinConfidenceForNudge = v
			}
		case "suppress_after_repeated_dismissals":
			cfg.Policy.SuppressAfterDismissals = val == "true"
		case "allow_high_severity_override":
			cfg.Policy.AllowHighSeverityOverride = val == "true"
		}
	case "streaming":
		switch key {
		case "sse_enabled":
			cfg.SSEEnabled = val == "true"
		case "sse_poll_interval_seconds":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.SSEPollSeconds = v
			}
		}
	}
}
