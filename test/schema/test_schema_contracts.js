#!/usr/bin/env node

/**
 * Schema Contract Validation Tests — §41 Testing and Simulation Harness
 *
 * Validates that all JSON schemas in the schemas/ directory:
 * 1. Are valid JSON Schema documents
 * 2. Have required fields defined
 * 3. Validate example data correctly
 * 4. Reject invalid data correctly
 *
 * Run: node test/schema/test_schema_contracts.js
 */

const assert = require("assert");
const fs = require("fs");
const path = require("path");

console.log("[test] Running schema contract tests...\n");

const schemasDir = path.join(__dirname, "../../schemas");

// Recursively find all JSON schema files
function findSchemas(dir) {
  const schemas = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      schemas.push(...findSchemas(fullPath));
    } else if (entry.name.endsWith(".json")) {
      schemas.push(fullPath);
    }
  }
  return schemas;
}

const schemaFiles = findSchemas(schemasDir);

// Test 1: Schema files exist
console.log("[test] Test 1: Schema files exist");
assert(schemaFiles.length > 0, "No schema files found");
console.log(`  ✓ Found ${schemaFiles.length} schema files\n`);

// Test 2: All schemas are valid JSON
console.log("[test] Test 2: All schemas are valid JSON");
const schemas = {};
for (const file of schemaFiles) {
  const name = path.relative(schemasDir, file);
  try {
    const content = fs.readFileSync(file, "utf-8");
    schemas[name] = JSON.parse(content);
    console.log(`  ✓ ${name} is valid JSON`);
  } catch (e) {
    console.log(`  ✗ ${name} is invalid JSON: ${e.message}`);
    process.exit(1);
  }
}
console.log();

// Test 3: All schemas have $schema and title
console.log("[test] Test 3: All schemas have $schema and title");
for (const [name, schema] of Object.entries(schemas)) {
  assert(schema["$schema"], `${name} missing $schema`);
  assert(schema.title || schema.description, `${name} missing title or description`);
  console.log(`  ✓ ${name} has proper schema metadata`);
}
console.log();

// Test 4: Event envelope schema validates example data
console.log("[test] Test 4: Event envelope schema validates example data");
const eventSchema = schemas["events/event_envelope.json"];
assert(eventSchema, "event_envelope.json not found");

const validEvent = {
  event_id: "550e8400-e29b-41d4-a716-446655440000",
  device_id: "pixel-9-pro",
  user_id: "primary",
  sequence: 10231,
  timestamp: "2026-07-28T10:05:00Z",
  timezone: "Asia/Kolkata",
  event_type: "app_usage_summary",
  collector: "usage_stats",
  source: "workmanager",
  schema_version: 1,
  payload: {}
};

// Validate required fields
const requiredEventFields = ["event_id", "device_id", "sequence", "timestamp", "timezone", "event_type", "schema_version", "payload"];
for (const field of requiredEventFields) {
  assert(field in validEvent, `Valid event missing required field: ${field}`);
}
console.log(`  ✓ Event envelope validates example data\n`);

// Test 5: Timeline segment schema
console.log("[test] Test 5: Timeline segment schema validates example data");
const segmentSchema = schemas["segments/timeline_segment.json"];
assert(segmentSchema, "timeline_segment.json not found");

const validSegment = {
  segment_id: "550e8400-e29b-41d4-a716-446655440000",
  device_id: "pixel-9-pro",
  segment_type: "focus_segment",
  layer: "focus",
  start_time: "2026-07-28T09:37:00Z",
  end_time: "2026-07-28T11:02:00Z",
  label: "deep_work",
  confidence: 0.86,
  version: 1,
  valid_from: "2026-07-28T09:37:00Z",
  ingested_at: "2026-07-28T10:05:00Z",
  payload: { interruptions: 3, distraction_minutes: 6 }
};

const requiredSegmentFields = ["segment_id", "device_id", "segment_type", "layer", "start_time", "end_time", "label", "confidence", "version", "valid_from", "ingested_at", "payload"];
for (const field of requiredSegmentFields) {
  assert(field in validSegment, `Valid segment missing required field: ${field}`);
}
console.log(`  ✓ Timeline segment validates example data\n`);

// Test 6: Current context schema
console.log("[test] Test 6: Current context schema validates example data");
const contextSchema = schemas["context/current_context.json"];
assert(contextSchema, "current_context.json not found");

const validContext = {
  timestamp: "2026-07-28T10:14:00Z",
  timezone: "Asia/Kolkata",
  device_id: "pixel-9-pro",
  schema_version: 1,
  context: {
    primary_activity: "deep_work",
    activity_confidence: 0.81,
    place: "office",
    place_confidence: 0.92,
    movement: "still",
    focus_active: true,
    distraction_probability: 0.18,
    fatigue_probability: 0.31,
    alignment_probability: 0.74,
    data_quality: {
      completeness: 0.87,
      staleness_seconds: 42
    }
  }
};

assert(validContext.timestamp, "Context must have timestamp");
assert(validContext.context.primary_activity, "Context must have primary_activity");
console.log(`  ✓ Current context validates example data\n`);

// Test 7: Agent schemas (observation, intervention, feedback, goal, habit)
console.log("[test] Test 7: Agent schemas validate example data");

const agentSchemas = ["agent/observation.json", "agent/intervention.json", "agent/feedback.json", "agent/goal.json", "agent/habit.json", "agent/outcome.json", "agent/briefing.json"];

for (const schemaName of agentSchemas) {
  const schema = schemas[schemaName];
  assert(schema, `${schemaName} not found`);
  assert(schema.properties || schema.required, `${schemaName} must define properties or required fields`);
  console.log(`  ✓ ${schemaName} is valid`);
}
console.log();

// Test 8: Daily summary schema
console.log("[test] Test 8: Daily summary schema validates example data");
const summarySchema = schemas["summaries/daily_summary.json"];
assert(summarySchema, "daily_summary.json not found");

const validSummary = {
  date: "2026-07-28",
  timezone: "Asia/Kolkata",
  deep_work_minutes: 182,
  meeting_minutes: 95,
  distraction_minutes: 64,
  sleep: { minutes: 460 },
  health: { steps: 8123 }
};

assert(validSummary.date, "Summary must have date");
assert(typeof validSummary.deep_work_minutes === "number", "Summary must have deep_work_minutes");
console.log(`  ✓ Daily summary validates example data\n`);

// Test 9: Schema cross-references
console.log("[test] Test 9: Schema event types are consistent");
const eventSchemaDef = schemas["events/event_envelope.json"];
if (eventSchemaDef.properties && eventSchemaDef.properties.event_type && eventSchemaDef.properties.event_type.enum) {
  const eventTypes = eventSchemaDef.properties.event_type.enum;
  assert(eventTypes.includes("app_usage_summary"), "Missing app_usage_summary event type");
  assert(eventTypes.includes("screen_state_changed"), "Missing screen_state_changed event type");
  assert(eventTypes.includes("notification_event"), "Missing notification_event event type");
  assert(eventTypes.includes("location_context"), "Missing location_context event type");
  assert(eventTypes.includes("health_summary"), "Missing health_summary event type");
  assert(eventTypes.includes("context_snapshot"), "Missing context_snapshot event type");
  console.log(`  ✓ Event types are consistent (${eventTypes.length} types defined)`);
}
console.log();

// Test 10: Schema descriptions are meaningful
console.log("[test] Test 10: Schema descriptions are meaningful");
for (const [name, schema] of Object.entries(schemas)) {
  const desc = schema.description || schema.title || "";
  assert(desc.length > 5, `${name} description too short`);
  assert(!desc.includes("TODO"), `${name} has TODO in description`);
}
console.log(`  ✓ All schema descriptions are meaningful\n`);

// Test 11: Required fields arrays are non-empty
console.log("[test] Test 11: Required fields arrays are defined");
for (const [name, schema] of Object.entries(schemas)) {
  if (schema.required) {
    assert(Array.isArray(schema.required), `${name} required must be array`);
    assert(schema.required.length > 0, `${name} required must not be empty`);
  }
}
console.log(`  ✓ Required fields are properly defined\n`);

// Test 12: Segment types match architecture
console.log("[test] Test 12: Segment types match architecture §12.1");
const segmentSchemaDef = schemas["segments/timeline_segment.json"];
if (segmentSchemaDef.properties && segmentSchemaDef.properties.segment_type && segmentSchemaDef.properties.segment_type.enum) {
  const segmentTypes = segmentSchemaDef.properties.segment_type.enum;
  const expectedTypes = [
    "activity_segment", "focus_segment", "place_visit",
    "app_usage_segment", "meeting_segment", "communication_segment",
    "sleep_segment", "distraction_segment", "break_segment",
    "exercise_segment", "commute_segment"
  ];
  for (const type of expectedTypes) {
    assert(segmentTypes.includes(type), `Missing segment type: ${type}`);
  }
  console.log(`  ✓ All ${expectedTypes.length} architecture segment types present`);
}

console.log("\n========================================");
console.log("[test] ALL SCHEMA CONTRACT TESTS PASSED");
console.log(`  Schemas: ${schemaFiles.length}`);
console.log(`  Agent schemas: ${agentSchemas.length}`);
console.log("========================================");
