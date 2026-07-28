package com.andrometry.summaries.hourly

import com.andrometry.core.database.AndrometryDatabase
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Hourly summary engine — aggregates five-minute summaries into hourly windows.
 * Runs hourly via WorkManager.
 *
 * Produces hourly summaries with:
 * - Total app usage by category
 * - Focus minutes
 * - Distraction minutes
 * - Notification load
 * - Screen on minutes
 * - Event counts
 */
@Singleton
class HourlySummaryEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class HourlySummary(
        val startTime: String,
        val endTime: String,
        val appUsageByCategory: Map<String, Int>,
        val focusMinutes: Int,
        val distractionMinutes: Int,
        val notificationCount: Int,
        val screenOnMinutes: Int,
        val totalEventCount: Int,
        val fiveMinuteWindows: Int
    )

    /**
     * Generates an hourly summary from five-minute summaries.
     */
    suspend fun generateSummary(hourStart: Date): HourlySummary {
        val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
        val startStr = fmt.format(hourStart)
        val hourEnd = Date(hourStart.time + 3600000)
        val endStr = fmt.format(hourEnd)

        // Get five-minute summaries for this hour
        val fiveMinSummaries = database.fiveMinuteSummaryDao().getInRange(startStr, endStr)

        val appUsageByCategory = mutableMapOf<String, Int>()
        var focusMinutes = 0
        var distractionMinutes = 0
        var notificationCount = 0
        var screenOnMinutes = 0
        var totalEventCount = 0

        for (summary in fiveMinSummaries) {
            try {
                val data = Gson().fromJson(summary.summary, Map::class.java) as? Map<String, Any> ?: continue

                // Aggregate app usage
                @Suppress("UNCHECKED_CAST")
                val appUsage = data["appUsage"] as? Map<String, Any> ?: emptyMap()
                for ((category, secs) in appUsage) {
                    if (secs is Number) {
                        appUsageByCategory[category] = (appUsageByCategory[category] ?: 0) + secs.toInt()
                    }
                }

                // Aggregate other metrics
                if (data["focusActive"] == true) focusMinutes += 5
                notificationCount += (data["notificationCount"] as? Number)?.toInt() ?: 0
                screenOnMinutes += (data["screenOnMinutes"] as? Number)?.toInt() ?: 0

            } catch (e: Exception) {
                // Skip malformed summaries
            }
        }

        // Calculate distraction from social/entertainment app usage
        val socialTime = (appUsageByCategory["social"] ?: 0) + (appUsageByCategory["entertainment"] ?: 0)
        distractionMinutes = socialTime / 60 // convert seconds to minutes

        return HourlySummary(
            startTime = startStr,
            endTime = endStr,
            appUsageByCategory = appUsageByCategory,
            focusMinutes = focusMinutes,
            distractionMinutes = distractionMinutes,
            notificationCount = notificationCount,
            screenOnMinutes = screenOnMinutes.coerceAtMost(60),
            totalEventCount = totalEventCount,
            fiveMinuteWindows = fiveMinSummaries.size
        )
    }

    /**
     * Stores an hourly summary in the database.
     */
    suspend fun storeSummary(summary: HourlySummary) {
        val entity = com.andrometry.core.database.HourlySummaryEntity(
            startTime = summary.startTime,
            endTime = summary.endTime,
            summary = Gson().toJson(summary),
            syncState = "pending"
        )
        database.hourlySummaryDao().upsert(entity)
    }
}
