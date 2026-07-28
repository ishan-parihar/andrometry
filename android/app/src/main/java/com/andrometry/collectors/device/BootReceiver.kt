package com.andrometry.collectors.device

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.sync.queue.SyncQueue
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject

/**
 * Receives BOOT_COMPLETED broadcast and:
 * 1. Records a boot event
 * 2. Reschedules periodic sync
 * 3. Restarts collectors
 */
@AndroidEntryPoint
class BootReceiver : BroadcastReceiver() {

    @Inject lateinit var sequenceGenerator: SequenceGenerator
    @Inject lateinit var database: com.andrometry.core.database.AndrometryDatabase
    @Inject lateinit var syncQueue: SyncQueue

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) return

        val pendingResult = goAsync()
        scope.launch {
            try {
                // Record boot event
                val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }.format(Date())

                val event = EventEntity(
                    eventId = UUID.randomUUID().toString(),
                    deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
                    sequence = sequenceGenerator.next(),
                    timestamp = now,
                    timezone = TimeZone.getDefault().id,
                    eventType = "boot_completed",
                    collector = "boot_broadcast",
                    source = "broadcast",
                    payload = com.google.gson.Gson().toJson(mapOf("reason" to "system_boot"))
                )

                database.eventDao().insert(event)

                // Reschedule periodic sync
                syncQueue.schedulePeriodicSync()

                // Schedule immediate sync to catch up
                syncQueue.scheduleImmediateSync()
            } finally {
                pendingResult.finish()
            }
        }
    }
}
