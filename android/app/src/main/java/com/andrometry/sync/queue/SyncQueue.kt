package com.andrometry.sync.queue

import android.content.Context
import androidx.work.*
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.TimelineSegmentEntity
import com.andrometry.core.network.BatchRequest
import com.andrometry.core.network.RelayClient
import dagger.hilt.android.qualifiers.ApplicationContext
import java.util.*
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages the sync queue — batches events and segments for upload.
 *
 * Upload triggers:
 * - Every 10-15 minutes
 * - Context changes significantly
 * - Wi-Fi and charging are available
 * - App opens
 * - Before device sleep if pending batch exists
 *
 * Retry policy: exponential backoff (1m, 5m, 15m, 30m, 60m)
 */
@Singleton
class SyncQueue @Inject constructor(
    @ApplicationContext private val context: Context,
    private val database: AndrometryDatabase,
    private val relayClient: RelayClient
) {
    companion object {
        const val WORK_NAME_PERIODIC = "andrometry_sync_periodic"
        const val WORK_NAME_ONE_TIME = "andrometry_sync_onetime"
    }

    /**
     * Schedules periodic sync work (every 15 minutes).
     */
    fun schedulePeriodicSync() {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .setRequiresBatteryNotLow(true)
            .build()

        val workRequest = PeriodicWorkRequestBuilder<SyncWorker>(
            15, TimeUnit.MINUTES
        )
            .setConstraints(constraints)
            .setBackoffCriteria(
                BackoffPolicy.EXPONENTIAL,
                1, TimeUnit.MINUTES
            )
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_NAME_PERIODIC,
            ExistingPeriodicWorkPolicy.KEEP,
            workRequest
        )
    }

    /**
     * Schedules an immediate one-time sync (e.g., on app open).
     */
    fun scheduleImmediateSync() {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val workRequest = OneTimeWorkRequestBuilder<SyncWorker>()
            .setConstraints(constraints)
            .setBackoffCriteria(
                BackoffPolicy.EXPONENTIAL,
                1, TimeUnit.MINUTES
            )
            .build()

        WorkManager.getInstance(context).enqueueUniqueWork(
            WORK_NAME_ONE_TIME,
            ExistingWorkPolicy.REPLACE,
            workRequest
        )
    }

    /**
     * Builds a batch from pending events and segments, sends it, and marks them as uploaded.
     */
    suspend fun syncPending(): SyncResult {
        // Gather pending events
        val pendingEvents = database.eventDao().getPendingEvents()
        val pendingSegments = database.segmentDao().getPendingSegments()

        if (pendingEvents.isEmpty() && pendingSegments.isEmpty()) {
            return SyncResult(0, 0, 0, null)
        }

        // Build batch
        val deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase()
        val eventsAsMaps = pendingEvents.map { event ->
            mapOf(
                "event_id" to event.eventId,
                "device_id" to event.deviceId,
                "schema_version" to event.schemaVersion,
                "event_version" to 1,
                "timestamp" to event.timestamp,
                "timezone" to event.timezone,
                "event_type" to event.eventType,
                "collector" to event.collector,
                "source" to event.source,
                "sequence" to event.sequence,
                "payload" to com.google.gson.Gson().fromJson(event.payload, Map::class.java)
            )
        }

        val segmentsAsMaps = pendingSegments.map { seg ->
            mapOf(
                "segment_id" to seg.segmentId,
                "segment_type" to seg.segmentType,
                "layer" to seg.layer,
                "start_time" to seg.startTime,
                "end_time" to seg.endTime,
                "label" to seg.label,
                "confidence" to seg.confidence,
                "version" to seg.version,
                "valid_from" to seg.validFrom,
                "evidence" to com.google.gson.Gson().fromJson(seg.evidence, Map::class.java),
                "payload" to com.google.gson.Gson().fromJson(seg.payload, Map::class.java)
            )
        }

        val lastSequence = pendingEvents.lastOrNull()?.sequence ?: 0
        val firstSequence = pendingEvents.firstOrNull()?.sequence ?: 0

        val batteryManager = context.getSystemService(Context.BATTERY_SERVICE) as android.os.BatteryManager
        val batteryLevel = batteryManager.getIntProperty(android.os.BatteryManager.BATTERY_PROPERTY_CAPACITY) / 100.0
        val isCharging = batteryManager.isCharging

        val batch = BatchRequest(
            device_id = deviceId,
            batch_id = UUID.randomUUID().toString(),
            client_version = "0.1.0",
            schema_version = 1,
            sequence_start = firstSequence,
            sequence_end = lastSequence,
            events = eventsAsMaps,
            segments = segmentsAsMaps,
            summaries = emptyList(),
            context = null,
            device_state = mapOf(
                "battery" to batteryLevel,
                "charging" to isCharging,
                "queue_size" to pendingEvents.size
            )
        )

        return try {
            val response = relayClient.postBatch(batch)

            // Mark uploaded
            database.eventDao().markUploaded(pendingEvents.map { it.eventId })
            database.segmentDao().markUploaded(pendingSegments.map { it.segmentId })

            SyncResult(
                acceptedEvents = response.accepted_events,
                acceptedSegments = response.accepted_segments,
                pendingInterventions = response.pending_interventions,
                error = null
            )
        } catch (e: Exception) {
            SyncResult(
                acceptedEvents = 0,
                acceptedSegments = 0,
                pendingInterventions = 0,
                error = e.message
            )
        }
    }

    data class SyncResult(
        val acceptedEvents: Int,
        val acceptedSegments: Int,
        val pendingInterventions: Int,
        val error: String?
    )
}

