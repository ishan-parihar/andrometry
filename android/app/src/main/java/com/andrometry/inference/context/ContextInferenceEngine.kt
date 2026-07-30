package com.andrometry.inference.context

import com.andrometry.core.database.CurrentContextEntity
import com.andrometry.core.database.EventEntity
import com.google.gson.Gson
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Context Inference Engine — the heart of the system.
 *
 * Produces a continuously updated current-context object from:
 * - Foreground app category
 * - App usage duration and switching rate
 * - Calendar state
 * - Focus session state
 * - Location
 * - Movement
 * - Time of day
 * - Notification load
 * - Health activity
 * - Sleep debt
 */
@Singleton
class ContextInferenceEngine @Inject constructor(
    private val preferencesBridge: PreferencesBridge? = null
) {

    data class ContextSnapshot(
        val primaryActivity: String,
        val activityConfidence: Double,
        val place: String,
        val placeConfidence: Double,
        val movement: String,
        val foregroundAppCategory: String,
        val screenState: String,
        val minutesSinceUnlock: Int,
        val focusActive: Boolean,
        val focusMinutesElapsed: Int,
        val focusSource: String?,
        val interruptionRate: String,
        val notificationsLast15m: Int,
        val distractionProbability: Double,
        val fatigueProbability: Double,
        val alignmentProbability: Double,
        val readinessScore: Double,
        val calendar: CalendarInfo?,
        val health: HealthInfo?,
        val sleep: SleepInfo?,
        val battery: BatteryInfo?,
        val dataQuality: DataQualityInfo
    )

    data class CalendarInfo(
        val currentEvent: String,
        val nextEventMinutes: Int?,
        val busyUntil: String?
    )

    data class HealthInfo(
        val stepsLastHour: Int,
        val activity: String,
        val heartRateAvg: Int?
    )

    data class SleepInfo(
        val lastSleepMinutes: Int?,
        val sleepDebtEstimateMinutes: Int?
    )

    data class BatteryInfo(
        val level: Double,
        val charging: Boolean
    )

    data class DataQualityInfo(
        val completeness: Double,
        val stalenessSeconds: Int,
        val usageStatsAvailable: Boolean,
        val notificationListenerAvailable: Boolean,
        val calendarAvailable: Boolean,
        val locationFresh: Boolean,
        val healthConnectAvailable: Boolean,
        val lastSyncMinutesAgo: Int,
        val missingSignals: List<String>,
        val missingCollectors: List<String>,
        val degradedCollectors: List<String>,
        val permissionIssues: List<String>,
        val clockDriftDetected: Boolean
    )

    /**
     * Minimum duration (seconds) before an app category can be used to classify activity.
     * Prevents transient app switches from affecting activity classification.
     */
    private val MINIMUM_ACTIVITY_DURATION_SECONDS = 60

    /**
     * Cooldown period (seconds) before switching activity type.
     * Prevents rapid oscillation between activities (e.g., social ↔ communication).
     * Must sustain new activity for this duration before it takes effect.
     */
    private val ACTIVITY_COOLDOWN_SECONDS = 120

    /**
     * Last classified activity for hysteresis.
     * Tracks the previous activity to apply cooldown logic.
     * Protected by hysteresisMutex for thread safety with suspend callers.
     */
    private val hysteresisMutex = Mutex()
    private var lastActivity: String? = null
    private var lastActivityTimestamp: Long = 0L

    /**
     * Applies temporal hysteresis: if switching to a different activity,
     * require that the new activity has been sustained for ACTIVITY_COOLDOWN_SECONDS.
     * Sensor-based detections (sleep, meeting, exercise, commute) bypass cooldown.
     * When sensor condition is no longer active, cooldown is bypassed to allow transition.
     */
    private suspend fun applyTemporalHysteresis(
        newActivity: String,
        newConfidence: Double,
        now: Long
    ): Pair<String, Double> {
        return hysteresisMutex.withLock {
        // Sensor-based activities bypass cooldown (they're high-confidence signals)
        val sensorActivities = setOf("sleep", "meeting", "exercise", "commute")
        if (newActivity in sensorActivities) {
            lastActivity = newActivity
            lastActivityTimestamp = now
            return newActivity to newConfidence
        }

        // If same activity, just update timestamp and return
        if (newActivity == lastActivity) {
            lastActivityTimestamp = now
            return newActivity to newConfidence
        }

        // If previous activity was sensor-based but no longer active, bypass cooldown
        if (lastActivity in sensorActivities) {
            lastActivity = newActivity
            lastActivityTimestamp = now
            return newActivity to newConfidence
        }

        // If switching to a new activity, check cooldown
        if (lastActivity != null && lastActivityTimestamp > 0) {
            val elapsedSeconds = (now - lastActivityTimestamp) / 1000L
            if (elapsedSeconds < ACTIVITY_COOLDOWN_SECONDS) {
                // Not enough time has passed — keep the previous activity
                // but reduce confidence slightly to signal uncertainty
                return lastActivity!! to (newConfidence * 0.8).coerceAtLeast(0.3)
            }
        }

        // Cooldown passed or first classification — accept new activity
        lastActivity = newActivity
        lastActivityTimestamp = now
        newActivity to newConfidence
        }
    }

    /**
     * Infers the current context from recent events.
     * 
     * Enhanced with:
     * - Calendar signals (meeting detection)
     * - Health signals (activity, heart rate)
     * - Location signals (place, movement)
     * - Screen unlock timing
     * - Sleep debt estimation
     * - Minimum duration thresholds
     * - Background audio app exclusion
     */
    suspend fun infer(
        recentEvents: List<EventEntity>,
        screenState: String = "on",
        currentAppCategory: String = "unknown",
        focusActive: Boolean = false,
        focusMinutes: Int = 0,
        focusSource: String? = null,
        batteryLevel: Double = 1.0,
        isCharging: Boolean = false,
        currentPlace: String = "unknown",
        placeConfidence: Double = 0.5,
        movement: String = "still",
        calendarInfo: CalendarInfo? = null,
        healthInfo: HealthInfo? = null,
        sleepInfo: SleepInfo? = null,
        minutesSinceUnlock: Int = 0,
        isBackgroundAudioApp: Boolean = false
    ): ContextSnapshot {
        val now = Calendar.getInstance()

        // Count app switches in last 5 minutes
        val fiveMinAgo = now.clone() as Calendar
        fiveMinAgo.add(Calendar.MINUTE, -5)
        val recentAppEvents = recentEvents.filter {
            it.eventType == "app_usage_summary"
        }
        val appSwitchCount = recentAppEvents.sumOf { event ->
            val payload = parsePayload(event.payload)
            (payload["app_switch_count"] as? Number)?.toInt() ?: 0
        }

        // Calculate foreground duration for current app category from most recent event only
        val foregroundDurationSeconds = recentAppEvents.lastOrNull()?.let { event ->
            val payload = parsePayload(event.payload)
            val byCategory = payload["by_category"] as? Map<*, *>
            (byCategory?.get(currentAppCategory) as? Number)?.toLong() ?: 0L
        } ?: 0L

        // Count notifications in last 15 minutes
        val fifteenMinAgo = now.clone() as Calendar
        fifteenMinAgo.add(Calendar.MINUTE, -15)
        val recentNotifEvents = recentEvents.filter {
            it.eventType == "notification_summary"
        }
        val notificationsLast15m = recentNotifEvents.sumOf { event ->
            val payload = parsePayload(event.payload)
            (payload["total"] as? Number)?.toInt() ?: 0
        }

        // Detect if user is in a meeting from calendar
        val isInMeeting = calendarInfo?.currentEvent != null && 
            calendarInfo.currentEvent != "none" &&
            calendarInfo.busyUntil != null

        // Detect exercise from health data
        val isExercising = healthInfo?.activity == "walking" || 
            healthInfo?.activity == "running" ||
            healthInfo?.activity == "cycling" ||
            healthInfo?.activity == "vehicle" // in transit

        // Detect sleep from screen off + time + location
        val isSleeping = screenState == "off" &&
            now.get(Calendar.HOUR_OF_DAY) in 0..6 &&
            currentPlace == "home" &&
            movement == "still"

        // Detect commute from movement + location
        val isCommute = movement == "in_transit" ||
            (currentPlace == "unknown" && healthInfo?.activity == "vehicle")

        // Classify activity with enhanced signals
        val classification = classifyActivity(
            focusActive = focusActive,
            currentAppCategory = currentAppCategory,
            appSwitchCount = appSwitchCount,
            notificationsLast15m = notificationsLast15m,
            isInMeeting = isInMeeting,
            isExercising = isExercising,
            isSleeping = isSleeping,
            isCommute = isCommute,
            foregroundDurationSeconds = foregroundDurationSeconds,
            isBackgroundAudioApp = isBackgroundAudioApp,
            currentPlace = currentPlace,
            healthInfo = healthInfo
        )

        // Calculate probabilities
        val distractionProb = calculateDistractionProbability(
            appSwitchCount = appSwitchCount,
            currentAppCategory = currentAppCategory,
            notificationsLast15m = notificationsLast15m,
            focusActive = focusActive,
            isInMeeting = isInMeeting
        )

        val fatigueProb = calculateFatigueProbability(
            now = now,
            sleepInfo = sleepInfo,
            healthInfo = healthInfo,
            minutesSinceUnlock = minutesSinceUnlock
        )

        val alignmentProb = calculateAlignmentProbability(
            focusActive = focusActive,
            currentAppCategory = currentAppCategory,
            calendarInfo = calendarInfo,
            isInMeeting = isInMeeting
        )

        val readinessScore = calculateReadinessScore(
            fatigueProbability = fatigueProb,
            batteryLevel = batteryLevel,
            sleepInfo = sleepInfo,
            healthInfo = healthInfo
        )

        // Calculate staleness
        val oldestEvent = recentEvents.minByOrNull {
            parseTimestamp(it.timestamp)?.time ?: 0L
        }
        val stalenessSeconds = if (oldestEvent != null) {
            val ts = parseTimestamp(oldestEvent.timestamp)
            if (ts != null) {
                val diffMs = now.timeInMillis - ts.time
                (diffMs / 1000L).toInt()
            } else 999
        } else 999

        // Data completeness
        val availableSignals = mutableListOf(
            "screen_state" to true,
            "app_usage" to recentAppEvents.isNotEmpty(),
            "notifications" to recentNotifEvents.isNotEmpty(),
            "battery" to true
        )
        if (calendarInfo != null) availableSignals.add("calendar" to true)
        if (healthInfo != null) availableSignals.add("health" to true)
        if (sleepInfo != null) availableSignals.add("sleep" to true)
        if (currentPlace != "unknown") availableSignals.add("location" to true)
        
        val completeness = availableSignals.count { it.second }.toDouble() / availableSignals.size

        // Track missing signals
        val missingSignals = mutableListOf<String>()
        if (calendarInfo == null) missingSignals.add("calendar")
        if (healthInfo == null) missingSignals.add("health")
        if (sleepInfo == null) missingSignals.add("sleep")
        if (currentPlace == "unknown") missingSignals.add("location")

        return ContextSnapshot(
            primaryActivity = classification.first,
            activityConfidence = classification.second,
            place = currentPlace,
            placeConfidence = placeConfidence,
            movement = movement,
            foregroundAppCategory = currentAppCategory,
            screenState = screenState,
            minutesSinceUnlock = minutesSinceUnlock,
            focusActive = focusActive,
            focusMinutesElapsed = focusMinutes,
            focusSource = focusSource,
            interruptionRate = when {
                notificationsLast15m > 10 -> "high"
                notificationsLast15m > 5 -> "medium"
                notificationsLast15m > 0 -> "low"
                else -> "none"
            },
            notificationsLast15m = notificationsLast15m,
            distractionProbability = distractionProb,
            fatigueProbability = fatigueProb,
            alignmentProbability = alignmentProb,
            readinessScore = readinessScore,
            calendar = calendarInfo,
            health = healthInfo,
            sleep = sleepInfo,
            battery = BatteryInfo(batteryLevel, isCharging),
            dataQuality = DataQualityInfo(
                completeness = completeness,
                stalenessSeconds = stalenessSeconds,
                usageStatsAvailable = true,
                notificationListenerAvailable = true,
                calendarAvailable = calendarInfo != null,
                locationFresh = currentPlace != "unknown",
                healthConnectAvailable = healthInfo != null,
                lastSyncMinutesAgo = 0,
                missingSignals = missingSignals,
                missingCollectors = emptyList(),
                degradedCollectors = emptyList(),
                permissionIssues = emptyList(),
                clockDriftDetected = false
            )
        )
    }

    /**
     * Classifies activity with enhanced signals.
     * Handles all 15 activity types from architecture §11.2.
     * 
     * Priority order:
     * 1. Sleep (screen off + night + home + still)
     * 2. Meeting (calendar busy)
     * 3. Exercise (health activity)
     * 4. Commute (movement/transit)
     * 5. Focus-based (deep_work, shallow_work)
     * 6. Category-based (communication, social, leisure, etc.)
     */
    private suspend fun classifyActivity(
        focusActive: Boolean,
        currentAppCategory: String,
        appSwitchCount: Int,
        notificationsLast15m: Int,
        isInMeeting: Boolean,
        isExercising: Boolean,
        isSleeping: Boolean,
        isCommute: Boolean,
        foregroundDurationSeconds: Long,
        isBackgroundAudioApp: Boolean,
        currentPlace: String,
        healthInfo: HealthInfo?
    ): Pair<String, Double> {
        val now = System.currentTimeMillis()

        // Priority 1-4: Sensor-based detection (high confidence, early return)
        val sensorResult = classifyFromSensors(isSleeping, isInMeeting, isExercising, isCommute)
        if (sensorResult != null) return applyTemporalHysteresis(sensorResult.first, sensorResult.second, now)

        // Minimum duration threshold for background audio apps
        if (foregroundDurationSeconds < MINIMUM_ACTIVITY_DURATION_SECONDS && isBackgroundAudioApp) {
            return applyTemporalHysteresis("unknown", 0.40, now)
        }

        // Priority 5: Focus-based classification
        val focusResult = classifyFromFocus(focusActive, currentAppCategory, appSwitchCount)
        if (focusResult != null) {
            val modified = applyConfidenceModifiers(focusResult, notificationsLast15m, focusActive)
            return applyTemporalHysteresis(modified.first, modified.second, now)
        }

        // Priority 6: Category-based classification
        val categoryResult = classifyFromCategory(currentAppCategory, focusActive, appSwitchCount)
        val modified = applyConfidenceModifiers(categoryResult, notificationsLast15m, focusActive)
        return applyTemporalHysteresis(modified.first, modified.second, now)
    }

    /**
     * Priority 1-4: Sensor-based detection (sleep, meeting, exercise, commute).
     * Returns null if none match, allowing fallthrough to app-based classification.
     */
    private fun classifyFromSensors(
        isSleeping: Boolean,
        isInMeeting: Boolean,
        isExercising: Boolean,
        isCommute: Boolean
    ): Pair<String, Double>? {
        if (isSleeping) return "sleep" to 0.85
        if (isInMeeting) return "meeting" to 0.82
        if (isExercising) return "exercise" to 0.80
        if (isCommute) return "commute" to 0.75
        return null
    }

    /**
     * Priority 5: Focus-based classification.
     * Returns null if focus is inactive or category doesn't match.
     */
    private fun classifyFromFocus(
        focusActive: Boolean,
        currentAppCategory: String,
        appSwitchCount: Int
    ): Pair<String, Double>? {
        if (!focusActive) return null

        return when {
            currentAppCategory == "productivity" && appSwitchCount < 4 -> "deep_work" to 0.81
            currentAppCategory == "productivity" -> "shallow_work" to 0.70
            currentAppCategory == "browser" && appSwitchCount > 5 -> "shallow_work" to 0.65
            currentAppCategory == "education" -> "learning" to 0.78
            currentAppCategory == "browser" -> "learning" to 0.60
            else -> null
        }
    }

    /**
     * Priority 6: Category-based classification (no focus active).
     */
    private fun classifyFromCategory(
        currentAppCategory: String,
        focusActive: Boolean,
        appSwitchCount: Int
    ): Pair<String, Double> {
        return when (currentAppCategory) {
            "communication", "email" -> "communication" to 0.75
            "social" -> if (focusActive) "distraction" to 0.80 else "social" to 0.70
            "entertainment" -> "leisure" to 0.65
            "browser" -> if (appSwitchCount > 5) "shallow_work" to 0.60 else "learning" to 0.60
            "news", "education" -> "learning" to if (currentAppCategory == "education") 0.72 else 0.55
            "health" -> "break" to 0.60
            "system", "tools" -> "admin" to 0.55
            "navigation" -> "commute" to 0.70
            "shopping" -> "break" to 0.55
            "calendar" -> "planning" to 0.70
            else -> "unknown" to 0.50
        }
    }

    /**
     * Applies notification penalty and focus bonus to confidence.
     */
    private fun applyConfidenceModifiers(
        result: Pair<String, Double>,
        notificationsLast15m: Int,
        focusActive: Boolean
    ): Pair<String, Double> {
        var confidence = result.second

        // Penalty for high notification load (interruptions)
        if (notificationsLast15m > 10) {
            confidence *= 0.9
        }

        // Bonus for consistent signals
        if (focusActive && result.first in listOf("deep_work", "shallow_work", "learning")) {
            confidence = (confidence * 1.1).coerceAtMost(0.95)
        }

        return result.first to confidence.coerceIn(0.0, 1.0)
    }

    private fun calculateDistractionProbability(
        appSwitchCount: Int,
        currentAppCategory: String,
        notificationsLast15m: Int,
        focusActive: Boolean,
        isInMeeting: Boolean
    ): Double {
        var prob = 0.1

        // High app switching increases distraction
        prob += appSwitchCount * 0.02

        // Social/entertainment apps are distracting
        if (currentAppCategory == "social") prob += 0.3
        if (currentAppCategory == "entertainment") prob += 0.2

        // High notifications increase distraction
        prob += notificationsLast15m * 0.02

        // Focus active reduces distraction probability
        if (focusActive) prob *= 0.7

        // Meeting context reduces distraction
        if (isInMeeting) prob *= 0.5

        return prob.coerceIn(0.0, 1.0)
    }

    /**
     * Calculates fatigue probability using multiple signals:
     * - Time of day (circadian rhythm)
     * - Sleep debt from previous night
     * - Health activity (exercise reduces fatigue)
     * - Screen time accumulation
     */
    private fun calculateFatigueProbability(
        now: Calendar,
        sleepInfo: SleepInfo?,
        healthInfo: HealthInfo?,
        minutesSinceUnlock: Int
    ): Double {
        val hour = now.get(Calendar.HOUR_OF_DAY)

        // Base fatigue from time of day (circadian rhythm)
        var baseFatigue = when {
            hour in 0..5 -> 0.7
            hour in 6..7 -> 0.4
            hour in 8..11 -> 0.2
            hour in 12..13 -> 0.35 // post-lunch dip
            hour in 14..17 -> 0.25
            hour in 18..20 -> 0.35
            hour in 21..22 -> 0.5
            else -> 0.65
        }

        // Sleep debt adjustment
        if (sleepInfo?.lastSleepMinutes != null) {
            val sleepDebt = sleepInfo.sleepDebtEstimateMinutes ?: 0
            // More sleep debt = higher fatigue
            baseFatigue += (sleepDebt / 60.0) * 0.1
        }

        // Health activity adjustment (exercise reduces fatigue)
        if (healthInfo?.activity == "walking" || healthInfo?.activity == "running") {
            baseFatigue *= 0.85 // 15% reduction for active state
        }

        // Screen time accumulation (long screen time increases fatigue)
        if (minutesSinceUnlock > 120) {
            baseFatigue += 0.1 // 10% increase after 2 hours of screen time
        } else if (minutesSinceUnlock > 60) {
            baseFatigue += 0.05 // 5% increase after 1 hour
        }

        return baseFatigue.coerceIn(0.0, 1.0)
    }

    /**
     * Calculates alignment probability based on:
     * - Focus state
     * - App category alignment
     * - Calendar context (meeting = aligned)
     */
    private suspend fun calculateAlignmentProbability(
        focusActive: Boolean,
        currentAppCategory: String,
        calendarInfo: CalendarInfo?,
        isInMeeting: Boolean
    ): Double {
        // Delegate to PreferencesBridge if available (uses server-configured weights)
        if (preferencesBridge != null) {
            return preferencesBridge.calculateAlignmentProbability(
                focusActive = focusActive,
                currentAppCategory = currentAppCategory,
                isInMeeting = isInMeeting
            )
        }

        // Fallback: hardcoded defaults (matches PreferencesBridge.defaultWeights)
        var alignment = 0.5
        if (focusActive) alignment += 0.3
        if (currentAppCategory in listOf("productivity", "education", "email", "calendar")) {
            alignment += 0.15
        }
        if (isInMeeting) alignment += 0.1
        if (currentAppCategory in listOf("social", "entertainment")) {
            alignment -= 0.2
        }
        return alignment.coerceIn(0.0, 1.0)
    }

    /**
     * Calculates readiness score using:
     * - Fatigue probability
     * - Battery level
     * - Sleep quality
     * - Health activity
     */
    private fun calculateReadinessScore(
        fatigueProbability: Double,
        batteryLevel: Double,
        sleepInfo: SleepInfo?,
        healthInfo: HealthInfo?
    ): Double {
        val fatigueComponent = (1.0 - fatigueProbability) * 0.5
        val batteryComponent = batteryLevel * 0.2

        var sleepComponent = 0.0
        if (sleepInfo?.lastSleepMinutes != null) {
            // Optimal sleep is 420-540 minutes (7-9 hours)
            val sleepMinutes = sleepInfo.lastSleepMinutes
            sleepComponent = when {
                sleepMinutes in 420..540 -> 0.2 // optimal
                sleepMinutes in 360..420 -> 0.15 // slightly short
                sleepMinutes in 540..600 -> 0.15 // slightly long
                else -> 0.1 // poor sleep
            }
        }

        var healthComponent = 0.0
        if (healthInfo?.activity == "walking" || healthInfo?.activity == "running") {
            healthComponent = 0.1 // active state boosts readiness
        }

        return (fatigueComponent + batteryComponent + sleepComponent + healthComponent).coerceIn(0.0, 1.0)
    }

    fun toCurrentContextEntity(snapshot: ContextSnapshot): CurrentContextEntity {
        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val contextMap = mapOf(
            "primary_activity" to snapshot.primaryActivity,
            "activity_confidence" to snapshot.activityConfidence,
            "place" to snapshot.place,
            "place_confidence" to snapshot.placeConfidence,
            "movement" to snapshot.movement,
            "foreground_app_category" to snapshot.foregroundAppCategory,
            "screen_state" to snapshot.screenState,
            "minutes_since_unlock" to snapshot.minutesSinceUnlock,
            "focus_active" to snapshot.focusActive,
            "focus_minutes_elapsed" to snapshot.focusMinutesElapsed,
            "focus_source" to snapshot.focusSource,
            "interruption_rate" to snapshot.interruptionRate,
            "notifications_last_15m" to snapshot.notificationsLast15m,
            "distraction_probability" to snapshot.distractionProbability,
            "fatigue_probability" to snapshot.fatigueProbability,
            "alignment_probability" to snapshot.alignmentProbability,
            "readiness_score" to snapshot.readinessScore,
            "calendar" to snapshot.calendar,
            "health" to snapshot.health,
            "sleep" to snapshot.sleep,
            "battery" to snapshot.battery,
            "data_quality" to snapshot.dataQuality,
            "timezone" to TimeZone.getDefault().id
        )

        val contextJson = Gson().toJson(contextMap)

        return CurrentContextEntity(
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            context = contextJson,
            syncState = "pending"
        )
    }

    private fun parsePayload(payload: String): Map<String, Any?> {
        return try {
            Gson().fromJson(payload, Map::class.java) as? Map<String, Any?> ?: emptyMap()
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun parseTimestamp(ts: String): Date? {
        return try {
            SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }.parse(ts)
        } catch (e: Exception) {
            null
        }
    }
}
