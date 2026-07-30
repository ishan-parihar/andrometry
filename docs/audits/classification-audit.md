# Andrometry — Activity Identification & App Classification Audit

**Date:** 2026-07-31  
**Scope:** Full audit of activity identification pipeline and app classification system  
**Status:** Implementation gaps identified — proceeding with fixes

---

## 1. Executive Summary

The andrometry system has a **three-layer classification pipeline** (collection → app categorization → activity inference) with significant implementation gaps between the architecture specification and the actual code. The system currently handles only **7 of 15** defined activity types, uses fragile substring matching for app classification, has duplicate classifiers that can disagree, and is missing ~60% of the context signals needed for accurate inference.

---

## 2. Current Architecture

### 2.1 Classification Pipeline

```
┌─────────────────────┐
│ UsageCollector.kt   │  ← Raw signal collection (UsageStatsManager)
│ (collectors/usage/) │
└─────────┬───────────┘
          │ app_usage_summary events
          ▼
┌─────────────────────┐
│ AppCategoryEngine.kt│  ← Package → Category mapping
│ (inference/classifier)│
└─────────┬───────────┘
          │ category labels
          ▼
┌──────────────────────────────┐
│ ContextInferenceEngine.kt    │  ← Rule-based activity classification
│ (inference/context/)         │
└─────────┬────────────────────┘
          │ ContextSnapshot
          ▼
┌──────────────────────────────┐
│ SegmentEngine.kt             │  ← Timeline segment generation
│ (inference/segments/)        │
└──────────────────────────────┘
```

### 2.2 Defined Categories (Architecture §9.1)

15 categories: `productivity`, `communication`, `calendar`, `email`, `browser`, `social`, `entertainment`, `news`, `finance`, `health`, `travel`, `shopping`, `maps`, `system`, `unknown`

### 2.3 Defined Activities (Architecture §11.2)

15 activities: `deep_work`, `shallow_work`, `meeting`, `communication`, `learning`, `admin`, `planning`, `break`, `leisure`, `social`, `distraction`, `exercise`, `commute`, `sleep`, `unknown`

---

## 3. Critical Issues (P0)

### 3.1 Duplicate Classifiers That Disagree

**Problem:** Two independent classification systems exist:

| Classifier | Location | Method |
|---|---|---|
| `AppCategoryEngine.kt` | `inference/classifier/` | Hardcoded map (~40 apps) + Android category |
| `classifyPackage()` in `UsageCollector.kt` | `collectors/usage/` | Substring matching on package name |

**Example conflict:** Package `com.meetup.app`:
- UsageCollector: `communication` (contains "meet")
- AppCategoryEngine: `unknown` (not in known list)

**Fix:** Unify into single classifier. UsageCollector must delegate to AppCategoryEngine.

### 3.2 Substring Matching Is Fragile

**Problem:** `UsageCollector.classifyPackage()` uses `packageName.contains()` which produces false positives:

| Package | Matches | Actual | Correct? |
|---|---|---|---|
| `com.google.android.apps.maps` | "maps" → travel | Maps | ✅ |
| `com.maps.android.screensaver` | "maps" → travel | Screensaver | ❌ |
| `com.paychex.employee` | "pay" → finance | HR app | ❌ |
| `com.notepadplusplus` | — → unknown | Notes | ❌ |

**Fix:** Replace with exact package matching + Android category fallback + user overrides.

### 3.3 Missing Context Signals (~60%)

**Problem:** `ContextInferenceEngine.infer()` hardcodes critical signals:

| Signal | Architecture Spec | Implemented |
|---|---|---|
| App usage | ✅ | ✅ |
| Focus state | ✅ | ✅ |
| Notifications | ✅ | ✅ |
| Calendar | ✅ | ❌ `calendar = null` |
| Health/Activity | ✅ | ❌ `health = null` |
| Location/Place | ✅ | ❌ `place = "unknown"` |
| Sleep | ✅ | ❌ `sleep = null` |
| Movement | ✅ | ❌ `movement = "still"` |
| Screen unlock timing | ✅ | ❌ `minutesSinceUnlock = 0` |

**Fix:** Wire CalendarCollector, HealthCollector, LocationCollector, ScreenReceiver into ContextInferenceEngine.

---

## 4. High-Priority Issues (P1)

### 4.1 Activity Classification Covers Only 7/15 Activities

**Unhandled activities:**

| Activity | Required Signal | Status |
|---|---|---|
| `meeting` | Calendar event + communication app | ❌ No calendar integration |
| `learning` | Education app category | ❌ Falls to "unknown" or "leisure" |
| `admin` | System/settings apps | ⚠️ Only via weak "system" category |
| `planning` | Calendar/todo apps | ❌ No handling |
| `break` | No app activity + screen on | ❌ No handling |
| `exercise` | Health app + step count | ❌ No handling |
| `commute` | Maps/travel + movement | ❌ No handling |
| `sleep` | Screen off + time + location | ❌ No handling |

### 4.2 No Minimum Duration Threshold

**Problem:** Transient app usage (opening Slack for 10 seconds) is classified the same as sustained usage (30 minutes of coding).

**Fix:** Add 60-second minimum before classifying activity from app category.

### 4.3 No Background App Exclusion

**Problem:** Spotify playing music in background while coding → classified as "entertainment" if Spotify was last foreground event.

**Fix:** Ignore background audio apps (Spotify, YouTube Music, podcast apps) from foreground classification.

### 4.4 No Temporal Transitions

**Problem:** Activity classification is snapshot-based with no hysteresis. Rapid app switching causes activity oscillation.

**Fix:** Add transition detection with cooldown periods (e.g., must sustain for 2+ minutes before switching activity).

---

## 5. Medium-Priority Issues (P2)

### 5.1 Hardcoded Alignment Values Are Opinionated

**Problem:** `AppCategoryEngine.kt` hardcodes alignment:
```kotlin
"com.twitter.android" → "misaligned"
"com.linkedin.android" → "neutral"
"com.spotify.music" → "neutral"
```

**Issues:**
- Twitter/X may be **aligned** for a journalist
- LinkedIn may be **misaligned** for someone avoiding work
- Spotify may be **aligned** for someone who codes with music
- No user-specific alignment learning exists

**Fix:** Make alignment user-configurable via DB overrides, add learning from outcomes.

### 5.2 Fatigue Model Is Time-of-Day Only

**Problem:** `calculateFatigueProbability()` only uses clock time:
```kotlin
hour in 0..5 -> 0.7
hour in 8..11 -> 0.2
```

**Missing fatigue signals:**
- Sleep debt from previous night
- Heart rate variability
- Step count / activity level
- Screen time accumulation
- Break duration

**Fix:** Integrate sleep, health, and screen time data into fatigue model.

### 5.3 No Learning From User Behavior

**Problem:** These components exist but don't feed back into classification:
- `UserCorrectionEngine.kt` — corrections don't update app alignment
- `OutcomeEngine.kt` — outcomes don't adjust distraction thresholds
- `BaselineEngine.kt` — baselines don't calibrate confidence scores
- `HabitEngine.kt` — habits don't inform activity inference

**Fix:** Wire correction/outcome data into classification feedback loop.

---

## 6. Edge Cases Not Handled

| Edge Case | Impact | Status |
|---|---|---|
| Split-screen / PiP | Only one app tracked | ❌ Not handled |
| Work profiles | Duplicate packages | ❌ Not handled |
| OEM bloatware | False substring matches | ❌ Not handled |
| App updates (package name changes) | Stale classifications | ❌ Not handled |
| Accessibility apps (TalkBack) | Misclassified as "system" | ❌ Not handled |
| Timezone changes | Wrong fatigue calculation | ❌ Not handled |
| Multi-device sync | Duplicate events | ❌ Not handled |

---

## 7. Implementation Plan

### Phase 1: P0 Fixes (Critical)
1. **Unify classifiers** — UsageCollector delegates to AppCategoryEngine
2. **Replace substring matching** — Exact package + Android category + user overrides
3. **Wire context signals** — Calendar, Health, Location, Screen into inference

### Phase 2: P1 Fixes (High Priority)
4. **Add missing activity rules** — meeting, exercise, commute, sleep, learning
5. **Add minimum duration** — 60s threshold before activity classification
6. **Add background exclusion** — Ignore background audio apps
7. **Add temporal transitions** — Hysteresis and cooldown periods

### Phase 3: P2 Fixes (Medium Priority)
8. **Make alignment configurable** — User overrides via DB
9. **Integrate fatigue signals** — Sleep, health, screen time
10. **Wire feedback loop** — Corrections → classification learning

---

## 8. Files to Modify

### Android (Kotlin)
- `android/app/src/main/java/com/andrometry/collectors/usage/UsageCollector.kt`
- `android/app/src/main/java/com/andrometry/inference/classifier/AppCategoryEngine.kt`
- `android/app/src/main/java/com/andrometry/inference/context/ContextInferenceEngine.kt`
- `android/app/src/main/java/com/andrometry/inference/segments/SegmentEngine.kt`
- `android/app/src/main/java/com/andrometry/inference/confidence/ConfidenceEngine.kt`
- `android/app/src/main/java/com/andrometry/inference/readiness/ReadinessEngine.kt`

### Server (Go) — Schema Updates
- `server/internal/models/models.go` — Update context/segment models
- `schemas/` — Update JSON schemas for new fields

---

## 9. Success Criteria

After implementation, the system should:
- [ ] Have a single unified classifier (no duplicate logic)
- [ ] Handle all 15 activity types from architecture §11.2
- [ ] Use exact package matching with <5% false positive rate
- [ ] Integrate calendar, health, location, and sleep signals
- [ ] Have configurable alignment values per user
- [ ] Include minimum duration thresholds for activity classification
- [ ] Exclude background audio apps from classification
- [ ] Have temporal transition detection with hysteresis
- [ ] Pass all existing tests with no regressions
- [ ] Have new tests for classification logic

---

## 10. References

- Architecture: `docs/architecture/architecture.md` (§9.1, §11.2, §12.3)
- Android Classifier: `android/app/src/main/java/com/andrometry/inference/classifier/AppCategoryEngine.kt`
- Usage Collector: `android/app/src/main/java/com/andrometry/collectors/usage/UsageCollector.kt`
- Context Inference: `android/app/src/main/java/com/andrometry/inference/context/ContextInferenceEngine.kt`
- Segment Engine: `android/app/src/main/java/com/andrometry/inference/segments/SegmentEngine.kt`
