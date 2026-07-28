package com.andrometry.inference.segments

import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.database.TimelineSegmentEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Timeline Segmentation Engine — produces semantic segments from raw events.
 *
 * Runs every 5-15 minutes via WorkManager.
 *
 * Segment types:
 * - activity_segment (deep_work, shallow_work, meeting, communication, break, leisure, distraction)
 * - focus_segment (focus_session, interrupted_focus, failed_focus)
 * - place_visit (home, office, gym, cafe, library, commute, unknown)
 * - meeting_segment
 * - distraction_segment
 * - break_segment
 * - commute_segment
 * - exercise_segment
 * - sleep_segment
 * - communication_segment
 * - app_usage_segment
 *
 * Classification rules (from architecture.md §12.3):
 *
 * If:
 *   productivity app usage > 20 minutes
 *   app switches low
 *   no meeting
 *   notifications low
 * Then:
 *   create deep_work segment
 *
 * If:
 *   social app usage > 10 minutes
 *   during intended focus
 *   app switches high
 * Then:
 *   create distraction segment
 */
@Singleton
class SegmentEngine @Inject constructor(
    private val sequenceGenerator: SequenceGenerator
) {
    data class SegmentCandidate(
        val type: String,
        val layer: String,
        val label: String,
        val confidence: Double,
        val startTime: String,
        val endTime: String,
        val payload: Map<String, Any?> = emptyMap()
    )

    /**
     * Generates timeline segments from recent events.
     */
    fun generateSegments(
        recentEvents: List<EventEntity>,
        contextSnapshot: Map<String, Any?>? = null
    ): List<TimelineSegmentEntity> {
        val candidates = mutableListOf<SegmentCandidate>()
        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        // Analyze app usage events
        val appUsageEvents = recentEvents.filter { it.eventType == "app_usage_summary" }
        val focusEvents = recentEvents.filter { it.eventType == "focus_session" }
        val notifEvents = recentEvents.filter { it.eventType == "notification_summary" }
        val locationEvents = recentEvents.filter { it.eventType == "location_context" }

        // Generate activity segments from app usage
        for (event in appUsageEvents) {
            val payload = parsePayload(event.payload)
            val byCategory = payload["by_category"] as? Map<*, *> ?: continue
            val appSwitchCount = (payload["app_switch_count"] as? Number)?.toInt() ?: 0

            // Determine dominant category
            val dominantCategory = byCategory.entries
                .filter { it.key != "_start" }
                .maxByOrNull { (it.value as? Number)?.toLong() ?: 0 }
                ?.key as? String ?: "unknown"

            // Classify activity
            val classification = classifyActivity(dominantCategory, appSwitchCount, focusEvents, notifEvents)

            candidates.add(SegmentCandidate(
                type = "activity_segment",
                layer = "activity",
                label = classification.first,
                confidence = classification.second,
                startTime = event.timestamp,
                endTime = event.timestamp, // Will be adjusted
                payload = mapOf(
                    "dominant_category" to dominantCategory,
                    "app_switch_count" to appSwitchCount
                )
            ))
        }

        // Generate focus segments
        for (event in focusEvents) {
            val payload = parsePayload(event.payload)
            val action = payload["action"] as? String

            if (action == "start") {
                // Find corresponding end event
                val endEvent = focusEvents.find {
                    val p = parsePayload(it.payload)
                    p["action"] == "end" && it.timestamp > event.timestamp
                }

                val endTime = endEvent?.timestamp ?: now
                val duration = parseTimestamp(endTime) - parseTimestamp(event.timestamp)
                val durationMinutes = (duration / 60000).toInt()

                candidates.add(SegmentCandidate(
                    type = "focus_segment",
                    layer = "focus",
                    label = "deep_work",
                    confidence = 0.85,
                    startTime = event.timestamp,
                    endTime = endTime,
                    payload = mapOf(
                        "source" to (payload["source"] ?: "manual"),
                        "intended_duration_minutes" to (payload["intended_duration_minutes"] ?: durationMinutes),
                        "actual_duration_minutes" to durationMinutes
                    )
                ))
            }
        }

        // Generate place visit segments from location events
        var currentPlace: String? = null
        var currentPlaceStart: String? = null

        for (event in locationEvents) {
            val payload = parsePayload(event.payload)
            val place = payload["place"] as? String ?: continue
            val changed = payload["changed"] as? Boolean ?: false

            if (changed && currentPlace != null && currentPlaceStart != null) {
                // End previous place visit
                candidates.add(SegmentCandidate(
                    type = "place_visit",
                    layer = "place",
                    label = currentPlace,
                    confidence = 0.88,
                    startTime = currentPlaceStart,
                    endTime = event.timestamp,
                    payload = mapOf("confidence" to (payload["confidence"] ?: 0.88))
                ))
                currentPlaceStart = event.timestamp
            } else if (currentPlace == null) {
                currentPlaceStart = event.timestamp
            }
            currentPlace = place
        }

        // Generate distraction segments if context indicates high distraction
        if (contextSnapshot != null) {
            val distractionProb = (contextSnapshot["distraction_probability"] as? Number)?.toDouble() ?: 0.0
            if (distractionProb > 0.4) {
                candidates.add(SegmentCandidate(
                    type = "distraction_segment",
                    layer = "activity",
                    label = "distraction",
                    confidence = distractionProb,
                    startTime = now,
                    endTime = now,
                    payload = mapOf("distraction_probability" to distractionProb)
                ))
            }
        }

        // Convert candidates to entities
        return candidates.map { candidate ->
            TimelineSegmentEntity(
                segmentId = UUID.randomUUID().toString(),
                segmentType = candidate.type,
                layer = candidate.layer,
                startTime = candidate.startTime,
                endTime = candidate.endTime,
                label = candidate.label,
                confidence = candidate.confidence,
                version = 1,
                validFrom = candidate.startTime,
                ingestedAt = now,
                evidence = Gson().toJson(mapOf("synthetic" to true)),
                payload = Gson().toJson(candidate.payload)
            )
        }
    }

    /**
     * Classifies activity from app category and context.
     * Returns (activity_label, confidence).
     */
    private fun classifyActivity(
        dominantCategory: String,
        appSwitchCount: Int,
        focusEvents: List<EventEntity>,
        notifEvents: List<EventEntity>
    ): Pair<String, Double> {
        val isFocusActive = focusEvents.any {
            val payload = parsePayload(it.payload)
            payload["action"] == "start"
        }

        val notifCount = notifEvents.size

        return when {
            isFocusActive && dominantCategory == "productivity" && appSwitchCount < 4 -> "deep_work" to 0.81
            isFocusActive && dominantCategory == "productivity" -> "shallow_work" to 0.70
            dominantCategory == "communication" -> "communication" to 0.75
            dominantCategory == "social" && isFocusActive -> "distraction" to 0.80
            dominantCategory == "social" -> "social" to 0.70
            dominantCategory == "entertainment" -> "leisure" to 0.65
            dominantCategory == "browser" && appSwitchCount > 5 -> "shallow_work" to 0.60
            dominantCategory == "system" -> "admin" to 0.55
            else -> "unknown" to 0.40
        }
    }

    private fun parsePayload(payload: String): Map<String, Any?> {
        return try {
            Gson().fromJson(payload, Map::class.java) as? Map<String, Any?> ?: emptyMap()
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun parseTimestamp(ts: String): Long {
        return try {
            SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }.parse(ts)?.time ?: 0L
        } catch (e: Exception) {
            0L
        }
    }
}
