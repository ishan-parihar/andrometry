package com.andrometry.collectors.screen

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
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
 * Collects screen on/off and unlock events via broadcast receivers.
 * These are event-driven (Tier 0) — extremely low battery impact.
 *
 * Uses @AndroidEntryPoint for Hilt injection in BroadcastReceiver.
 */
@AndroidEntryPoint
class ScreenCollector : BroadcastReceiver() {

    @Inject lateinit var sequenceGenerator: SequenceGenerator
    @Inject lateinit var database: AndrometryDatabase

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onReceive(context: Context, intent: Intent) {
        val pendingResult = goAsync()
        scope.launch {
            try {
                when (intent.action) {
                    Intent.ACTION_SCREEN_ON -> {
                        emitEvent(context, "screen_state_changed", mapOf("state" to "on"))
                    }
                    Intent.ACTION_SCREEN_OFF -> {
                        emitEvent(context, "screen_state_changed", mapOf("state" to "off"))
                    }
                    Intent.ACTION_USER_PRESENT -> {
                        emitEvent(context, "device_unlocked", mapOf("session_start" to true))
                    }
                }
            } finally {
                pendingResult.finish()
            }
        }
    }

    private suspend fun emitEvent(context: Context, eventType: String, payload: Map<String, Any?>) {
        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val tz = TimeZone.getDefault()
        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = getDeviceId(context),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = tz.id,
            eventType = eventType,
            collector = "screen_broadcast",
            source = "broadcast",
            payload = com.google.gson.Gson().toJson(payload)
        )

        database.eventDao().insert(event)
    }

    private fun getDeviceId(context: Context): String {
        return android.os.Build.MODEL.replace(" ", "-").lowercase()
    }

    companion object {
        fun createIntentFilter(): IntentFilter {
            return IntentFilter().apply {
                addAction(Intent.ACTION_SCREEN_ON)
                addAction(Intent.ACTION_SCREEN_OFF)
                addAction(Intent.ACTION_USER_PRESENT)
            }
        }
    }
}
