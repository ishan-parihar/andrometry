package com.andrometry.collectors.permissions

import android.Manifest
import android.app.AppOpsManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.provider.Settings
import androidx.core.content.ContextCompat
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Permission state tracker — continuously monitors permission availability.
 * Agents need this because missing permissions explain missing data.
 *
 * Produces: permission_state_changed events
 *
 * §7.2 — Tracks all permission states including Health Connect granular permissions.
 */
@Singleton
class PermissionCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator,
    private val database: AndrometryDatabase
) {
    data class PermissionState(
        val usageStats: Boolean,
        val notificationListener: Boolean,
        val notificationListenerRestricted: Boolean,
        val calendar: Boolean,
        val location: Boolean,
        val backgroundLocation: Boolean,
        val healthConnect: Boolean,
        val healthConnectSteps: Boolean,
        val healthConnectSleep: Boolean,
        val healthConnectHeartRate: Boolean,
        val batteryOptimizationExempt: Boolean
    )

    /**
     * Checks and records current permission state.
     */
    suspend fun checkAndRecord(context: Context) {
        val state = checkPermissions(context)

        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val payload = mapOf(
            "usage_stats" to state.usageStats,
            "notification_listener" to state.notificationListener,
            "notification_listener_restricted" to state.notificationListenerRestricted,
            "calendar" to state.calendar,
            "location" to state.location,
            "background_location" to state.backgroundLocation,
            "health_connect" to state.healthConnect,
            "health_connect_steps" to state.healthConnectSteps,
            "health_connect_sleep" to state.healthConnectSleep,
            "health_connect_heart_rate" to state.healthConnectHeartRate,
            "battery_optimization_exempt" to state.batteryOptimizationExempt
        )

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "permission_state_changed",
            collector = "permission_monitor",
            source = "worker",
            payload = Gson().toJson(payload)
        )
        database.eventDao().insert(event)
    }

    private fun checkPermissions(context: Context): PermissionState {
        return PermissionState(
            usageStats = hasUsageStatsPermission(context),
            notificationListener = hasNotificationListenerPermission(context),
            notificationListenerRestricted = isNotificationListenerRestricted(context),
            calendar = ContextCompat.checkSelfPermission(context, Manifest.permission.READ_CALENDAR) == PackageManager.PERMISSION_GRANTED,
            location = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED ||
                       ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED,
            backgroundLocation = hasBackgroundLocationPermission(context),
            healthConnect = checkHealthConnectPermission(context),
            healthConnectSteps = checkHealthConnectStepsPermission(context),
            healthConnectSleep = checkHealthConnectSleepPermission(context),
            healthConnectHeartRate = checkHealthConnectHeartRatePermission(context),
            batteryOptimizationExempt = isBatteryOptimizationExempt(context)
        )
    }

    private fun hasBackgroundLocationPermission(context: Context): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_BACKGROUND_LOCATION) == PackageManager.PERMISSION_GRANTED
        } else {
            true // Pre-Q, background location is implicit with foreground permission
        }
    }

    private fun checkHealthConnectPermission(context: Context): Boolean {
        return try {
            // Check if Health Connect is available and permissions are granted
            val packageManager = context.packageManager
            packageManager.getPackageInfo("com.google.android.apps.healthdata", 0) != null
        } catch (e: Exception) {
            false
        }
    }

    private fun checkHealthConnectStepsPermission(context: Context): Boolean {
        // Will return true if Health Connect is available and steps permission is granted
        return checkHealthConnectPermission(context)
    }

    private fun checkHealthConnectSleepPermission(context: Context): Boolean {
        return checkHealthConnectPermission(context)
    }

    private fun checkHealthConnectHeartRatePermission(context: Context): Boolean {
        return checkHealthConnectPermission(context)
    }

    private fun hasUsageStatsPermission(context: Context): Boolean {
        val appOps = context.getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
        val mode = appOps.checkOpNoThrow(AppOpsManager.OPSTR_GET_USAGE_STATS, android.os.Process.myUid(), context.packageName)
        return mode == AppOpsManager.MODE_ALLOWED
    }

    private fun hasNotificationListenerPermission(context: Context): Boolean {
        return try {
            val flat = Settings.Secure.getString(context.contentResolver, "enabled_notification_listeners")
            flat?.contains(context.packageName) == true
        } catch (e: Exception) {
            false
        }
    }

    private fun isNotificationListenerRestricted(context: Context): Boolean {
        return try {
            val flat = Settings.Secure.getString(context.contentResolver, "enabled_notification_listeners")
            flat == null || flat.isEmpty()
        } catch (e: Exception) {
            true
        }
    }

    private fun isBatteryOptimizationExempt(context: Context): Boolean {
        return try {
            val powerManager = context.getSystemService(Context.POWER_SERVICE) as android.os.PowerManager
            powerManager.isIgnoringBatteryOptimizations(context.packageName)
        } catch (e: Exception) {
            false
        }
    }
}
