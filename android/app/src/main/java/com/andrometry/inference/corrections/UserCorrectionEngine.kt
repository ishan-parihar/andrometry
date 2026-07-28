package com.andrometry.inference.corrections

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.database.TimelineSegmentEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * User Correction Engine — implements §37 User Correction Loop.
 *
 * Allows users to correct timeline segment labels when the classification is wrong.
 * Corrections are stored and used to:
 * 1. Immediately fix the segment
 * 2. Track correction patterns
 * 3. Adjust classification weights over time
 *
 * §37.1 — Correction examples:
 * - "This was deep work, not shallow work"
 * - "I was at the cafe, not the office"
 * - "This was a break, not distraction"
 *
 * §37.2 — Correction effects:
 * - Immediate segment relabeling
 * - Version bump on the segment
 * - Correction event logged for learning
 * - Future classification adjustments
 */
@Singleton
class UserCorrectionEngine @Inject constructor(
    private val database: AndrometryDatabase,
    private val sequenceGenerator: SequenceGenerator
) {

    data class Correction(
        val correctionId: String,
        val segmentId: String,
        val oldLabel: String,
        val newLabel: String,
        val oldConfidence: Double,
        val correctionType: String, // "label", "merge", "split", "delete"
        val reason: String,
        val timestamp: String
    )

    data class CorrectionPattern(
        val fromLabel: String,
        val toLabel: String,
        val count: Int,
        val lastOccurrence: String
    )

    /**
     * Applies a user correction to a timeline segment.
     *
     * @param segmentId The segment to correct
     * @param newLabel The corrected label
     * @param reason Optional reason for the correction
     * @return The correction record, or null if segment not found
     */
    suspend fun applyCorrection(
        segmentId: String,
        newLabel: String,
        reason: String = ""
    ): Correction? {
        // Find the segment
        val segment = database.segmentDao().getById(segmentId) ?: return null

        val oldLabel = segment.label
        val oldConfidence = segment.confidence
        val now = getCurrentTimestamp()

        // Create correction record
        val correction = Correction(
            correctionId = UUID.randomUUID().toString(),
            segmentId = segmentId,
            oldLabel = oldLabel,
            newLabel = newLabel,
            oldConfidence = oldConfidence,
            correctionType = "label",
            reason = reason,
            timestamp = now
        )

        // Update the segment with new label and version bump
        val updatedSegment = segment.copy(
            label = newLabel,
            confidence = 1.0, // User correction = high confidence
            version = segment.version + 1,
            revisedAt = now,
            revisionReason = "user_correction"
        )
        database.segmentDao().upsert(updatedSegment)

        // Log the correction event
        val correctionEvent = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "user_correction",
            collector = "correction_engine",
            source = "user_input",
            payload = Gson().toJson(correction)
        )
        database.eventDao().insert(correctionEvent)

        return correction
    }

    /**
     * Gets correction patterns for a user — shows what labels are frequently corrected.
     */
    suspend fun getCorrectionPatterns(userId: String = "primary", limit: Int = 20): List<CorrectionPattern> {
        val events = database.eventDao().getByType("user_correction", limit * 10)

        return events.groupBy { event ->
            val correction = Gson().fromJson(event.payload, Correction::class.java)
            "${correction.oldLabel} -> ${correction.newLabel}"
        }.map { (key, events) ->
            val first = Gson().fromJson(events.first().payload, Correction::class.java)
            CorrectionPattern(
                fromLabel = first.oldLabel,
                toLabel = first.newLabel,
                count = events.size,
                lastOccurrence = events.maxOf { it.timestamp }
            )
        }.sortedByDescending { it.count }.take(limit)
    }

    /**
     * Splits a segment into two at a given timestamp.
     */
    suspend fun splitSegment(
        segmentId: String,
        splitAtTimestamp: String,
        reason: String = ""
    ): Pair<Correction?, Correction?> {
        val segment = database.segmentDao().getById(segmentId) ?: return Pair(null, null)

        // End the original segment at the split point
        val firstCorrection = applyCorrection(segmentId, segment.label, "split_first_half: $reason")
        val firstSegment = segment.copy(
            endTime = splitAtTimestamp,
            version = segment.version + 1,
            revisedAt = getCurrentTimestamp(),
            revisionReason = "user_split"
        )
        database.segmentDao().upsert(firstSegment)

        // Create a second segment from the split point to original end
        val secondSegment = segment.copy(
            segmentId = UUID.randomUUID().toString(),
            startTime = splitAtTimestamp,
            version = 1,
            ingestedAt = getCurrentTimestamp(),
            revisedAt = null,
            revisionReason = null
        )
        database.segmentDao().upsert(secondSegment)

        return Pair(firstCorrection, null)
    }

    /**
     * Deletes a segment (marks as invalid).
     */
    suspend fun deleteSegment(segmentId: String, reason: String = ""): Correction? {
        val segment = database.segmentDao().getById(segmentId) ?: return null

        val now = getCurrentTimestamp()
        val correction = Correction(
            correctionId = UUID.randomUUID().toString(),
            segmentId = segmentId,
            oldLabel = segment.label,
            newLabel = "[deleted]",
            oldConfidence = segment.confidence,
            correctionType = "delete",
            reason = reason,
            timestamp = now
        )

        // Soft delete — set label to [deleted] and confidence to 0
        val deletedSegment = segment.copy(
            label = "[deleted]",
            confidence = 0.0,
            version = segment.version + 1,
            revisedAt = now,
            revisionReason = "user_deleted"
        )
        database.segmentDao().upsert(deletedSegment)

        // Log the deletion event
        val correctionEvent = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "user_correction",
            collector = "correction_engine",
            source = "user_input",
            payload = Gson().toJson(correction)
        )
        database.eventDao().insert(correctionEvent)

        return correction
    }

    private fun getCurrentTimestamp(): String {
        return SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())
    }
}
