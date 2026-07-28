package com.andrometry.core.database

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.sequenceDataStore: DataStore<Preferences> by preferencesDataStore(name = "sequence_generator")

/**
 * Generates monotonically increasing sequence numbers per device.
 * Persists the last sequence across reboots via DataStore.
 */
@Singleton
class SequenceGenerator @Inject constructor(
    @ApplicationContext private val context: Context
) {
    companion object {
        private val KEY_LAST_SEQUENCE = longPreferencesKey("last_sequence")
    }

    private val dataStore = context.sequenceDataStore

    /**
     * Returns the next sequence number. Always monotonic.
     */
    suspend fun next(): Long {
        var lastSequence = 0L
        dataStore.data.map { prefs ->
            prefs[KEY_LAST_SEQUENCE] ?: 0L
        }.first().also { lastSequence = it }

        val nextSequence = lastSequence + 1

        dataStore.edit { prefs ->
            prefs[KEY_LAST_SEQUENCE] = nextSequence
        }

        return nextSequence
    }

    /**
     * Returns the current sequence without incrementing.
     */
    suspend fun current(): Long {
        return dataStore.data.map { prefs ->
            prefs[KEY_LAST_SEQUENCE] ?: 0L
        }.first()
    }
}
