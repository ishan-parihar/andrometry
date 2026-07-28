package com.andrometry.collectors.calendar

import android.content.Context
import android.database.ContentObserver
import android.net.Uri
import android.provider.CalendarContract
import android.os.Handler
import android.os.Looper
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
 * Calendar collector using ContentObserver for real-time meeting detection.
 * Stores only busy/free status and next event time — never meeting titles by default.
 *
 * Tier 0 (observer) + Tier 1 (periodic reconciliation).
 * Produces: calendar_context events
 */
@AndroidEntryPoint
class CalendarCollector : ContentObserver(Handler(Looper.getMainLooper())) {

    @Inject lateinit var sequenceGenerator: SequenceGenerator
    @Inject lateinit var database: AndrometryDatabase

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var context: Context? = null

    fun start(context: Context) {
        this.context = context
        val uri: Uri = CalendarContract.Events.CONTENT_URI
        context.contentResolver.registerContentObserver(uri, true, this)
    }

    fun stop() {
        context?.contentResolver?.unregisterContentObserver(this)
    }

    override fun onChange(selfChange: Boolean) {
        super.onChange(selfChange)
        scope.launch {
            syncCalendarContext()
        }
    }

    private suspend fun syncCalendarContext() {
        val ctx = context ?: return
        val now = System.currentTimeMillis()
        val endOfDay = Calendar.getInstance().apply {
            set(Calendar.HOUR_OF_DAY, 23)
            set(Calendar.MINUTE, 59)
            set(Calendar.SECOND, 59)
        }.timeInMillis

        // Query events in next 24 hours
        val projection = arrayOf(
            CalendarContract.Events.DTSTART,
            CalendarContract.Events.DTEND,
            CalendarContract.Events.TITLE,
            CalendarContract.Events.ALL_DAY
        )

        val selection = "${CalendarContract.Events.DTSTART} >= ? AND ${CalendarContract.Events.DTSTART} <= ?"
        val selectionArgs = arrayOf(now.toString(), endOfDay.toString())

        var currentEvent = "none"
        var nextEventMinutes = -1
        var busyUntil: String? = null

        try {
            ctx.contentResolver.query(
                CalendarContract.Events.CONTENT_URI,
                projection,
                selection,
                selectionArgs,
                "${CalendarContract.Events.DTSTART} ASC"
            )?.use { cursor ->
                var foundNext = false
                while (cursor.moveToNext()) {
                    val start = cursor.getLong(0)
                    val end = cursor.getLong(1)
                    val allDay = cursor.getInt(3) == 1

                    if (allDay) continue

                    // Check if currently in event
                    if (start <= now && end > now) {
                        currentEvent = "meeting"
                        busyUntil = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                            timeZone = TimeZone.getTimeZone("UTC")
                        }.format(Date(end))
                    }

                    // Find next upcoming event
                    if (!foundNext && start > now) {
                        foundNext = true
                        nextEventMinutes = ((start - now) / 60000).toInt()
                    }
                }
            }
        } catch (e: SecurityException) {
            // Calendar permission not granted
        }

        val nowFormatted = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val payload = mapOf(
            "current_event" to currentEvent,
            "next_event_minutes" to nextEventMinutes,
            "busy_until" to busyUntil
        )

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = nowFormatted,
            timezone = TimeZone.getDefault().id,
            eventType = "calendar_context",
            collector = "calendar_observer",
            source = "content_observer",
            payload = com.google.gson.Gson().toJson(payload)
        )

        database.eventDao().insert(event)
    }
}
