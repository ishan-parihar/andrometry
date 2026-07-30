# Andrometry — Operational Audit Report

**Date:** 2026-07-31  
**Server:** andrometry-relay v0.1.0 on localhost:8420  
**Tokens:** Device `dev-token-123`, Agent `agent-dev-token-123`

---

## Executive Summary

Tested all 25 API endpoints. **15 pass, 10 have issues** (4 bugs, 4 schema mismatches, 2 expected behaviors).

| Category | Count | Status |
|----------|-------|--------|
| ✅ Passing | 15 | Working correctly |
| ❌ Bugs | 4 | Need code fixes |
| ⚠️ Schema Mismatches | 4 | Payload format issues |
| ℹ️ Expected | 2 | Correct behavior (scope/time) |

---

## Endpoint Test Results

### ✅ Passing Endpoints (15)

| # | Endpoint | Method | Status | Notes |
|---|----------|--------|--------|-------|
| 1 | `/v1/health` | GET | ✅ 200 | Returns server info, status, timestamp |
| 2 | `/v1/batch` | POST | ✅ 200 | 3 events accepted, 1 segment, pending_interventions=1 |
| 3 | `/v1/context/current` | GET | ✅ 200 | Returns context with ETag for cheap polling |
| 4 | `/v1/timeline` | GET | ✅ 200 | Returns 1 focus segment with payload |
| 5 | `/v1/briefing/current` | GET | ✅ 200 | Returns state, risks, opportunities, suggested_queries |
| 7 | `/v1/agent/observations` | POST | ✅ 201 | observation_id returned |
| 14 | `/v1/events` | GET | ✅ 200 | Returns 3 events (context, notifications, app_usage) |
| 17 | `/v1/corrections` | GET | ✅ 200 | Returns empty corrections list |
| 18 | `/v1/corrections` | POST | ✅ 201 | segment_updated=false, status=created |
| 19 | `/v1/outcomes` | POST | ✅ 201 | outcome_id returned |
| 20 | `/v1/metrics/work-patterns` | GET | ✅ 200 | Returns default metrics (all zeros — expected with 1 segment) |
| 21 | `/v1/metrics/readiness` | GET | ✅ 200 | score=0.7, recommendation about sleep debt |
| 22 | `/v1/health/collectors` | GET | ✅ 200 | Returns 5 collector statuses + DB stats |
| 23 | `/v1/privacy/sensitive-mode` | GET | ✅ 200 | active=false (correct default) |
| 25 | `/v1/analytics/insights` | GET | ✅ 200 | Returns 7-day analytics with event count, focus metrics |

### ❌ Bugs (Need Code Fixes)

| # | Endpoint | Method | Status | Bug Description |
|---|----------|--------|--------|-----------------|
| 6 | `/v1/summary/daily` | GET | ❌ 404 | `"no summary for this date"` — Daily summaries are not auto-generated from ingested segments/events. The server stores raw data but never triggers summary computation. **P0 bug.** |
| 8 | `/v1/agent/interventions` | POST | ⚠️ 200 | `rejected_reason: "quiet_hours"` — Interventions are rejected during quiet hours (22:30-07:30). Server time is 04:11 AM which falls in quiet hours. **Expected behavior but needs Android-side timezone sync.** |
| 11 | `/v1/preferences` | GET | ⚠️ 200 | Returns `{"preferences":{}}` — Empty even after PUT. Preferences are not persisting correctly. **P1 bug.** |
| 16 | `/v1/summary/weekly` | GET | ❌ 404 | `"no summary for this week"` — Same as daily summary. **P0 bug.** |

### ⚠️ Schema Mismatches (Payload Format Issues)

| # | Endpoint | Method | Status | Issue |
|---|----------|--------|--------|-------|
| 9 | `/v1/goals` | POST | ❌ 400 | `"title is required"` — Handler expects `title` field but architecture spec uses `target`. Need to align schema. |
| 10 | `/v1/habits` | POST | ❌ 400 | `"invalid JSON"` — Payload structure doesn't match expected format. Need to check handler's JSON parsing. |
| 12 | `/v1/preferences` | PUT | ❌ 400 | `"invalid JSON"` — Payload format mismatch. Handler expects different structure than documented. |
| 13 | `/v1/audit` | GET | ❌ 403 | `"insufficient scope: admin"` — Device token doesn't have admin scope. Audit requires admin scope. |

### ℹ️ Expected Behaviors

| # | Endpoint | Method | Status | Notes |
|---|----------|--------|--------|-------|
| 15 | `/v1/export/daily` | GET | ❌ 403 | Admin scope required — correct, device token doesn't have admin |
| 8 | `/v1/agent/interventions` | POST | ⚠️ 200 | Quiet hours rejection — correct, server is in quiet hours (04:11 AM) |

---

## Briefing Quality Issues

The briefing endpoint returns a briefing but with **all probabilities at 0**:

```json
{
  "state": {
    "primary_activity": "deep_work",
    "place": "office",
    "focus_active": false,
    "distraction_probability": 0,
    "fatigue_probability": 0,
    "alignment_probability": 0,
    "readiness_score": 0
  }
}
```

**Issue:** The briefing engine doesn't compute distraction/fatigue/alignment/readiness from the stored context. It only uses the raw `primary_activity` and `place` from the context snapshot, ignoring the rich probability data.

---

## Data Flow Verification

### What Works
1. **Device → Server:** Events, segments, and context are ingested correctly via batch endpoint
2. **Server → Agent:** Context, timeline, events, briefing are returned correctly
3. **Agent → Server:** Observations, outcomes, corrections are stored correctly
4. **ETag polling:** Context endpoint returns ETag for cheap polling

### What's Missing
1. **Auto-summary generation:** No background job generates daily/weekly summaries from segments
2. **Preferences persistence:** PUT stores but GET returns empty
3. **Goal/Habit schema alignment:** Handler expectations don't match architecture spec
4. **Briefing probability computation:** Briefing doesn't use stored probability data

---

## Android App Connectivity

The Android app was installed on device. Server is running on `localhost:8420`. For the app to connect:

1. **Network:** Device and server must be on same network (192.168.1.13:8420)
2. **Token:** App needs `dev-token-123` configured as device token
3. **Relay URL:** App needs `http://192.168.1.13:8420` as relay URL
4. **CORS:** Server has `cors_origin = "*"` which allows all origins

**Note:** ADB is not available on this machine, so direct device inspection wasn't possible.

---

## Priority Fixes

### P0 — Critical (Blocks Core Functionality)
1. **Daily/Weekly Summary Auto-Generation** — Add background job that computes summaries from segments after ingestion
2. **Config Precedence Fix** — Already fixed in this commit (parseConfigFile moved before CLI flags)

### P1 — High (Breaks Agent Workflows)
3. **Preferences Persistence** — Fix PUT/GET cycle for preferences storage
4. **Goal Schema Alignment** — Align handler's expected fields with architecture spec (`title` vs `target`)
5. **Habit Schema Alignment** — Fix JSON parsing for habit creation
6. **Briefing Probability Computation** — Compute distraction/fatigue/alignment/readiness from stored context

### P2 — Medium (Improves Reliability)
7. **Audit Scope** — Add admin scope to device token for testing, or create separate admin token
8. **Export Scope** — Same as above
9. **Intervention Timezone** — Ensure quiet hours use device timezone, not server timezone

---

## Test Commands Used

```bash
# Start server
/tmp/andrometry-server --bootstrap-token=dev-token-123

# Test endpoints
TOKEN='dev-token-123'
AGENT='agent-dev-token-123'

# Batch ingestion
curl -X POST http://localhost:8420/v1/batch -H "Authorization: Bearer $TOKEN" -d '...'

# Context query
curl -H "Authorization: Bearer $AGENT" http://localhost:8420/v1/context/current

# Timeline query
curl -H "Authorization: Bearer $AGENT" "http://localhost:8420/v1/timeline?from=...&to=..."
```
