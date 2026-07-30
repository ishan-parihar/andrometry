package com.andrometry.collectors.usage

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import com.andrometry.core.config.SensitiveModeManager
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.inference.classifier.AppCategoryEngine
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject

/**
 * Collects app usage summaries from UsageStatsManager.
 * Runs periodically (Tier 1) via WorkManager.
 *
 * Respects sensitive mode: when active, top_apps are redacted to categories only.
 * Default cadence: 15 minutes (balanced), 1-5 minutes (active mode).
 *
 * Classification is delegated to AppCategoryEngine for unified categorization.
 */
class UsageCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator,
    private val sensitiveModeManager: SensitiveModeManager,
    private val appCategoryEngine: AppCategoryEngine
) {
    data class AppUsage(
        val packageName: String,
        val category: String,
        val seconds: Long
    )

    data class UsageSummary(
        val byCategory: Map<String, Long>,
        val topApps: List<AppUsage>,
        val appSwitchCount: Int
    )

    /**
     * Collects app usage for the given time window.
     */
    suspend fun collect(context: Context, startTimeMs: Long, endTimeMs: Long): EventEntity? {
        val usageStatsManager = context.getSystemService(Context.USAGE_STATS_SERVICE) as? UsageStatsManager
            ?: return null

        val events = usageStatsManager.queryEvents(startTimeMs, endTimeMs)
        if (events == null || !events.hasNextEvent()) return null

        val usageMap = mutableMapOf<String, Long>() // package -> foreground ms
        var appSwitchCount = 0
        var lastPackage = ""

        val event = UsageEvents.Event()
        while (events.hasNextEvent()) {
            events.getNextEvent(event)

            if (event.eventType == UsageEvents.Event.MOVE_TO_FOREGROUND) {
                val pkg = event.packageName
                if (pkg != lastPackage) {
                    appSwitchCount++
                    lastPackage = pkg
                }
                // Track foreground start time
                usageMap[pkg + "_start"] = event.timeStamp
            } else if (event.eventType == UsageEvents.Event.MOVE_TO_BACKGROUND) {
                val pkg = event.packageName
                val startKey = pkg + "_start"
                val start = usageMap.remove(startKey) ?: continue
                val duration = event.timeStamp - start
                usageMap[pkg] = (usageMap[pkg] ?: 0L) + duration
            }
        }

        // Classify by category using unified AppCategoryEngine
        val byCategory = mutableMapOf<String, Long>()
        val topApps = mutableListOf<AppUsage>()

        for ((pkg, durationMs) in usageMap) {
            if (pkg.endsWith("_start")) continue
            val seconds = durationMs / 1000
            val category = classifyPackage(context, pkg)
            byCategory[category] = (byCategory[category] ?: 0L) + seconds
            topApps.add(AppUsage(pkg, category, seconds))
        }

        topApps.sortByDescending { it.seconds }

        val summary = UsageSummary(
            byCategory = byCategory,
            topApps = topApps.take(10),
            appSwitchCount = appSwitchCount
        )

        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val tz = TimeZone.getDefault()
        val redactAppUsage = sensitiveModeManager.shouldRedactAppUsage()
        val payload = mapOf(
            "by_category" to byCategory,
            "top_apps" to topApps.take(10).map { app ->
                mapOf(
                    "package" to if (redactAppUsage) "[redacted]" else app.packageName,
                    "category" to app.category,
                    "seconds" to app.seconds
                )
            },
            "app_switch_count" to appSwitchCount
        )

        return EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = tz.id,
            eventType = "app_usage_summary",
            collector = "usage_stats",
            source = "workmanager",
            payload = com.google.gson.Gson().toJson(payload)
        )
    }

    /**
     * Classifies a package into a category.
     * Delegates to AppCategoryEngine for unified classification.
     * Uses exact package matching + Android category + user overrides.
     */
    private suspend fun classifyPackage(context: Context, packageName: String): String {
        val classification = appCategoryEngine.classify(context, packageName)
        return classification.category
    }
}
