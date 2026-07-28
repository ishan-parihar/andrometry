package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"


	"github.com/andrometry/server/internal/audit"
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
	mux.Handle("PATCH /v1/agent/interventions/", auth(middleware.RequireScope("agent:interventions:write")(http.HandlerFunc(h.UpdateInterventionStatus))))

	// Feedback
	mux.Handle("POST /v1/feedback", auth(middleware.RequireScope("agent:feedback:read")(http.HandlerFunc(h.PostFeedback))))

	// Goals
	mux.Handle("GET /v1/goals", auth(middleware.RequireScope("agent:goals:read")(http.HandlerFunc(h.GetGoals))))
	mux.Handle("POST /v1/goals", auth(middleware.RequireScope("agent:goals:write")(http.HandlerFunc(h.PostGoal))))

	// Habits
	mux.Handle("GET /v1/habits", auth(middleware.RequireScope("agent:habits:read")(http.HandlerFunc(h.GetHabits))))
	mux.Handle("POST /v1/habits", auth(middleware.RequireScope("agent:habits:write")(http.HandlerFunc(h.PostHabit))))

	// Outcomes
	mux.Handle("POST /v1/outcomes", auth(middleware.RequireScope("agent:outcomes:write")(http.HandlerFunc(h.PostOutcome))))
	mux.Handle("GET /v1/outcomes", auth(middleware.RequireScope("agent:feedback:read")(http.HandlerFunc(h.GetOutcomes))))

	// Audit log
	mux.Handle("GET /v1/audit", auth(middleware.RequireScope("admin")(http.HandlerFunc(h.GetAuditLog))))

	// Apply global middleware
	handler := middleware.CORS(middleware.Logging(mux))

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
	fmt.Println("\n=== ANDROMETRY BOOTSTRAP COMPLETE ===")
	fmt.Printf("Device token: %s\n", rawToken)
	fmt.Printf("Agent token:  agent-%s\n", rawToken)
	fmt.Println("\nAdd to your environment:")
	fmt.Printf("  ANDROMETRY_DEVICE_TOKEN=%s\n", rawToken)
	fmt.Printf("  ANDROMETRY_AGENT_TOKEN=agent-%s\n", rawToken)
	fmt.Println("====================================\n")
}
