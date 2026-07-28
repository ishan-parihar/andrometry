#!/usr/bin/env node
/**
 * Synthetic event generator for Andrometry testing.
 *
 * Generates a realistic day of events, segments, and context snapshots
 * that can be sent to the relay server via POST /v1/batch.
 *
 * Usage:
 *   node generate.js --token <device_token> --relay http://localhost:8420 --date 2026-07-28
 *   node generate.js --output synthetic_day.json
 */

import { writeFileSync } from "fs";
import { randomUUID } from "crypto";

const DEVICE_ID = "pixel-9-pro";
const CLIENT_VERSION = "0.1.0";
const SCHEMA_VERSION = 1;
const TIMEZONE = "Asia/Kolkata";
const UTC_OFFSET = 19800;

// ============================================================
// Helper functions
// ============================================================

function uuid() {
  return randomUUID();
}

function iso(date) {
  return date.toISOString();
}

function localISO(date) {
  // Convert to IST for display
  const offset = 5.5 * 60 * 60 * 1000;
  const local = new Date(date.getTime() + offset);
  return local.toISOString().replace("Z", "+05:30");
}

function dateOnly(date) {
  return date.toISOString().split("T")[0];
}

// ============================================================
// Synthetic day generator
// ============================================================

function generateDay(dateStr) {
  const baseDate = new Date(dateStr + "T00:00:00Z");
  const events = [];
  const segments = [];
  let sequence = 10000;

  function addEvent(type, timestamp, payload, collector = "synthetic", source = "inferred") {
    events.push({
      event_id: uuid(),
      device_id: DEVICE_ID,
      schema_version: SCHEMA_VERSION,
      event_version: 1,
      client_version: CLIENT_VERSION,
      timestamp: iso(new Date(timestamp)),
      timezone: TIMEZONE,
      utc_offset_seconds: UTC_OFFSET,
      event_type: type,
      collector,
      source,
      sequence: sequence++,
      payload,
    });
  }

  function addSegment(type, layer, start, end, label, confidence, payload = {}) {
    const segId = uuid();
    segments.push({
      segment_id: segId,
      device_id: DEVICE_ID,
      segment_type: type,
      layer,
      start_time: iso(new Date(start)),
      end_time: iso(new Date(end)),
      label,
      confidence,
      version: 1,
      valid_from: iso(new Date(start)),
      ingested_at: iso(new Date(start)),
      evidence: { synthetic: true },
      payload,
    });
    return segId;
  }

  // --- Boot ---
  const bootTime = new Date(baseDate);
  bootTime.setUTCHours(7, 5, 0, 0);
  addEvent("boot_completed", bootTime, { reason: "scheduled_reboot" }, "broadcast", "broadcast");

  // --- Screen on/unlock ---
  const firstUnlock = new Date(baseDate);
  firstUnlock.setUTCHours(7, 12, 0, 0);
  addEvent("device_unlocked", firstUnlock, { session_start: true }, "broadcast", "broadcast");
  addEvent("screen_state_changed", firstUnlock, { state: "on" }, "broadcast", "broadcast");

  // --- Morning routine (home) ---
  addSegment("place_visit", "place", firstUnlock, addMinutes(firstUnlock, 45), "home", 0.95);
  addSegment("activity_segment", "activity", firstUnlock, addMinutes(firstUnlock, 45), "break", 0.7, { source: "inferred_morning" });

  // --- Commute to office ---
  const commuteStart = addMinutes(firstUnlock, 45);
  addSegment("place_visit", "place", commuteStart, addMinutes(commuteStart, 25), "commute", 0.88);
  addSegment("commute_segment", "activity", commuteStart, addMinutes(commuteStart, 25), "commute", 0.88);

  // --- Office: Deep work session 1 ---
  const deepWork1Start = addMinutes(commuteStart, 25);
  addSegment("place_visit", "place", deepWork1Start, addMinutes(deepWork1Start, 180), "office", 0.93);
  addSegment("focus_segment", "focus", deepWork1Start, addMinutes(deepWork1Start, 180), "deep_work", 0.85, {
    interruptions: 2,
    distraction_minutes: 4,
    success_score: 0.82,
    source: "manual",
  });
  addSegment("activity_segment", "activity", deepWork1Start, addMinutes(deepWork1Start, 180), "deep_work", 0.85);

  // App usage during deep work
  addEvent("app_usage_summary", deepWork1Start, {
    by_category: { productivity: 680, communication: 60, browser: 40 },
    top_apps: [
      { package: "com.notion.id", category: "productivity", seconds: 420 },
      { package: "com.slack", category: "communication", seconds: 60 },
      { package: "com.google.chrome", category: "browser", seconds: 40 },
    ],
    app_switch_count: 12,
  }, "usage_stats", "workmanager");

  // --- Meeting ---
  const meetingStart = addMinutes(deepWork1Start, 180);
  addSegment("meeting_segment", "calendar", meetingStart, addMinutes(meetingStart, 45), "meeting", 0.88, {
    event_title_category: "meeting",
    attendee_count: 4,
  });
  addSegment("activity_segment", "activity", meetingStart, addMinutes(meetingStart, 45), "meeting", 0.88);

  // --- Lunch break ---
  const lunchStart = addMinutes(meetingStart, 45);
  addSegment("break_segment", "activity", lunchStart, addMinutes(lunchStart, 40), "break", 0.9);
  addSegment("place_visit", "place", lunchStart, addMinutes(lunchStart, 40), "cafe", 0.7);

  // --- Deep work session 2 ---
  const deepWork2Start = addMinutes(lunchStart, 40);
  addSegment("focus_segment", "focus", deepWork2Start, addMinutes(deepWork2Start, 150), "deep_work", 0.78, {
    interruptions: 5,
    distraction_minutes: 12,
    success_score: 0.68,
    source: "inferred",
  });
  addSegment("activity_segment", "activity", deepWork2Start, addMinutes(deepWork2Start, 150), "shallow_work", 0.72);

  // Distraction during session 2
  const distractionStart = addMinutes(deepWork2Start, 90);
  addSegment("distraction_segment", "activity", distractionStart, addMinutes(distractionStart, 8), "social", 0.8, {
    app: "com.twitter.android",
    category: "social",
  });

  // --- Health data ---
  addEvent("health_summary", addMinutes(deepWork1Start, 60), {
    steps: 210,
    active_minutes: 7,
    heart_rate_avg: 72,
    heart_rate_min: 62,
    heart_rate_max: 92,
  }, "health_connect", "workmanager");

  // --- Sleep summary (previous night) ---
  const sleepStart = new Date(baseDate);
  sleepStart.setUTCDate(sleepStart.getUTCDate() - 1);
  sleepStart.setUTCHours(23, 41, 0, 0);
  const wakeTime = new Date(baseDate);
  wakeTime.setUTCHours(7, 21, 0, 0);
  addEvent("sleep_summary", wakeTime, {
    date: dateOnly(new Date(baseDate.getTime() - 86400000)),
    sleep_minutes: 460,
    bedtime: iso(sleepStart),
    wake_time: iso(wakeTime),
    regularity_score: 0.82,
    in_bed_minutes: 480,
  }, "health_connect", "workmanager");

  // --- Notification summary ---
  addEvent("notification_summary", addMinutes(deepWork1Start, 30), {
    total: 14,
    by_category: { communication: 8, work: 4, social: 2 },
    interruption_level: "medium",
    response_latency_seconds_avg: 84,
  }, "notification_listener", "workmanager");

  // --- Location context ---
  addEvent("location_context", deepWork1Start, {
    place: "office",
    confidence: 0.93,
    movement: "still",
    changed: false,
  }, "geofence", "geofence");

  // --- Calendar context ---
  addEvent("calendar_context", deepWork1Start, {
    current_event: "none",
    next_event_in_minutes: 180,
    next_event_category: "meeting",
    busy_until: iso(addMinutes(meetingStart, 45)),
  }, "content_observer", "content_observer");

  // --- Battery state ---
  addEvent("battery_state_changed", firstUnlock, {
    level: 0.92,
    charging: false,
  }, "broadcast", "broadcast");

  addEvent("battery_state_changed", addMinutes(deepWork1Start, 120), {
    level: 0.74,
    charging: false,
  }, "broadcast", "broadcast");

  // --- Permission state ---
  addEvent("permission_state_changed", firstUnlock, {
    usage_stats: true,
    notification_listener: true,
    notification_listener_restricted: false,
    calendar: true,
    location: true,
    background_location: true,
    health_connect: true,
    health_connect_steps: true,
    health_connect_sleep: true,
    health_connect_heart_rate: false,
    battery_optimization_exempt: true,
  }, "manual", "manual");

  // --- Context snapshot ---
  const contextTime = addMinutes(deepWork1Start, 37);
  const currentContext = {
    primary_activity: "deep_work",
    activity_confidence: 0.81,
    place: "office",
    place_confidence: 0.92,
    movement: "still",
    foreground_app_category: "productivity",
    screen_state: "on",
    minutes_since_unlock: 37,
    focus_active: true,
    focus_minutes_elapsed: 37,
    focus_source: "manual",
    interruption_rate: "low",
    notifications_last_15m: 3,
    distraction_probability: 0.18,
    fatigue_probability: 0.31,
    alignment_probability: 0.74,
    readiness_score: 0.68,
    calendar: {
      current_event: "none",
      next_event_minutes: 143,
      busy_until: iso(addMinutes(meetingStart, 45)),
    },
    health: {
      steps_last_hour: 210,
      activity: "still",
      heart_rate_avg: 72,
    },
    sleep: {
      last_sleep_minutes: 460,
      sleep_debt_estimate_minutes: 35,
    },
    battery: {
      level: 0.74,
      charging: false,
    },
    data_quality: {
      completeness: 0.87,
      staleness_seconds: 42,
      usage_stats_available: true,
      notification_listener_available: true,
      calendar_available: true,
      location_fresh: true,
      health_connect_available: true,
      last_sync_minutes_ago: 7,
      missing_signals: [],
      missing_collectors: [],
      degraded_collectors: ["health_connect"],
      permission_issues: [],
      clock_drift_detected: false,
    },
    timezone: TIMEZONE,
  };

  // --- Daily summary ---
  const dailySummary = {
    date: dateStr,
    timezone: TIMEZONE,
    sleep: {
      minutes: 460,
      regularity_score: 0.82,
      bedtime: iso(sleepStart),
      wake_time: iso(wakeTime),
    },
    health: {
      steps: 8123,
      exercise_minutes: 35,
      active_minutes: 58,
    },
    work: {
      deep_work_minutes: 182,
      shallow_work_minutes: 96,
      meeting_minutes: 95,
      admin_minutes: 41,
    },
    distraction: {
      total_minutes: 64,
      social_minutes: 38,
      browser_hopping_minutes: 14,
      during_focus_minutes: 11,
    },
    interruptions: {
      notification_count: 143,
      unlock_count: 58,
      app_switch_count: 312,
    },
    location: {
      home_minutes: 620,
      office_minutes: 510,
      gym_minutes: 55,
      commute_minutes: 70,
    },
    focus: {
      sessions: 3,
      successful_sessions: 2,
      average_interruptions: 2.3,
    },
    work_patterns: {
      average_focus_bout_minutes: 41,
      switching_rate_per_hour: 18,
      meeting_fragmentation_score: 0.43,
      late_night_work_minutes: 22,
    },
    scores: {
      alignment: 0.74,
      productivity: 0.69,
      wellbeing: 0.71,
      focus_control: 0.66,
      sleep_quality: 0.78,
      readiness: 0.68,
    },
    baselines: {
      deep_work_minutes_7d_avg: 158,
      distraction_minutes_7d_avg: 71,
      sleep_minutes_7d_avg: 442,
    },
  };

  return {
    batch: {
      device_id: DEVICE_ID,
      batch_id: uuid(),
      client_version: CLIENT_VERSION,
      schema_version: SCHEMA_VERSION,
      sequence_start: 10000,
      sequence_end: sequence - 1,
      events,
      segments,
      summaries: [dailySummary],
      context: currentContext,
      device_state: {
        battery: 0.74,
        charging: false,
        storage_free_mb: 42000,
        queue_size: events.length + segments.length,
      },
    },
    currentContext,
    dailySummary,
    eventCount: events.length,
    segmentCount: segments.length,
  };
}

function addMinutes(date, minutes) {
  return new Date(date.getTime() + minutes * 60 * 1000);
}

// ============================================================
// CLI
// ============================================================

const args = process.argv.slice(2);
const dateStr =
  args.find((a) => a.startsWith("--date="))?.split("=")[1] ||
  new Date().toISOString().split("T")[0];
const token = args.find((a) => a.startsWith("--token="))?.split("=")[1];
const relay = args.find((a) => a.startsWith("--relay="))?.split("=")[1];
const outputFile = args.find((a) => a.startsWith("--output="))?.split("=")[1];

const { batch, eventCount, segmentCount } = generateDay(dateStr);

// Log to stderr so stdout is clean JSON
console.error(`[synthetic] Generating day: ${dateStr}`);
console.error(`[synthetic] Generated ${eventCount} events, ${segmentCount} segments`);

if (outputFile) {
  writeFileSync(outputFile, JSON.stringify(batch, null, 2));
  console.error(`[synthetic] Written to ${outputFile}`);
}

if (token && relay) {
  console.error(`[synthetic] Sending batch to ${relay}/v1/batch ...`);
  const resp = await fetch(`${relay}/v1/batch`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(batch),
  });
  const result = await resp.json();
  console.error(`[synthetic] Response:`, JSON.stringify(result, null, 2));
} else if (!outputFile) {
  // Output clean JSON to stdout
  console.log(JSON.stringify(batch, null, 2));
}
