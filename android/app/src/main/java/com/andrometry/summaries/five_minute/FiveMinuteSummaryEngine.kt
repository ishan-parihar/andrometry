package com.andrometry.summaries.five_minute

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.FiveMinuteSummaryEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Five-minute summary engine — aggregates raw events into 5-minute windows.
 * Runs every 5 minutes via WorkManager.
 *
 * Produces summary objects with:
 * - App usage by category
 * - Notification count and interruption level
 * - Screen state
 * - Focus session status
 * - Event counts by type
 */
@Singleton
class FiveMinuteSummaryEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class FiveMinuteSummary(
        val startTime: String,
        val endTime: String,
        val appUsage: Map<String, Int>,
        val notificationCount: Int,
        val interruptionLevel: String,
        val screenOnMinutes: Int,
        val focusActive: Boolean,
        val eventCounts: Map<String, Int>,
        val topApps: List<TopApp>
    )

    data class TopApp(
        val packageName: String,
        val category: String,
        val seconds: Int
    )

    /**
     * Generates a 5-minute summary from recent events.
     */
    suspend fun generateSummary(windowStart: Date, windowEnd: Date): FiveMinuteSummary {
        val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
        val startStr = fmt.format(windowStart)
        val endStr = fmt.format(windowEnd)

        val events = database.eventDao().getEventsInRange(startStr, endStr)

        val appUsage = mutableMapOf<String, Int>()
        var notificationCount = 0
        var screenOnMinutes = 0
        val eventCounts = mutableMapOf<String, Int>()
        val appSeconds = mutableMapOf<String, Int>()

        for (event in events) {
            eventCounts[event.eventType] = (eventCounts[event.eventType] ?: 0) + 1

            when (event.eventType) {
                "app_usage_summary" -> {
                    val payload = parsePayload(event.payload)
                    @Suppress("UNCHECKED_CAST")
                    val byCategory = payload["by_category"] as? Map<String, Any> ?: emptyMap()
                    for ((cat, secs) in byCategory) {
                        if (secs is Number) {
                            appUsage[cat] = (appUsage[cat] ?: 0) + secs.toInt()
                        }
                    }
                    @Suppress("UNCHECKED_CAST")
                    val topApps = payload["top_apps"] as? List<Map<String, Any>> ?: emptyList()
                    for (app in topApps) {
                        val pkg = app["package"] as? String ?: continue
                        val secs = (app["seconds"] as? Number)?.toInt() ?: 0
                        appSeconds[pkg] = (appSeconds[pkg] ?: 0) + secs
                    }
                }
                "notification_event" -> {
                    notificationCount++
                }
                "screen_state_changed" -> {
                    val payload = parsePayload(event.payload)
                    if (payload["state"] == "on") {
                        screenOnMinutes += 5 // approximate per window
                    }
                }
                "focus_session" -> {
                    val payload = parsePayload(event.payload)
                    if (payload["action"] == "start") {
                        // Focus active during this window
                    }
                }
            }
        }

        val interruptionLevel = when {
            notificationCount > 10 -> "high"
            notificationCount > 5 -> "medium"
            notificationCount > 0 -> "low"
            else -> "none"
        }

        val topAppsList = appSeconds.entries
            .sortedByDescending { it.value }
            .take(5)
            .map { TopApp(it.key, categorizePackage(it.key), it.value) }

        return FiveMinuteSummary(
            startTime = startStr,
            endTime = endStr,
            appUsage = appUsage,
            notificationCount = notificationCount,
            interruptionLevel = interruptionLevel,
            screenOnMinutes = screenOnMinutes.coerceAtMost(5),
            focusActive = eventCounts.containsKey("focus_session"),
            eventCounts = eventCounts,
            topApps = topAppsList
        )
    }

    /**
     * Stores a 5-minute summary in the database.
     */
    suspend fun storeSummary(summary: FiveMinuteSummary) {
        val entity = FiveMinuteSummaryEntity(
            startTime = summary.startTime,
            endTime = summary.endTime,
            summary = Gson().toJson(summary),
            syncState = "pending"
        )
        database.fiveMinuteSummaryDao().upsert(entity)
    }

    private fun parsePayload(payload: String): Map<String, Any> {
        return try {
            Gson().fromJson(payload, Map::class.java) as? Map<String, Any> ?: emptyMap()
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun categorizePackage(packageName: String): String {
        return when {
            packageName.contains("slack") || packageName.contains("teams") ||
            packageName.contains("discord") -> "communication"
            packageName.contains("notion") || packageName.contains("todoist") ||
            packageName.contains("asana") -> "productivity"
            packageName.contains("chrome") || packageName.contains("firefox") ||
            packageName.contains("samsung.internet") -> "browser"
            packageName.contains("twitter") || packageName.contains("instagram") ||
            packageName.contains("facebook") || packageName.contains("reddit") -> "social"
            packageName.contains("youtube") || packageName.contains("netflix") ||
            packageName.contains("spotify") -> "entertainment"
            else -> "other"
        }
    }
}
