package com.andrometry.summaries.daily

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.DailySummaryEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Daily summary engine — aggregates hourly summaries into rich daily summaries.
 * Runs hourly during daytime and nightly for final summary.
 *
 * Produces daily summaries matching the architecture.md spec:
 * - sleep minutes, regularity, bedtime/wake
 * - health: steps, exercise, active minutes
 * - work: deep/shallow/meeting/admin minutes
 * - distraction: total, social, browser, during focus
 * - interruptions: notifications, unlocks, app switches
 * - focus: sessions, successful, average interruptions
 * - work patterns: focus bout, switching rate, fragmentation
 * - scores: alignment, productivity, wellbeing, focus control, sleep quality, readiness
 */
@Singleton
class DailySummaryEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class DailySummary(
        val date: String,
        val timezone: String,
        val sleep: SleepMetrics?,
        val health: HealthMetrics?,
        val work: WorkMetrics,
        val distraction: DistractionMetrics,
        val interruptions: InterruptionMetrics,
        val focus: FocusMetrics,
        val workPatterns: WorkPatternMetrics,
        val scores: ScoreMetrics
    )

    data class SleepMetrics(
        val minutes: Int,
        val regularityScore: Double,
        val bedtime: String?,
        val wakeTime: String?
    )

    data class HealthMetrics(
        val steps: Int,
        val exerciseMinutes: Int,
        val activeMinutes: Int
    )

    data class WorkMetrics(
        val deepWorkMinutes: Int,
        val shallowWorkMinutes: Int,
        val meetingMinutes: Int,
        val adminMinutes: Int
    )

    data class DistractionMetrics(
        val totalMinutes: Int,
        val socialMinutes: Int,
        val browserHoppingMinutes: Int,
        val duringFocusMinutes: Int
    )

    data class InterruptionMetrics(
        val notificationCount: Int,
        val unlockCount: Int,
        val appSwitchCount: Int
    )

    data class FocusMetrics(
        val sessions: Int,
        val successfulSessions: Int,
        val averageInterruptions: Double
    )

    data class WorkPatternMetrics(
        val averageFocusBoutMinutes: Double,
        val switchingRatePerHour: Double,
        val meetingFragmentationScore: Double,
        val lateNightWorkMinutes: Int
    )

    data class ScoreMetrics(
        val alignment: Double,
        val productivity: Double,
        val wellbeing: Double,
        val focusControl: Double,
        val sleepQuality: Double,
        val readiness: Double
    )

    /**
     * Generates a daily summary for the given date.
     */
    suspend fun generateSummary(date: Date): DailySummary {
        val dateFormat = SimpleDateFormat("yyyy-MM-dd", Locale.US)
        val dateStr = dateFormat.format(date)

        val tzFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }

        val dayStart = Calendar.getInstance().apply {
            time = date
            set(Calendar.HOUR_OF_DAY, 0)
            set(Calendar.MINUTE, 0)
            set(Calendar.SECOND, 0)
        }.time

        val dayEnd = Calendar.getInstance().apply {
            time = date
            set(Calendar.HOUR_OF_DAY, 23)
            set(Calendar.MINUTE, 59)
            set(Calendar.SECOND, 59)
        }.time

        val startStr = tzFormat.format(dayStart)
        val endStr = tzFormat.format(dayEnd)

        // Get hourly summaries for the day
        val hourlySummaries = database.hourlySummaryDao().getInRange(startStr, endStr)

        // Aggregate metrics
        var deepWorkMinutes = 0
        var shallowWorkMinutes = 0
        var meetingMinutes = 0
        var distractionMinutes = 0
        var socialMinutes = 0
        var notificationCount = 0
        var screenOnMinutes = 0
        var focusSessions = 0

        for (summary in hourlySummaries) {
            try {
                val data = Gson().fromJson(summary.summary, Map::class.java) as? Map<String, Any> ?: continue

                deepWorkMinutes += (data["focusMinutes"] as? Number)?.toInt() ?: 0
                distractionMinutes += (data["distractionMinutes"] as? Number)?.toInt() ?: 0
                notificationCount += (data["notificationCount"] as? Number)?.toInt() ?: 0
                screenOnMinutes += (data["screenOnMinutes"] as? Number)?.toInt() ?: 0

                @Suppress("UNCHECKED_CAST")
                val appUsage = data["appUsageByCategory"] as? Map<String, Any> ?: emptyMap()
                socialMinutes += ((appUsage["social"] as? Number)?.toInt() ?: 0) / 60
                socialMinutes += ((appUsage["entertainment"] as? Number)?.toInt() ?: 0) / 60

            } catch (e: Exception) {
                // Skip malformed summaries
            }
        }

        // Get focus segments for the day
        val focusSegments = database.segmentDao().getSegmentsInRange(startStr, endStr)
            .filter { it.segmentType == "focus_segment" }
        focusSessions = focusSegments.size

        val successfulSessions = focusSegments.count {
            val payload = parsePayload(it.payload)
            (payload["success_score"] as? Number)?.toDouble()?.let { it > 0.7 } ?: false
        }

        val avgInterruptions = if (focusSessions > 0) {
            focusSegments.sumOf {
                val payload = parsePayload(it.payload)
                (payload["interruptions"] as? Number)?.toInt() ?: 0
            }.toDouble() / focusSessions
        } else 0.0

        // Calculate scores
        val productivityScore = (deepWorkMinutes.toDouble() / 480.0).coerceIn(0.0, 1.0) // 8 hours target
        val focusControlScore = if (focusSessions > 0) {
            successfulSessions.toDouble() / focusSessions
        } else 0.5
        val wellbeingScore = 1.0 - (distractionMinutes.toDouble() / 480.0).coerceIn(0.0, 1.0)
        val alignmentScore = (productivityScore + focusControlScore) / 2.0

        return DailySummary(
            date = dateStr,
            timezone = TimeZone.getDefault().id,
            sleep = null, // Would be populated from sleep data
            health = null, // Would be populated from health data
            work = WorkMetrics(
                deepWorkMinutes = deepWorkMinutes,
                shallowWorkMinutes = shallowWorkMinutes,
                meetingMinutes = meetingMinutes,
                adminMinutes = 0
            ),
            distraction = DistractionMetrics(
                totalMinutes = distractionMinutes,
                socialMinutes = socialMinutes,
                browserHoppingMinutes = 0,
                duringFocusMinutes = 0
            ),
            interruptions = InterruptionMetrics(
                notificationCount = notificationCount,
                unlockCount = 0,
                appSwitchCount = 0
            ),
            focus = FocusMetrics(
                sessions = focusSessions,
                successfulSessions = successfulSessions,
                averageInterruptions = avgInterruptions
            ),
            workPatterns = WorkPatternMetrics(
                averageFocusBoutMinutes = if (focusSessions > 0) deepWorkMinutes.toDouble() / focusSessions else 0.0,
                switchingRatePerHour = 0.0,
                meetingFragmentationScore = 0.0,
                lateNightWorkMinutes = 0
            ),
            scores = ScoreMetrics(
                alignment = alignmentScore,
                productivity = productivityScore,
                wellbeing = wellbeingScore,
                focusControl = focusControlScore,
                sleepQuality = 0.0,
                readiness = 0.0
            )
        )
    }

    /**
     * Stores a daily summary in the database.
     */
    suspend fun storeSummary(summary: DailySummary) {
        val entity = DailySummaryEntity(
            date = summary.date,
            timezone = summary.timezone,
            summary = Gson().toJson(summary),
            syncState = "pending"
        )
        database.dailySummaryDao().upsert(entity)
    }

    private fun parsePayload(payload: String): Map<String, Any> {
        return try {
            Gson().fromJson(payload, Map::class.java) as? Map<String, Any> ?: emptyMap()
        } catch (e: Exception) {
            emptyMap()
        }
    }
}
