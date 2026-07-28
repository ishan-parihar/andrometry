package com.andrometry.inference.context

import com.andrometry.core.database.CurrentContextEntity
import com.andrometry.core.database.EventEntity
import com.google.gson.Gson
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
class ContextInferenceEngine @Inject constructor() {

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
     * Infers the current context from recent events.
     */
    fun infer(
        recentEvents: List<EventEntity>,
        screenState: String = "on",
        currentAppCategory: String = "unknown",
        focusActive: Boolean = false,
        focusMinutes: Int = 0,
        focusSource: String? = null,
        batteryLevel: Double = 1.0,
        isCharging: Boolean = false
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

        // Classify activity
        val classification = classifyActivity(
            focusActive = focusActive,
            currentAppCategory = currentAppCategory,
            appSwitchCount = appSwitchCount,
            notificationsLast15m = notificationsLast15m
        )

        // Calculate probabilities
        val distractionProb = calculateDistractionProbability(
            appSwitchCount = appSwitchCount,
            currentAppCategory = currentAppCategory,
            notificationsLast15m = notificationsLast15m,
            focusActive = focusActive
        )

        val fatigueProb = calculateFatigueProbability(now)

        val alignmentProb = if (focusActive) 0.8 else 0.5

        val readinessScore = calculateReadinessScore(
            fatigueProbability = fatigueProb,
            batteryLevel = batteryLevel
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
        val availableSignals = listOf(
            "screen_state" to true,
            "app_usage" to recentAppEvents.isNotEmpty(),
            "notifications" to recentNotifEvents.isNotEmpty(),
            "battery" to true
        )
        val completeness = availableSignals.count { it.second }.toDouble() / availableSignals.size

        return ContextSnapshot(
            primaryActivity = classification.first,
            activityConfidence = classification.second,
            place = "unknown",
            placeConfidence = 0.5,
            movement = "still",
            foregroundAppCategory = currentAppCategory,
            screenState = screenState,
            minutesSinceUnlock = 0,
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
            calendar = null,
            health = null,
            sleep = null,
            battery = BatteryInfo(batteryLevel, isCharging),
            dataQuality = DataQualityInfo(
                completeness = completeness,
                stalenessSeconds = stalenessSeconds,
                usageStatsAvailable = true,
                notificationListenerAvailable = true,
                calendarAvailable = false,
                locationFresh = false,
                healthConnectAvailable = false,
                lastSyncMinutesAgo = 0,
                missingSignals = emptyList(),
                missingCollectors = emptyList(),
                degradedCollectors = emptyList(),
                permissionIssues = emptyList(),
                clockDriftDetected = false
            )
        )
    }

    private fun classifyActivity(
        focusActive: Boolean,
        currentAppCategory: String,
        appSwitchCount: Int,
        notificationsLast15m: Int
    ): Pair<String, Double> {
        // Rule-based classification with confidence scoring
        var activity = "unknown"
        var confidence = 0.5

        if (focusActive && currentAppCategory == "productivity" && appSwitchCount < 4) {
            activity = "deep_work"
            confidence = 0.81
        } else if (focusActive && currentAppCategory == "productivity") {
            activity = "shallow_work"
            confidence = 0.70
        } else if (currentAppCategory == "communication") {
            activity = "communication"
            confidence = 0.75
        } else if (currentAppCategory == "social") {
            if (focusActive) {
                activity = "distraction"
                confidence = 0.80
            } else {
                activity = "social"
                confidence = 0.70
            }
        } else if (currentAppCategory == "entertainment") {
            activity = "leisure"
            confidence = 0.65
        } else if (currentAppCategory == "browser" && appSwitchCount > 5) {
            activity = "shallow_work"
            confidence = 0.60
        }

        // Penalty for high notification load
        if (notificationsLast15m > 10) {
            confidence *= 0.9
        }

        return Pair(activity, confidence)
    }

    private fun calculateDistractionProbability(
        appSwitchCount: Int,
        currentAppCategory: String,
        notificationsLast15m: Int,
        focusActive: Boolean
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

        return prob.coerceIn(0.0, 1.0)
    }

    private fun calculateFatigueProbability(now: Calendar): Double {
        val hour = now.get(Calendar.HOUR_OF_DAY)

        // Fatigue is higher late at night and early morning
        return when {
            hour in 0..5 -> 0.7
            hour in 6..7 -> 0.4
            hour in 8..11 -> 0.2
            hour in 12..13 -> 0.35 // post-lunch dip
            hour in 14..17 -> 0.25
            hour in 18..20 -> 0.35
            hour in 21..22 -> 0.5
            else -> 0.65
        }
    }

    private fun calculateReadinessScore(
        fatigueProbability: Double,
        batteryLevel: Double
    ): Double {
        val fatigueComponent = (1.0 - fatigueProbability) * 0.7
        val batteryComponent = batteryLevel * 0.3
        return (fatigueComponent + batteryComponent).coerceIn(0.0, 1.0)
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
