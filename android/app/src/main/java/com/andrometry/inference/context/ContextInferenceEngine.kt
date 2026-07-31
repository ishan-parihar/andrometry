package com.andrometry.inference.context

import com.andrometry.core.database.CurrentContextEntity
import com.andrometry.core.database.EventEntity
import com.andrometry.inference.readiness.ReadinessEngine
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
 *
 * Delegates to ReadinessEngine (§40) for fatigue/readiness scoring,
 * providing richer output including:
 * - cognitiveLoad
 * - suggestedBreak
 * - bestNextAction
 * - sleepDebtMinutes
 * - recommendedFocusIntensity
 */
@Singleton
class ContextInferenceEngine @Inject constructor(
    private val preferencesBridge: PreferencesBridge? = null,
    private val readinessEngine: ReadinessEngine
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
        val cognitiveLoad: String,
        val suggestedBreak: Boolean,
        val bestNextAction: String,
        val sleepDebtMinutes: Int,
        val recommendedFocusIntensity: String,
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

    private val MINIMUM_ACTIVITY_DURATION_SECONDS = 60
    private val ACTIVITY_COOLDOWN_SECONDS = 120

    private val hysteresisMutex = Mutex()
    private var lastActivity: String? = null
    private var lastActivityTimestamp: Long = 0L

    private suspend fun applyTemporalHysteresis(
        newActivity: String,
        newConfidence: Double,
        now: Long
    ): Pair<String, Double> {
        return hysteresisMutex.withLock {
            val sensorActivities = setOf("sleep", "meeting", "exercise", "commute")
            if (newActivity in sensorActivities) {
                lastActivity = newActivity
                lastActivityTimestamp = now
                return@withLock newActivity to newConfidence
            }

            if (newActivity == lastActivity) {
                lastActivityTimestamp = now
                return@withLock newActivity to newConfidence
            }

            if (lastActivity in sensorActivities) {
                lastActivity = newActivity
                lastActivityTimestamp = now
                return@withLock newActivity to newConfidence
            }

            if (lastActivity != null && lastActivityTimestamp > 0) {
                val elapsedSeconds = (now - lastActivityTimestamp) / 1000L
                if (elapsedSeconds < ACTIVITY_COOLDOWN_SECONDS) {
                    return@withLock lastActivity!! to (newConfidence * 0.8).coerceAtLeast(0.3)
                }
            }

            lastActivity = newActivity
            lastActivityTimestamp = now
            newActivity to newConfidence
        }
    }

    /**
     * Infers the current context from recent events.
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

        val recentAppEvents = recentEvents.filter { it.eventType == "app_usage_summary" }
        val appSwitchCount = recentAppEvents.sumOf { event ->
            val payload = parsePayload(event.payload)
            (payload["app_switch_count"] as? Number)?.toInt() ?: 0
        }

        val foregroundDurationSeconds = recentAppEvents.lastOrNull()?.let { event ->
            val payload = parsePayload(event.payload)
            val byCategory = payload["by_category"] as? Map<*, *>
            (byCategory?.get(currentAppCategory) as? Number)?.toLong() ?: 0L
        } ?: 0L

        val recentNotifEvents = recentEvents.filter { it.eventType == "notification_summary" }
        val notificationsLast15m = recentNotifEvents.sumOf { event ->
            val payload = parsePayload(event.payload)
            (payload["total"] as? Number)?.toInt() ?: 0
        }

        val isInMeeting = calendarInfo?.currentEvent != null &&
            calendarInfo.currentEvent != "none" &&
            calendarInfo.busyUntil != null

        val isExercising = healthInfo?.activity in listOf("walking", "running", "cycling", "vehicle")

        val isSleeping = screenState == "off" &&
            now.get(Calendar.HOUR_OF_DAY) in 0..6 &&
            currentPlace == "home" &&
            movement == "still"

        val isCommute = movement == "in_transit" ||
            (currentPlace == "unknown" && healthInfo?.activity == "vehicle")

        val classification = classifyActivity(
            focusActive, currentAppCategory, appSwitchCount, notificationsLast15m,
            isInMeeting, isExercising, isSleeping, isCommute,
            foregroundDurationSeconds, isBackgroundAudioApp, currentPlace, healthInfo
        )

        val distractionProb = calculateDistractionProbability(
            appSwitchCount, currentAppCategory, notificationsLast15m, focusActive, isInMeeting
        )

        val alignmentProb = calculateAlignmentProbability(focusActive, currentAppCategory, calendarInfo, isInMeeting)

        // Delegate to ReadinessEngine (§40) for comprehensive fatigue/readiness scoring
        val hoursSinceWake = calculateHoursSinceWake(now)
        val readinessOutput = readinessEngine.computeReadiness(
            ReadinessEngine.ReadinessInput(
                sleepDurationMinutes = sleepInfo?.lastSleepMinutes ?: 0,
                sleepRegularityScore = 0.5,
                hoursSinceWake = hoursSinceWake,
                lateNightScreenMinutes = if (now.get(Calendar.HOUR_OF_DAY) in 22..23 || now.get(Calendar.HOUR_OF_DAY) in 0..6) minutesSinceUnlock else 0,
                heartRateAvg = healthInfo?.heartRateAvg ?: 0,
                stepsToday = (healthInfo?.stepsLastHour ?: 0) * 6,
                focusMinutesToday = focusMinutes,
                meetingMinutesToday = 0
            )
        )

        val oldestEvent = recentEvents.minByOrNull { parseTimestamp(it.timestamp)?.time ?: 0L }
        val stalenessSeconds = if (oldestEvent != null) {
            val ts = parseTimestamp(oldestEvent.timestamp)
            if (ts != null) ((now.timeInMillis - ts.time) / 1000L).toInt() else 999
        } else 999

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
            fatigueProbability = readinessOutput.fatigueProbability,
            alignmentProbability = alignmentProb,
            readinessScore = readinessOutput.readinessScore,
            cognitiveLoad = readinessOutput.cognitiveLoad,
            suggestedBreak = readinessOutput.suggestedBreak,
            bestNextAction = readinessOutput.bestNextAction,
            sleepDebtMinutes = readinessOutput.sleepDebtMinutes,
            recommendedFocusIntensity = readinessOutput.recommendedFocusIntensity,
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

    private fun calculateHoursSinceWake(now: Calendar): Double {
        val hour = now.get(Calendar.HOUR_OF_DAY)
        val wakeHour = 7
        return if (hour >= wakeHour) (hour - wakeHour).toDouble() else (24 - wakeHour + hour).toDouble()
    }

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

        val sensorResult = classifyFromSensors(isSleeping, isInMeeting, isExercising, isCommute)
        if (sensorResult != null) return applyTemporalHysteresis(sensorResult.first, sensorResult.second, now)

        if (foregroundDurationSeconds < MINIMUM_ACTIVITY_DURATION_SECONDS && isBackgroundAudioApp) {
            return applyTemporalHysteresis("unknown", 0.40, now)
        }

        val focusResult = classifyFromFocus(focusActive, currentAppCategory, appSwitchCount)
        if (focusResult != null) {
            val modified = applyConfidenceModifiers(focusResult, notificationsLast15m, focusActive)
            return applyTemporalHysteresis(modified.first, modified.second, now)
        }

        val categoryResult = classifyFromCategory(currentAppCategory, focusActive, appSwitchCount)
        val modified = applyConfidenceModifiers(categoryResult, notificationsLast15m, focusActive)
        return applyTemporalHysteresis(modified.first, modified.second, now)
    }

    private fun classifyFromSensors(isSleeping: Boolean, isInMeeting: Boolean, isExercising: Boolean, isCommute: Boolean): Pair<String, Double>? {
        if (isSleeping) return "sleep" to 0.85
        if (isInMeeting) return "meeting" to 0.82
        if (isExercising) return "exercise" to 0.80
        if (isCommute) return "commute" to 0.75
        return null
    }

    private fun classifyFromFocus(focusActive: Boolean, currentAppCategory: String, appSwitchCount: Int): Pair<String, Double>? {
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

    private fun classifyFromCategory(currentAppCategory: String, focusActive: Boolean, appSwitchCount: Int): Pair<String, Double> {
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

    private fun applyConfidenceModifiers(result: Pair<String, Double>, notificationsLast15m: Int, focusActive: Boolean): Pair<String, Double> {
        var confidence = result.second
        if (notificationsLast15m > 10) confidence *= 0.9
        if (focusActive && result.first in listOf("deep_work", "shallow_work", "learning")) {
            confidence = (confidence * 1.1).coerceAtMost(0.95)
        }
        return result.first to confidence.coerceIn(0.0, 1.0)
    }

    private fun calculateDistractionProbability(appSwitchCount: Int, currentAppCategory: String, notificationsLast15m: Int, focusActive: Boolean, isInMeeting: Boolean): Double {
        var prob = 0.1
        prob += appSwitchCount * 0.02
        if (currentAppCategory == "social") prob += 0.3
        if (currentAppCategory == "entertainment") prob += 0.2
        prob += notificationsLast15m * 0.02
        if (focusActive) prob *= 0.7
        if (isInMeeting) prob *= 0.5
        return prob.coerceIn(0.0, 1.0)
    }

    private suspend fun calculateAlignmentProbability(focusActive: Boolean, currentAppCategory: String, calendarInfo: CalendarInfo?, isInMeeting: Boolean): Double {
        if (preferencesBridge != null) {
            return preferencesBridge.calculateAlignmentProbability(focusActive, currentAppCategory, isInMeeting)
        }
        var alignment = 0.5
        if (focusActive) alignment += 0.3
        if (currentAppCategory in listOf("productivity", "education", "email", "calendar")) alignment += 0.15
        if (isInMeeting) alignment += 0.1
        if (currentAppCategory in listOf("social", "entertainment")) alignment -= 0.2
        return alignment.coerceIn(0.0, 1.0)
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
            "cognitive_load" to snapshot.cognitiveLoad,
            "suggested_break" to snapshot.suggestedBreak,
            "best_next_action" to snapshot.bestNextAction,
            "sleep_debt_minutes" to snapshot.sleepDebtMinutes,
            "recommended_focus_intensity" to snapshot.recommendedFocusIntensity,
            "calendar" to snapshot.calendar,
            "health" to snapshot.health,
            "sleep" to snapshot.sleep,
            "battery" to snapshot.battery,
            "data_quality" to snapshot.dataQuality,
            "timezone" to TimeZone.getDefault().id
        )

        return CurrentContextEntity(
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            context = Gson().toJson(contextMap),
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
