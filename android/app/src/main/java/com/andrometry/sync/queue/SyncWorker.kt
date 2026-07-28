package com.andrometry.sync.queue

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.*
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.network.BatchRequest
import com.andrometry.core.network.RelayClient
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.util.*
import java.util.concurrent.TimeUnit

/**
 * HiltWorker for periodic sync — properly integrates with Hilt DI.
 */
@HiltWorker
class SyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val database: AndrometryDatabase,
    private val relayClient: RelayClient
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val pendingEvents = database.eventDao().getPendingEvents()
            val pendingSegments = database.segmentDao().getPendingSegments()

            if (pendingEvents.isEmpty() && pendingSegments.isEmpty()) {
                return Result.success()
            }

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

            val batteryManager = applicationContext.getSystemService(Context.BATTERY_SERVICE) as android.os.BatteryManager
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

            val response = relayClient.postBatch(batch)

            database.eventDao().markUploaded(pendingEvents.map { it.eventId })
            database.segmentDao().markUploaded(pendingSegments.map { it.segmentId })

            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}
