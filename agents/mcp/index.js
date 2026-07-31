#!/usr/bin/env node
/**
 * Andrometry MCP Server
 *
 * Wraps the Andrometry relay API as MCP tools for AI agents.
 * Agents can query context, timeline, summaries, write observations,
 * propose interventions, and track outcomes — all through MCP.
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

// Configuration
const RELAY_URL = process.env.ANDROMETRY_RELAY_URL || "http://localhost:8420";
const AGENT_TOKEN = process.env.ANDROMETRY_AGENT_TOKEN || "";

// Helper to call the relay API
async function relayFetch(path, options = {}) {
  const url = `${RELAY_URL}${path}`;
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${AGENT_TOKEN}`,
    ...options.headers,
  };

  const response = await fetch(url, { ...options, headers });

  if (response.status === 304) {
    return { _notModified: true };
  }

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Relay API error ${response.status}: ${text}`);
  }

  return response.json();
}

// Create MCP server
const server = new McpServer({
  name: "andrometry",
  version: "0.1.0",
});

// ============================================================
// Context Tools
// ============================================================

server.tool(
  "get_current_context",
  "Get the user's current context — what they're doing, where, focus state, distraction/fatigue probability, and data quality",
  {},
  async () => {
    const data = await relayFetch("/v1/context/current");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_current_briefing",
  "Get a high-level agent briefing with state, risks, opportunities, and suggested queries",
  {
    horizon: z.string().optional().describe("Time horizon (e.g., '2h')"),
    agent_id: z.string().optional().describe("Agent ID for personalized briefing"),
  },
  async ({ horizon, agent_id }) => {
    const params = new URLSearchParams();
    if (horizon) params.set("horizon", horizon);
    if (agent_id) params.set("agent_id", agent_id);
    const qs = params.toString() ? `?${params}` : "";
    const data = await relayFetch(`/v1/briefing/current${qs}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Timeline Tools
// ============================================================

server.tool(
  "get_timeline",
  "Get semantic timeline segments — activity, focus, place visits, meetings, distraction, etc.",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    segment_type: z
      .enum([
        "activity_segment",
        "focus_segment",
        "place_visit",
        "app_usage_segment",
        "meeting_segment",
        "communication_segment",
        "sleep_segment",
        "distraction_segment",
        "break_segment",
        "exercise_segment",
        "commute_segment",
      ])
      .optional()
      .describe("Filter by segment type"),
    limit: z.number().optional().describe("Max segments to return (default 200)"),
  },
  async ({ from, to, segment_type, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    if (segment_type) params.set("segment_type", segment_type);
    if (limit) params.set("limit", String(limit));
    const qs = params.toString() ? `?${params}` : "";
    const data = await relayFetch(`/v1/timeline${qs}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_focus_sessions",
  "Get today's focus sessions from the timeline",
  {},
  async () => {
    const now = new Date();
    const startOfDay = new Date(now);
    startOfDay.setHours(0, 0, 0, 0);
    const data = await relayFetch(
      `/v1/timeline?from=${startOfDay.toISOString()}&to=${now.toISOString()}&segment_type=focus_segment`
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_place_visits",
  "Get place visit segments (home, office, gym, cafe, etc.)",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("segment_type", "place_visit");
    const data = await relayFetch(`/v1/timeline?${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_distraction_segments",
  "Get distraction segments to understand when focus was broken",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("segment_type", "distraction_segment");
    const data = await relayFetch(`/v1/timeline?${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Summary Tools
// ============================================================

server.tool(
  "get_daily_summary",
  "Get a rich daily summary with sleep, health, work, distraction, focus, scores, and baselines",
  {
    date: z
      .string()
      .optional()
      .describe("Date in YYYY-MM-DD format (default: today)"),
  },
  async ({ date }) => {
    const params = date ? `?date=${date}` : "";
    const data = await relayFetch(`/v1/summary/daily${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_weekly_summary",
  "Get a weekly summary with trends and patterns",
  {
    week: z
      .string()
      .optional()
      .describe("Week in YYYY-WXX format (default: current week)"),
  },
  async ({ week }) => {
    const params = week ? `?week=${week}` : "";
    const data = await relayFetch(`/v1/summary/weekly${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Event Tools (drill-down)
// ============================================================

server.tool(
  "search_events",
  "Search raw events by type and time range (drill-down — prefer timeline/summaries first)",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    event_type: z.string().optional().describe("Filter by event type"),
    limit: z.number().optional().describe("Max events (default 200)"),
  },
  async ({ from, to, event_type, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    if (event_type) params.set("event_type", event_type);
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Agent Loop Tools
// ============================================================

server.tool(
  "add_agent_observation",
  "Write an observation about the user's behavior, patterns, or risks",
  {
    agent_id: z.string().describe("Your agent identifier"),
    observation_type: z
      .enum(["risk", "pattern", "alignment", "anomaly", "suggestion", "reflection", "milestone"])
      .describe("Type of observation"),
    severity: z
      .enum(["info", "low", "medium", "high", "critical"])
      .describe("Severity level"),
    message: z.string().describe("Observation message"),
    evidence: z.record(z.any()).optional().describe("Supporting evidence"),
  },
  async ({ agent_id, observation_type, severity, message, evidence }) => {
    const data = await relayFetch("/v1/agent/observations", {
      method: "POST",
      body: JSON.stringify({
        agent_id,
        observation_type,
        severity,
        message,
        evidence: evidence || {},
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "propose_intervention",
  "Propose an intervention (nudge, suggestion, warning, break suggestion, etc.)",
  {
    agent_id: z.string().describe("Your agent identifier"),
    intervention_type: z
      .enum([
        "nudge",
        "suggestion",
        "warning",
        "reflection_prompt",
        "goal_reminder",
        "habit_check",
        "break_suggestion",
        "focus_extend",
        "energy_check",
      ])
      .describe("Type of intervention"),
    title: z.string().describe("Short intervention title"),
    message: z.string().describe("Intervention message"),
    suggested_action: z.string().optional().describe("Suggested action"),
    severity: z.enum(["low", "medium", "high"]).describe("Severity level"),
    evidence: z.record(z.any()).optional().describe("Supporting evidence"),
  },
  async ({
    agent_id,
    intervention_type,
    title,
    message,
    suggested_action,
    severity,
    evidence,
  }) => {
    const data = await relayFetch("/v1/agent/interventions", {
      method: "POST",
      body: JSON.stringify({
        agent_id,
        intervention_type,
        title,
        message,
        suggested_action: suggested_action || "",
        severity,
        evidence: evidence || {},
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "record_outcome",
  "Record the outcome of an intervention — did it work?",
  {
    intervention_id: z.string().describe("The intervention ID"),
    outcome_type: z
      .enum([
        "accepted",
        "dismissed",
        "ignored",
        "completed",
        "behavior_change",
        "metric_improved",
        "no_effect",
        "negative_effect",
        "self_reported_helpful",
      ])
      .describe("Type of outcome"),
    effectiveness_score: z
      .number()
      .min(0)
      .max(1)
      .optional()
      .describe("Effectiveness 0-1"),
    evidence: z.record(z.any()).optional().describe("Supporting evidence"),
  },
  async ({ intervention_id, outcome_type, effectiveness_score, evidence }) => {
    const data = await relayFetch("/v1/outcomes", {
      method: "POST",
      body: JSON.stringify({
        intervention_id,
        outcome_type,
        effectiveness_score,
        evidence: evidence || {},
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_intervention_outcomes",
  "Get outcomes for interventions to evaluate effectiveness",
  {
    intervention_id: z.string().optional().describe("Specific intervention ID"),
    limit: z.number().optional().describe("Max outcomes (default 50)"),
  },
  async ({ intervention_id, limit }) => {
    const params = new URLSearchParams();
    if (intervention_id) params.set("intervention_id", intervention_id);
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/outcomes?${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Goal/Habit Tools
// ============================================================

server.tool(
  "get_goals",
  "Get the user's active goals",
  {},
  async () => {
    const data = await relayFetch("/v1/goals");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "create_goal",
  "Create a new goal for the user",
  {
    goal_type: z
      .enum([
        "deep_work_minutes_daily",
        "distraction_limit",
        "sleep_target",
        "exercise_target",
        "focus_session_count",
        "custom",
      ])
      .describe("Goal type"),
    title: z.string().describe("Goal title"),
    target: z.record(z.any()).optional().describe("Target values"),
  },
  async ({ goal_type, title, target }) => {
    const data = await relayFetch("/v1/goals", {
      method: "POST",
      body: JSON.stringify({ goal_type, title, target: target || {} }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "update_goal",
  "Update a goal's status, title, or target",
  {
    goal_id: z.string().describe("Goal ID"),
    status: z.enum(["active", "paused", "completed", "abandoned"]).optional().describe("New status"),
    title: z.string().optional().describe("Updated title"),
    target: z.record(z.any()).optional().describe("Updated target values"),
  },
  async ({ goal_id, status, title, target }) => {
    const body = {};
    if (status) body.status = status;
    if (title) body.title = title;
    if (target) body.target = target;
    const data = await relayFetch(`/v1/goals/${goal_id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "delete_goal",
  "Delete a goal",
  {
    goal_id: z.string().describe("Goal ID to delete"),
  },
  async ({ goal_id }) => {
    const data = await relayFetch(`/v1/goals/${goal_id}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_habits",
  "Get the user's tracked habits with streaks and strength",
  {},
  async () => {
    const data = await relayFetch("/v1/habits");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "create_habit",
  "Create a new habit to track",
  {
    habit_type: z
      .enum([
        "morning_routine",
        "deep_work_block",
        "exercise",
        "reading",
        "meditation",
        "early_sleep",
        "no_social_media_morning",
        "custom",
      ])
      .describe("Habit type"),
    title: z.string().describe("Habit title"),
    target: z
      .object({
        metric: z.string(),
        minimum: z.number(),
        unit: z.string().optional(),
      })
      .describe("Target metric"),
    schedule: z
      .object({
        days: z.array(z.string()),
        window: z.string(),
      })
      .describe("Schedule"),
    evidence_sources: z
      .array(z.string())
      .describe("Evidence sources (focus_segment, health_summary, etc.)"),
  },
  async ({ habit_type, title, target, schedule, evidence_sources }) => {
    const data = await relayFetch("/v1/habits", {
      method: "POST",
      body: JSON.stringify({
        habit_type,
        title,
        target,
        schedule,
        evidence_sources,
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "update_habit",
  "Update a habit's status, type, or streak",
  {
    habit_id: z.string().describe("Habit ID"),
    status: z.enum(["active", "paused", "completed", "abandoned"]).optional().describe("New status"),
    streak_days: z.number().optional().describe("Updated streak days"),
    habit_strength: z.number().optional().describe("Updated habit strength (0-1)"),
  },
  async ({ habit_id, status, streak_days, habit_strength }) => {
    const body = {};
    if (status) body.status = status;
    if (streak_days !== undefined) body.streak_days = streak_days;
    if (habit_strength !== undefined) body.habit_strength = habit_strength;
    const data = await relayFetch(`/v1/habits/${habit_id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "delete_habit",
  "Delete a habit",
  {
    habit_id: z.string().describe("Habit ID to delete"),
  },
  async ({ habit_id }) => {
    const data = await relayFetch(`/v1/habits/${habit_id}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "update_habit_streak",
  "Update a habit's streak count and strength score",
  {
    habit_id: z.string().describe("Habit ID"),
    streak_days: z.number().describe("Number of consecutive days"),
    habit_strength: z.number().min(0).max(1).describe("Habit strength 0-1"),
  },
  async ({ habit_id, streak_days, habit_strength }) => {
    const data = await relayFetch(`/v1/habits/${habit_id}/streak`, {
      method: "POST",
      body: JSON.stringify({ streak_days, habit_strength }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "submit_feedback",
  "Submit user feedback about an intervention, observation, or context",
  {
    target_type: z.enum(["intervention", "observation", "context", "segment", "habit", "goal"]).describe("What the feedback is about"),
    target_id: z.string().describe("ID of the target being rated"),
    feedback_type: z.enum(["helpful", "not_helpful", "annoying", "too_late", "too_early", "correct", "incorrect", "accepted", "dismissed"]).describe("Type of feedback"),
    rating: z.number().min(1).max(5).optional().describe("Rating 1-5"),
    note: z.string().optional().describe("Optional note about the feedback"),
  },
  async ({ target_type, target_id, feedback_type, rating, note }) => {
    const data = await relayFetch("/v1/feedback", {
      method: "POST",
      body: JSON.stringify({
        target_type,
        target_id,
        feedback_type,
        rating: rating || 0,
        note: note || "",
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_feedback",
  "Get user feedback, optionally filtered by target",
  {
    target_type: z.string().optional().describe("Filter by target type (intervention, observation, etc.)"),
    target_id: z.string().optional().describe("Filter by target ID"),
    limit: z.number().optional().describe("Max results (default 50)"),
  },
  async ({ target_type, target_id, limit }) => {
    const params = new URLSearchParams();
    if (target_type) params.set("target_type", target_type);
    if (target_id) params.set("target_id", target_id);
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/feedback?${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Baseline Tools
// ============================================================

server.tool(
  "get_baselines",
  "Get all baseline metrics for comparison — average focus time, distraction rate, etc.",
  {},
  async () => {
    const data = await relayFetch("/v1/baselines");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "create_baseline",
  "Create or update a baseline metric for comparison",
  {
    metric_name: z.string().describe("Metric name (e.g., 'deep_work_minutes_daily', 'steps_daily')"),
    value: z.number().describe("Baseline value"),
    window_days: z.number().optional().describe("Rolling window in days (default 7)"),
  },
  async ({ metric_name, value, window_days }) => {
    const data = await relayFetch("/v1/baselines", {
      method: "POST",
      body: JSON.stringify({
        metric_name,
        value,
        window_days: window_days || 7,
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "delete_baseline",
  "Delete a baseline metric",
  {
    baseline_id: z.string().describe("Baseline ID to delete"),
  },
  async ({ baseline_id }) => {
    const data = await relayFetch(`/v1/baselines/${baseline_id}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Correction Tools
// ============================================================

server.tool(
  "get_corrections",
  "Get user corrections to timeline segments — label fixes, splits, merges, etc.",
  {
    limit: z.number().optional().describe("Max results (default 50)"),
  },
  async ({ limit }) => {
    const params = limit ? `?limit=${limit}` : "";
    const data = await relayFetch(`/v1/corrections${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "submit_correction",
  "Submit a correction to a timeline segment (label fix, split, merge, delete, move)",
  {
    correction_type: z
      .enum(["label", "split", "delete", "merge", "move"])
      .describe("Type of correction"),
    segment_id: z.string().describe("Segment ID to correct"),
    new_label: z.string().optional().describe("New label (for label corrections)"),
    split_at: z.string().optional().describe("Timestamp to split at (for split corrections)"),
    reason: z.string().optional().describe("Reason for the correction"),
  },
  async ({ correction_type, segment_id, new_label, split_at, reason }) => {
    const body = { correction_type, segment_id };
    if (new_label) body.new_label = new_label;
    if (split_at) body.split_at = split_at;
    if (reason) body.reason = reason;
    const data = await relayFetch("/v1/corrections", {
      method: "POST",
      body: JSON.stringify(body),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Observation Read Tools
// ============================================================

server.tool(
  "get_agent_observations",
  "Get recent agent observations — risks, patterns, anomalies, milestones",
  {
    limit: z.number().optional().describe("Max observations (default 50)"),
  },
  async ({ limit }) => {
    const params = limit ? `?limit=${limit}` : "";
    const data = await relayFetch(`/v1/agent/observations${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Granular Summary Tools
// ============================================================

server.tool(
  "get_five_minute_summaries",
  "Get five-minute resolution summaries for detailed analysis",
  {
    from: z.string().optional().describe("Start time (ISO 8601, default: 2h ago)"),
    to: z.string().optional().describe("End time (ISO 8601, default: now)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    const qs = params.toString() ? `?${params}` : "";
    const data = await relayFetch(`/v1/summary/five_minute${qs}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_hourly_summaries",
  "Get hourly summaries for trend analysis",
  {
    from: z.string().optional().describe("Start time (ISO 8601, default: 24h ago)"),
    to: z.string().optional().describe("End time (ISO 8601, default: now)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    const qs = params.toString() ? `?${params}` : "";
    const data = await relayFetch(`/v1/summary/hourly${qs}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Privacy Tools
// ============================================================

server.tool(
  "get_sensitive_mode",
  "Check if sensitive mode is active (redacts location, health, and communication data)",
  {},
  async () => {
    const data = await relayFetch("/v1/privacy/sensitive-mode");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "set_sensitive_mode",
  "Enable or disable sensitive mode — when active, location/health/communication data is redacted",
  {
    active: z.boolean().describe("true to enable sensitive mode, false to disable"),
  },
  async ({ active }) => {
    const data = await relayFetch("/v1/privacy/sensitive-mode", {
      method: "POST",
      body: JSON.stringify({ active }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Audit Tools
// ============================================================

server.tool(
  "get_audit_log",
  "Get the audit log of all agent and system actions",
  {
    limit: z.number().optional().describe("Max entries (default 100)"),
  },
  async ({ limit }) => {
    const params = limit ? `?limit=${limit}` : "";
    const data = await relayFetch(`/v1/audit${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Health Tools
// ============================================================

server.tool(
  "get_server_health",
  "Get detailed server health with database statistics",
  {},
  async () => {
    const data = await relayFetch("/v1/health/detailed");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Metrics & Analytics Tools (§39, §40)
// ============================================================

server.tool(
  "get_work_pattern_metrics",
  "Get computed work pattern analytics — focus bout lengths, switching rates, fragmentation, late-night work",
  {
    from: z.string().optional().describe("Start time (ISO 8601, default: 24h ago)"),
    to: z.string().optional().describe("End time (ISO 8601, default: now)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    const qs = params.toString() ? `?${params}` : "";
    const data = await relayFetch(`/v1/metrics/work-patterns${qs}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_readiness_score",
  "Get the user's current readiness/fatigue score based on sleep, activity, and workload",
  {},
  async () => {
    const data = await relayFetch("/v1/metrics/readiness");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_collector_health",
  "Get per-collector health status — which data sources are healthy, stale, or degraded",
  {},
  async () => {
    const data = await relayFetch("/v1/health/collectors");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "evaluate_goal_progress",
  "Evaluate progress toward a specific goal — compares current metrics against baselines",
  {
    goal_id: z.string().describe("Goal ID to evaluate"),
  },
  async ({ goal_id }) => {
    const data = await relayFetch(`/v1/goals/${goal_id}/progress`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "evaluate_habit_completion",
  "Evaluate habit adherence — assesses streak, strength, and formation stage",
  {
    habit_id: z.string().describe("Habit ID to evaluate"),
  },
  async ({ habit_id }) => {
    const data = await relayFetch(`/v1/habits/${habit_id}/evaluation`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "update_intervention_status",
  "Update the status of an intervention (delivered, accepted, dismissed, ignored, completed, expired)",
  {
    intervention_id: z.string().describe("Intervention ID"),
    status: z.enum(["delivered", "accepted", "dismissed", "ignored", "completed", "failed", "expired"]).describe("New status"),
  },
  async ({ intervention_id, status }) => {
    const data = await relayFetch(`/v1/agent/interventions/${intervention_id}`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "get_intervention_effectiveness",
  "Compute intervention effectiveness score from outcomes and feedback (§35)",
  {
    intervention_id: z.string().describe("Intervention ID to evaluate"),
  },
  async ({ intervention_id }) => {
    const data = await relayFetch(`/v1/interventions/${intervention_id}/effectiveness`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Analytics Tools (§20)
// ============================================================

server.tool(
  "get_analytics_insights",
  "Get cross-domain analytics insights — focus trends, habit stats, intervention effectiveness, baseline deviations",
  {
    period: z.enum(["1d", "7d", "30d"]).optional().describe("Analysis period (default: 7d)"),
  },
  async ({ period }) => {
    const params = period ? `?period=${period}` : "";
    const data = await relayFetch(`/v1/analytics/insights${params}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Data Export & Privacy Tools (§27, §22.6)
// ============================================================

server.tool(
  "export_daily_data",
  "Export all data for a specific day as a structured JSON bundle (§27 Data Export)",
  {
    date: z.string().describe("Date in YYYY-MM-DD format"),
  },
  async ({ date }) => {
    const data = await relayFetch(`/v1/export/daily?date=${date}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "export_weekly_data",
  "Export all data for a specific week as a structured JSON bundle (§27 Data Export)",
  {
    week: z.string().describe("Week in YYYY-WXX format (e.g., '2026-W31')"),
  },
  async ({ week }) => {
    const data = await relayFetch(`/v1/export/weekly?week=${week}`);
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

server.tool(
  "wipe_all_data",
  "Erase ALL user data from the server — irreversible kill switch (§22.6)",
  {},
  async () => {
    const data = await relayFetch("/v1/privacy/wipe", {
      method: "POST",
      body: JSON.stringify({}),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  }
);

// ============================================================
// Convenience Segment Tools (§18.1)
// ============================================================

server.tool(
  "get_activity_segments",
  "Get activity segments from the timeline",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("segment_type", "activity_segment");
    const data = await relayFetch(`/v1/timeline?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_communication_segments",
  "Get communication segments — calls, messages, email, chat",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("segment_type", "communication_segment");
    const data = await relayFetch(`/v1/timeline?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_sleep_segments",
  "Get sleep segments from the timeline",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
  },
  async ({ from, to }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("segment_type", "sleep_segment");
    const data = await relayFetch(`/v1/timeline?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_app_usage",
  "Get app usage summary events",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    limit: z.number().optional().describe("Max events (default 50)"),
  },
  async ({ from, to, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("event_type", "app_usage_summary");
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_notification_load",
  "Get notification events to assess notification load",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    limit: z.number().optional().describe("Max events (default 50)"),
  },
  async ({ from, to, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("event_type", "notification_received");
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_health_summary_events",
  "Get health data events (steps, heart rate, sleep)",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    limit: z.number().optional().describe("Max events (default 50)"),
  },
  async ({ from, to, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("event_type", "health_data");
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_sleep_summary_events",
  "Get sleep summary events",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    limit: z.number().optional().describe("Max events (default 50)"),
  },
  async ({ from, to, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("event_type", "sleep_summary");
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_calendar_context_events",
  "Get calendar events for context",
  {
    from: z.string().optional().describe("Start time (ISO 8601)"),
    to: z.string().optional().describe("End time (ISO 8601)"),
    limit: z.number().optional().describe("Max events (default 50)"),
  },
  async ({ from, to, limit }) => {
    const params = new URLSearchParams();
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    params.set("event_type", "calendar_event");
    if (limit) params.set("limit", String(limit));
    const data = await relayFetch(`/v1/events?${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_data_quality",
  "Get data quality signals — completeness, staleness, missing collectors",
  {},
  async () => {
    const data = await relayFetch("/v1/context/data-quality");
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_agent_effectiveness",
  "Get agent effectiveness score — delivery rate, acceptance rate, behavior change rate",
  {
    agent_id: z.string().describe("Agent identifier to evaluate"),
  },
  async ({ agent_id }) => {
    const data = await relayFetch(`/v1/agents/${encodeURIComponent(agent_id)}/effectiveness`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_habit_metrics",
  "Get habit metrics — streaks, habit strength, completion rates",
  {},
  async () => {
    const data = await relayFetch("/v1/habits");
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_baseline_comparison",
  "Get baseline metrics for comparison with current performance",
  {},
  async () => {
    const data = await relayFetch("/v1/baselines");
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

// ============================================================
// Preferences Tools (P1: configurable alignment weights)
// ============================================================

server.tool(
  "get_preferences",
  "Get user preferences — alignment weights, classification overrides, and custom settings",
  {
    category: z.string().optional().describe("Filter by preference category (e.g., 'alignment', 'classification')"),
  },
  async ({ category }) => {
    const params = category ? `?category=${category}` : "";
    const data = await relayFetch(`/v1/preferences${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "set_preferences",
  "Set user preferences — supports array format [{category,key,value}] or category+map format {category, preferences:{key:value}}",
  {
    preferences: z.array(z.object({
      category: z.string().describe("Preference category"),
      key: z.string().describe("Preference key"),
      value: z.string().describe("Preference value"),
    })).optional().describe("Array of preferences to upsert"),
    category: z.string().optional().describe("Category for map format"),
    preferences_map: z.record(z.any()).optional().describe("Key-value map for category+map format"),
  },
  async ({ preferences, category, preferences_map }) => {
    let body;
    if (preferences && preferences.length > 0) {
      body = { preferences };
    } else if (category && preferences_map) {
      body = { category, preferences: preferences_map };
    } else {
      return { content: [{ type: "text", text: "Error: provide either 'preferences' array or 'category' + 'preferences_map'" }] };
    }
    const data = await relayFetch("/v1/preferences", {
      method: "PUT",
      body: JSON.stringify(body),
    });
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "delete_preference",
  "Delete a single user preference by category and key",
  {
    category: z.string().describe("Preference category"),
    key: z.string().describe("Preference key to delete"),
  },
  async ({ category, key }) => {
    const data = await relayFetch(`/v1/preferences/${encodeURIComponent(category)}/${encodeURIComponent(key)}`, {
      method: "DELETE",
    });
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

// ============================================================
// Correction Feedback Tools (P1: feedback loop)
// ============================================================

server.tool(
  "get_correction_feedback",
  "Get correction feedback statistics — label correction counts, correction rate, suggested classification adjustments",
  {
    limit: z.number().optional().describe("Max corrections to analyze (default 100)"),
  },
  async ({ limit }) => {
    const params = limit ? `?limit=${limit}` : "";
    const data = await relayFetch(`/v1/corrections/feedback${params}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "apply_correction_feedback",
  "Apply a label correction to a segment and record it for the classification feedback loop",
  {
    segment_id: z.string().describe("Segment ID to correct"),
    new_label: z.string().describe("New label for the segment"),
    old_label: z.string().optional().describe("Old label (auto-detected if not provided)"),
    reason: z.string().optional().describe("Reason for the correction"),
  },
  async ({ segment_id, new_label, old_label, reason }) => {
    const body = { segment_id, new_label };
    if (old_label) body.old_label = old_label;
    if (reason) body.reason = reason;
    const data = await relayFetch("/v1/corrections/feedback", {
      method: "POST",
      body: JSON.stringify(body),
    });
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

// ============================================================
// Intervention Lifecycle Tools
// ============================================================

server.tool(
  "get_pending_interventions",
  "Get pending interventions for the device — shows interventions waiting to be delivered to the user",
  {},
  async () => {
    const data = await relayFetch("/v1/device/interventions");
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

server.tool(
  "get_interventions",
  "List all interventions for the agent with optional status filter (proposed, approved_by_policy, delivered, accepted, dismissed, expired)",
  {
    status: z.string().optional().describe("Filter by intervention status"),
    limit: z.number().optional().describe("Max results (default 50, max 200)"),
  },
  async ({ status, limit }) => {
    const params = new URLSearchParams();
    if (status) params.set("status", status);
    if (limit) params.set("limit", String(limit));
    const qs = params.toString();
    const data = await relayFetch(`/v1/agent/interventions${qs ? "?" + qs : ""}`);
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  }
);

// ============================================================
// Start the server
// ============================================================

async function main() {
  if (!AGENT_TOKEN) {
    console.error(
      "[andrometry-mcp] WARNING: ANDROMETRY_AGENT_TOKEN not set. Set it to authenticate with the relay."
    );
  }

  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("[andrometry-mcp] MCP server running on stdio");
  console.error(`[andrometry-mcp] Relay URL: ${RELAY_URL}`);
}

main().catch((err) => {
  console.error("[andrometry-mcp] Fatal error:", err);
  process.exit(1);
});
