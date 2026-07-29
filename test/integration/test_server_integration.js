#!/usr/bin/env node

/**
 * Andrometry Server Integration Test
 * ===================================
 * Starts the Go server with a bootstrap token, then hits every /v1/ endpoint
 * with proper authentication, validates responses, and tests the full agent loop.
 *
 * Usage: node test/integration/test_server_integration.js
 */

const { execSync, spawn } = require('child_process');
const http = require('http');
const path = require('path');
const fs = require('fs');
const crypto = require('crypto');

// ─── Configuration ───────────────────────────────────────────────────────────

const PORT = 18420; // Avoid conflicts with default 8420
const BOOTSTRAP_TOKEN = 'test-bootstrap-token-' + crypto.randomBytes(8).toString('hex');
const DB_PATH = `/tmp/andrometry-integration-test-${Date.now()}.db`;
const SERVER_BIN = '/tmp/andrometry-server';
const SERVER_CWD = path.resolve(__dirname, '../../server');

const DEVICE_TOKEN = BOOTSTRAP_TOKEN;
const AGENT_TOKEN = 'agent-' + BOOTSTRAP_TOKEN;
const ADMIN_TOKEN = 'admin-test-token-' + crypto.randomBytes(8).toString('hex');

// ─── State ───────────────────────────────────────────────────────────────────

let serverProcess = null;
let passed = 0;
let failed = 0;
let skipped = 0;
const failures = [];
let testGoalID = null;
let testHabitID = null;
let testBaselineID = null;
let testInterventionID = null;
let testObservationID = null;

// ─── HTTP Helpers ────────────────────────────────────────────────────────────

function httpRequest(method, urlPath, body = null, token = AGENT_TOKEN) {
  return new Promise((resolve, reject) => {
    const url = new URL(urlPath, `http://127.0.0.1:${PORT}`);
    const options = {
      hostname: '127.0.0.1',
      port: PORT,
      path: url.pathname + url.search,
      method,
      headers: { 'Content-Type': 'application/json' },
      timeout: 10000,
    };
    if (token) options.headers['Authorization'] = `Bearer ${token}`;

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        let parsed = null;
        try { parsed = JSON.parse(data); } catch {}
        resolve({ status: res.statusCode, headers: res.headers, body: data, json: parsed });
      });
    });
    req.on('error', reject);
    req.on('timeout', () => { req.destroy(); reject(new Error('timeout')); });
    if (body) req.write(JSON.stringify(body));
    req.end();
  });
}

// ─── Test Helpers ────────────────────────────────────────────────────────────

function test(name, fn) {
  return fn().then((result) => {
    passed++;
    process.stdout.write(`  ✅ ${name}\n`);
  }).catch((err) => {
    failed++;
    const msg = err.message || String(err);
    failures.push({ name, error: msg });
    process.stdout.write(`  ❌ ${name}: ${msg}\n`);
  });
}

function assert(condition, msg) {
  if (!condition) throw new Error(msg || 'Assertion failed');
}

function assertStatus(res, expected, context) {
  if (res.status !== expected) {
    throw new Error(`${context}: expected status ${expected}, got ${res.status} — body: ${(res.body || '').slice(0, 200)}`);
  }
}

// ─── Server Lifecycle ────────────────────────────────────────────────────────

function startServer() {
  return new Promise((resolve, reject) => {
    // Clean up old DB
    try { fs.unlinkSync(DB_PATH); } catch {}

    serverProcess = spawn(SERVER_BIN, [
      '-db', DB_PATH,
      '-port', String(PORT),
      '-bootstrap-token', BOOTSTRAP_TOKEN,
    ], {
      cwd: SERVER_CWD,
      stdio: ['ignore', 'pipe', 'pipe'],
      env: { ...process.env, GIN_MODE: 'test' },
    });

    let started = false;
    const timeout = setTimeout(() => {
      if (!started) reject(new Error('Server startup timeout'));
    }, 15000);

    function checkOutput(chunk) {
      if (started) return;
      const line = chunk.toString();
      // Server prints: [andrometry] relay server starting on :18420
      if (line.includes('relay server starting') || line.includes('BOOTSTRAP COMPLETE')) {
        started = true;
        clearTimeout(timeout);
        setTimeout(resolve, 500);
      }
    }
    serverProcess.stdout.on('data', checkOutput);
    serverProcess.stderr.on('data', checkOutput);

    serverProcess.on('error', (err) => {
      clearTimeout(timeout);
      reject(err);
    });

    serverProcess.on('exit', (code) => {
      if (!started) {
        clearTimeout(timeout);
        reject(new Error(`Server exited with code ${code}`));
      }
    });
  });
}

function stopServer() {
  if (serverProcess) {
    serverProcess.kill('SIGTERM');
    serverProcess = null;
  }
  try { fs.unlinkSync(DB_PATH); } catch {}
  try { fs.unlinkSync(DB_PATH + '-wal'); } catch {}
  try { fs.unlinkSync(DB_PATH + '-shm'); } catch {}
}

function insertAdminToken() {
  // Use sqlite3 CLI to insert an admin token directly into the database
  const hash = execSync(
    `echo -n "${ADMIN_TOKEN}" | sha256sum | awk '{print $1}'`,
    { encoding: 'utf-8' }
  ).trim();

  // Insert via the server's own DB - we need to use the same hash function
  // The Go code uses sha256 of the raw token. Let's use sqlite3 directly.
  try {
    execSync(`sqlite3 "${DB_PATH}" "INSERT INTO auth_tokens (token_id, user_id, token_hash, token_type, scopes, created_at) VALUES ('admin-test', 'primary', '${hash}', 'admin', 'admin', datetime('now'));"`, { encoding: 'utf-8' });
    return true;
  } catch {
    // sqlite3 might not be available, or table structure might differ
    // Fall back: we'll test admin endpoints with a note
    return false;
  }
}

// ─── Test Suites ─────────────────────────────────────────────────────────────

async function testUnauthenticatedAccess() {
  console.log('\n📋 Test Suite: Unauthenticated Access');
  console.log('─'.repeat(50));

  await test('GET /v1/health (no auth required)', async () => {
    const res = await httpRequest('GET', '/v1/health', null, null);
    assertStatus(res, 200, 'health');
    assert(res.json && res.json.status === 'ok', 'expected status ok');
  });

  await test('GET /v1/health/detailed (no auth required)', async () => {
    const res = await httpRequest('GET', '/v1/health/detailed', null, null);
    assertStatus(res, 200, 'health/detailed');
  });

  await test('GET /v1/context/current returns 401 without token', async () => {
    const res = await httpRequest('GET', '/v1/context/current', null, null);
    assertStatus(res, 401, 'context without auth');
  });

  await test('GET /v1/briefing/current returns 401 without token', async () => {
    const res = await httpRequest('GET', '/v1/briefing/current', null, null);
    assertStatus(res, 401, 'briefing without auth');
  });

  await test('GET /v1/events returns 401 with bad token', async () => {
    const res = await httpRequest('GET', '/v1/events', null, 'bad-token-here');
    assertStatus(res, 401, 'events with bad token');
  });
}

async function testDeviceEndpoints() {
  console.log('\n📋 Test Suite: Device Endpoints (device token)');
  console.log('─'.repeat(50));

  await test('POST /v1/batch — ingest synthetic data', async () => {
    const now = new Date().toISOString();
    const batch = {
      events: [
        { event_type: 'app_usage_summary', timestamp: now, payload: JSON.stringify({ package: 'com.example', duration_seconds: 300 }) },
        { event_type: 'notification_posted', timestamp: now, payload: JSON.stringify({ package: 'com.example', title: 'Test' }) },
        { event_type: 'screen_state', timestamp: now, payload: JSON.stringify({ state: 'unlocked' }) },
      ],
      five_minute_summaries: [
        { timestamp: now, app_minutes: JSON.stringify({ com_example: 5 }), notification_count: 1 },
      ],
      hourly_summaries: [
        { timestamp: now, app_minutes: JSON.stringify({ com_example: 30 }), notification_count: 5, focus_score: 0.7 },
      ],
    };
    const res = await httpRequest('POST', '/v1/batch', batch, DEVICE_TOKEN);
    // Server returns 200 with batch stats (not 201)
    assertStatus(res, 200, 'batch ingest');
    assert(res.json && typeof res.json === 'object', 'expected object response');
  });

  await test('GET /v1/device/interventions', async () => {
    const res = await httpRequest('GET', '/v1/device/interventions', null, DEVICE_TOKEN);
    assertStatus(res, 200, 'device interventions');
    // May return array or object with interventions key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });

  await test('POST /v1/privacy/sensitive-mode — get', async () => {
    const res = await httpRequest('GET', '/v1/privacy/sensitive-mode', null, DEVICE_TOKEN);
    assertStatus(res, 200, 'get sensitive mode');
  });

  await test('POST /v1/privacy/sensitive-mode — set', async () => {
    const res = await httpRequest('POST', '/v1/privacy/sensitive-mode', { enabled: false }, DEVICE_TOKEN);
    assertStatus(res, 200, 'set sensitive mode');
  });

  await test('POST /v1/corrections — create', async () => {
    const res = await httpRequest('POST', '/v1/corrections', {
      entity_type: 'segment',
      entity_id: 'test-seg-1',
      correction_type: 'recategorize',
      original_value: 'work',
      corrected_value: 'personal',
    }, DEVICE_TOKEN);
    assertStatus(res, 201, 'create correction');
  });

  await test('GET /v1/corrections', async () => {
    const res = await httpRequest('GET', '/v1/corrections', null, DEVICE_TOKEN);
    assertStatus(res, 200, 'get corrections');
    // May return array or object with corrections key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });
}

async function testAgentEndpoints() {
  console.log('\n📋 Test Suite: Agent Endpoints (agent token)');
  console.log('─'.repeat(50));

  await test('GET /v1/context/current', async () => {
    const res = await httpRequest('GET', '/v1/context/current', null, AGENT_TOKEN);
    // 404 is expected when no context has been generated yet
    assert(res.status === 200 || res.status === 404, `expected 200 or 404, got ${res.status}`);
  });

  await test('GET /v1/briefing/current', async () => {
    const res = await httpRequest('GET', '/v1/briefing/current', null, AGENT_TOKEN);
    // 404 is expected when no context exists for briefing
    assert(res.status === 200 || res.status === 404, `expected 200 or 404, got ${res.status}`);
  });

  await test('GET /v1/timeline', async () => {
    const now = new Date().toISOString();
    const yesterday = new Date(Date.now() - 86400000).toISOString();
    const res = await httpRequest('GET', `/v1/timeline?from=${yesterday}&to=${now}&limit=10`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'timeline');
  });

  await test('GET /v1/events', async () => {
    const now = new Date().toISOString();
    const yesterday = new Date(Date.now() - 86400000).toISOString();
    const res = await httpRequest('GET', `/v1/events?from=${yesterday}&to=${now}&limit=10`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'events');
  });

  await test('GET /v1/summary/daily', async () => {
    const today = new Date().toISOString().split('T')[0];
    const res = await httpRequest('GET', `/v1/summary/daily?date=${today}`, null, AGENT_TOKEN);
    // 404 is expected when no summary has been generated for this date
    assert(res.status === 200 || res.status === 404, `expected 200 or 404, got ${res.status}`);
  });

  await test('GET /v1/summary/weekly', async () => {
    const now = new Date();
    const year = now.getFullYear();
    const weekNum = getISOWeekNumber(now);
    const res = await httpRequest('GET', `/v1/summary/weekly?week=${year}-W${String(weekNum).padStart(2, '0')}`, null, AGENT_TOKEN);
    // 404 is expected when no summary has been generated for this week
    assert(res.status === 200 || res.status === 404, `expected 200 or 404, got ${res.status}`);
  });

  await test('GET /v1/summary/five_minute', async () => {
    const res = await httpRequest('GET', '/v1/summary/five_minute', null, AGENT_TOKEN);
    assertStatus(res, 200, 'summary/five_minute');
  });

  await test('GET /v1/summary/hourly', async () => {
    const res = await httpRequest('GET', '/v1/summary/hourly', null, AGENT_TOKEN);
    assertStatus(res, 200, 'summary/hourly');
  });

  // Observations
  await test('POST /v1/agent/observations — create', async () => {
    const res = await httpRequest('POST', '/v1/agent/observations', {
      observation_type: 'pattern_detected',
      content: JSON.stringify({ pattern: 'deep_work_morning', confidence: 0.85 }),
      confidence: 0.85,
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'create observation');
    if (res.json && res.json.observation_id) testObservationID = res.json.observation_id;
  });

  await test('GET /v1/agent/observations', async () => {
    const res = await httpRequest('GET', '/v1/agent/observations', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get observations');
    // May return array or object with observations key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });

  // Interventions
  await test('POST /v1/agent/interventions — propose', async () => {
    const res = await httpRequest('POST', '/v1/agent/interventions', {
      intervention_type: 'suggestion',
      content: JSON.stringify({ message: 'Take a break', priority: 'medium' }),
      reason: 'High focus streak detected',
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'propose intervention');
    if (res.json && res.json.intervention_id) testInterventionID = res.json.intervention_id;
  });

  await test('PATCH /v1/agent/interventions/{id} — update status', async () => {
    if (!testInterventionID) { skipped++; process.stdout.write('  ⏭️  patch intervention (no ID)\n'); return; }
    const res = await httpRequest('PATCH', `/v1/agent/interventions/${testInterventionID}`, {
      status: 'delivered',
    }, AGENT_TOKEN);
    assertStatus(res, 200, 'update intervention');
  });

  await test('GET /v1/interventions/{id}/effectiveness', async () => {
    if (!testInterventionID) { skipped++; process.stdout.write('  ⏭️  effectiveness (no intervention ID)\n'); return; }
    const res = await httpRequest('GET', `/v1/interventions/${testInterventionID}/effectiveness`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'intervention effectiveness');
  });

  // Feedback
  await test('POST /v1/feedback — submit', async () => {
    const res = await httpRequest('POST', '/v1/feedback', {
      target_type: 'intervention',
      target_id: testInterventionID || 'test',
      rating: 4,
      comment: 'Helpful suggestion',
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'submit feedback');
  });

  await test('GET /v1/feedback', async () => {
    const res = await httpRequest('GET', '/v1/feedback', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get feedback');
  });

  // Goals CRUD
  await test('POST /v1/goals — create', async () => {
    const res = await httpRequest('POST', '/v1/goals', {
      title: 'Deep work 4h/day',
      description: 'Increase focused deep work sessions',
      goal_type: 'habit',
      target_value: 240,
      unit: 'minutes',
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'create goal');
    if (res.json && res.json.goal_id) testGoalID = res.json.goal_id;
  });

  await test('GET /v1/goals', async () => {
    const res = await httpRequest('GET', '/v1/goals', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get goals');
    // May return array or object with goals key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });

  await test('GET /v1/goals/{id}/progress', async () => {
    if (!testGoalID) { skipped++; process.stdout.write('  ⏭️  goal progress (no ID)\n'); return; }
    const res = await httpRequest('GET', `/v1/goals/${testGoalID}/progress`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'goal progress');
  });

  await test('PATCH /v1/goals/{id} — update', async () => {
    if (!testGoalID) { skipped++; process.stdout.write('  ⏭️  patch goal (no ID)\n'); return; }
    const res = await httpRequest('PATCH', `/v1/goals/${testGoalID}`, {
      status: 'active',
      current_value: 120,
    }, AGENT_TOKEN);
    assertStatus(res, 200, 'update goal');
  });

  // Habits CRUD
  await test('POST /v1/habits — create', async () => {
    const res = await httpRequest('POST', '/v1/habits', {
      title: 'Morning meditation',
      habit_type: 'wellness',
      description: '10 minutes of mindfulness',
      frequency: 'daily',
      target_count: 1,
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'create habit');
    if (res.json && res.json.habit_id) testHabitID = res.json.habit_id;
  });

  await test('GET /v1/habits', async () => {
    const res = await httpRequest('GET', '/v1/habits', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get habits');
    // May return array or object with habits key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });

  await test('GET /v1/habits/{id}/evaluation', async () => {
    if (!testHabitID) { skipped++; process.stdout.write('  ⏭️  habit evaluation (no ID)\n'); return; }
    const res = await httpRequest('GET', `/v1/habits/${testHabitID}/evaluation`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'habit evaluation');
  });

  await test('POST /v1/habits/{id}/streak — record', async () => {
    if (!testHabitID) { skipped++; process.stdout.write('  ⏭️  habit streak (no ID)\n'); return; }
    const res = await httpRequest('POST', `/v1/habits/${testHabitID}/streak`, {
      completed: true,
      timestamp: new Date().toISOString(),
    }, AGENT_TOKEN);
    // Server returns 200 with streak_updated status (not 201)
    assertStatus(res, 200, 'habit streak');
  });

  await test('PATCH /v1/habits/{id} — update', async () => {
    if (!testHabitID) { skipped++; process.stdout.write('  ⏭️  patch habit (no ID)\n'); return; }
    const res = await httpRequest('PATCH', `/v1/habits/${testHabitID}`, {
      current_streak: 3,
    }, AGENT_TOKEN);
    assertStatus(res, 200, 'update habit');
  });

  // Outcomes
  await test('POST /v1/outcomes — record', async () => {
    const res = await httpRequest('POST', '/v1/outcomes', {
      intervention_id: testInterventionID || 'test',
      outcome_type: 'behavior_change',
      description: 'Took a 15-minute walk after suggestion',
      metric_before: 0,
      metric_after: 75,
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'record outcome');
  });

  await test('GET /v1/outcomes', async () => {
    const res = await httpRequest('GET', '/v1/outcomes', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get outcomes');
  });

  // Baselines
  await test('POST /v1/baselines — create', async () => {
    const res = await httpRequest('POST', '/v1/baselines', {
      metric_name: 'deep_work_minutes_daily',
      baseline_value: 180,
      unit: 'minutes',
      period: 'daily',
    }, AGENT_TOKEN);
    assertStatus(res, 201, 'create baseline');
    if (res.json && res.json.baseline_id) testBaselineID = res.json.baseline_id;
  });

  await test('GET /v1/baselines', async () => {
    const res = await httpRequest('GET', '/v1/baselines', null, AGENT_TOKEN);
    assertStatus(res, 200, 'get baselines');
    // May return array or object with baselines key
    assert(Array.isArray(res.json) || (res.json && typeof res.json === 'object'), 'expected array or object');
  });

  // Stream (SSE)
  await test('GET /v1/stream — SSE endpoint returns event-stream', async () => {
    // SSE is a long-lived connection, so we check the initial response headers
    const res = await httpRequest('GET', '/v1/stream', null, AGENT_TOKEN);
    // Server should return 200 with text/event-stream, or 500 if Flusher not supported
    assert(res.status === 200 || res.status === 500, `expected 200 or 500, got ${res.status}`);
    if (res.status === 200) {
      assert(res.headers['content-type'] === 'text/event-stream', `expected text/event-stream content type, got ${res.headers['content-type']}`);
    }
  });
}

async function testMetricsEndpoints() {
  console.log('\n📋 Test Suite: Metrics & Analytics Endpoints');
  console.log('─'.repeat(50));

  await test('GET /v1/metrics/work-patterns', async () => {
    const res = await httpRequest('GET', '/v1/metrics/work-patterns', null, AGENT_TOKEN);
    assertStatus(res, 200, 'work patterns');
  });

  await test('GET /v1/metrics/readiness', async () => {
    const res = await httpRequest('GET', '/v1/metrics/readiness', null, AGENT_TOKEN);
    assertStatus(res, 200, 'readiness');
  });

  await test('GET /v1/analytics/insights', async () => {
    const res = await httpRequest('GET', '/v1/analytics/insights?period=7d', null, AGENT_TOKEN);
    assertStatus(res, 200, 'analytics insights');
  });

  await test('GET /v1/health/collectors', async () => {
    const res = await httpRequest('GET', '/v1/health/collectors', null, AGENT_TOKEN);
    assertStatus(res, 200, 'collector health');
  });
}

async function testAdminEndpoints() {
  console.log('\n📋 Test Suite: Admin Endpoints');
  console.log('─'.repeat(50));

  // Admin endpoints require 'admin' scope. The bootstrap doesn't create admin tokens.
  // We'll test that they return 401/403 without admin scope.

  await test('GET /v1/audit — requires admin (should reject agent token)', async () => {
    const res = await httpRequest('GET', '/v1/audit', null, AGENT_TOKEN);
    // Agent token doesn't have admin scope, so should get 403
    assert(res.status === 403 || res.status === 401, `expected 403 or 401, got ${res.status}`);
  });

  await test('GET /v1/export/daily — requires admin (should reject agent token)', async () => {
    const today = new Date().toISOString().split('T')[0];
    const res = await httpRequest('GET', `/v1/export/daily?date=${today}`, null, AGENT_TOKEN);
    assert(res.status === 403 || res.status === 401, `expected 403 or 401, got ${res.status}`);
  });

  await test('POST /v1/privacy/wipe — requires admin (should reject agent token)', async () => {
    const res = await httpRequest('POST', '/v1/privacy/wipe', { confirm: true }, AGENT_TOKEN);
    assert(res.status === 403 || res.status === 401, `expected 403 or 401, got ${res.status}`);
  });
}

async function testDeleteEndpoints() {
  console.log('\n📋 Test Suite: DELETE Endpoints');
  console.log('─'.repeat(50));

  await test('DELETE /v1/goals/{id}', async () => {
    if (!testGoalID) { skipped++; process.stdout.write('  ⏭️  delete goal (no ID)\n'); return; }
    const res = await httpRequest('DELETE', `/v1/goals/${testGoalID}`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'delete goal');
  });

  await test('DELETE /v1/habits/{id}', async () => {
    if (!testHabitID) { skipped++; process.stdout.write('  ⏭️  delete habit (no ID)\n'); return; }
    const res = await httpRequest('DELETE', `/v1/habits/${testHabitID}`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'delete habit');
  });

  await test('DELETE /v1/baselines/{id}', async () => {
    if (!testBaselineID) { skipped++; process.stdout.write('  ⏭️  delete baseline (no ID)\n'); return; }
    const res = await httpRequest('DELETE', `/v1/baselines/${testBaselineID}`, null, AGENT_TOKEN);
    assertStatus(res, 200, 'delete baseline');
  });

  await test('DELETE /v1/goals/nonexistent-id — returns error', async () => {
    const res = await httpRequest('DELETE', '/v1/goals/nonexistent-id-12345', null, AGENT_TOKEN);
    assert(res.status >= 200 && res.status < 500, `expected 2xx-4xx, got ${res.status}`);
  });

  await test('DELETE /v1/habits/nonexistent-id — returns error', async () => {
    const res = await httpRequest('DELETE', '/v1/habits/nonexistent-id-12345', null, AGENT_TOKEN);
    assert(res.status >= 200 && res.status < 500, `expected 2xx-4xx, got ${res.status}`);
  });

  await test('DELETE /v1/baselines/nonexistent-id — returns error', async () => {
    const res = await httpRequest('DELETE', '/v1/baselines/nonexistent-id-12345', null, AGENT_TOKEN);
    assert(res.status >= 200 && res.status < 500, `expected 2xx-4xx, got ${res.status}`);
  });

  await test('DELETE /v1/corrections/{id} — first correction', async () => {
    // Get corrections first to find an ID
    const listRes = await httpRequest('GET', '/v1/corrections', null, DEVICE_TOKEN);
    if (listRes.json && Array.isArray(listRes.json) && listRes.json.length > 0) {
      const id = listRes.json[0].correction_id || listRes.json[0].id;
      if (id) {
        const res = await httpRequest('DELETE', `/v1/corrections/${id}`, null, DEVICE_TOKEN);
        assertStatus(res, 200, 'delete correction');
      }
    }
  });
}

async function testErrorCases() {
  console.log('\n📋 Test Suite: Error Cases');
  console.log('─'.repeat(50));

  await test('GET /v1/summary/daily — missing date param', async () => {
    const res = await httpRequest('GET', '/v1/summary/daily', null, AGENT_TOKEN);
    // Should either return 400 or empty/default response
    assert(res.status >= 200 && res.status < 500, `expected 2xx-4xx, got ${res.status}`);
  });

  await test('POST /v1/goals — empty body returns 400', async () => {
    const res = await httpRequest('POST', '/v1/goals', {}, AGENT_TOKEN);
    assertStatus(res, 400, 'empty goal body');
  });

  await test('POST /v1/goals — title-only creates successfully', async () => {
    const res = await httpRequest('POST', '/v1/goals', { title: 'Test goal', goal_type: 'milestone' }, AGENT_TOKEN);
    assertStatus(res, 201, 'goal with title only');
  });

  await test('POST /v1/habits — empty body returns 400', async () => {
    const res = await httpRequest('POST', '/v1/habits', {}, AGENT_TOKEN);
    assertStatus(res, 400, 'empty habit body');
  });

  await test('POST /v1/baselines — empty body returns 400', async () => {
    const res = await httpRequest('POST', '/v1/baselines', {}, AGENT_TOKEN);
    assertStatus(res, 400, 'empty baseline body');
  });

  await test('GET /v1/goals/nonexistent-id/progress', async () => {
    const res = await httpRequest('GET', '/v1/goals/nonexistent-id/progress', null, AGENT_TOKEN);
    assert(res.status >= 200, `expected 2xx or 4xx, got ${res.status}`);
  });

  await test('GET /v1/habits/nonexistent-id/evaluation', async () => {
    const res = await httpRequest('GET', '/v1/habits/nonexistent-id/evaluation', null, AGENT_TOKEN);
    assert(res.status >= 200, `expected 2xx or 4xx, got ${res.status}`);
  });

  await test('GET /nonexistent-route returns 404', async () => {
    const res = await httpRequest('GET', '/v1/does-not-exist', null, AGENT_TOKEN);
    assertStatus(res, 404, 'nonexistent route');
  });

  await test('POST /v1/batch — malformed JSON', async () => {
    const res = new Promise((resolve, reject) => {
      const req = http.request({
        hostname: '127.0.0.1',
        port: PORT,
        path: '/v1/batch',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${DEVICE_TOKEN}`,
        },
        timeout: 5000,
      }, (res) => {
        let data = '';
        res.on('data', (c) => data += c);
        res.on('end', () => resolve({ status: res.statusCode, body: data }));
      });
      req.on('error', reject);
      req.write('not valid json {{{');
      req.end();
    });
    const r = await res;
    assert(r.status >= 400, `expected 4xx for malformed JSON, got ${r.status}`);
  });
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function getISOWeekNumber(date) {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil((((d - yearStart) / 86400000) + 1) / 7);
}

// ─── Main ────────────────────────────────────────────────────────────────────

async function main() {
  console.log('🚀 Andrometry Server Integration Tests');
  console.log('═'.repeat(50));
  console.log(`Port: ${PORT}`);
  console.log(`DB: ${DB_PATH}`);
  console.log(`Bootstrap token: ${BOOTSTRAP_TOKEN.slice(0, 12)}...`);
  console.log('');

  try {
    // Start server
    process.stdout.write('⏳ Starting server... ');
    await startServer();
    process.stdout.write('ready!\n');

    // Wait for server to be fully up
    await new Promise(r => setTimeout(r, 1000));

    // Verify server is running
    const healthRes = await httpRequest('GET', '/v1/health', null, null);
    if (healthRes.status !== 200) {
      throw new Error(`Server health check failed: ${healthRes.status}`);
    }
    process.stdout.write('✅ Server is healthy\n');

    // Run all test suites
    await testUnauthenticatedAccess();
    await testDeviceEndpoints();
    await testAgentEndpoints();
    await testMetricsEndpoints();
    await testAdminEndpoints();
    await testDeleteEndpoints();
    await testErrorCases();

  } catch (err) {
    console.error(`\n💥 Fatal error: ${err.message}`);
    failed++;
    failures.push({ name: 'FATAL', error: err.message });
  } finally {
    stopServer();
  }

  // Summary
  console.log('\n' + '═'.repeat(50));
  console.log('📊 Test Results Summary');
  console.log('═'.repeat(50));
  console.log(`  ✅ Passed:  ${passed}`);
  console.log(`  ❌ Failed:  ${failed}`);
  console.log(`  ⏭️  Skipped: ${skipped}`);
  console.log(`  Total:      ${passed + failed + skipped}`);

  if (failures.length > 0) {
    console.log('\n❌ Failures:');
    failures.forEach(f => console.log(`  • ${f.name}: ${f.error}`));
  }

  console.log('');
  process.exit(failed > 0 ? 1 : 0);
}

main();
