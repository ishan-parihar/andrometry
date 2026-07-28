#!/usr/bin/env node
/**
 * Tests for the synthetic event generator.
 * Validates that generated batches conform to the event envelope schema.
 */

import { strict as assert } from "assert";
import { readFileSync } from "fs";
import { execSync } from "child_process";

console.log("[test] Running synthetic generator tests...\n");

// Run the generator and capture output
let batch;
try {
  const output = execSync("node scripts/generate.js --date=2026-07-28", {
    cwd: process.cwd(),
    encoding: "utf-8",
  });
  batch = JSON.parse(output);
} catch (e) {
  console.error("[test] Failed to run generator:", e.message);
  process.exit(1);
}

// ============================================================
// Test 1: Batch structure
// ============================================================
console.log("[test] Test 1: Batch structure");

assert(batch.device_id, "batch must have device_id");
assert(batch.batch_id, "batch must have batch_id");
assert(batch.client_version, "batch must have client_version");
assert(typeof batch.schema_version === "number", "batch must have schema_version");
assert(typeof batch.sequence_start === "number", "batch must have sequence_start");
assert(typeof batch.sequence_end === "number", "batch must have sequence_end");
assert(Array.isArray(batch.events), "batch must have events array");
assert(Array.isArray(batch.segments), "batch must have segments array");
assert(Array.isArray(batch.summaries), "batch must have summaries array");
assert(batch.context, "batch must have context");
assert(batch.device_state, "batch must have device_state");

console.log("  ✓ Batch structure is valid\n");

// ============================================================
// Test 2: Event envelope fields
// ============================================================
console.log("[test] Test 2: Event envelope fields");

const requiredEventFields = [
  "event_id", "device_id", "schema_version", "event_version",
  "client_version", "timestamp", "timezone", "utc_offset_seconds",
  "event_type", "collector", "source", "sequence", "payload"
];

for (const event of batch.events) {
  for (const field of requiredEventFields) {
    assert(field in event, `Event ${event.event_id} missing field: ${field}`);
  }
  assert(typeof event.event_id === "string" && event.event_id.length > 0, "event_id must be non-empty string");
  assert(typeof event.sequence === "number", "sequence must be a number");
  assert(typeof event.payload === "object", "payload must be an object");
}

console.log(`  ✓ All ${batch.events.length} events have valid envelope\n`);

// ============================================================
// Test 3: Sequence monotonicity
// ============================================================
console.log("[test] Test 3: Sequence monotonicity");

const sequences = batch.events.map(e => e.sequence);
for (let i = 1; i < sequences.length; i++) {
  assert(sequences[i] > sequences[i-1], `Sequence not monotonic: ${sequences[i-1]} >= ${sequences[i]}`);
}

console.log("  ✓ Sequences are monotonically increasing\n");

// ============================================================
// Test 4: No duplicate event IDs
// ============================================================
console.log("[test] Test 4: No duplicate event IDs");

const eventIds = new Set(batch.events.map(e => e.event_id));
assert(eventIds.size === batch.events.length, "Duplicate event IDs found");

console.log("  ✓ No duplicate event IDs\n");

// ============================================================
// Test 5: Segment structure
// ============================================================
console.log("[test] Test 5: Segment structure");

const requiredSegmentFields = [
  "segment_id", "device_id", "segment_type", "layer",
  "start_time", "end_time", "label", "confidence",
  "version", "valid_from", "ingested_at", "evidence", "payload"
];

const validSegmentTypes = [
  "activity_segment", "focus_segment", "place_visit",
  "app_usage_segment", "meeting_segment", "communication_segment",
  "sleep_segment", "distraction_segment", "break_segment",
  "exercise_segment", "commute_segment"
];

const validLayers = ["activity", "place", "focus", "communication", "health", "calendar", "device"];

for (const seg of batch.segments) {
  for (const field of requiredSegmentFields) {
    assert(field in seg, `Segment ${seg.segment_id} missing field: ${field}`);
  }
  assert(validSegmentTypes.includes(seg.segment_type), `Invalid segment type: ${seg.segment_type}`);
  assert(validLayers.includes(seg.layer), `Invalid layer: ${seg.layer}`);
  assert(typeof seg.confidence === "number" && seg.confidence >= 0 && seg.confidence <= 1, "confidence must be 0-1");
  assert(new Date(seg.start_time) < new Date(seg.end_time), "start_time must be before end_time");
}

console.log(`  ✓ All ${batch.segments.length} segments are valid\n`);

// ============================================================
// Test 6: Segment type coverage
// ============================================================
console.log("[test] Test 6: Segment type coverage");

const segmentTypes = new Set(batch.segments.map(s => s.segment_type));
const expectedTypes = ["activity_segment", "focus_segment", "place_visit", "meeting_segment", "break_segment"];
for (const type of expectedTypes) {
  assert(segmentTypes.has(type), `Missing expected segment type: ${type}`);
}

console.log(`  ✓ Segment types covered: ${[...segmentTypes].join(", ")}\n`);

// ============================================================
// Test 7: Daily summary structure
// ============================================================
console.log("[test] Test 7: Daily summary structure");

assert(batch.summaries.length > 0, "Must have at least one summary");
const summary = batch.summaries[0];
assert(summary.date, "Summary must have date");
assert(summary.timezone, "Summary must have timezone");
assert(summary.sleep, "Summary must have sleep");
assert(summary.work, "Summary must have work");
assert(summary.distraction, "Summary must have distraction");
assert(summary.focus, "Summary must have focus");
assert(summary.scores, "Summary must have scores");

console.log("  ✓ Daily summary structure is valid\n");

// ============================================================
// Test 8: Context object
// ============================================================
console.log("[test] Test 8: Context object");

const ctx = batch.context;
assert(ctx.primary_activity, "Context must have primary_activity");
assert(typeof ctx.activity_confidence === "number", "Context must have activity_confidence");
assert(ctx.place, "Context must have place");
assert(typeof ctx.distraction_probability === "number", "Context must have distraction_probability");
assert(typeof ctx.fatigue_probability === "number", "Context must have fatigue_probability");
assert(typeof ctx.alignment_probability === "number", "Context must have alignment_probability");
assert(ctx.data_quality, "Context must have data_quality");
assert(typeof ctx.data_quality.completeness === "number", "Data quality must have completeness");

console.log("  ✓ Context object is valid\n");

// ============================================================
// Test 9: Timezone handling
// ============================================================
console.log("[test] Test 9: Timezone handling");

assert(batch.events[0].timezone === "Asia/Kolkata", "Default timezone should be Asia/Kolkata");
assert(batch.events[0].utc_offset_seconds === 19800, "UTC offset should be 19800 for IST");

console.log("  ✓ Timezone handling is correct\n");

// ============================================================
// Test 10: Event type diversity
// ============================================================
console.log("[test] Test 10: Event type diversity");

const eventTypes = new Set(batch.events.map(e => e.event_type));
const expectedEventTypes = [
  "boot_completed", "screen_state_changed", "device_unlocked",
  "app_usage_summary", "notification_summary", "health_summary",
  "sleep_summary", "battery_state_changed", "permission_state_changed",
  "location_context", "calendar_context"
];

let found = 0;
for (const type of expectedEventTypes) {
  if (eventTypes.has(type)) found++;
}

assert(found >= 8, `Expected at least 8 of ${expectedEventTypes.length} event types, got ${found}`);

console.log(`  ✓ Event type diversity: ${found}/${expectedEventTypes.length} expected types present\n`);

// ============================================================
// Summary
// ============================================================
console.log("========================================");
console.log(`[test] ALL TESTS PASSED`);
console.log(`  Events: ${batch.events.length}`);
console.log(`  Segments: ${batch.segments.length}`);
console.log(`  Summaries: ${batch.summaries.length}`);
console.log(`  Sequence range: ${batch.sequence_start}-${batch.sequence_end}`);
console.log("========================================");
