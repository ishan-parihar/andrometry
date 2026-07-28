package com.andrometry.collectors.focus

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.database.TimelineSegmentEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/**
 * Focus session collector — tracks manual, calendar-inferred, and agent-suggested focus sessions.
 * Thread-safe using Mutex for concurrent coroutine access.
 */
@Singleton
class FocusCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator,
    private val database: AndrometryDatabase
) {
    data class FocusSession(
        val sessionId: String,
        val source: String,
        val startTime: Long,
        var endTime: Long? = null,
        var interruptions: Int = 0,
        var distractionMinutes: Int = 0
    )

    private val mutex = Mutex()
    private var activeSession: FocusSession? = null

    suspend fun startSession(source: String = "manual", goal: String = "deep work") = mutex.withLock {
        val session = FocusSession(
            sessionId = UUID.randomUUID().toString(),
            source = source,
            startTime = System.currentTimeMillis()
        )
        activeSession = session

        val now = formatTimestamp(Date())
        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "focus_session",
            collector = "focus_manager",
            source = source,
            payload = Gson().toJson(mapOf(
                "action" to "start",
                "source" to source,
                "goal" to goal,
                "session_id" to session.sessionId
            ))
        )
        database.eventDao().insert(event)
    }

    suspend fun endSession(reason: String = "manual_end") = mutex.withLock {
        val session = activeSession ?: return@withLock
        session.endTime = System.currentTimeMillis()

        val durationMinutes = ((session.endTime!! - session.startTime) / 60000).toInt()
        val successScore = calculateSuccessScore(session, durationMinutes)

        val now = formatTimestamp(Date())
        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "focus_session",
            collector = "focus_manager",
            source = session.source,
            payload = Gson().toJson(mapOf(
                "action" to "end",
                "reason" to reason,
                "session_id" to session.sessionId,
                "duration_minutes" to durationMinutes,
                "interruptions" to session.interruptions,
                "distraction_minutes" to session.distractionMinutes,
                "success_score" to successScore
            ))
        )
        database.eventDao().insert(event)

        val segment = TimelineSegmentEntity(
            segmentId = session.sessionId,
            segmentType = "focus_segment",
            layer = "focus",
            startTime = formatTimestamp(Date(session.startTime)),
            endTime = formatTimestamp(Date(session.endTime!!)),
            label = if (successScore > 0.7) "deep_work" else "shallow_work",
            confidence = successScore,
            version = 1,
            validFrom = formatTimestamp(Date(session.startTime)),
            ingestedAt = now,
            evidence = Gson().toJson(mapOf(
                "source" to session.source,
                "interruptions" to session.interruptions,
                "distraction_minutes" to session.distractionMinutes
            )),
            payload = Gson().toJson(mapOf(
                "success_score" to successScore,
                "duration_minutes" to durationMinutes
            ))
        )
        database.segmentDao().upsert(segment)

        activeSession = null
    }

    suspend fun recordInterruption(distractionMinutes: Int = 1) = mutex.withLock {
        val session = activeSession ?: return@withLock
        session.interruptions++
        session.distractionMinutes += distractionMinutes
    }

    suspend fun isSessionActive(): Boolean = mutex.withLock {
        activeSession != null
    }

    suspend fun getActiveSessionId(): String? = mutex.withLock {
        activeSession?.sessionId
    }

    private fun calculateSuccessScore(session: FocusSession, durationMinutes: Int): Double {
        val interruptionPenalty = (session.interruptions * 0.05).coerceAtMost(0.3)
        val distractionPenalty = (session.distractionMinutes.toDouble() / durationMinutes.coerceAtLeast(1) * 0.3).coerceAtMost(0.3)
        val durationBonus = if (durationMinutes >= 25) 0.1 else 0.0
        return (0.8 - interruptionPenalty - distractionPenalty + durationBonus).coerceIn(0.0, 1.0)
    }

    private fun formatTimestamp(date: Date): String {
        return SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(date)
    }
}
