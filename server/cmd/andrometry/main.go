package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/governor"
	"github.com/andrometry/server/internal/handlers"
	"github.com/andrometry/server/internal/middleware"
	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

// Config holds server configuration loaded from config.toml or defaults.
type Config struct {
	Address          string
	DatabasePath     string
	BootstrapToken   string
	MaxBatchEvents   int
	MaxBatchSegments int
	MaxBatchSummaries int
	Retention        db.RetentionPolicy
	Policy           governor.Policy
	SSEEnabled       bool
	SSEPollSeconds   int
}

// DefaultConfig returns production-ready defaults matching config.toml.
func DefaultConfig() Config {
	return Config{
		Address:           "0.0.0.0:8420",
		DatabasePath:      "andrometry.db",
		MaxBatchEvents:    500,
		MaxBatchSegments:  100,
		MaxBatchSummaries: 50,
		Retention:         db.DefaultRetentionPolicy(),
		Policy:            governor.DefaultPolicy(),
		SSEEnabled:        true,
		SSEPollSeconds:    5,
	}
}

func main() {
	// CLI flags override config defaults
	configPath := flag.String("config", "", "Path to config.toml (optional)")
	dbPath := flag.String("db", "", "SQLite database path (overrides config)")
	port := flag.Int("port", 0, "HTTP server port (overrides config)")
	bootstrapToken := flag.String("bootstrap-token", "", "Bootstrap a device token on first run")
	flag.Parse()

	// Load config (config.toml support — if file exists, parse it; otherwise use defaults)
	cfg := DefaultConfig()

	// CLI flags take precedence over config
	if *dbPath != "" {
		cfg.DatabasePath = *dbPath
	}
	if *port > 0 {
		cfg.Address = fmt.Sprintf(":%d", *port)
	}
	if *bootstrapToken != "" {
		cfg.BootstrapToken = *bootstrapToken
	}

	// Parse config.toml if provided, or look for default locations
	configFile := *configPath
	if configFile == "" {
		if _, err := os.Stat("config.toml"); err == nil {
			configFile = "config.toml"
		}
	}
	if configFile != "" {
		parseConfigFile(configFile, &cfg)
	}

	// Open database
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	// Bootstrap token if requested
	if cfg.BootstrapToken != "" {
		bootstrapDeviceToken(database, cfg.BootstrapToken)
	}

	h := handlers.NewHandler(database)
	h.SetBatchLimits(cfg.MaxBatchEvents, cfg.MaxBatchSegments, cfg.MaxBatchSummaries)
	auth := middleware.Auth(database)

	// Router
	mux := http.NewServeMux()

	// Health (no auth)
	mux.HandleFunc("GET /v1/health", h.HealthCheck)
	mux.HandleFunc("GET /v1/health/detailed", h.HealthCheckDetailed)

	// Device ingestion (device:ingest scope)
	mux.Handle("POST /v1/batch", auth(middleware.RequireScope("device:ingest")(http.HandlerFunc(h.BatchIngestion))))

	// Device: read pending interventions
	mux.Handle("GET /v1/device/interventions", auth(middleware.RequireScope("device:read_interventions")(http.HandlerFunc(h.GetPendingInterventions))))

	// Context
	mux.Handle("GET /v1/context/current", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.GetCurrentContext))))

	// Briefing
	mux.Handle("GET /v1/briefing/current", auth(middleware.RequireScope("agent:briefing:read")(http.HandlerFunc(h.GetCurrentBriefing))))

	// Timeline
	mux.Handle("GET /v1/timeline", auth(middleware.RequireScope("agent:timeline:read")(http.HandlerFunc(h.GetTimeline))))

	// Events (drill-down)
	mux.Handle("GET /v1/events", auth(middleware.RequireScope("agent:events:read")(http.HandlerFunc(h.GetEvents))))

	// Summaries
	mux.Handle("GET /v1/summary/daily", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetDailySummary))))
	mux.Handle("GET /v1/summary/weekly", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetWeeklySummary))))

	// Agent observations
	mux.Handle("POST /v1/agent/observations", auth(middleware.RequireScope("agent:observations:write")(http.HandlerFunc(h.PostObservation))))
	mux.Handle("GET /v1/agent/observations", auth(middleware.RequireScope("agent:observations:write")(http.HandlerFunc(h.GetObservations))))

	// Interventions
	mux.Handle("POST /v1/agent/interventions", auth(middleware.RequireScope("agent:interventions:write")(http.HandlerFunc(h.PostIntervention))))
	mux.Handle("PATCH /v1/agent/interventions/{id}", auth(middleware.RequireScope("agent:interventions:write")(http.HandlerFunc(h.UpdateInterventionStatus))))

	// Feedback
	mux.Handle("POST /v1/feedback", auth(middleware.RequireScope("agent:feedback:read")(http.HandlerFunc(h.PostFeedback))))
	mux.Handle("GET /v1/feedback", auth(middleware.RequireScope("agent:feedback:read")(http.HandlerFunc(h.GetFeedback))))

	// Goals
	mux.Handle("GET /v1/goals", auth(middleware.RequireScope("agent:goals:read")(http.HandlerFunc(h.GetGoals))))
	mux.Handle("POST /v1/goals", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.PostGoal))))
	mux.Handle("PATCH /v1/goals/{id}", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.UpdateGoal))))
	mux.Handle("DELETE /v1/goals/{id}", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.DeleteGoal))))

	// Habits
	mux.Handle("GET /v1/habits", auth(middleware.RequireScope("agent:habits:read")(http.HandlerFunc(h.GetHabits))))
	mux.Handle("POST /v1/habits", auth(middleware.RequireScope("agent:habits:write")(http.HandlerFunc(h.PostHabit))))
	mux.Handle("PATCH /v1/habits/{id}", auth(middleware.RequireScope("agent:habits:write")(http.HandlerFunc(h.UpdateHabit))))
	mux.Handle("DELETE /v1/habits/{id}", auth(middleware.RequireScope("agent:habits:write")(http.HandlerFunc(h.DeleteHabit))))
	mux.Handle("POST /v1/habits/{id}/streak", auth(middleware.RequireScope("agent:habits:write")(http.HandlerFunc(h.UpdateHabitStreak))))

	// Outcomes
	mux.Handle("POST /v1/outcomes", auth(middleware.RequireScope("agent:outcomes:write")(http.HandlerFunc(h.PostOutcome))))
	mux.Handle("GET /v1/outcomes", auth(middleware.RequireScope("agent:feedback:read")(http.HandlerFunc(h.GetOutcomes))))

	// Audit log
	mux.Handle("GET /v1/audit", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.GetAuditLog))))

	// Five-minute and hourly summaries
	mux.Handle("GET /v1/summary/five_minute", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetFiveMinuteSummaries))))
	mux.Handle("GET /v1/summary/hourly", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetHourlySummaries))))

	// Baselines
	mux.Handle("GET /v1/baselines", auth(middleware.RequireScope("agent:goals:read")(http.HandlerFunc(h.GetBaselines))))
	mux.Handle("POST /v1/baselines", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.PostBaseline))))
	mux.Handle("DELETE /v1/baselines/{id}", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.DeleteBaseline))))

	// User corrections
	mux.Handle("GET /v1/corrections", auth(middleware.RequireScope("device:write_feedback")(http.HandlerFunc(h.GetUserCorrections))))
	mux.Handle("DELETE /v1/corrections/{id}", auth(middleware.RequireScope("device:write_feedback")(http.HandlerFunc(h.DeleteUserCorrection))))
	mux.Handle("POST /v1/corrections", auth(middleware.RequireScope("device:write_feedback")(http.HandlerFunc(h.PostUserCorrection))))

	// Token rotation and revocation (§28)
	mux.Handle("POST /v1/auth/rotate", auth(middleware.RequireScope("device:ingest")(http.HandlerFunc(h.PostTokenRotation))))
	mux.Handle("POST /v1/auth/revoke", auth(middleware.RequireScope("device:ingest")(http.HandlerFunc(h.PostTokenRevoke))))

	// SSE stream for real-time context updates (§21)
	if cfg.SSEEnabled {
		h.SetSSEPollInterval(time.Duration(cfg.SSEPollSeconds) * time.Second)
		mux.Handle("GET /v1/stream", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.StreamContext))))
	}

	// Work Pattern Metrics (§39)
	mux.Handle("GET /v1/metrics/work-patterns", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetWorkPatternMetrics))))

	// Readiness & Fatigue (§40)
	mux.Handle("GET /v1/metrics/readiness", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.GetReadinessScore))))

	// Collector Health (§23.5)
	mux.Handle("GET /v1/health/collectors", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.GetCollectorStatus))))

	// Goal Progress (§36)
	mux.Handle("GET /v1/goals/{id}/progress", auth(middleware.RequireScope("agent:goals:read")(http.HandlerFunc(h.EvaluateGoalProgress))))

	// Habit Evaluation (§38)
	mux.Handle("GET /v1/habits/{id}/evaluation", auth(middleware.RequireScope("agent:habits:read")(http.HandlerFunc(h.EvaluateHabitCompletion))))

	// Intervention Effectiveness (§35)
	mux.Handle("GET /v1/interventions/{id}/effectiveness", auth(middleware.RequireScope("agent:interventions:write")(http.HandlerFunc(h.GetInterventionEffectiveness))))

	// Analytics Insights (§20)
	mux.Handle("GET /v1/analytics/insights", auth(middleware.RequireScope("agent:summaries:read")(http.HandlerFunc(h.GetAnalyticsInsights))))

	// Data Export (§27)
	mux.Handle("GET /v1/export/daily", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.ExportDaily))))
	mux.Handle("GET /v1/export/weekly", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.ExportWeekly))))

	// Data Quality (§11.4)
	mux.Handle("GET /v1/context/data-quality", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.GetDataQuality))))

	// Agent Effectiveness (§35)
	mux.Handle("GET /v1/agents/{agent_id}/effectiveness", auth(middleware.RequireScope("agent:interventions:read")(http.HandlerFunc(h.GetAgentEffectiveness))))

	// Privacy / Sensitive mode
	mux.Handle("GET /v1/privacy/sensitive-mode", auth(middleware.RequireScope("device:read_config")(http.HandlerFunc(h.GetSensitiveMode))))
	mux.Handle("POST /v1/privacy/sensitive-mode", auth(middleware.RequireScope("device:read_config")(http.HandlerFunc(h.SetSensitiveMode))))
	mux.Handle("POST /v1/privacy/wipe", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.WipeData))))

	// Apply global middleware
	handler := middleware.CORS(middleware.Logging(mux))

	// Enforce retention policy on startup
	if err := database.EnforceRetentionPolicy(cfg.Retention); err != nil {
		log.Printf("[retention] warning: failed to enforce policy: %v", err)
	}

	// Periodic retention enforcement every 24 hours
	retentionDone := make(chan struct{})
	go func() {
		defer close(retentionDone)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := database.EnforceRetentionPolicy(cfg.Retention); err != nil {
				log.Printf("[retention] warning: periodic enforcement failed: %v", err)
			} else {
				log.Println("[retention] periodic enforcement complete")
			}
		}
	}()

	// Graceful shutdown (§31)
	addr := cfg.Address
	log.Printf("[andrometry] relay server starting on %s", addr)
	log.Printf("[andrometry] database: %s", cfg.DatabasePath)
	log.Printf("[andrometry] batch limits: events=%d, segments=%d, summaries=%d",
		cfg.MaxBatchEvents, cfg.MaxBatchSegments, cfg.MaxBatchSummaries)
	log.Printf("[andrometry] policy: max_interventions=%d, cooldown=%dm, quiet_hours=%s-%s",
		cfg.Policy.MaxInterventionsPerDay,
		cfg.Policy.CooldownMinutes,
		cfg.Policy.QuietHoursStart,
		cfg.Policy.QuietHoursEnd,
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[andrometry] received signal %v, shutting down gracefully...", sig)

	// Shutdown with 30s deadline
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[andrometry] forced shutdown: %v", err)
	}

	log.Println("[andrometry] server stopped")
}

func bootstrapDeviceToken(database *db.DB, rawToken string) {
	hash := db.HashToken(rawToken)

	// Check if token already exists — skip if so
	if existing, err := database.ValidateToken(hash); err == nil && existing != nil {
		log.Printf("[bootstrap] device token already exists (ID: %s), skipping", existing.TokenID)
		fmt.Println("=== ANDROMETRY BOOTSTRAP SKIPPED ===")
		fmt.Println("Token already exists in database. No action taken.")
		fmt.Println("====================================")
		return
	}

	tokenID := uuid.New().String()

	token := &models.AuthToken{
		TokenID:   tokenID,
		UserID:    "primary",
		TokenHash: hash,
		TokenType: "device",
		Scopes:    "device:ingest,device:read_config,device:read_interventions,device:write_feedback",
	}

	if err := database.CreateToken(token); err != nil {
		log.Printf("[bootstrap] failed to create device token: %v", err)
		return
	}
	log.Printf("[bootstrap] device token created: %s (ID: %s)", rawToken[:8]+"...", tokenID)

	// Also create an agent token
	agentTokenID := uuid.New().String()
	agentHash := db.HashToken("agent-" + rawToken)
	if existing, err := database.ValidateToken(agentHash); err == nil && existing != nil {
		log.Printf("[bootstrap] agent token already exists (ID: %s), skipping", existing.TokenID)
	} else {
		agentToken := &models.AuthToken{
			TokenID:   agentTokenID,
			UserID:    "primary",
			TokenHash: agentHash,
			TokenType: "agent",
			Scopes:    "agent:context:read,agent:briefing:read,agent:timeline:read,agent:events:read,agent:summaries:read,agent:observations:write,agent:interventions:read,agent:interventions:write,agent:feedback:read,agent:goals:read,agent:goals:write,agent:habits:read,agent:habits:write,agent:outcomes:write",
		}
		if err := database.CreateToken(agentToken); err != nil {
			log.Printf("[bootstrap] failed to create agent token: %v", err)
		} else {
			log.Printf("[bootstrap] agent token created: agent-%s (ID: %s)", rawToken[:8]+"...", agentTokenID)
		}
	}

	// Print instructions
	fmt.Println("=== ANDROMETRY BOOTSTRAP COMPLETE ===")
	fmt.Printf("Device token: %s\n", rawToken)
	fmt.Printf("Agent token:  agent-%s\n", rawToken)
	fmt.Println("\nAdd to your environment:")
	fmt.Printf("  ANDROMETRY_DEVICE_TOKEN=%s\n", rawToken)
	fmt.Printf("  ANDROMETRY_AGENT_TOKEN=agent-%s\n", rawToken)
	fmt.Println("====================================")
}
