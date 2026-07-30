package com.andrometry.inference.context

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.inference.classifier.AppCategoryEngine
import com.andrometry.inference.context.ContextInferenceEngine.*
import com.google.gson.Gson
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Context Inference Bridge — connects collector outputs to ContextInferenceEngine.
 *
 * Extracts structured signals from raw events produced by:
 * - CalendarCollector → CalendarInfo
 * - HealthCollector → HealthInfo, SleepInfo
 * - LocationCollector → place, movement
 * - ScreenReceiver → screenState, minutesSinceUnlock
 * - FocusCollector → focusActive, focusMinutes
 * - UsageCollector → currentAppCategory, appSwitchCount
 *
 * This bridge is called by the sync worker before uploading context to the server.
 */
@Singleton
class ContextInferenceBridge @Inject constructor(
    private val database: AndrometryDatabase,
    private val appCategoryEngine: AppCategoryEngine,
    private val contextInferenceEngine: ContextInferenceEngine
) {
    /**
     * Extracts CalendarInfo from the most recent calendar_context event.
     */
    fun extractCalendarInfo(recentEvents: List<EventEntity>): CalendarInfo? {
        val calendarEvent = recentEvents
            .filter { it.eventType == "calendar_context" }
            .maxByOrNull { it.timestamp } ?: return null

        val payload = parsePayload(calendarEvent.payload)
        val currentEvent = payload["current_event"] as? String ?: "none"
        val nextEventMinutes = (payload["next_event_minutes"] as? Number)?.toInt()
        val busyUntil = payload["busy_until"] as? String

        return CalendarInfo(
            currentEvent = currentEvent,
            nextEventMinutes = nextEventMinutes,
            busyUntil = busyUntil
        )
    }

    /**
     * Extracts HealthInfo from the most recent health_summary event.
     */
    fun extractHealthInfo(recentEvents: List<EventEntity>): HealthInfo? {
        val healthEvent = recentEvents
            .filter { it.eventType == "health_summary" }
            .maxByOrNull { it.timestamp } ?: return null

        val payload = parsePayload(healthEvent.payload)
        val steps = (payload["steps"] as? Number)?.toInt() ?: 0
        val heartRateAvg = (payload["heart_rate_avg"] as? Number)?.toInt()

        // Determine activity from recent exercise_session events (within last 30 minutes)
        val now = java.util.Date()
        val thirtyMinAgo = java.util.Date(now.time - 30 * 60 * 1000)
        val exerciseEvent = recentEvents
            .filter { it.eventType == "exercise_session" }
            .filter { parseTimestamp(it.timestamp)?.after(thirtyMinAgo) == true }
            .maxByOrNull { it.timestamp }

        val activity = if (exerciseEvent != null) {
            val exercisePayload = parsePayload(exerciseEvent.payload)
            val activities = exercisePayload["activities"] as? List<*>
            when {
                activities?.any { it.toString().contains("run", ignoreCase = true) } == true -> "running"
                activities?.any { it.toString().contains("walk", ignoreCase = true) } == true -> "walking"
                activities?.any { it.toString().contains("cycle", ignoreCase = true) } == true -> "cycling"
                activities?.any { it.toString().contains("drive", ignoreCase = true) ||
                    it.toString().contains("vehicle", ignoreCase = true) } == true -> "vehicle"
                else -> "exercise"
            }
        } else {
            "still"
        }

        return HealthInfo(
            stepsLastHour = steps,
            activity = activity,
            heartRateAvg = heartRateAvg
        )
    }

    /**
     * Extracts SleepInfo from the most recent sleep_summary event.
     */
    fun extractSleepInfo(recentEvents: List<EventEntity>): SleepInfo? {
        val sleepEvent = recentEvents
            .filter { it.eventType == "sleep_summary" }
            .maxByOrNull { it.timestamp } ?: return null

        val payload = parsePayload(sleepEvent.payload)
        val sleepMinutes = (payload["sleep_minutes"] as? Number)?.toInt()

        // Estimate sleep debt: optimal is 480 minutes (8 hours)
        val sleepDebt = if (sleepMinutes != null && sleepMinutes < 480) {
            480 - sleepMinutes
        } else {
            0
        }

        return SleepInfo(
            lastSleepMinutes = sleepMinutes,
            sleepDebtEstimateMinutes = sleepDebt
        )
    }

    /**
     * Extracts place and movement from the most recent location_context event.
     */
    fun extractLocationInfo(recentEvents: List<EventEntity>): Pair<String, String> {
        val locationEvent = recentEvents
            .filter { it.eventType == "location_context" }
            .maxByOrNull { it.timestamp } ?: return "unknown" to "still"

        val payload = parsePayload(locationEvent.payload)
        val place = payload["place"] as? String ?: "unknown"
        val movement = payload["movement"] as? String ?: "still"

        return place to movement
    }

    /**
     * Extracts screen state from the most recent screen_state_changed event.
     */
    fun extractScreenState(recentEvents: List<EventEntity>): String {
        val screenEvent = recentEvents
            .filter { it.eventType == "screen_state_changed" }
            .maxByOrNull { it.timestamp } ?: return "on"

        val payload = parsePayload(screenEvent.payload)
        return payload["state"] as? String ?: "on"
    }

    /**
     * Extracts focus state from the most recent focus_session event.
     */
    fun extractFocusInfo(recentEvents: List<EventEntity>): Triple<Boolean, Int, String?> {
        val focusEvents = recentEvents
            .filter { it.eventType == "focus_session" }
            .sortedByDescending { it.timestamp }

        if (focusEvents.isEmpty()) return Triple(false, 0, null)

        val latestFocus = focusEvents.first()
        val payload = parsePayload(latestFocus.payload)
        val action = payload["action"] as? String

        if (action == "end") {
            return Triple(false, 0, null)
        }

        // Calculate elapsed minutes since focus start
        val startTime = parseTimestamp(latestFocus.timestamp)
        val now = java.util.Date()
        val elapsedMinutes = if (startTime != null) {
            ((now.time - startTime.time) / 60000).toInt()
        } else {
            0
        }

        val source = payload["source"] as? String ?: "manual"

        return Triple(true, elapsedMinutes, source)
    }

    /**
     * Extracts current app category from the most recent app_usage_summary event.
     * Uses AppCategoryEngine for unified classification.
     */
    fun extractCurrentAppCategory(recentEvents: List<EventEntity>): String {
        val appEvent = recentEvents
            .filter { it.eventType == "app_usage_summary" }
            .maxByOrNull { it.timestamp } ?: return "unknown"

        val payload = parsePayload(appEvent.payload)
        val byCategory = payload["by_category"] as? Map<*, *> ?: return "unknown"

        // Find the dominant category from by_category map
        return byCategory.entries
            .filter { it.key != "_start" }
            .maxByOrNull { (it.value as? Number)?.toLong() ?: 0L }
            ?.key as? String ?: "unknown"
    }

    /**
     * Extracts minutes since last unlock from device_unlocked events.
     */
    fun extractMinutesSinceUnlock(recentEvents: List<EventEntity>): Int {
        val unlockEvent = recentEvents
            .filter { it.eventType == "device_unlocked" }
            .maxByOrNull { it.timestamp } ?: return 0

        val startTime = parseTimestamp(unlockEvent.timestamp)
        val now = java.util.Date()

        return if (startTime != null) {
            ((now.time - startTime.time) / 60000).toInt()
        } else {
            0
        }
    }

    /**
     * Checks if the current foreground app is a background audio app.
     */
    fun isBackgroundAudioApp(recentEvents: List<EventEntity>): Boolean {
        val appEvent = recentEvents
            .filter { it.eventType == "app_usage_summary" }
            .maxByOrNull { it.timestamp } ?: return false

        val payload = parsePayload(appEvent.payload)
        val topApps = payload["top_apps"] as? List<*>
        val topApp = topApps?.firstOrNull() as? Map<*, *>
        val packageName = topApp?.get("package") as? String ?: return false

        return appCategoryEngine.isBackgroundAudioApp(packageName)
    }

    /**
     * Full context inference — extracts all signals from events and produces a ContextSnapshot.
     * This is the main entry point for the sync worker.
     */
    suspend fun inferFullContext(
        recentEvents: List<EventEntity>,
        batteryLevel: Double = 1.0,
        isCharging: Boolean = false
    ): ContextSnapshot {
        val (place, movement) = extractLocationInfo(recentEvents)
        val calendarInfo = extractCalendarInfo(recentEvents)
        val healthInfo = extractHealthInfo(recentEvents)
        val sleepInfo = extractSleepInfo(recentEvents)
        val screenState = extractScreenState(recentEvents)
        val (focusActive, focusMinutes, focusSource) = extractFocusInfo(recentEvents)
        val currentAppCategory = extractCurrentAppCategory(recentEvents)
        val minutesSinceUnlock = extractMinutesSinceUnlock(recentEvents)
        val isBackgroundAudio = isBackgroundAudioApp(recentEvents)

        return contextInferenceEngine.infer(
            recentEvents = recentEvents,
            screenState = screenState,
            currentAppCategory = currentAppCategory,
            focusActive = focusActive,
            focusMinutes = focusMinutes,
            focusSource = focusSource,
            batteryLevel = batteryLevel,
            isCharging = isCharging,
            currentPlace = place,
            placeConfidence = if (place != "unknown") 0.9 else 0.5,
            movement = movement,
            calendarInfo = calendarInfo,
            healthInfo = healthInfo,
            sleepInfo = sleepInfo,
            minutesSinceUnlock = minutesSinceUnlock,
            isBackgroundAudioApp = isBackgroundAudio
        )
    }

    private fun parsePayload(payload: String): Map<String, Any?> {
        return try {
            Gson().fromJson(payload, Map::class.java) as? Map<String, Any?> ?: emptyMap()
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun parseTimestamp(ts: String): java.util.Date? {
        return try {
            java.text.SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", java.util.Locale.US).apply {
                timeZone = java.util.TimeZone.getTimeZone("UTC")
            }.parse(ts)
        } catch (e: Exception) {
            null
        }
    }
}
