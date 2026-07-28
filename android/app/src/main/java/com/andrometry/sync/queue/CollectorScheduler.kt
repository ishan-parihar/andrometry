package com.andrometry.sync.queue

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.*
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.collectors.usage.UsageCollector
import com.andrometry.collectors.health.HealthCollector
import com.andrometry.collectors.permissions.PermissionCollector

import com.andrometry.inference.segments.SegmentEngine
import com.andrometry.inference.context.ContextInferenceEngine
import com.andrometry.summaries.five_minute.FiveMinuteSummaryEngine
import com.andrometry.summaries.hourly.HourlySummaryEngine
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.text.SimpleDateFormat
import java.util.*
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Schedules all collector workers per architecture §8 tiered collection strategy.
 *
 * Tier 1 (Periodic WorkManager):
 * - App usage summary: 15 min
 * - Calendar sync: 15 min
 * - Health Connect sync: 30 min
 * - Context snapshot: 15 min (via inference)
 * - Timeline segmentation: 15 min
 * - Five-minute summary: 15 min (min WorkManager interval)
 * - Hourly summary: 60 min
 *
 * Note: WorkManager minimum periodic interval is 15 minutes.
 * For 1-5 min granularity, use a foreground service (Tier 2 / active mode).
 */
@Singleton
class CollectorScheduler @Inject constructor() {

    companion object {
        const val WORK_USAGE = "andrometry_collector_usage"
        const val WORK_HEALTH = "andrometry_collector_health"
        const val WORK_FIVE_MIN = "andrometry_collector_five_minute"
        const val WORK_HOURLY = "andrometry_collector_hourly"
        const val WORK_SEGMENT = "andrometry_collector_segment"
        const val WORK_CONTEXT = "andrometry_collector_context"        const val WORK_PERMISSIONS = "andrometry_collector_permissions"
    }

    fun scheduleAll(context: Context) {
        scheduleUsageCollector(context)
        scheduleHealthCollector(context)
        scheduleFiveMinuteSummary(context)
        scheduleHourlySummary(context)
        scheduleSegmentation(context)
        scheduleContextSnapshot(context)        schedulePermissionCheck(context)
    }

    /**
     * App usage: every 15 min (Tier 1 balanced)
     */
    private fun scheduleUsageCollector(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .setRequiresBatteryNotLow(true)
            .build()

        val work = PeriodicWorkRequestBuilder<UsageCollectorWorker>(
            15, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 1, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_USAGE, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Health Connect: every 30 min (Tier 1)
     */
    private fun scheduleHealthCollector(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .setRequiresBatteryNotLow(true)
            .build()

        val work = PeriodicWorkRequestBuilder<HealthCollectorWorker>(
            30, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 2, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_HEALTH, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Five-minute summary: every 15 min (WorkManager minimum)
     */
    private fun scheduleFiveMinuteSummary(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<FiveMinuteSummaryWorker>(
            15, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_FIVE_MIN, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Hourly summary: every 60 min
     */
    private fun scheduleHourlySummary(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<HourlySummaryWorker>(
            60, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_HOURLY, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Timeline segmentation: every 15 min
     */
    private fun scheduleSegmentation(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<SegmentationWorker>(
            15, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_SEGMENT, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Context snapshot: every 15 min
     */
    private fun scheduleContextSnapshot(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<ContextSnapshotWorker>(
            15, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_CONTEXT, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Outcome processing: every 60 min (§35)
     */
    private fun scheduleOutcomeProcessing(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<OutcomeProcessingWorker>(
            60, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_OUTCOMES, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }

    /**
     * Permission state check: every 15 min (§7.2)
     */
    private fun schedulePermissionCheck(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<PermissionCheckWorker>(
            15, TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_PERMISSIONS, ExistingPeriodicWorkPolicy.KEEP, work
        )
    }



// ============================================================
// HiltWorker implementations for each collector
// ============================================================

@HiltWorker
class UsageCollectorWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val usageCollector: UsageCollector,
    private val database: AndrometryDatabase,
    private val sequenceGenerator: SequenceGenerator
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = System.currentTimeMillis()
            val fifteenMinAgo = now - 15 * 60 * 1000
            val event = usageCollector.collect(applicationContext, fifteenMinAgo, now)
            if (event != null) {
                database.eventDao().insert(event)
            }
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker
class HealthCollectorWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val healthCollector: HealthCollector,
    private val database: AndrometryDatabase
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = System.currentTimeMillis()
            val thirtyMinAgo = now - 30 * 60 * 1000
            val events = healthCollector.collect(applicationContext, thirtyMinAgo, now)
            for (event in events) {
                database.eventDao().insert(event)
            }
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker
class FiveMinuteSummaryWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val fiveMinuteSummaryEngine: FiveMinuteSummaryEngine
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = Date()
            val fifteenMinAgo = Date(now.time - 15 * 60 * 1000)
            val summary = fiveMinuteSummaryEngine.generateSummary(fifteenMinAgo, now)
            fiveMinuteSummaryEngine.storeSummary(summary)
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker
class HourlySummaryWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val hourlySummaryEngine: HourlySummaryEngine
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = Date()
            val hourAgo = Date(now.time - 3600000)
            val summary = hourlySummaryEngine.generateSummary(hourAgo)
            hourlySummaryEngine.storeSummary(summary)
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker
class SegmentationWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val segmentEngine: SegmentEngine,
    private val database: AndrometryDatabase
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = Date()
            val fifteenMinAgo = Date(now.time - 15 * 60 * 1000)
            val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }
            val recentEvents = database.eventDao().getEventsInRange(
                fmt.format(fifteenMinAgo), fmt.format(now), 100
            )
            val segments = segmentEngine.generateSegments(recentEvents)
            for (seg in segments) {
                database.segmentDao().upsert(seg)
            }
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker
class ContextSnapshotWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val contextInferenceEngine: ContextInferenceEngine,
    private val database: AndrometryDatabase
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            val now = Date()
            val fifteenMinAgo = Date(now.time - 15 * 60 * 1000)
            val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }
            val recentEvents = database.eventDao().getEventsInRange(
                fmt.format(fifteenMinAgo), fmt.format(now), 100
            )
            val snapshot = contextInferenceEngine.infer(recentEvents)
            val entity = contextInferenceEngine.toCurrentContextEntity(snapshot)
            database.contextDao().upsert(entity)
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

// ============================================================
// Inference engine workers (§35 Outcome, §38 Habit)
// ============================================================



// ============================================================
// Collector workers (§7 Permission, §9.7 Communication, §9.8 Focus)
// ============================================================

@HiltWorker
class PermissionCheckWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val permissionCollector: PermissionCollector
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            permissionCollector.checkAndRecord(applicationContext)
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

@HiltWorker



