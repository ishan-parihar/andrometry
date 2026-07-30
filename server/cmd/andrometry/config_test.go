package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestParseConfigFile_AllSections(t *testing.T) {
	cfg := DefaultConfig()
	path := writeTempConfig(t, `
# Server settings
[server]
address = "127.0.0.1:9000"
database_path = "/data/test.db"

# Auth settings
[auth]
bootstrap_token = "my-secret-token"

# Sync settings
[sync]
max_batch_events = 1000
max_batch_segments = 200
max_batch_summaries = 100

# Retention settings
[retention]
raw_events_days = 14
timeline_segments_days = 180
daily_summaries_days = 365
agent_observations_days = 30
interventions_days = 60
audit_log_days = 90

# Policy settings
[policy]
max_interventions_per_day = 3
cooldown_minutes = 15
quiet_hours_start = "23:00"
quiet_hours_end = "07:00"
min_confidence_for_nudge = 0.8
suppress_after_repeated_dismissals = true
allow_high_severity_override = false

# Streaming settings
[streaming]
sse_enabled = false
sse_poll_interval_seconds = 10
`)

	parseConfigFile(path, &cfg)

	// Server
	if cfg.Address != "127.0.0.1:9000" {
		t.Errorf("Address = %q, want %q", cfg.Address, "127.0.0.1:9000")
	}
	if cfg.DatabasePath != "/data/test.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/data/test.db")
	}

	// Auth
	if cfg.BootstrapToken != "my-secret-token" {
		t.Errorf("BootstrapToken = %q, want %q", cfg.BootstrapToken, "my-secret-token")
	}

	// Sync
	if cfg.MaxBatchEvents != 1000 {
		t.Errorf("MaxBatchEvents = %d, want 1000", cfg.MaxBatchEvents)
	}
	if cfg.MaxBatchSegments != 200 {
		t.Errorf("MaxBatchSegments = %d, want 200", cfg.MaxBatchSegments)
	}
	if cfg.MaxBatchSummaries != 100 {
		t.Errorf("MaxBatchSummaries = %d, want 100", cfg.MaxBatchSummaries)
	}

	// Retention
	if cfg.Retention.RawEventsDays != 14 {
		t.Errorf("RawEventsDays = %d, want 14", cfg.Retention.RawEventsDays)
	}
	if cfg.Retention.TimelineSegmentsDays != 180 {
		t.Errorf("TimelineSegmentsDays = %d, want 180", cfg.Retention.TimelineSegmentsDays)
	}
	if cfg.Retention.DailySummariesDays != 365 {
		t.Errorf("DailySummariesDays = %d, want 365", cfg.Retention.DailySummariesDays)
	}
	if cfg.Retention.AgentObservationsDays != 30 {
		t.Errorf("AgentObservationsDays = %d, want 30", cfg.Retention.AgentObservationsDays)
	}
	if cfg.Retention.InterventionsDays != 60 {
		t.Errorf("InterventionsDays = %d, want 60", cfg.Retention.InterventionsDays)
	}
	if cfg.Retention.AuditLogDays != 90 {
		t.Errorf("AuditLogDays = %d, want 90", cfg.Retention.AuditLogDays)
	}

	// Policy
	if cfg.Policy.MaxInterventionsPerDay != 3 {
		t.Errorf("MaxInterventionsPerDay = %d, want 3", cfg.Policy.MaxInterventionsPerDay)
	}
	if cfg.Policy.CooldownMinutes != 15 {
		t.Errorf("CooldownMinutes = %d, want 15", cfg.Policy.CooldownMinutes)
	}
	if cfg.Policy.QuietHoursStart != "23:00" {
		t.Errorf("QuietHoursStart = %q, want %q", cfg.Policy.QuietHoursStart, "23:00")
	}
	if cfg.Policy.QuietHoursEnd != "07:00" {
		t.Errorf("QuietHoursEnd = %q, want %q", cfg.Policy.QuietHoursEnd, "07:00")
	}
	if cfg.Policy.MinConfidenceForNudge != 0.8 {
		t.Errorf("MinConfidenceForNudge = %f, want 0.8", cfg.Policy.MinConfidenceForNudge)
	}
	if !cfg.Policy.SuppressAfterDismissals {
		t.Error("SuppressAfterDismissals = false, want true")
	}
	if cfg.Policy.AllowHighSeverityOverride {
		t.Error("AllowHighSeverityOverride = true, want false")
	}

	// Streaming
	if cfg.SSEEnabled {
		t.Error("SSEEnabled = true, want false")
	}
	if cfg.SSEPollSeconds != 10 {
		t.Errorf("SSEPollSeconds = %d, want 10", cfg.SSEPollSeconds)
	}
}

func TestParseConfigFile_InlineComments(t *testing.T) {
	cfg := DefaultConfig()
	path := writeTempConfig(t, `
[server]
address = "127.0.0.1:8420" # default address
database_path = "andrometry.db" # local db

[policy]
quiet_hours_start = "22:30" # evening
quiet_hours_end = "07:00" # morning
max_interventions_per_day = 5 # per day
`)

	parseConfigFile(path, &cfg)

	if cfg.Address != "127.0.0.1:8420" {
		t.Errorf("Address = %q (inline comment not stripped)", cfg.Address)
	}
	if cfg.DatabasePath != "andrometry.db" {
		t.Errorf("DatabasePath = %q (inline comment not stripped)", cfg.DatabasePath)
	}
	if cfg.Policy.QuietHoursStart != "22:30" {
		t.Errorf("QuietHoursStart = %q (inline comment not stripped)", cfg.Policy.QuietHoursStart)
	}
	if cfg.Policy.QuietHoursEnd != "07:00" {
		t.Errorf("QuietHoursEnd = %q (inline comment not stripped)", cfg.Policy.QuietHoursEnd)
	}
	if cfg.Policy.MaxInterventionsPerDay != 5 {
		t.Errorf("MaxInterventionsPerDay = %d, want 5", cfg.Policy.MaxInterventionsPerDay)
	}
}

func TestParseConfigFile_MissingFile(t *testing.T) {
	cfg := DefaultConfig()
	// Should not panic — just log and use defaults
	parseConfigFile("/nonexistent/config.toml", &cfg)

	if cfg.Address != "0.0.0.0:8420" {
		t.Errorf("Address changed on missing file: %q", cfg.Address)
	}
}

func TestParseConfigFile_EmptyFile(t *testing.T) {
	cfg := DefaultConfig()
	path := writeTempConfig(t, "")
	parseConfigFile(path, &cfg)

	if cfg.Address != "0.0.0.0:8420" {
		t.Errorf("Address changed on empty file: %q", cfg.Address)
	}
}

func TestParseConfigFile_InvalidLines(t *testing.T) {
	cfg := DefaultConfig()
	path := writeTempConfig(t, `
this is not valid
[key without equals
[server]
address = "127.0.0.1:9000"
`)
	parseConfigFile(path, &cfg)

	if cfg.Address != "127.0.0.1:9000" {
		t.Errorf("Address = %q, want valid value despite invalid lines", cfg.Address)
	}
}

func TestParseConfigFile_UnquotedValues(t *testing.T) {
	cfg := DefaultConfig()
	path := writeTempConfig(t, `
[server]
address = 0.0.0.0:8420

[policy]
quiet_hours_start = 22:30
quiet_hours_end = 07:00
suppress_after_repeated_dismissals = true
`)
	parseConfigFile(path, &cfg)

	if cfg.Address != "0.0.0.0:8420" {
		t.Errorf("Address = %q", cfg.Address)
	}
	if cfg.Policy.QuietHoursStart != "22:30" {
		t.Errorf("QuietHoursStart = %q", cfg.Policy.QuietHoursStart)
	}
	if !cfg.Policy.SuppressAfterDismissals {
		t.Error("SuppressAfterDismissals = false, want true")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Address != "0.0.0.0:8420" {
		t.Errorf("default Address = %q", cfg.Address)
	}
	if cfg.DatabasePath != "andrometry.db" {
		t.Errorf("default DatabasePath = %q", cfg.DatabasePath)
	}
	if cfg.MaxBatchEvents != 500 {
		t.Errorf("default MaxBatchEvents = %d", cfg.MaxBatchEvents)
	}
	if !cfg.SSEEnabled {
		t.Error("default SSEEnabled = false")
	}
	if cfg.SSEPollSeconds != 5 {
		t.Errorf("default SSEPollSeconds = %d", cfg.SSEPollSeconds)
	}
}
