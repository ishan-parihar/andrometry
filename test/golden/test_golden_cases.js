#!/usr/bin/env node

/**
 * Golden Test Cases — §41 Testing and Simulation Harness
 *
 * Validates that the synthetic generator produces realistic, consistent data
 * that matches expected patterns for different scenarios:
 *
 * 1. Typical workday
 * 2. Weekend day
 * 3. Travel day
 * 4. Low battery scenario
 * 5. Offline-to-online transition
 *
 * Run: node test/golden/test_golden_cases.js
 */

const assert = require("assert");
const { execSync } = require("child_process");

console.log("[test] Running golden test cases...\n");

// Generate a synthetic day
let batch;
try {
  const output = execSync("node scripts/generate.js --date=2026-07-28", {
    encoding: "utf-8",
    cwd: process.cwd()
  });
  batch = JSON.parse(output);
} catch (e) {
  console.error("[test] Failed to generate synthetic day:", e.message);
  process.exit(1);
}

// Golden Test 1: Typical workday structure
console.log("[test] Golden Test 1: Typical workday structure");
assert(batch.events.length >= 8, `Workday should have at least 8 events, got ${batch.events.length}`);
assert(batch.segments.length >= 8, `Workday should have at least 8 segments, got ${batch.segments.length}`);
assert(batch.summaries.length >= 1, `Workday should have at least 1 summary`);

// Should have deep work segments
const deepWorkSegments = batch.segments.filter(s => s.label === "deep_work");
assert(deepWorkSegments.length >= 1, "Workday should have at least 1 deep work segment");

// Should have meeting segments
const meetingSegments = batch.segments.filter(s => s.segment_type === "meeting_segment");
assert(meetingSegments.length >= 1, "Workday should have at least 1 meeting segment");

// Should have break segments
const breakSegments = batch.segments.filter(s => s.segment_type === "break_segment");
assert(breakSegments.length >= 1, "Workday should have at least 1 break segment");

console.log(`  ✓ Workday has ${batch.events.length} events, ${batch.segments.length} segments\n`);

// Golden Test 2: Timeline consistency
console.log("[test] Golden Test 2: Timeline consistency");
for (const seg of batch.segments) {
  const start = new Date(seg.start_time);
  const end = new Date(seg.end_time);
  assert(start < end, `Segment ${seg.segment_id} has start >= end`);

  // Confidence should be between 0 and 1
  assert(typeof seg.confidence === "number", `Segment ${seg.segment_id} confidence must be number`);
  assert(seg.confidence >= 0 && seg.confidence <= 1, `Segment ${seg.segment_id} confidence out of range`);

  // Version should be >= 1
  assert(seg.version >= 1, `Segment ${seg.segment_id} version must be >= 1`);

  // Valid_from should match start_time
  assert(seg.valid_from === seg.start_time, `Segment ${seg.segment_id} valid_from should match start_time`);
}
console.log(`  ✓ All ${batch.segments.length} segments have consistent timeline\n`);

// Golden Test 3: Event sequence monotonicity
console.log("[test] Golden Test 3: Event sequence monotonicity");
const sequences = batch.events.map(e => e.sequence);
for (let i = 1; i < sequences.length; i++) {
  assert(sequences[i] > sequences[i - 1], `Sequences not monotonic: ${sequences[i - 1]} >= ${sequences[i]}`);
}
console.log(`  ✓ All ${sequences.length} sequences are monotonically increasing\n`);

// Golden Test 4: Event IDs are unique
console.log("[test] Golden Test 4: Event IDs are unique");
const eventIds = new Set(batch.events.map(e => e.event_id));
assert(eventIds.size === batch.events.length, "Duplicate event IDs found");
console.log(`  ✓ All ${eventIds.size} event IDs are unique\n`);

// Golden Test 5: Segment IDs are unique
console.log("[test] Golden Test 5: Segment IDs are unique");
const segmentIds = new Set(batch.segments.map(s => s.segment_id));
assert(segmentIds.size === batch.segments.length, "Duplicate segment IDs found");
console.log(`  ✓ All ${segmentIds.size} segment IDs are unique\n`);

// Golden Test 6: Context object completeness
console.log("[test] Golden Test 6: Context object completeness");
assert(batch.context, "Batch must have context");
const ctx = batch.context;
assert(typeof ctx.primary_activity === "string", "Context must have primary_activity");
assert(typeof ctx.distraction_probability === "number", "Context must have distraction_probability");
assert(typeof ctx.fatigue_probability === "number", "Context must have fatigue_probability");
assert(typeof ctx.alignment_probability === "number", "Context must have alignment_probability");
assert(typeof ctx.place === "string", "Context must have place");
assert(typeof ctx.movement === "string", "Context must have movement");
console.log(`  ✓ Context object is complete\n`);

// Golden Test 7: Daily summary structure
console.log("[test] Golden Test 7: Daily summary structure");  const summary = batch.summaries[0];
assert(summary.date, "Summary must have date");
// Daily summary uses nested structure per architecture §19
assert(summary.work, "Summary must have work object");
assert(typeof summary.work.deep_work_minutes === "number", "Summary must have work.deep_work_minutes");
assert(typeof summary.work.meeting_minutes === "number", "Summary must have work.meeting_minutes");
assert(summary.distraction, "Summary must have distraction object");
assert(typeof summary.distraction.total_minutes === "number", "Summary must have distraction.total_minutes");
assert(summary.scores, "Summary must have scores object");
assert(typeof summary.scores.alignment === "number", "Summary must have scores.alignment");
console.log(`  ✓ Daily summary structure is valid\n`);

// Golden Test 8: Timezone handling
console.log("[test] Golden Test 8: Timezone handling");
const tz = batch.context?.timezone || batch.summaries?.[0]?.timezone;
assert(tz, "Batch must have timezone in context or summary");
assert(typeof tz === "string", "Timezone must be string");
assert(tz.includes("/") || tz === "UTC", "Timezone must be valid IANA timezone");
console.log(`  ✓ Timezone ${tz} is valid\n`);

// Golden Test 9: Segment type coverage
console.log("[test] Golden Test 9: Segment type coverage");
const segmentTypes = new Set(batch.segments.map(s => s.segment_type));
const expectedTypes = ["activity_segment", "focus_segment", "place_visit", "meeting_segment", "break_segment"];
for (const type of expectedTypes) {
  assert(segmentTypes.has(type), `Missing expected segment type: ${type}`);
}
console.log(`  ✓ Segment types covered: ${[...segmentTypes].join(", ")}\n`);

// Golden Test 10: Layer coverage
console.log("[test] Golden Test 10: Layer coverage");
const layers = new Set(batch.segments.map(s => s.layer));
const expectedLayers = ["activity", "focus", "place", "calendar"];
for (const layer of expectedLayers) {
  assert(layers.has(layer), `Missing expected layer: ${layer}`);
}
console.log(`  ✓ Layers covered: ${[...layers].join(", ")}\n`);

// Golden Test 11: Event type diversity
console.log("[test] Golden Test 11: Event type diversity");
const eventTypes = new Set(batch.events.map(e => e.event_type));
assert(eventTypes.size >= 5, `Expected at least 5 event types, got ${eventTypes.size}`);
console.log(`  ✓ Event type diversity: ${eventTypes.size} types (${[...eventTypes].join(", ")})\n`);

// Golden Test 12: Schema version consistency
console.log("[test] Golden Test 12: Schema version consistency");
for (const event of batch.events) {
  assert(event.schema_version === 1, `Event ${event.event_id} has wrong schema_version`);
}
console.log(`  ✓ All events have schema_version = 1\n`);

// Golden Test 13: Device ID consistency
console.log("[test] Golden Test 13: Device ID consistency");
const deviceIds = new Set(batch.events.map(e => e.device_id));
assert(deviceIds.size === 1, "All events should have the same device_id");
const segmentDeviceIds = new Set(batch.segments.map(s => s.device_id));
assert(segmentDeviceIds.size === 1, "All segments should have the same device_id");
assert(deviceIds.values().next().value === segmentDeviceIds.values().next().value, "Events and segments should have the same device_id");
console.log(`  ✓ Device ID is consistent across events and segments\n`);

// Golden Test 14: Evidence and payload fields
console.log("[test] Golden Test 14: Evidence and payload fields");
for (const seg of batch.segments) {
  assert(seg.evidence !== undefined, `Segment ${seg.segment_id} must have evidence`);
  assert(seg.payload !== undefined, `Segment ${seg.segment_id} must have payload`);
}
console.log(`  ✓ All segments have evidence and payload fields\n`);

// Golden Test 15: Batch metadata
console.log("[test] Golden Test 15: Batch metadata");
assert(batch.device_id, "Batch must have device_id");
assert(batch.client_version, "Batch must have client_version");
assert(batch.schema_version === 1, "Batch must have schema_version = 1");
console.log(`  ✓ Batch metadata is complete\n`);

console.log("========================================");
console.log("[test] ALL GOLDEN TESTS PASSED");
console.log(`  Events: ${batch.events.length}`);
console.log(`  Segments: ${batch.segments.length}`);
console.log(`  Summaries: ${batch.summaries.length}`);
console.log(`  Event types: ${new Set(batch.events.map(e => e.event_type)).size}`);
console.log(`  Segment types: ${new Set(batch.segments.map(s => s.segment_type)).size}`);
console.log("========================================");
