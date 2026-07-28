package com.andrometry.inference.baselines

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.BaselineEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Baseline engine — computes rolling averages for key metrics.
 * Handles nested JSON structures from daily summaries.
 */
@Singleton
class BaselineEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class Baseline(
        val metricName: String,
        val value: Double,
        val windowDays: Int,
        val computedAt: String
    )

    suspend fun computeBaselines(userId: String = "primary", windowDays: Int = 7): List<Baseline> {
        val summaries = database.dailySummaryDao().getRecent(windowDays)
        if (summaries.isEmpty()) return emptyList()

        val baselines = mutableListOf<Baseline>()
        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val metricValues = mutableMapOf<String, MutableList<Double>>()

        for (summary in summaries) {
            try {
                val data = Gson().fromJson(summary.summary, Map::class.java) as? Map<String, Any> ?: continue
                flattenAndExtract(data, "", metricValues)
            } catch (e: Exception) {
                // Skip malformed summaries
            }
        }

        for ((metric, values) in metricValues) {
            if (values.isNotEmpty()) {
                val avg = values.average()
                val baseline = BaselineEntity(
                    baselineId = UUID.randomUUID().toString(),
                    metricName = metric,
                    windowDays = windowDays,
                    value = avg,
                    computedAt = now
                )
                database.baselineDao().upsert(baseline)
                baselines.add(Baseline(metric, avg, windowDays, now))
            }
        }

        return baselines
    }

    private fun flattenAndExtract(data: Map<String, Any>, prefix: String, result: MutableMap<String, MutableList<Double>>) {
        for ((key, value) in data) {
            val fullKey = if (prefix.isEmpty()) key else "${prefix}_$key"
            when (value) {
                is Number -> {
                    val metricName = mapToStandardMetric(fullKey)
                    if (metricName != null) {
                        result.getOrPut(metricName) { mutableListOf() }.add(value.toDouble())
                    }
                }
                is Map<*, *> -> {
                    @Suppress("UNCHECKED_CAST")
                    flattenAndExtract(value as Map<String, Any>, fullKey, result)
                }
            }
        }
    }

    private fun mapToStandardMetric(rawKey: String): String? {
        return when {
            rawKey.contains("deep_work_minutes") -> "deep_work_minutes_daily"
            rawKey.contains("distraction_minutes") -> "distraction_minutes_daily"
            rawKey.contains("sleep_minutes") -> "sleep_minutes"
            rawKey.contains("steps") -> "steps_daily"
            rawKey.contains("meeting_minutes") -> "meeting_minutes_daily"
            rawKey.contains("notification_count") -> "notification_count_hourly"
            rawKey.contains("unlock_count") -> "unlock_count_daily"
            rawKey.contains("focus_session") && rawKey.contains("count") -> "focus_session_count_daily"
            else -> null
        }
    }

    suspend fun compareToBaseline(
        metricName: String,
        currentValue: Double,
        userId: String = "primary"
    ): BaselineComparison {
        val baseline = database.baselineDao().getByMetric(userId, metricName)
            ?: return BaselineComparison(
                metricName = metricName,
                currentValue = currentValue,
                baselineValue = null,
                deviation = null,
                deviationPercent = null,
                interpretation = "no_baseline"
            )

        val deviation = currentValue - baseline.value
        val deviationPercent = if (baseline.value != 0.0) (deviation / baseline.value * 100) else 0.0

        val interpretation = when {
            deviationPercent > 20 -> "significantly_above"
            deviationPercent > 5 -> "above"
            deviationPercent > -5 -> "at_baseline"
            deviationPercent > -20 -> "below"
            else -> "significantly_below"
        }

        return BaselineComparison(
            metricName = metricName,
            currentValue = currentValue,
            baselineValue = baseline.value,
            deviation = deviation,
            deviationPercent = deviationPercent,
            interpretation = interpretation
        )
    }

    data class BaselineComparison(
        val metricName: String,
        val currentValue: Double,
        val baselineValue: Double?,
        val deviation: Double?,
        val deviationPercent: Double?,
        val interpretation: String
    )
}
