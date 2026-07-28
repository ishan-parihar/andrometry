package com.andrometry.inference.data_quality

import javax.inject.Inject
import javax.inject.Singleton

/**
 * Data quality engine — tracks data completeness, staleness, and collector health.
 * Attached to every context snapshot so agents know when not to trust a conclusion.
 *
 * Implements clock drift detection (§23.3) and app health telemetry (§23.5).
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

    data class AppHealthTelemetry(
        val queueDepth: Int,
        val collectorHealth: Map<String, CollectorHealth>,
        val syncFailureCount: Int,
        val lastSyncSuccessMs: Long,
        val dbSizeBytes: Long,
        val uptimeMs: Long
    )

    private val collectorHealthMap = mutableMapOf<String, CollectorHealth>()
    private var syncFailureCount = 0
    private var lastSyncSuccessMs = 0L
    private var lastServerTimeMs = 0L
    private val driftHistory = mutableListOf<Long>()

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
     * Records a sync result for telemetry.
     */
    fun recordSyncResult(success: Boolean) {
        if (success) {
            lastSyncSuccessMs = System.currentTimeMillis()
            syncFailureCount = 0
        } else {
            syncFailureCount++
        }
    }

    /**
     * Checks for clock drift between device and server time (§23.3).
     * Uses a rolling window of 5 samples to detect persistent drift.
     */
    fun checkClockDrift(serverTimeMs: Long, deviceTimeMs: Long, thresholdMs: Long = 300000): Boolean {
        val drift = deviceTimeMs - serverTimeMs
        driftHistory.add(drift)
        if (driftHistory.size > 5) driftHistory.removeAt(0)

        lastServerTimeMs = serverTimeMs

        // Persistent drift: if average drift exceeds threshold for multiple samples
        if (driftHistory.size >= 3) {
            val avgDrift = driftHistory.average()
            return kotlin.math.abs(avgDrift) > thresholdMs
        }

        return kotlin.math.abs(drift) > thresholdMs
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

        // Check clock drift using last known server time
        val clockDriftDetected = if (lastServerTimeMs > 0) {
            checkClockDrift(lastServerTimeMs, System.currentTimeMillis())
        } else {
            false
        }

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
            clockDriftDetected = clockDriftDetected
        )
    }

    /**
     * Generates app health telemetry (§23.5).
     */
    fun getHealthTelemetry(queueDepth: Int = 0, dbSizeBytes: Long = 0): AppHealthTelemetry {
        return AppHealthTelemetry(
            queueDepth = queueDepth,
            collectorHealth = collectorHealthMap.toMap(),
            syncFailureCount = syncFailureCount,
            lastSyncSuccessMs = lastSyncSuccessMs,
            dbSizeBytes = dbSizeBytes,
            uptimeMs = System.currentTimeMillis()
        )
    }
}
