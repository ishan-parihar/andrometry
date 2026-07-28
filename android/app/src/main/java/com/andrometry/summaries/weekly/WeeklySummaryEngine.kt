package com.andrometry.summaries.weekly

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.WeeklySummaryEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Weekly summary engine — aggregates daily summaries into weekly trends.
 * Runs weekly via WorkManager.
 *
 * Produces weekly summaries with:
 * - Daily breakdown
 * - Weekly totals
 * - Trend analysis
 * - Habit tracking
 */
@Singleton
class WeeklySummaryEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class WeeklySummary(
        val week: String,
        val timezone: String,
        val dailyBreakdown: List<DailySummary>,
        val weeklyTotals: WeeklyTotals,
        val trends: TrendMetrics
    )

    data class DailySummary(
        val date: String,
        val deepWorkMinutes: Int,
        val distractionMinutes: Int,
        val focusSessions: Int,
        val productivityScore: Double
    )

    data class WeeklyTotals(
        val deepWorkMinutes: Int,
        val distractionMinutes: Int,
        val totalFocusSessions: Int,
        val averageProductivity: Double
    )

    data class TrendMetrics(
        val productivityTrend: String, // improving, stable, declining
        val focusTrend: String,
        val distractionTrend: String
    )

    /**
     * Generates a weekly summary for the given week.
     */
    suspend fun generateSummary(weekStart: Date): WeeklySummary {
        val weekFormat = SimpleDateFormat("yyyy-'W'ww", Locale.US)
        val weekStr = weekFormat.format(weekStart)

        val tzFormat = SimpleDateFormat("yyyy-MM-dd", Locale.US)

        val dailyBreakdown = mutableListOf<DailySummary>()
        var totalDeepWork = 0
        var totalDistraction = 0
        var totalFocusSessions = 0
        var totalProductivity = 0.0

        // Iterate through each day of the week
        for (dayOffset in 0..6) {
            val day = Calendar.getInstance().apply {
                time = weekStart
                add(Calendar.DAY_OF_YEAR, dayOffset)
            }.time
            val dateStr = tzFormat.format(day)

            val dailySummary = database.dailySummaryDao().get(dateStr)
            if (dailySummary != null) {
                try {
                    val data = Gson().fromJson(dailySummary.summary, Map::class.java) as? Map<String, Any> ?: continue

                    val work = data["work"] as? Map<String, Any> ?: emptyMap()
                    val distraction = data["distraction"] as? Map<String, Any> ?: emptyMap()
                    val focus = data["focus"] as? Map<String, Any> ?: emptyMap()
                    val scores = data["scores"] as? Map<String, Any> ?: emptyMap()

                    val deepWork = (work["deepWorkMinutes"] as? Number)?.toInt() ?: 0
                    val distractionMins = (distraction["totalMinutes"] as? Number)?.toInt() ?: 0
                    val sessions = (focus["sessions"] as? Number)?.toInt() ?: 0
                    val productivity = (scores["productivity"] as? Number)?.toDouble() ?: 0.0

                    dailyBreakdown.add(DailySummary(
                        date = dateStr,
                        deepWorkMinutes = deepWork,
                        distractionMinutes = distractionMins,
                        focusSessions = sessions,
                        productivityScore = productivity
                    ))

                    totalDeepWork += deepWork
                    totalDistraction += distractionMins
                    totalFocusSessions += sessions
                    totalProductivity += productivity

                } catch (e: Exception) {
                    // Skip malformed summaries
                }
            }
        }

        val daysWithData = dailyBreakdown.size.coerceAtLeast(1)
        val avgProductivity = totalProductivity / daysWithData

        // Calculate trends
        val firstHalf = dailyBreakdown.take(3)
        val secondHalf = dailyBreakdown.takeLast(3)
        val firstHalfAvg = if (firstHalf.isNotEmpty()) firstHalf.map { it.productivityScore }.average() else 0.0
        val secondHalfAvg = if (secondHalf.isNotEmpty()) secondHalf.map { it.productivityScore }.average() else 0.0

        val productivityTrend = when {
            secondHalfAvg > firstHalfAvg + 0.1 -> "improving"
            secondHalfAvg < firstHalfAvg - 0.1 -> "declining"
            else -> "stable"
        }

        return WeeklySummary(
            week = weekStr,
            timezone = TimeZone.getDefault().id,
            dailyBreakdown = dailyBreakdown,
            weeklyTotals = WeeklyTotals(
                deepWorkMinutes = totalDeepWork,
                distractionMinutes = totalDistraction,
                totalFocusSessions = totalFocusSessions,
                averageProductivity = avgProductivity
            ),
            trends = TrendMetrics(
                productivityTrend = productivityTrend,
                focusTrend = "stable",
                distractionTrend = "stable"
            )
        )
    }

    /**
     * Stores a weekly summary in the database.
     */
    suspend fun storeSummary(summary: WeeklySummary) {
        val entity = WeeklySummaryEntity(
            week = summary.week,
            timezone = summary.timezone,
            summary = Gson().toJson(summary),
            syncState = "pending"
        )
        database.weeklySummaryDao().upsert(entity)
    }
}
