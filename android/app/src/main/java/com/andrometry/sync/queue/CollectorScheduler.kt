package com.andrometry.sync.queue

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.*
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.collectors.usage.UsageCollector
import com.andrometry.collectors.health.HealthCollector
import com.andrometry.collectors.permissions.PermissionCollector

import com.andrometry.core.config.BatteryProfileManager
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
 * Now uses BatteryProfileManager (§24) to get adaptive intervals:
 * - Eco: Longer intervals, minimal collection
 * - Balanced: Default intervals (15 min)
 * - High Fidelity: Short intervals (2-5 min)
 *
 * Tier 1 (Periodic WorkManager):
 * - App usage summary: 15 min (balanced) / 2 min (high fidelity) / 30 min (eco)
 * - Health Connect sync: 30 min (balanced) / 15 min (high fidelity) / 60 min (eco)
 * - Five-minute summary: 15 min (WorkManager minimum)
 * - Hourly summary: 60 min
 * - Timeline segmentation: 15 min
 * - Context snapshot: 15 min (balanced) / 5 min (high fidelity) / 30 min (eco)
 * - Permission state check: 15 min
 */
@Singleton
class CollectorScheduler @Inject constructor(
    private val batteryProfileManager: BatteryProfileManager
) {

    companion object {
        const val WORK_USAGE = "andrometry_collector_usage"
        const val WORK_HEALTH = "andrometry_collector_health"
        const val WORK_FIVE_MIN = "andrometry_collector_five_minute"
        const val WORK_HOURLY = "andrometry_collector_hourly"
        const val WORK_SEGMENT = "andrometry_collector_segment"
        const val WORK_CONTEXT = "andrometry_collector_context"
        const val WORK_PERMISSIONS = "andrometry_collector_permissions"
    }

    fun scheduleAll(context: Context) {
        val profile = batteryProfileManager.updateProfile(context)
        val config = batteryProfileManager.getConfig()

        // WorkManager enforces minimum 15-minute interval (§8.3).
        // Sub-15-minute intervals require a foreground service (Tier 2).
        val usageInterval = config.appUsageIntervalMinutes.coerceAtLeast(15)
        val healthInterval = config.healthSyncIntervalMinutes.coerceAtLeast(15)
        val segmentInterval = config.contextSnapshotIntervalMinutes.coerceAtLeast(15)
        val contextInterval = config.contextSnapshotIntervalMinutes.coerceAtLeast(15)

        scheduleUsageCollector(context, usageInterval)
        scheduleHealthCollector(context, healthInterval)
        scheduleFiveMinuteSummary(context)
        scheduleHourlySummary(context)
        scheduleSegmentation(context, segmentInterval)
        scheduleContextSnapshot(context, contextInterval)
        schedulePermissionCheck(context)
    }

    /**
     * App usage: adaptive interval from battery profile
     */
    private fun scheduleUsageCollector(context: Context, intervalMinutes: Int) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .setRequiresBatteryNotLow(true)
            .build()

        val work = PeriodicWorkRequestBuilder<UsageCollectorWorker>(
            intervalMinutes.toLong(), TimeUnit.MINUTES
        ).setConstraints(constraints)
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 1, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_USAGE, ExistingPeriodicWorkPolicy.UPDATE, work
        )
    }

    /**
     * Health Connect: adaptive interval from battery profile
     */
    private fun scheduleHealthCollector(context: Context, intervalMinutes: Int) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .setRequiresBatteryNotLow(true)
            .build()

        val work = PeriodicWorkRequestBuilder<HealthCollectorWorker>(
            intervalMinutes.toLong(), TimeUnit.MINUTES
        ).setConstraints(constraints)
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 2, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_HEALTH, ExistingPeriodicWorkPolicy.UPDATE, work
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
     * Timeline segmentation: adaptive interval from battery profile
     */
    private fun scheduleSegmentation(context: Context, intervalMinutes: Int) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<SegmentationWorker>(
            intervalMinutes.toLong(), TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_SEGMENT, ExistingPeriodicWorkPolicy.UPDATE, work
        )
    }

    /**
     * Context snapshot: adaptive interval from battery profile
     */
    private fun scheduleContextSnapshot(context: Context, intervalMinutes: Int) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.NOT_REQUIRED)
            .build()

        val work = PeriodicWorkRequestBuilder<ContextSnapshotWorker>(
            intervalMinutes.toLong(), TimeUnit.MINUTES
        ).setConstraints(constraints)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_CONTEXT, ExistingPeriodicWorkPolicy.UPDATE, work
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
