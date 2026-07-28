package com.andrometry.inference.data_quality

import javax.inject.Inject
import javax.inject.Singleton

/**
 * Data quality engine — tracks data completeness, staleness, and collector health.
 * Attached to every context snapshot so agents know when not to trust a conclusion.
 */
@Singleton
class DataQualityEngine @Inject constructor() {

    data class DataQuality(
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

    data class CollectorHealth(
        val name: String,
        val active: Boolean,
        val lastUpdateMs: Long,
        val errorCount: Int = 0,
        val degraded: Boolean = false
    )

    private val collectorHealthMap = mutableMapOf<String, CollectorHealth>()

    /**
     * Records a collector update.
     */
    fun recordCollectorUpdate(name: String, active: Boolean = true, error: Boolean = false) {
        val existing = collectorHealthMap[name]
        collectorHealthMap[name] = CollectorHealth(
            name = name,
            active = active,
            lastUpdateMs = System.currentTimeMillis(),
            errorCount = (existing?.errorCount ?: 0) + if (error) 1 else 0,
            degraded = error || (existing?.errorCount ?: 0) > 3
        )
    }

    /**
     * Calculates current data quality based on collector states.
     */
    fun calculateDataQuality(
        permissionState: Map<String, Boolean>,
        lastSyncMinutesAgo: Int = 0,
        expectedCollectors: List<String> = listOf(
            "usage_stats", "notification_listener", "calendar",
            "location", "health_connect", "screen", "device"
        )
    ): DataQuality {
        val activeCollectors = collectorHealthMap.values.filter { it.active }.map { it.name }
        val degradedCollectors = collectorHealthMap.values.filter { it.degraded }.map { it.name }

        val completeness = if (expectedCollectors.isEmpty()) 1.0
                          else activeCollectors.size.toDouble() / expectedCollectors.size

        val missingCollectors = expectedCollectors.filter { it !in activeCollectors }

        val permissionIssues = permissionState.filter { !it.value }.keys.toList()

        val missingSignals = mutableListOf<String>()
        if (permissionState["usage_stats"] != true) missingSignals.add("app_usage")
        if (permissionState["notification_listener"] != true) missingSignals.add("notifications")
        if (permissionState["calendar"] != true) missingSignals.add("calendar")
        if (permissionState["location"] != true) missingSignals.add("location")
        if (permissionState["health_connect"] != true) missingSignals.add("health")

        val locationFresh = (collectorHealthMap["location"]?.lastUpdateMs ?: 0) >
                            (System.currentTimeMillis() - 900000) // 15 min

        return DataQuality(
            completeness = completeness,
            stalenessSeconds = lastSyncMinutesAgo * 60,
            usageStatsAvailable = permissionState["usage_stats"] == true,
            notificationListenerAvailable = permissionState["notification_listener"] == true,
            calendarAvailable = permissionState["calendar"] == true,
            locationFresh = locationFresh,
            healthConnectAvailable = permissionState["health_connect"] == true,
            lastSyncMinutesAgo = lastSyncMinutesAgo,
            missingSignals = missingSignals,
            missingCollectors = missingCollectors,
            degradedCollectors = degradedCollectors,
            permissionIssues = permissionIssues,
            clockDriftDetected = false
        )
    }

    /**
     * Checks for clock drift between device and server time.
     */
    fun checkClockDrift(serverTimeMs: Long, deviceTimeMs: Long, thresholdMs: Long = 300000): Boolean {
        return kotlin.math.abs(serverTimeMs - deviceTimeMs) > thresholdMs
    }
}
