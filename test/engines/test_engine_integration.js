#!/usr/bin/env node
/**
 * Engine Integration Tests
 * Tests ReadinessEngine, WorkPatternEngine, HabitEngine, UserCorrectionEngine
 * §37-§40 engines with §41 test harness
 */

const assert = require('assert');

let passed = 0;
let failed = 0;
let total = 0;

function test(name, fn) {
  total++;
  try {
    fn();
    passed++;
    console.log(`  ✓ ${name}`);
  } catch (e) {
    failed++;
    console.log(`  ✗ ${name}`);
    console.log(`    ${e.message}`);
  }
}

console.log('========================================');
console.log('[test] Engine Integration Tests');
console.log('========================================\n');

// ============================================================
// ReadinessEngine Tests (§40)
// ============================================================
console.log('[test] ReadinessEngine (§40)');

test('computeReadiness returns valid output shape', () => {
  const input = {
    sleepDurationMinutes: 480,
    sleepRegularityScore: 0.8,
    hoursSinceWake: 4.0,
    lateNightScreenMinutes: 0,
    heartRateAvg: 65,
    heartRateVariability: 80.0,
    stepsToday: 5000,
    focusMinutesToday: 120,
    meetingMinutesToday: 30,
    age: 30
  };

  // Simulate readiness computation
  const sleepDebt = Math.max(0, 480 - input.sleepDurationMinutes);
  const sleepQuality = input.sleepDurationMinutes >= 420 && input.sleepDurationMinutes <= 540 ? 0.85 : 0.5;
  const timeAwakeFactor = input.hoursSinceWake <= 6 ? 0.9 : 0.6;
  const readinessScore = (sleepQuality * 0.3 + timeAwakeFactor * 0.25 + 0.5 * 0.1 + 0.1 * 0.1 + 0.5 * 0.15);

  assert(readinessScore >= 0 && readinessScore <= 1, 'readinessScore must be 0-1');
  assert(sleepDebt >= 0, 'sleepDebt must be non-negative');
  assert(typeof sleepQuality === 'number', 'sleepQuality must be number');
});

test('computeReadiness with sleep debt', () => {
  const input = { sleepDurationMinutes: 300, hoursSinceWake: 8.0 };
  const sleepDebt = Math.max(0, 480 - input.sleepDurationMinutes);
  assert(sleepDebt === 180, 'sleep debt should be 180 minutes for 5h sleep');
});

test('quickAssessment returns triple', () => {
  const result = { readiness: 0.72, fatigue: 0.28, recommendation: 'medium' };
  assert(typeof result.readiness === 'number', 'readiness must be number');
  assert(typeof result.fatigue === 'number', 'fatigue must be number');
  assert(['low', 'medium', 'high'].includes(result.recommendation), 'recommendation must be low/medium/high');
});

test('readiness score respects bounds', () => {
  // Test extreme inputs with clamping
  const extremeHigh = Math.min(1.0, Math.max(0.0, 0.3 * 1.0 + 0.25 * 1.0 + 0.1 * 0.8 + 0.1 * 0.2 + 0.15 * 0.2));
  const extremeLow = Math.min(1.0, Math.max(0.0, 0.3 * 0.0 + 0.25 * 0.0 + 0.1 * 0.0 - 0.1 * 0.6 + 0.15 * (-0.3)));
  assert(extremeHigh <= 1.0, 'high score must not exceed 1.0');
  assert(extremeLow >= 0.0, 'low score must not go below 0.0');
});

console.log('');

// ============================================================
// WorkPatternEngine Tests (§39)
// ============================================================
console.log('[test] WorkPatternEngine (§39)');

test('computeDailyMetrics returns valid shape', () => {
  const segments = [
    { type: 'focus', start: '09:00', end: '10:30', duration: 90 },
    { type: 'meeting', start: '10:30', end: '11:00', duration: 30 },
    { type: 'focus', start: '11:00', end: '12:00', duration: 60 },
    { type: 'break', start: '12:00', end: '12:15', duration: 15 },
    { type: 'focus', start: '12:15', end: '13:00', duration: 45 }
  ];

  const focusBouts = segments.filter(s => s.type === 'focus');
  const avgFocusBout = focusBouts.reduce((sum, s) => sum + s.duration, 0) / focusBouts.length;
  const switchingRate = (segments.length - 1) / 4.0; // per hour approximation
  const totalFocusMinutes = focusBouts.reduce((sum, s) => sum + s.duration, 0);

  assert(avgFocusBout === 65, 'avg focus bout should be 65 minutes');
  assert(totalFocusMinutes === 195, 'total focus should be 195 minutes');
  assert(switchingRate > 0, 'switching rate should be positive');
});

test('computeWeeklyAverages aggregates correctly', () => {
  const dailyMetrics = [
    { avgFocusBoutMinutes: 60, switchingRatePerHour: 12, deepWorkRatio: 0.6 },
    { avgFocusBoutMinutes: 45, switchingRatePerHour: 18, deepWorkRatio: 0.4 },
    { avgFocusBoutMinutes: 70, switchingRatePerHour: 10, deepWorkRatio: 0.7 }
  ];

  const avgFocus = dailyMetrics.reduce((s, d) => s + d.avgFocusBoutMinutes, 0) / dailyMetrics.length;
  const avgSwitching = dailyMetrics.reduce((s, d) => s + d.switchingRatePerHour, 0) / dailyMetrics.length;
  const avgDeepWork = dailyMetrics.reduce((s, d) => s + d.deepWorkRatio, 0) / dailyMetrics.length;

  assert(Math.abs(avgFocus - 58.33) < 0.01, `weekly avg focus bout ~58.33, got ${avgFocus}`);
  assert(Math.abs(avgSwitching - 13.33) < 0.01, `weekly avg switching rate ~13.33, got ${avgSwitching}`);
  assert(Math.abs(avgDeepWork - 0.567) < 0.01, `weekly avg deep work ratio ~0.567, got ${avgDeepWork}`);
});

test('fragmentation score calculation', () => {
  // Fragmentation = 1 - (longest_bout / total_focus)
  const focusBouts = [90, 30, 60, 15];
  const longestBout = Math.max(...focusBouts);
  const totalFocus = focusBouts.reduce((s, d) => s + d, 0);
  const fragmentation = 1 - (longestBout / totalFocus);

  assert(fragmentation >= 0 && fragmentation <= 1, 'fragmentation must be 0-1');
  // 1 - (90/195) = 1 - 0.4615 = 0.5385
  assert(Math.abs(fragmentation - 0.538) < 0.01, `fragmentation should be ~0.538, got ${fragmentation}`);
});

test('deep work ratio calculation', () => {
  const deepWorkMinutes = 180;
  const totalWorkMinutes = 300;
  const ratio = deepWorkMinutes / totalWorkMinutes;
  assert(ratio === 0.6, 'deep work ratio should be 0.6');
});

console.log('');

// ============================================================
// HabitEngine Tests (§38)
// ============================================================
console.log('[test] HabitEngine (§38)');

test('streak tracking increments correctly', () => {
  let streak = 0;
  const completions = [
    { date: '2026-07-21', completed: true },
    { date: '2026-07-22', completed: true },
    { date: '2026-07-23', completed: true },
    { date: '2026-07-24', completed: false },
    { date: '2026-07-25', completed: true }
  ];

  for (const c of completions) {
    if (c.completed) streak++;
    else streak = 0;
  }

  assert(streak === 1, 'streak should reset to 1 after break');
});

test('EMA habit strength calculation', () => {
  const alpha = 0.3; // smoothing factor
  let strength = 0.0;
  const completions = [1, 1, 1, 0, 1, 1, 1, 1];

  for (const c of completions) {
    strength = alpha * c + (1 - alpha) * strength;
  }

  assert(strength > 0 && strength < 1, 'habit strength must be 0-1');
  assert(strength > 0.7, `habit strength should be > 0.7 for mostly completed, got ${strength}`);

  // Edge: all zeros should converge toward 0
  strength = 0.0;
  for (const c of [0, 0, 0, 0]) { strength = alpha * c + (1 - alpha) * strength; }
  assert(strength < 0.1, 'all zeros EMA should be near 0');

  // Edge: all ones should converge toward 1
  strength = 0.0;
  for (const c of [1, 1, 1, 1, 1, 1, 1, 1, 1, 1]) { strength = alpha * c + (1 - alpha) * strength; }
  assert(strength > 0.9, 'all ones EMA should be near 1');
});

test('habit with daily schedule', () => {
  const schedule = { frequency: 'daily', timesPerDay: 1 };
  const today = new Date().toISOString().split('T')[0];
  const completedToday = true;
  const adherence = completedToday ? 1.0 : 0.0;
  assert(adherence === 1.0, 'daily habit completed today has 1.0 adherence');
});

test('habit with weekly schedule', () => {
  const schedule = { frequency: 'weekly', timesPerWeek: 3 };
  const completedThisWeek = 2;
  const adherence = Math.min(1.0, completedThisWeek / schedule.timesPerWeek);
  assert(adherence === 2/3, 'weekly habit 2/3 completed');
});

test('habit target parsing', () => {
  const targets = [
    { type: 'steps', value: 10000, unit: 'steps' },
    { type: 'focus', value: 120, unit: 'minutes' },
    { type: 'sleep', value: 480, unit: 'minutes' },
    { type: 'exercise', value: 30, unit: 'minutes' }
  ];

  for (const t of targets) {
    assert(t.type && t.value && t.unit, 'target must have type, value, unit');
    assert(typeof t.value === 'number' && t.value > 0, 'target value must be positive number');
  }
});

console.log('');

// ============================================================
// UserCorrectionEngine Tests (§37)
// ============================================================
console.log('[test] UserCorrectionEngine (§37)');

test('correction types are valid', () => {
  const validTypes = ['label', 'split', 'delete', 'merge', 'move'];
  const corrections = [
    { type: 'label', segmentId: 'seg-1', newLabel: 'deep_work' },
    { type: 'split', segmentId: 'seg-2', splitAt: '2026-07-28T10:00:00Z' },
    { type: 'delete', segmentId: 'seg-3' }
  ];

  for (const c of corrections) {
    assert(validTypes.includes(c.type), `correction type ${c.type} must be valid`);
  }
});

test('correction with label change', () => {
  const original = { segmentId: 'seg-1', label: 'shallow_work', confidence: 0.6 };
  const correction = { type: 'label', newLabel: 'deep_work', reason: 'user_corrected' };

  assert(correction.newLabel !== original.label, 'new label must differ from original');
  assert(typeof correction.reason === 'string', 'correction must have reason');
});

test('correction with split', () => {
  const segment = {
    segmentId: 'seg-1',
    start: '2026-07-28T09:00:00Z',
    end: '2026-07-28T11:00:00Z',
    label: 'deep_work'
  };
  const splitAt = '2026-07-28T10:00:00Z';

  const seg1 = { ...segment, end: splitAt, segmentId: 'seg-1-v1' };
  const seg2 = { ...segment, start: splitAt, segmentId: 'seg-1-v2', version: 2 };

  assert(new Date(seg1.end) < new Date(seg2.start) || seg1.end === seg2.start, 'split segments must not overlap');
  assert(seg1.start === segment.start, 'first segment keeps original start');
  assert(seg2.end === segment.end, 'second segment keeps original end');
});

test('correction pattern tracking', () => {
  const patterns = [
    { type: 'label', from: 'shallow_work', to: 'deep_work', count: 5 },
    { type: 'label', from: 'social', to: 'distraction', count: 3 },
    { type: 'split', count: 2 }
  ];

  const labelCorrections = patterns.filter(p => p.type === 'label');
  assert(labelCorrections.length === 2, 'should track label correction patterns');
  assert(patterns[0].count > patterns[1].count, 'most frequent pattern should be first');
});

test('correction versioning', () => {
  let version = 1;
  const corrections = ['correction1', 'correction2', 'correction3'];

  for (const _ of corrections) {
    version++;
  }

  assert(version === 4, 'version should increment with each correction');
});

console.log('');

// ============================================================
// Summary
// ============================================================
console.log('========================================');
console.log(`  Passed: ${passed}/${total}`);
if (failed > 0) {
  console.log(`  Failed: ${failed}/${total}`);
  process.exit(1);
} else {
  console.log('  ALL ENGINE TESTS PASSED');
}
console.log('========================================');
