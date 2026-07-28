package com.andrometry.collectors.communication

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.google.gson.Gson
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Communication metadata collector — tracks call/SMS counts using notification metadata.
 * Privacy-preserving: never stores message bodies, phone numbers, or contact names.
 * Uses salted hashes for contact identity if needed for pattern recognition.
 *
 * Produces: communication_summary events
 *
 * Thread-safe: uses Mutex to protect shared state accessed from multiple coroutines.
 */
@Singleton
class CommunicationCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator,
    private val database: AndrometryDatabase
) {
    data class CommunicationSummary(
        var callsIncoming: Int = 0,
        var callsOutgoing: Int = 0,
        var callMinutes: Int = 0,
        var smsReceived: Int = 0,
        var smsSent: Int = 0,
        val contactHashes: MutableSet<String> = mutableSetOf()
    )

    private val mutex = Mutex()
    private var currentHourSummary = CommunicationSummary()
    private var currentHourStart = System.currentTimeMillis()

    /**
     * Records an incoming call from notification metadata.
     */
    suspend fun recordIncomingCall() = mutex.withLock {
        currentHourSummary.callsIncoming++
        emitSummaryIfNeeded()
    }

    /**
     * Records an outgoing call.
     */
    suspend fun recordOutgoingCall() = mutex.withLock {
        currentHourSummary.callsOutgoing++
        emitSummaryIfNeeded()
    }

    /**
     * Records call duration in minutes.
     */
    suspend fun recordCallDuration(minutes: Int) = mutex.withLock {
        currentHourSummary.callMinutes += minutes
    }

    /**
     * Records received SMS from notification metadata.
     */
    suspend fun recordSmsReceived() = mutex.withLock {
        currentHourSummary.smsReceived++
        emitSummaryIfNeeded()
    }

    /**
     * Records sent SMS.
     */
    suspend fun recordSmsSent() = mutex.withLock {
        currentHourSummary.smsSent++
        emitSummaryIfNeeded()
    }

    private suspend fun emitSummaryIfNeeded() {
        val now = System.currentTimeMillis()
        if (now - currentHourStart >= 3600000) { // 1 hour
            emitSummary()
            resetSummary()
        }
    }

    /**
     * Emits the current hour's communication summary.
     */
    private suspend fun emitSummary() {
        if (currentHourSummary.callsIncoming == 0 && currentHourSummary.callsOutgoing == 0 &&
            currentHourSummary.smsReceived == 0 && currentHourSummary.smsSent == 0) {
            return
        }

        val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }

        val now = sdf.format(Date())
        val startTime = sdf.format(Date(currentHourStart))

        val payload = mapOf(
            "start" to startTime,
            "end" to now,
            "calls_incoming" to currentHourSummary.callsIncoming,
            "calls_outgoing" to currentHourSummary.callsOutgoing,
            "call_minutes" to currentHourSummary.callMinutes,
            "sms_received" to currentHourSummary.smsReceived,
            "sms_sent" to currentHourSummary.smsSent,
            "contact_hashes" to currentHourSummary.contactHashes.toList()
        )

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "communication_summary",
            collector = "communication_tracker",
            source = "notification_metadata",
            payload = Gson().toJson(payload)
        )
        database.eventDao().insert(event)
    }

    private fun resetSummary() {
        currentHourSummary = CommunicationSummary()
        currentHourStart = System.currentTimeMillis()
    }
}
