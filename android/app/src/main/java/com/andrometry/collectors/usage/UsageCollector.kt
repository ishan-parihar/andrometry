package com.andrometry.collectors.usage

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import com.andrometry.core.database.AppClassificationEntity
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject

/**
 * Collects app usage summaries from UsageStatsManager.
 * Runs periodically (Tier 1) via WorkManager.
 *
 * Default cadence: 15 minutes (balanced), 1-5 minutes (active mode).
 */
class UsageCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator
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

        // Classify by category
        val byCategory = mutableMapOf<String, Long>()
        val topApps = mutableListOf<AppUsage>()

        for ((pkg, durationMs) in usageMap) {
            if (pkg.endsWith("_start")) continue
            val seconds = durationMs / 1000
            val category = classifyPackage(pkg)
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
        val payload = mapOf(
            "by_category" to byCategory,
            "top_apps" to topApps.take(10).map {
                mapOf("package" to it.packageName, "category" to it.category, "seconds" to it.seconds)
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
     * Uses Android's app category + known package mappings.
     */
    private fun classifyPackage(packageName: String): String {
        // Known package classifications
        return when {
            // Productivity
            packageName.contains("notion") || packageName.contains("todoist") ||
            packageName.contains("asana") || packageName.contains("trello") ||
            packageName.contains("jira") || packageName.contains("confluence") ||
            packageName.contains("google.docs") || packageName.contains("google.sheets") ||
            packageName.contains("google.slides") || packageName.contains("ms.office") ||
            packageName.contains("obsidian") -> "productivity"

            // Communication
            packageName.contains("slack") || packageName.contains("teams") ||
            packageName.contains("discord") || packageName.contains("telegram") ||
            packageName.contains("whatsapp") || packageName.contains("messenger") ||
            packageName.contains("signal") || packageName.contains("zoom") ||
            packageName.contains("meet") || packageName.contains("skype") ||
            packageName.contains("gmail") || packageName.contains("outlook") -> "communication"

            // Browser
            packageName.contains("chrome") || packageName.contains("firefox") ||
            packageName.contains("samsung.internet") || packageName.contains("brave") ->
                "browser"

            // Social
            packageName.contains("twitter") || packageName.contains("instagram") ||
            packageName.contains("facebook") || packageName.contains("reddit") ||
            packageName.contains("tiktok") || packageName.contains("linkedin") ||
            packageName.contains("threads") -> "social"

            // Entertainment
            packageName.contains("youtube") || packageName.contains("netflix") ||
            packageName.contains("spotify") || packageName.contains("prime") ||
            packageName.contains("disney") || packageName.contains("twitch") ->
                "entertainment"

            // News
            packageName.contains("news") || packageName.contains("bbc") ||
            packageName.contains("cnn") || packageName.contains("reuters") ||
            packageName.contains("flipboard") -> "news"

            // Finance
            packageName.contains("bank") || packageName.contains("finance") ||
            packageName.contains("wallet") || packageName.contains("pay") ||
            packageName.contains("crypto") -> "finance"

            // Health
            packageName.contains("health") || packageName.contains("fit") ||
            packageName.contains("strava") || packageName.contains("fitness") ->
                "health"

            // Maps/Travel
            packageName.contains("maps") || packageName.contains("uber") ||
            packageName.contains("lyft") || packageName.contains("travel") ||
            packageName.contains("booking") -> "travel"

            // System
            packageName.contains("android") || packageName.contains("google.android") ||
            packageName.contains("samsung.android") -> "system"

            else -> "unknown"
        }
    }
}
