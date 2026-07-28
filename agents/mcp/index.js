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
