package com.andrometry.collectors.health

import android.content.Context
import androidx.health.connect.client.HealthConnectClient
import androidx.health.connect.client.permission.HealthPermission
import androidx.health.connect.client.records.*
import androidx.health.connect.client.request.ReadRecordsRequest
import androidx.health.connect.client.time.TimeRangeFilter
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.time.Duration
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.ZoneId
import java.util.*
import kotlin.math.abs
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Health Connect collector for steps, sleep, heart rate, and exercise sessions.
 * Runs periodically (Tier 1) via WorkManager.
 *
 * Default cadence: 30-60 minutes
 * Produces: health_summary, sleep_summary events
 */
@Singleton
class HealthCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator
) {
    data class HealthSummary(
        val steps: Int,
        val activeMinutes: Int,
        val heartRateAvg: Int?,
        val heartRateMin: Int?,
        val heartRateMax: Int?,
        val exerciseMinutes: Int
    )

    data class SleepSummary(
        val date: String,
        val sleepMinutes: Int,
        val bedtime: String?,
        val wakeTime: String?,
        val regularityScore: Double,
        val inBedMinutes: Int
    )

    private val requiredPermissions = setOf(
        HealthPermission.getReadPermission(StepsRecord::class),
        HealthPermission.getReadPermission(SleepSessionRecord::class),
        HealthPermission.getReadPermission(HeartRateRecord::class),
        HealthPermission.getReadPermission(ExerciseSessionRecord::class),
        HealthPermission.getReadPermission(ActiveCaloriesBurnedRecord::class)
    )

    /**
     * Collects health data for the given time window.
     */
    suspend fun collect(context: Context, startMs: Long, endMs: Long): List<EventEntity> {
        val events = mutableListOf<EventEntity>()
        val healthClient = HealthConnectClient.getOrCreate(context)

        // Check availability
        if (healthClient.permissionController != null) {
            // Health Connect is available
        } else {
            return events // Health Connect not available on this device
        }

        val start = Instant.ofEpochMilli(startMs)
        val end = Instant.ofEpochMilli(endMs)
        val timeRange = TimeRangeFilter.between(start, end)

        // Collect steps
        val stepsEvent = collectSteps(healthClient, timeRange, startMs, endMs)
        if (stepsEvent != null) events.add(stepsEvent)

        // Collect heart rate
        val hrEvent = collectHeartRate(healthClient, timeRange, startMs, endMs)
        if (hrEvent != null) events.add(hrEvent)

        // Collect exercise
        val exerciseEvent = collectExercise(healthClient, timeRange, startMs, endMs)
        if (exerciseEvent != null) events.add(exerciseEvent)

        return events
    }

    /**
     * Collects sleep data for the previous night.
     */
    suspend fun collectSleep(context: Context, date: LocalDate): EventEntity? {
        val healthClient = HealthConnectClient.getOrCreate(context)

        val startOfDay = date.atStartOfDay(ZoneId.systemDefault()).toInstant()
        val endOfDay = date.plusDays(1).atStartOfDay(ZoneId.systemDefault()).toInstant()

        val timeRange = TimeRangeFilter.between(startOfDay, endOfDay)

        return try {
            val response = healthClient.readRecords(
                ReadRecordsRequest(SleepSessionRecord::class, timeRange)
            )

            if (response.records.isEmpty()) return null

            // Find the main sleep session (longest)
            val mainSleep = response.records.maxByOrNull {
                Duration.between(it.startTime, it.endTime).toMinutes()
            } ?: return null

            val sleepMinutes = Duration.between(mainSleep.startTime, mainSleep.endTime).toMinutes().toInt()

            // Calculate regularity (simplified — compare to ideal bedtime)
            val bedtime = LocalDateTime.ofInstant(mainSleep.startTime, ZoneId.systemDefault())
            val wakeTime = LocalDateTime.ofInstant(mainSleep.endTime, ZoneId.systemDefault())
            val regularityScore = calculateSleepRegularity(bedtime, wakeTime)

            val payload = mapOf(
                "date" to date.toString(),
                "sleep_minutes" to sleepMinutes,
                "bedtime" to mainSleep.startTime.toString(),
                "wake_time" to mainSleep.endTime.toString(),
                "regularity_score" to regularityScore,
                "in_bed_minutes" to sleepMinutes + 20 // Approximate in-bed time
            )

            EventEntity(
                eventId = UUID.randomUUID().toString(),
                deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
                sequence = sequenceGenerator.next(),
                timestamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }.format(Date()),
                timezone = TimeZone.getDefault().id,
                eventType = "sleep_summary",
                collector = "health_connect",
                source = "workmanager",
                payload = Gson().toJson(payload)
            )
        } catch (e: Exception) {
            null
        }
    }

    private suspend fun collectSteps(
        healthClient: HealthConnectClient,
        timeRange: TimeRangeFilter,
        startMs: Long,
        endMs: Long
    ): EventEntity? {
        return try {
            val response = healthClient.readRecords(
                ReadRecordsRequest(StepsRecord::class, timeRange)
            )

            val totalSteps = response.records.sumOf { it.count }.toInt()
            if (totalSteps == 0) return null

            val payload = mapOf(
                "steps" to totalSteps,
                "active_minutes" to 0, // Will be calculated from exercise
                "heart_rate_avg" to null,
                "heart_rate_min" to null,
                "heart_rate_max" to null
            )

            EventEntity(
                eventId = UUID.randomUUID().toString(),
                deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
                sequence = sequenceGenerator.next(),
                timestamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }.format(Date()),
                timezone = TimeZone.getDefault().id,
                eventType = "health_summary",
                collector = "health_connect",
                source = "workmanager",
                payload = Gson().toJson(payload)
            )
        } catch (e: Exception) {
            null
        }
    }

    private suspend fun collectHeartRate(
        healthClient: HealthConnectClient,
        timeRange: TimeRangeFilter,
        startMs: Long,
        endMs: Long
    ): EventEntity? {
        return try {
            val response = healthClient.readRecords(
                ReadRecordsRequest(HeartRateRecord::class, timeRange)
            )

            if (response.records.isEmpty()) return null

            val allSamples = response.records.flatMap { it.samples }
            if (allSamples.isEmpty()) return null

            val avg = allSamples.map { it.beatsPerMinute }.average().toInt()
            val min = allSamples.minOf { it.beatsPerMinute }
            val max = allSamples.maxOf { it.beatsPerMinute }

            val payload = mapOf(
                "heart_rate_avg" to avg,
                "heart_rate_min" to min,
                "heart_rate_max" to max,
                "sample_count" to allSamples.size
            )

            EventEntity(
                eventId = UUID.randomUUID().toString(),
                deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
                sequence = sequenceGenerator.next(),
                timestamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }.format(Date()),
                timezone = TimeZone.getDefault().id,
                eventType = "health_summary",
                collector = "health_connect",
                source = "workmanager",
                payload = Gson().toJson(payload)
            )
        } catch (e: Exception) {
            null
        }
    }

    private suspend fun collectExercise(
        healthClient: HealthConnectClient,
        timeRange: TimeRangeFilter,
        startMs: Long,
        endMs: Long
    ): EventEntity? {
        return try {
            val response = healthClient.readRecords(
                ReadRecordsRequest(ExerciseSessionRecord::class, timeRange)
            )

            if (response.records.isEmpty()) return null

            val totalMinutes = response.records.sumOf {
                Duration.between(it.startTime, it.endTime).toMinutes()
            }.toInt()

            if (totalMinutes == 0) return null

            val payload = mapOf(
                "exercise_minutes" to totalMinutes,
                "session_count" to response.records.size,
                "activities" to response.records.map { it.title ?: "exercise" }
            )

            EventEntity(
                eventId = UUID.randomUUID().toString(),
                deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
                sequence = sequenceGenerator.next(),
                timestamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }.format(Date()),
                timezone = TimeZone.getDefault().id,
                eventType = "exercise_session",
                collector = "health_connect",
                source = "workmanager",
                payload = Gson().toJson(payload)
            )
        } catch (e: Exception) {
            null
        }
    }

    /**
     * Calculates sleep regularity score (0-1).
     * Compares actual bedtime/wake time to ideal (23:00 bedtime, 07:00 wake).
     */
    private fun calculateSleepRegularity(bedtime: LocalDateTime, wakeTime: LocalDateTime): Double {
        val idealBedtime = LocalTime.of(23, 0)
        val idealWakeTime = LocalTime.of(7, 0)

        val bedtimeDiff = abs(Duration.between(bedtime.toLocalTime(), idealBedtime).toMinutes())
        val wakeTimeDiff = abs(Duration.between(wakeTime.toLocalTime(), idealWakeTime).toMinutes())

        // Perfect = 0 minutes deviation, terrible = 120+ minutes
        val score = 1.0 - ((bedtimeDiff + wakeTimeDiff) / 240.0).coerceIn(0.0, 1.0)
        return score
    }

    /**
     * Checks which health permissions are available.
     */
    suspend fun checkPermissions(context: Context): Map<String, Boolean> {
        return try {
            val healthClient = HealthConnectClient.getOrCreate(context)
            val granted = healthClient.permissionController.getGrantedPermissions()
            mapOf(
                "steps" to (HealthPermission.getReadPermission(StepsRecord::class) in granted),
                "sleep" to (HealthPermission.getReadPermission(SleepSessionRecord::class) in granted),
                "heart_rate" to (HealthPermission.getReadPermission(HeartRateRecord::class) in granted),
                "exercise" to (HealthPermission.getReadPermission(ExerciseSessionRecord::class) in granted)
            )
        } catch (e: Exception) {
            mapOf(
                "steps" to false,
                "sleep" to false,
                "heart_rate" to false,
                "exercise" to false
            )
        }
    }
}
