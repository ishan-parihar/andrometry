package com.andrometry.core.config

import android.content.Context
import android.os.BatteryManager
import android.os.PowerManager
import java.time.LocalTime
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Battery profile manager for adaptive collection fidelity.
 *
 * Profiles:
 * - Eco: Long-term passive tracking, very low battery impact
 * - Balanced: Recommended default, strong agent usefulness
 * - High Fidelity: Temporary, high precision, higher battery cost
 *
 * Adaptive switching rules:
 * - If charging and work hours → Balanced or High Fidelity
 * - If battery < 20% → Eco
 * - If focus session starts → temporarily increase fidelity
 * - If device idle overnight → reduce polling
 */
@Singleton
class BatteryProfileManager @Inject constructor() {

    enum class Profile {
        ECO,
        BALANCED,
        HIGH_FIDELITY
    }

    data class ProfileConfig(
        val name: String,
        val appUsageIntervalMinutes: Int,
        val contextSnapshotIntervalMinutes: Int,
        val locationIntervalMinutes: Int,
        val healthSyncIntervalMinutes: Int,
        val syncIntervalMinutes: Int,
        val rawEventRetentionDays: Int,
        val useForegroundService: Boolean
    ) {
        companion object {
            val ECO = ProfileConfig(
                name = "eco",
                appUsageIntervalMinutes = 30,
                contextSnapshotIntervalMinutes = 15,
                locationIntervalMinutes = 60, // geofence only
                healthSyncIntervalMinutes = 60,
                syncIntervalMinutes = 30,
                rawEventRetentionDays = 7,
                useForegroundService = false
            )

            val BALANCED = ProfileConfig(
                name = "balanced",
                appUsageIntervalMinutes = 15,
                contextSnapshotIntervalMinutes = 5,
                locationIntervalMinutes = 15,
                healthSyncIntervalMinutes = 30,
                syncIntervalMinutes = 10,
                rawEventRetentionDays = 14,
                useForegroundService = false
            )

            val HIGH_FIDELITY = ProfileConfig(
                name = "high_fidelity",
                appUsageIntervalMinutes = 2,
                contextSnapshotIntervalMinutes = 1,
                locationIntervalMinutes = 5,
                healthSyncIntervalMinutes = 15,
                syncIntervalMinutes = 5,
                rawEventRetentionDays = 30,
                useForegroundService = true
            )
        }
    }

    private var currentProfile: Profile = Profile.BALANCED
    private var isWorkHours: Boolean = false
    private var isFocusActive: Boolean = false

    /**
     * Determines the appropriate profile based on current conditions.
     */
    fun determineProfile(context: Context): Profile {
        val batteryLevel = getBatteryLevel(context)
        val isCharging = isCharging(context)
        val hour = Calendar.getInstance().get(Calendar.HOUR_OF_DAY)

        isWorkHours = hour in 8..18
        val isNight = hour in 22..23 || hour in 0..6

        // Priority rules
        return when {
            // Battery critical → always Eco
            batteryLevel < 0.15 -> Profile.ECO

            // Night time → Eco
            isNight && !isFocusActive -> Profile.ECO

            // Charging during work hours → High Fidelity
            isCharging && isWorkHours -> Profile.HIGH_FIDELITY

            // Focus session active → High Fidelity (temporarily)
            isFocusActive -> Profile.HIGH_FIDELITY

            // Work hours → Balanced
            isWorkHours -> Profile.BALANCED

            // Default → Eco for off-hours
            else -> Profile.ECO
        }
    }

    /**
     * Returns the config for the current profile.
     */
    fun getConfig(): ProfileConfig {
        return when (currentProfile) {
            Profile.ECO -> ProfileConfig.ECO
            Profile.BALANCED -> ProfileConfig.BALANCED
            Profile.HIGH_FIDELITY -> ProfileConfig.HIGH_FIDELITY
        }
    }

    /**
     * Updates the current profile if conditions changed.
     */
    fun updateProfile(context: Context): Profile {
        val newProfile = determineProfile(context)
        if (newProfile != currentProfile) {
            val oldProfile = currentProfile
            currentProfile = newProfile
            // Log profile change — would be sent as an event
        }
        return currentProfile
    }

    /**
     * Temporarily boosts fidelity for focus sessions.
     */
    fun onFocusSessionStart() {
        isFocusActive = true
    }

    /**
     * Returns to normal profile after focus session ends.
     */
    fun onFocusSessionEnd() {
        isFocusActive = false
    }

    private fun getBatteryLevel(context: Context): Double {
        val batteryManager = context.getSystemService(Context.BATTERY_SERVICE) as BatteryManager
        return batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY) / 100.0
    }

    private fun isCharging(context: Context): Boolean {
        val batteryManager = context.getSystemService(Context.BATTERY_SERVICE) as BatteryManager
        return batteryManager.isCharging
    }

    private fun isPowerSaveMode(context: Context): Boolean {
        val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
        return powerManager.isPowerSaveMode
    }
}
