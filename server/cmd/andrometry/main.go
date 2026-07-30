package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/governor"
	"github.com/andrometry/server/internal/handlers"
	"github.com/andrometry/server/internal/middleware"
	"github.com/andrometry/server/internal/models"
	"github.com/google/uuid"
)

func main() {
	dbPath := flag.String("db", "andrometry.db", "SQLite database path")
	port := flag.Int("port", 8420, "HTTP server port")
	bootstrapToken := flag.String("bootstrap-token", "", "Bootstrap a device token on first run")
	flag.Parse()

	// Open database
	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	// Bootstrap token if requested
	if *bootstrapToken != "" {
		bootstrapDeviceToken(database, *bootstrapToken)
	}

	h := handlers.NewHandler(database)
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
	// NOTE: These endpoints use device:ingest scope. Agent tokens cannot rotate themselves.
	// Design: device rotates, agent reads. If agents need self-rotation, add separate routes.
	mux.Handle("POST /v1/auth/rotate", auth(middleware.RequireScope("device:ingest")(http.HandlerFunc(h.PostTokenRotation))))
	mux.Handle("POST /v1/auth/revoke", auth(middleware.RequireScope("device:ingest")(http.HandlerFunc(h.PostTokenRevoke))))

	// SSE stream for real-time context updates (§21)
	mux.Handle("GET /v1/stream", auth(middleware.RequireScope("agent:context:read")(http.HandlerFunc(h.StreamContext))))

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
	mux.Handle("GET /v1/agents/{agent_id}/effectiveness", auth(middleware.RequireScope("agent:interventions:write")(http.HandlerFunc(h.GetAgentEffectiveness))))

	// Privacy / Sensitive mode
	mux.Handle("GET /v1/privacy/sensitive-mode", auth(middleware.RequireScope("device:read_config")(http.HandlerFunc(h.GetSensitiveMode))))
	mux.Handle("POST /v1/privacy/sensitive-mode", auth(middleware.RequireScope("device:read_config")(http.HandlerFunc(h.SetSensitiveMode))))
	mux.Handle("POST /v1/privacy/wipe", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.WipeData))))

	// Apply global middleware
	handler := middleware.CORS(middleware.Logging(mux))

	// Enforce retention policy on startup
	if err := database.EnforceRetentionPolicy(db.DefaultRetentionPolicy()); err != nil {
		log.Printf("[retention] warning: failed to enforce policy: %v", err)
	}

	// Periodic retention enforcement every 24 hours
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := database.EnforceRetentionPolicy(db.DefaultRetentionPolicy()); err != nil {
				log.Printf("[retention] warning: periodic enforcement failed: %v", err)
			} else {
				log.Println("[retention] periodic enforcement complete")
			}
		}
	}()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[andrometry] relay server starting on %s", addr)
	log.Printf("[andrometry] database: %s", *dbPath)
	log.Printf("[andrometry] default policy: max_interventions=%d, cooldown=%dm, quiet_hours=%s-%s",
		governor.DefaultPolicy().MaxInterventionsPerDay,
		governor.DefaultPolicy().CooldownMinutes,
		governor.DefaultPolicy().QuietHoursStart,
		governor.DefaultPolicy().QuietHoursEnd,
	)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func bootstrapDeviceToken(database *db.DB, rawToken string) {
	tokenID := uuid.New().String()
	hash := db.HashToken(rawToken)

	token := &models.AuthToken{
		TokenID:   tokenID,
		UserID:    "primary",
		TokenHash: hash,
		TokenType: "device",
		Scopes:    "device:ingest,device:read_config,device:read_interventions,device:write_feedback",
	}

	if err := database.CreateToken(token); err != nil {
		log.Printf("[bootstrap] token may already exist: %v", err)
	} else {
		log.Printf("[bootstrap] device token created: %s (ID: %s)", rawToken[:8]+"...", tokenID)
	}

	// Also create an agent token
	agentTokenID := uuid.New().String()
	agentHash := db.HashToken("agent-" + rawToken)
	agentToken := &models.AuthToken{
		TokenID:   agentTokenID,
		UserID:    "primary",
		TokenHash: agentHash,
		TokenType: "agent",
		Scopes:    "agent:context:read,agent:briefing:read,agent:timeline:read,agent:events:read,agent:summaries:read,agent:observations:write,agent:interventions:write,agent:feedback:read,agent:goals:read,agent:goals:write,agent:habits:read,agent:habits:write,agent:outcomes:write",
	}

	if err := database.CreateToken(agentToken); err != nil {
		log.Printf("[bootstrap] agent token may already exist: %v", err)
	} else {
		log.Printf("[bootstrap] agent token created: agent-%s (ID: %s)", rawToken[:8]+"...", agentTokenID)
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
