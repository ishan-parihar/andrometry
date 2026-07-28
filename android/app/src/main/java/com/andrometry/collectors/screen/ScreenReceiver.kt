package com.andrometry.collectors.screen

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject

/**
 * Screen state broadcast receiver with proper Hilt injection.
 *
 * Declared in AndroidManifest.xml. Receives:
 * - ACTION_SCREEN_ON
 * - ACTION_SCREEN_OFF
 * - ACTION_USER_PRESENT (unlock)
 *
 * This is Tier 0 collection — event-driven, extremely low battery impact.
 */
@AndroidEntryPoint
class ScreenReceiver : BroadcastReceiver() {

    @Inject lateinit var sequenceGenerator: SequenceGenerator
    @Inject lateinit var database: AndrometryDatabase

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onReceive(context: Context, intent: Intent) {
        val pendingResult = goAsync()
        scope.launch {
            try {
                when (intent.action) {
                    Intent.ACTION_SCREEN_ON -> emitEvent("screen_state_changed", mapOf("state" to "on"))
                    Intent.ACTION_SCREEN_OFF -> emitEvent("screen_state_changed", mapOf("state" to "off"))
                    Intent.ACTION_USER_PRESENT -> emitEvent("device_unlocked", mapOf("session_start" to true))
                }
            } finally {
                pendingResult.finish()
            }
        }
    }

    private suspend fun emitEvent(eventType: String, payload: Map<String, Any?>) {
        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = eventType,
            collector = "screen_broadcast",
            source = "broadcast",
            payload = com.google.gson.Gson().toJson(payload)
        )

        database.eventDao().insert(event)
    }
}
