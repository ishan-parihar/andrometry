package com.andrometry.core.config

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.sensitiveModeDataStore: DataStore<Preferences> by preferencesDataStore(name = "sensitive_mode")

/**
 * Manages sensitive mode state and redaction settings.
 *
 * When sensitive mode is active:
 * - Sync is paused (no data leaves the device)
 * - Location data is redacted
 * - App usage is redacted (categories only, no package names)
 * - Calendar titles are redacted
 *
 * Persists across reboots via DataStore.
 *
 * Architecture refs: §7.7, §33 Privacy Governor
 */
@Singleton
class SensitiveModeManager @Inject constructor(
    @ApplicationContext private val context: Context
) {
    companion object {
        val KEY_SENSITIVE_MODE = booleanPreferencesKey("sensitive_mode_active")
        val KEY_REDACT_LOCATION = booleanPreferencesKey("redact_location")
        val KEY_REDACT_APP_USAGE = booleanPreferencesKey("redact_app_usage")
        val KEY_REDACT_CALENDAR = booleanPreferencesKey("redact_calendar")
    }

    private val dataStore = context.sensitiveModeDataStore

    /** Observe sensitive mode state as a Flow. */
    val isSensitiveModeActive: Flow<Boolean> = dataStore.data.map { prefs ->
        prefs[KEY_SENSITIVE_MODE] ?: false
    }

    /** Observe redaction settings. */
    val redactLocation: Flow<Boolean> = dataStore.data.map { prefs ->
        prefs[KEY_REDACT_LOCATION] ?: false
    }

    val redactAppUsage: Flow<Boolean> = dataStore.data.map { prefs ->
        prefs[KEY_REDACT_APP_USAGE] ?: false
    }

    val redactCalendar: Flow<Boolean> = dataStore.data.map { prefs ->
        prefs[KEY_REDACT_CALENDAR] ?: true // Calendar titles redacted by default
    }

    /** Check if sync should be paused. */
    suspend fun shouldPauseSync(): Boolean {
        return dataStore.data.first()[KEY_SENSITIVE_MODE] ?: false
    }

    /** Check if location should be redacted. */
    suspend fun shouldRedactLocation(): Boolean {
        val sensitive = dataStore.data.first()[KEY_SENSITIVE_MODE] ?: false
        val setting = dataStore.data.first()[KEY_REDACT_LOCATION] ?: false
        return sensitive || setting
    }

    /** Check if app usage should be redacted (show categories only). */
    suspend fun shouldRedactAppUsage(): Boolean {
        val sensitive = dataStore.data.first()[KEY_SENSITIVE_MODE] ?: false
        val setting = dataStore.data.first()[KEY_REDACT_APP_USAGE] ?: false
        return sensitive || setting
    }

    /** Check if calendar titles should be redacted. */
    suspend fun shouldRedactCalendar(): Boolean {
        val sensitive = dataStore.data.first()[KEY_SENSITIVE_MODE] ?: false
        val setting = dataStore.data.first()[KEY_REDACT_CALENDAR] ?: true
        return sensitive || setting
    }

    /** Enable sensitive mode. */
    suspend fun enableSensitiveMode() {
        dataStore.edit { prefs ->
            prefs[KEY_SENSITIVE_MODE] = true
            prefs[KEY_REDACT_LOCATION] = true
            prefs[KEY_REDACT_APP_USAGE] = true
            prefs[KEY_REDACT_CALENDAR] = true
        }
    }

    /** Disable sensitive mode. */
    suspend fun disableSensitiveMode() {
        dataStore.edit { prefs ->
            prefs[KEY_SENSITIVE_MODE] = false
            prefs[KEY_REDACT_LOCATION] = false
            prefs[KEY_REDACT_APP_USAGE] = false
            prefs[KEY_REDACT_CALENDAR] = true // Keep calendar redacted by default
        }
    }

    /** Update individual redaction settings. */
    suspend fun setRedactLocation(enabled: Boolean) {
        dataStore.edit { prefs -> prefs[KEY_REDACT_LOCATION] = enabled }
    }

    suspend fun setRedactAppUsage(enabled: Boolean) {
        dataStore.edit { prefs -> prefs[KEY_REDACT_APP_USAGE] = enabled }
    }

    suspend fun setRedactCalendar(enabled: Boolean) {
        dataStore.edit { prefs -> prefs[KEY_REDACT_CALENDAR] = enabled }
    }
}
