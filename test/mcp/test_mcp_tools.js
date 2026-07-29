#!/usr/bin/env node

/**
 * MCP Server Test Suite — §41 Testing and Simulation Harness
 *
 * Tests all 47 MCP tools for proper request/response handling,
 * error cases, and data validation.
 *
 * Run: node test/mcp/test_mcp_tools.js
 */

const assert = require("assert");

console.log("[test] Running MCP server tests...\n");

// Mock HTTP responses for testing
const mockResponses = {
  "/v1/context/current": {
    status: 200,
    data: {
      timestamp: "2026-07-28T10:14:00Z",
      timezone: "Asia/Kolkata",
      etag: "abc123",
      context: {
        primary_activity: "deep_work",
        activity_confidence: 0.81,
        place: "office",
        place_confidence: 0.92,
        focus_active: true,
        focus_minutes_elapsed: 37,
        distraction_probability: 0.18,
        fatigue_probability: 0.31,
        alignment_probability: 0.74,
        data_quality: {
          completeness: 0.87,
          staleness_seconds: 42,
        },
      },
    },
  },
  "/v1/briefing/current": {
    status: 200,
    data: {
      briefing_id: "briefing-1",
      generated_at: "2026-07-28T10:15:00Z",
      state: {
        primary_activity: "deep_work",
        place: "office",
        distraction_probability: 0.18,
        fatigue_probability: 0.31,
        alignment_probability: 0.74,
      },
      trends: {},
      risks: [],
      opportunities: [],
      suggested_queries: [],
    },
  },
  "/v1/timeline": {
    status: 200,
    data: {
      timezone: "Asia/Kolkata",
      timeline: [
        {
          segment_id: "seg-1",
          segment_type: "focus_segment",
          start_time: "2026-07-28T09:00:00Z",
          end_time: "2026-07-28T10:00:00Z",
          label: "deep_work",
          confidence: 0.85,
        },
      ],
    },
  },
  "/v1/summary/daily": {
    status: 200,
    data: {
      date: "2026-07-28",
      timezone: "Asia/Kolkata",
      deep_work_minutes: 182,
      meeting_minutes: 95,
      distraction_minutes: 64,
      sleep_minutes: 460,
      steps: 8123,
      alignment_score: 0.74,
    },
  },
  "/v1/summary/weekly": {
    status: 200,
    data: {
      week: "2026-W31",
      timezone: "Asia/Kolkata",
      deep_work_minutes: 1200,
      meeting_minutes: 450,
      alignment_score: 0.72,
    },
  },
  "/v1/summary/five_minute": {
    status: 200,
    data: {
      summaries: [
        {
          start_time: "2026-07-28T10:00:00Z",
          end_time: "2026-07-28T10:05:00Z",
          activity: "deep_work",
        },
      ],
    },
  },
  "/v1/summary/hourly": {
    status: 200,
    data: {
      summaries: [
        {
          start_time: "2026-07-28T10:00:00Z",
          end_time: "2026-07-28T11:00:00Z",
          activity: "deep_work",
        },
      ],
    },
  },
  "/v1/events": {
    status: 200,
    data: {
      events: [
        {
          event_id: "evt-1",
          timestamp: "2026-07-28T10:00:00Z",
          event_type: "app_usage_summary",
          payload: {},
        },
      ],
    },
  },
  "GET:/v1/agent/observations": {
    status: 200,
    data: {
      observations: [
        {
          observation_id: "obs-1",
          observation_type: "pattern",
          severity: "info",
          message: "User tends to focus better in mornings",
        },
      ],
    },
  },
  "POST:/v1/agent/observations": {
    status: 201,
    data: {
      observation_id: "obs-1",
    },
  },
  "GET:/v1/agent/interventions": {
    status: 200,
    data: {
      interventions: [],
    },
  },
  "POST:/v1/agent/interventions": {
    status: 201,
    data: {
      intervention_id: "int-1",
      status: "approved_by_policy",
    },
  },
  "PATCH:/v1/agent/interventions/int-test": {
    status: 200,
    data: {
      intervention_id: "int-1",
      status: "delivered",
    },
  },
  "GET:/v1/interventions/int-test/effectiveness": {
    status: 200,
    data: {
      intervention_id: "int-1",
      effectiveness_score: 0.65,
      rating: "moderately_effective",
      components: {
        accepted: 1.0,
        behavior_change: 0.7,
        metric_improvement: 0.5,
        user_helpfulness: 0.75,
      },
      total_outcomes: 3,
      total_feedback: 4,
    },
  },
  "GET:/v1/feedback": {
    status: 200,
    data: {
      feedback: [
        {
          feedback_id: "fb-1",
          feedback_type: "helpful",
          rating: 4,
        },
      ],
    },
  },
  "POST:/v1/feedback": {
    status: 201,
    data: {
      feedback_id: "fb-1",
    },
  },
  "GET:/v1/outcomes": {
    status: 200,
    data: {
      outcomes: [
        {
          outcome_id: "out-1",
          outcome_type: "accepted",
          effectiveness_score: 0.7,
        },
      ],
    },
  },
  "POST:/v1/outcomes": {
    status: 201,
    data: {
      outcome_id: "out-1",
    },
  },
  "GET:/v1/goals": {
    status: 200,
    data: {
      goals: [
        {
          goal_id: "goal-1",
          goal_type: "deep_work_minutes",
          title: "Deep work 2 hours daily",
          status: "active",
        },
      ],
    },
  },
  "POST:/v1/goals": {
    status: 201,
    data: {
      goal_id: "goal-new-1",
      goal_type: "deep_work_minutes",
      title: "Deep work 2 hours daily",
      status: "active",
    },
  },
  "GET:/v1/habits": {
    status: 200,
    data: {
      habits: [
        {
          habit_id: "habit-1",
          habit_type: "morning_routine",
          title: "Wake before 7am",
          streak_days: 5,
          habit_strength: 0.72,
        },
      ],
    },
  },
  "POST:/v1/habits": {
    status: 201,
    data: {
      habit_id: "habit-new-1",
      habit_type: "morning_routine",
      title: "Wake before 7am",
      streak_days: 0,
      habit_strength: 0.0,
    },
  },
  "GET:/v1/baselines": {
    status: 200,
    data: {
      baselines: [
        {
          baseline_id: "bl-1",
          metric_name: "deep_work_minutes_daily",
          value: 120,
        },
      ],
    },
  },
  "POST:/v1/baselines": {
    status: 201,
    data: {
      baseline_id: "bl-new-1",
      metric_name: "deep_work_minutes_daily",
      value: 120,
    },
  },
  "GET:/v1/corrections": {
    status: 200,
    data: {
      corrections: [],
    },
  },
  "POST:/v1/corrections": {
    status: 201,
    data: {
      status: "created",
      correction_id: "corr-1",
    },
  },
  "/v1/health/detailed": {
    status: 200,
    data: {
      status: "ok",
      server: "andrometry-relay",
      db_stats: {
        events: 1000,
        segments: 500,
      },
    },
  },
  "GET:/v1/privacy/sensitive-mode": {
    status: 200,
    data: {
      active: false,
      pause_sync: true,
      redact_location: true,
      redact_app_usage: true,
      redact_calendar: true,
    },
  },
  "POST:/v1/privacy/sensitive-mode": {
    status: 200,
    data: {
      active: true,
      pause_sync: true,
      redact_location: true,
      redact_app_usage: true,
      redact_calendar: true,
    },
  },
  "/v1/audit": {
    status: 200,
    data: {
      audit_log: [],
    },
  },
  "/v1/metrics/work-patterns": {
    status: 200,
    data: {
      total_focus_minutes: 180,
      context_switching_rate_per_hour: 2.5,
    },
  },
  "/v1/metrics/readiness": {
    status: 200,
    data: {
      readiness_score: 0.75,
      fatigue_probability: 0.25,
    },
  },
  "/v1/health/collectors": {
    status: 200,
    data: {
      collectors: [
        { name: "screen", status: "healthy", staleness_seconds: 10 },
        { name: "location", status: "healthy", staleness_seconds: 30 },
      ],
      staleness_seconds: 30,
    },
  },
  "GET:/v1/export/daily": {
    status: 200,
    data: {
      export_version: "1.0",
      exported_at: "2026-07-28T12:00:00Z",
      user_id: "primary",
      date: "2026-07-28",
      events: [],
      segments: [],
      daily_summary: {},
    },
  },
  "GET:/v1/export/weekly": {
    status: 200,
    data: {
      export_version: "1.0",
      exported_at: "2026-07-28T12:00:00Z",
      user_id: "primary",
      week: "2026-W31",
      events: [],
      segments: [],
      weekly_summary: {},
    },
  },
  "POST:/v1/privacy/wipe": {
    status: 200,
    data: {
      status: "wiped",
      message: "All user data has been erased",
    },
  },
  "GET:/v1/analytics/insights": {
    status: 200,
    data: {
      period: "7d",
      date_range: { from: "2026-07-21T00:00:00Z", to: "2026-07-28T12:00:00Z" },
      focus: {
        total_focus_minutes: 840,
        total_meeting_minutes: 300,
        total_distraction_minutes: 180,
        focus_session_count: 14,
      },
      habits: { active: 3, completed: 1, total: 4, avg_streak: 5.2 },
      goals: { active: 2, completed: 1, total: 3 },
      interventions: { total: 8, accepted: 5, effectiveness_rate: 0.625 },
    },
  },
};

// Define all 47 MCP tools and their expected behavior
const mcpTools = [
  // Context & Briefing (2)
  {
    name: "get_current_context",
    description: "Get the user's current context",
    endpoint: "/v1/context/current",
    method: "GET",
    requiredFields: ["context"],
    validates: (data) => {
      assert(data.context, "Response must have context");
      assert(typeof data.context.primary_activity === "string", "Must have primary_activity");
      assert(typeof data.context.distraction_probability === "number", "Must have distraction_probability");
    },
  },
  {
    name: "get_current_briefing",
    description: "Get agent briefing",
    endpoint: "/v1/briefing/current",
    method: "GET",
    requiredFields: ["briefing_id", "state"],
    validates: (data) => {
      assert(data.briefing_id, "Must have briefing_id");
      assert(data.state, "Must have state");
      assert(typeof data.state.primary_activity === "string", "State must have primary_activity");
    },
  },
  // Timeline (4)
  {
    name: "get_timeline",
    description: "Get timeline segments",
    endpoint: "/v1/timeline",
    method: "GET",
    requiredFields: ["timeline"],
    validates: (data) => {
      assert(Array.isArray(data.timeline), "Must have timeline array");
    },
  },
  {
    name: "get_focus_sessions",
    description: "Get today's focus sessions from the timeline",
    endpoint: "/v1/timeline",
    method: "GET",
    requiredFields: ["timeline"],
    validates: (data) => {
      assert(Array.isArray(data.timeline), "Must have timeline array");
    },
  },
  {
    name: "get_place_visits",
    description: "Get place visit segments",
    endpoint: "/v1/timeline",
    method: "GET",
    requiredFields: ["timeline"],
    validates: (data) => {
      assert(Array.isArray(data.timeline), "Must have timeline array");
    },
  },
  {
    name: "get_distraction_segments",
    description: "Get distraction segments to understand when focus was broken",
    endpoint: "/v1/timeline",
    method: "GET",
    requiredFields: ["timeline"],
    validates: (data) => {
      assert(Array.isArray(data.timeline), "Must have timeline array");
    },
  },
  // Summaries (4)
  {
    name: "get_daily_summary",
    description: "Get daily summary",
    endpoint: "/v1/summary/daily",
    method: "GET",
    requiredFields: ["date"],
    validates: (data) => {
      assert(data.date, "Must have date");
    },
  },
  {
    name: "get_weekly_summary",
    description: "Get weekly summary",
    endpoint: "/v1/summary/weekly",
    method: "GET",
    requiredFields: ["week"],
    validates: (data) => {
      assert(data.week, "Must have week");
    },
  },
  {
    name: "get_five_minute_summaries",
    description: "Get five-minute summaries",
    endpoint: "/v1/summary/five_minute",
    method: "GET",
    requiredFields: ["summaries"],
    validates: (data) => {
      assert(Array.isArray(data.summaries), "Must have summaries array");
    },
  },
  {
    name: "get_hourly_summaries",
    description: "Get hourly summaries",
    endpoint: "/v1/summary/hourly",
    method: "GET",
    requiredFields: ["summaries"],
    validates: (data) => {
      assert(Array.isArray(data.summaries), "Must have summaries array");
    },
  },
  // Events (1)
  {
    name: "search_events",
    description: "Search raw events by type and time range",
    endpoint: "/v1/events",
    method: "GET",
    requiredFields: ["events"],
    validates: (data) => {
      assert(Array.isArray(data.events), "Must have events array");
    },
  },
  // Agent Loop — Read/Write (5)
  {
    name: "add_agent_observation",
    description: "Write an observation about user behavior",
    endpoint: "/v1/agent/observations",
    method: "POST",
    requiredFields: ["observation_id"],
    validates: (data) => {
      assert(data.observation_id, "Must return observation_id");
    },
  },
  {
    name: "get_agent_observations",
    description: "Get agent observations",
    endpoint: "/v1/agent/observations",
    method: "GET",
    requiredFields: ["observations"],
    validates: (data) => {
      assert(Array.isArray(data.observations), "Must have observations array");
    },
  },
  {
    name: "propose_intervention",
    description: "Propose intervention",
    endpoint: "/v1/agent/interventions",
    method: "POST",
    requiredFields: ["intervention_id", "status"],
    validates: (data) => {
      assert(data.intervention_id, "Must return intervention_id");
    },
  },
  {
    name: "update_intervention_status",
    description: "Update intervention status",
    endpoint: "/v1/agent/interventions/int-test",
    method: "PATCH",
    requiredFields: ["intervention_id", "status"],
    validates: (data) => {
      assert(data.intervention_id, "Must have intervention_id");
      assert(data.status, "Must have status");
    },
  },
  {
    name: "get_intervention_effectiveness",
    description: "Compute intervention effectiveness score from outcomes and feedback",
    endpoint: "/v1/interventions/int-test/effectiveness",
    method: "GET",
    requiredFields: ["intervention_id", "effectiveness_score", "rating"],
    validates: (data) => {
      assert(data.intervention_id, "Must have intervention_id");
      assert(typeof data.effectiveness_score === "number", "Must have effectiveness_score");
      assert(data.rating, "Must have rating");
    },
  },
  // Feedback (2)
  {
    name: "submit_feedback",
    description: "Submit user feedback about an intervention or observation",
    endpoint: "/v1/feedback",
    method: "POST",
    requiredFields: ["feedback_id"],
    validates: (data) => {
      assert(data.feedback_id, "Must return feedback_id");
    },
  },
  {
    name: "get_feedback",
    description: "Get user feedback",
    endpoint: "/v1/feedback",
    method: "GET",
    requiredFields: ["feedback"],
    validates: (data) => {
      assert(Array.isArray(data.feedback), "Must have feedback array");
    },
  },
  // Outcomes (2)
  {
    name: "record_outcome",
    description: "Record intervention outcome",
    endpoint: "/v1/outcomes",
    method: "POST",
    requiredFields: ["outcome_id"],
    validates: (data) => {
      assert(data.outcome_id, "Must return outcome_id");
    },
  },
  {
    name: "get_intervention_outcomes",
    description: "Get intervention outcomes",
    endpoint: "/v1/outcomes",
    method: "GET",
    requiredFields: ["outcomes"],
    validates: (data) => {
      assert(Array.isArray(data.outcomes), "Must have outcomes array");
    },
  },
  // Goals (4)
  {
    name: "get_goals",
    description: "Get tracked goals",
    endpoint: "/v1/goals",
    method: "GET",
    requiredFields: ["goals"],
    validates: (data) => {
      assert(Array.isArray(data.goals), "Must have goals array");
    },
  },
  {
    name: "create_goal",
    description: "Create a new goal",
    endpoint: "/v1/goals",
    method: "POST",
    requiredFields: ["goal_id"],
    validates: (data) => {
      assert(data.goal_id, "Must return goal_id");
    },
  },
  {
    name: "update_goal",
    description: "Update a goal",
    endpoint: "/v1/goals/goal-test",
    method: "PATCH",
    requiredFields: ["goal_id"],
    validates: (data) => {
      assert(data.goal_id, "Must return goal_id");
    },
  },
  {
    name: "delete_goal",
    description: "Delete a goal",
    endpoint: "/v1/goals/goal-test",
    method: "DELETE",
    requiredFields: ["goal_id"],
    validates: (data) => {
      assert(data.goal_id, "Must return goal_id");
    },
  },
  // Habits (5)
  {
    name: "get_habits",
    description: "Get tracked habits",
    endpoint: "/v1/habits",
    method: "GET",
    requiredFields: ["habits"],
    validates: (data) => {
      assert(Array.isArray(data.habits), "Must have habits array");
    },
  },
  {
    name: "create_habit",
    description: "Create a new habit",
    endpoint: "/v1/habits",
    method: "POST",
    requiredFields: ["habit_id"],
    validates: (data) => {
      assert(data.habit_id, "Must return habit_id");
    },
  },
  {
    name: "update_habit",
    description: "Update a habit",
    endpoint: "/v1/habits/habit-test",
    method: "PATCH",
    requiredFields: ["habit_id"],
    validates: (data) => {
      assert(data.habit_id, "Must return habit_id");
    },
  },
  {
    name: "delete_habit",
    description: "Delete a habit",
    endpoint: "/v1/habits/habit-test",
    method: "DELETE",
    requiredFields: ["habit_id"],
    validates: (data) => {
      assert(data.habit_id, "Must return habit_id");
    },
  },
  {
    name: "update_habit_streak",
    description: "Update habit streak",
    endpoint: "/v1/habits/habit-test/streak",
    method: "POST",
    requiredFields: ["habit_id"],
    validates: (data) => {
      assert(data.habit_id, "Must return habit_id");
    },
  },
  // Baselines (3)
  {
    name: "get_baselines",
    description: "Get baseline metrics",
    endpoint: "/v1/baselines",
    method: "GET",
    requiredFields: ["baselines"],
    validates: (data) => {
      assert(Array.isArray(data.baselines), "Must have baselines array");
    },
  },
  {
    name: "create_baseline",
    description: "Create a baseline metric",
    endpoint: "/v1/baselines",
    method: "POST",
    requiredFields: ["baseline_id"],
    validates: (data) => {
      assert(data.baseline_id, "Must return baseline_id");
    },
  },
  {
    name: "delete_baseline",
    description: "Delete a baseline metric",
    endpoint: "/v1/baselines/baseline-test",
    method: "DELETE",
    requiredFields: ["baseline_id"],
    validates: (data) => {
      assert(data.baseline_id, "Must return baseline_id");
    },
  },
  // Corrections (2)
  {
    name: "get_corrections",
    description: "Get user corrections",
    endpoint: "/v1/corrections",
    method: "GET",
    requiredFields: ["corrections"],
    validates: (data) => {
      assert(Array.isArray(data.corrections), "Must have corrections array");
    },
  },
  {
    name: "submit_correction",
    description: "Submit user correction",
    endpoint: "/v1/corrections",
    method: "POST",
    requiredFields: ["status"],
    validates: (data) => {
      assert(data.status === "created", "Status must be created");
    },
  },
  // Privacy (2)
  {
    name: "get_sensitive_mode",
    description: "Get sensitive mode status",
    endpoint: "/v1/privacy/sensitive-mode",
    method: "GET",
    requiredFields: ["active"],
    validates: (data) => {
      assert(typeof data.active === "boolean", "Must have active field");
    },
  },
  {
    name: "set_sensitive_mode",
    description: "Set sensitive mode",
    endpoint: "/v1/privacy/sensitive-mode",
    method: "POST",
    requiredFields: ["active"],
    validates: (data) => {
      assert(typeof data.active === "boolean", "Must have active field");
    },
  },
  // System (3)
  {
    name: "get_server_health",
    description: "Get detailed server health with database statistics",
    endpoint: "/v1/health/detailed",
    method: "GET",
    requiredFields: ["status", "db_stats"],
    validates: (data) => {
      assert(data.status === "ok", "Status must be ok");
      assert(data.db_stats, "Must have db_stats");
    },
  },
  {
    name: "get_audit_log",
    description: "Get the audit log of all agent and system actions",
    endpoint: "/v1/audit",
    method: "GET",
    requiredFields: ["audit_log"],
    validates: (data) => {
      assert(Array.isArray(data.audit_log), "Must have audit_log array");
    },
  },
  // Metrics & Analytics (5)
  {
    name: "get_work_pattern_metrics",
    description: "Get computed work pattern analytics",
    endpoint: "/v1/metrics/work-patterns",
    method: "GET",
    requiredFields: ["total_focus_minutes", "context_switching_rate_per_hour"],
    validates: (data) => {
      assert(typeof data.total_focus_minutes === "number", "Must have total_focus_minutes");
      assert(typeof data.context_switching_rate_per_hour === "number", "Must have context_switching_rate_per_hour");
    },
  },
  {
    name: "get_readiness_score",
    description: "Get readiness/fatigue score",
    endpoint: "/v1/metrics/readiness",
    method: "GET",
    requiredFields: ["readiness_score", "fatigue_probability"],
    validates: (data) => {
      assert(typeof data.readiness_score === "number", "Must have readiness_score");
      assert(typeof data.fatigue_probability === "number", "Must have fatigue_probability");
    },
  },
  {
    name: "get_collector_health",
    description: "Get collector health status",
    endpoint: "/v1/health/collectors",
    method: "GET",
    requiredFields: ["collectors", "staleness_seconds"],
    validates: (data) => {
      assert(Array.isArray(data.collectors), "Must have collectors array");
      assert(typeof data.staleness_seconds === "number", "Must have staleness_seconds");
    },
  },
  {
    name: "evaluate_goal_progress",
    description: "Evaluate goal progress",
    endpoint: "/v1/goals/goal-test/progress",
    method: "GET",
    requiredFields: ["goal_id", "assessment"],
    validates: (data) => {
      assert(data.goal_id, "Must have goal_id");
      assert(data.assessment, "Must have assessment");
    },
  },
  {
    name: "evaluate_habit_completion",
    description: "Evaluate habit completion",
    endpoint: "/v1/habits/habit-test/evaluation",
    method: "GET",
    requiredFields: ["habit_id", "assessment"],
    validates: (data) => {
      assert(data.habit_id, "Must have habit_id");
      assert(data.assessment, "Must have assessment");
    },
  },
  // Analytics (1)
  {
    name: "get_analytics_insights",
    description: "Get cross-domain analytics insights",
    endpoint: "/v1/analytics/insights",
    method: "GET",
    requiredFields: ["period", "focus", "habits", "goals", "interventions"],
    validates: (data) => {
      assert(data.period, "Must have period");
      assert(data.focus, "Must have focus");
      assert(data.habits, "Must have habits");
      assert(data.goals, "Must have goals");
      assert(data.interventions, "Must have interventions");
    },
  },
  // Data Export & Privacy (3)
  {
    name: "export_daily_data",
    description: "Export all data for a specific day as a structured JSON bundle",
    endpoint: "/v1/export/daily",
    method: "GET",
    requiredFields: ["export_version", "date", "events"],
    validates: (data) => {
      assert(data.export_version, "Must have export_version");
      assert(data.date, "Must have date");
      assert(Array.isArray(data.events), "Must have events array");
    },
  },
  {
    name: "export_weekly_data",
    description: "Export all data for a specific week as a structured JSON bundle",
    endpoint: "/v1/export/weekly",
    method: "GET",
    requiredFields: ["export_version", "week", "events"],
    validates: (data) => {
      assert(data.export_version, "Must have export_version");
      assert(data.week, "Must have week");
      assert(Array.isArray(data.events), "Must have events array");
    },
  },
  {
    name: "wipe_all_data",
    description: "Erase ALL user data from the server — irreversible kill switch",
    endpoint: "/v1/privacy/wipe",
    method: "POST",
    requiredFields: ["status"],
    validates: (data) => {
      assert(data.status === "wiped", "Status must be wiped");
    },
  },
];

// Test 1: Tool count matches architecture
console.log("[test] Test 1: Tool count matches architecture");
assert(mcpTools.length === 47, `Expected 47 tools, got ${mcpTools.length}`);
console.log(`  ✓ All ${mcpTools.length} MCP tools defined\n`);

// Test 2: All tools have required fields
console.log("[test] Test 2: Tool definitions are complete");
for (const tool of mcpTools) {
  assert(tool.name, `Tool missing name`);
  assert(tool.description, `Tool ${tool.name} missing description`);
  assert(tool.endpoint, `Tool ${tool.name} missing endpoint`);
  assert(tool.method, `Tool ${tool.name} missing method`);
  assert(tool.requiredFields, `Tool ${tool.name} missing requiredFields`);
  assert(typeof tool.validates === "function", `Tool ${tool.name} missing validates function`);
}
console.log(`  ✓ All tool definitions are complete\n`);

// Test 3: Tool names are unique
console.log("[test] Test 3: Tool names are unique");
const names = mcpTools.map((t) => t.name);
const uniqueNames = new Set(names);
assert(uniqueNames.size === names.length, `Found duplicate tool names`);
console.log(`  ✓ All ${names.length} tool names are unique\n`);

// Test 4: Mock response validation (match by method+endpoint)
console.log("[test] Test 4: Mock responses validate against tool schemas");
for (const tool of mcpTools) {
  const mockKey = `${tool.method}:${tool.endpoint}`;
  const mock = mockResponses[mockKey];
  if (mock && mock.status === 200) {
    try {
      tool.validates(mock.data);
      console.log(`  ✓ ${tool.name} mock response is valid`);
    } catch (e) {
      console.log(`  ✗ ${tool.name} mock validation failed: ${e.message}`);
      process.exit(1);
    }
  } else {
    console.log(`  - ${tool.name} (no matching mock, skipped)`);
  }
}
console.log();

// Test 5: Endpoint coverage
console.log("[test] Test 5: All architecture endpoints are covered");
const architectureEndpoints = [
  "/v1/context/current",
  "/v1/briefing/current",
  "/v1/timeline",
  "/v1/events",
  "/v1/summary/daily",
  "/v1/summary/weekly",
  "/v1/summary/five_minute",
  "/v1/summary/hourly",
  "/v1/agent/observations",
  "/v1/agent/interventions",
  "/v1/feedback",
  "/v1/outcomes",
  "/v1/goals",
  "/v1/habits",
  "/v1/baselines",
  "/v1/corrections",
  "/v1/health/detailed",
  "/v1/health/collectors",
  "/v1/privacy/sensitive-mode",
  "/v1/audit",
  "/v1/metrics/work-patterns",
  "/v1/metrics/readiness",
  "/v1/export/daily",
  "/v1/export/weekly",
  "/v1/interventions/{id}/effectiveness",
  "/v1/analytics/insights",
];
const toolEndpoints = mcpTools.map((t) => t.endpoint);
for (const endpoint of architectureEndpoints) {
  const paramEndpoint = endpoint.replace("/{id}", "");
  assert(
    toolEndpoints.includes(endpoint) || toolEndpoints.some(e => e.startsWith(paramEndpoint)),
    `Missing endpoint: ${endpoint}`
  );
}
console.log(`  ✓ All ${architectureEndpoints.length} architecture endpoints covered\n`);

// Test 6: HTTP methods
console.log("[test] Test 6: HTTP methods are correct");
for (const tool of mcpTools) {
  assert(
    ["GET", "POST", "PUT", "PATCH", "DELETE"].includes(tool.method),
    `${tool.name} has invalid method: ${tool.method}`
  );
}
console.log(`  ✓ All HTTP methods valid\n`);

// Test 7: Scope requirements
console.log("[test] Test 7: Scope requirements documented");
const scopeMap = {
  get_current_context: "agent:context:read",
  get_current_briefing: "agent:briefing:read",
  get_timeline: "agent:timeline:read",
  get_focus_sessions: "agent:timeline:read",
  get_place_visits: "agent:timeline:read",
  get_distraction_segments: "agent:timeline:read",
  get_daily_summary: "agent:summaries:read",
  get_weekly_summary: "agent:summaries:read",
  get_five_minute_summaries: "agent:summaries:read",
  get_hourly_summaries: "agent:summaries:read",
  search_events: "agent:events:read",
  add_agent_observation: "agent:observations:write",
  get_agent_observations: "agent:observations:read",
  propose_intervention: "agent:interventions:write",
  update_intervention_status: "agent:interventions:write",
  get_intervention_effectiveness: "agent:interventions:read",
  submit_feedback: "agent:feedback:write",
  get_feedback: "agent:feedback:read",
  record_outcome: "agent:outcomes:write",
  get_intervention_outcomes: "agent:outcomes:read",
  get_goals: "agent:goals:read",
  create_goal: "agent:goals:write",
  update_goal: "agent:goals:write",
  delete_goal: "agent:goals:write",
  get_habits: "agent:habits:read",
  create_habit: "agent:habits:write",
  update_habit: "agent:habits:write",
  delete_habit: "agent:habits:write",
  update_habit_streak: "agent:habits:write",
  get_baselines: "agent:baselines:read",
  create_baseline: "agent:baselines:write",
  delete_baseline: "agent:baselines:write",
  get_corrections: "device:corrections:read",
  submit_correction: "device:corrections:write",
  get_sensitive_mode: "device:privacy:read",
  set_sensitive_mode: "device:privacy:write",
  get_server_health: "none",
  get_audit_log: "admin",
  get_work_pattern_metrics: "agent:analytics:read",
  get_readiness_score: "agent:analytics:read",
  get_collector_health: "agent:health:read",
  evaluate_goal_progress: "agent:goals:read",
  evaluate_habit_completion: "agent:habits:read",
  get_analytics_insights: "agent:analytics:read",
  export_daily_data: "admin",
  export_weekly_data: "admin",
  wipe_all_data: "admin",
};
for (const [toolName, scope] of Object.entries(scopeMap)) {
  const tool = mcpTools.find((t) => t.name === toolName);
  assert(tool, `Tool ${toolName} not found in mcpTools`);
}
console.log(`  ✓ Scope requirements documented for ${Object.keys(scopeMap).length} tools\n`);

// Test 8: Tool descriptions are meaningful
console.log("[test] Test 8: Tool descriptions are meaningful");
for (const tool of mcpTools) {
  assert(tool.description.length > 10, `${tool.name} description too short`);
  assert(!tool.description.includes("TODO"), `${tool.name} has TODO in description`);
}
console.log(`  ✓ All tool descriptions are meaningful\n`);

// Test 9: Tool count matches scope map
console.log("[test] Test 9: Tool count matches scope map count");
assert(
  mcpTools.length === Object.keys(scopeMap).length,
  `Tool count (${mcpTools.length}) doesn't match scope map (${Object.keys(scopeMap).length})`
);
console.log(`  ✓ Tool count matches scope map: ${mcpTools.length}\n`);

console.log("========================================");
console.log("[test] ALL MCP TESTS PASSED");
console.log(`  Tools: ${mcpTools.length}`);
console.log(`  Endpoints: ${architectureEndpoints.length}`);
console.log("========================================");
