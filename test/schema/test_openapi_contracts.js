#!/usr/bin/env node
/**
 * OpenAPI Spec Validation Test
 * Validates that the OpenAPI spec covers all server routes.
 */
const fs = require('fs');
const path = require('path');

const specPath = path.resolve(__dirname, '../../api/openapi.yaml');
const mainGoPath = path.resolve(__dirname, '../../server/cmd/andrometry/main.go');

let passed = 0;
let failed = 0;

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

try {
  const spec = fs.readFileSync(specPath, 'utf8');
  const mainGo = fs.readFileSync(mainGoPath, 'utf8');

  // Extract paths from OpenAPI spec (lines like '  /v1/something:')
  const specPaths = new Set();
  for (const line of spec.split('\n')) {
    const m = line.match(/^\s{2}(\/v1\/[^:]+):/);
    if (m) specPaths.add(m[1]);
  }

  // Extract route patterns from main.go (GET /v1/..., POST /v1/..., etc.)
  const routePatterns = new Set();
  for (const line of mainGo.split('\n')) {
    const m = line.match(/HandleFunc\("(GET|POST|PATCH|DELETE|PUT) (\/v1\/[^"\\]+)/);
    if (m) routePatterns.add(m[2]);
  }

  // Normalize route patterns to match spec format
  const normalizedRoutes = new Set();
  for (const route of routePatterns) {
    // Convert {id} to {id} (already in correct format)
    normalizedRoutes.add(route);
  }

  console.log('[test] OpenAPI Spec Validation');
  console.log('  Spec paths: ' + specPaths.size);
  console.log('  Server routes: ' + normalizedRoutes.size);

  // Check each server route has a spec path
  let missing = [];
  for (const route of normalizedRoutes) {
    if (!specPaths.has(route)) {
      missing.push(route);
    }
  }

  assert(missing.length === 0, 'Missing spec paths: ' + missing.join(', '));
  passed++;
  console.log('  ✅ All server routes are documented in OpenAPI spec');

  // Check spec doesn't have extra paths not in server
  let extra = [];
  for (const p of specPaths) {
    if (!normalizedRoutes.has(p)) extra.push(p);
  }
  // Extra paths in spec are OK (might be parameterized differently)
  if (extra.length > 0) {
    console.log('  ℹ️  Spec has extra paths (parameterized): ' + extra.join(', '));
  }

  console.log('\n========================================');
  console.log('[test] ALL OPENAPI TESTS PASSED');
  console.log('========================================');
} catch (err) {
  failed++;
  console.error('  ❌ ' + err.message);
  console.log('\n========================================');
  console.log('[test] OPENAPI TEST FAILED');
  console.log('========================================');
  process.exit(1);
}
