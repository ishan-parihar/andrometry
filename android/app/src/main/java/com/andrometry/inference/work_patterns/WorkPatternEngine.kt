package com.andrometry.inference.work_patterns

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.TimelineSegmentEntity
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Work Pattern Engine — implements §39 Work Pattern Metrics.
 *
 * Analyzes focus sessions, task switching, and work fragmentation to provide
 * actionable insights about productivity patterns.
 *
 * §39.1 — Metrics:
 * - Average focus bout duration
 * - Task switching rate per hour
 * - Meeting fragmentation score
 * - Deep work to shallow work ratio
 * - Focus session success rate
 * - Distraction frequency and duration
 *
 * §39.2 — Work pattern object:
 * These metrics are included in daily summaries and context snapshots
 * so agents can detect pattern changes and suggest improvements.
 */
@Singleton
class WorkPatternEngine @Inject constructor(
    private val database: AndrometryDatabase
) {

    data class WorkPatternMetrics(
        val averageFocusBoutMinutes: Double,
        val switchingRatePerHour: Double,
        val meetingFragmentationScore: Double,
        val deepWorkRatio: Double,
        val focusSessionSuccessRate: Double,
        val distractionFrequencyPerHour: Double,
        val averageDistractionMinutes: Double,
        val longestFocusBoutMinutes: Int,
        val totalFocusMinutes: Int,
        val totalMeetingMinutes: Int,
        val totalDistractionMinutes: Int
    )

    data class FocusBout(
        val startTime: String,
        val endTime: String,
        val durationMinutes: Int,
        val label: String,
        val successScore: Double
    )

    /**
     * Computes work pattern metrics for a given day.
     *
     * @param dateStr Date string in YYYY-MM-DD format
     * @param userId User ID (default: "primary")
     * @return Work pattern metrics for the day
     */
    suspend fun computeDailyMetrics(dateStr: String, userId: String = "primary"): WorkPatternMetrics {
        val dayStart = "${dateStr}T00:00:00Z"
        val dayEnd = "${dateStr}T23:59:59Z"

        val segments = database.segmentDao().getSegmentsInRange(userId, dayStart, dayEnd)

        // Extract focus segments
        val focusSegments = segments.filter {
            it.segmentType == "focus_segment" || it.label in listOf("deep_work", "shallow_work")
        }

        // Extract meeting segments
        val meetingSegments = segments.filter {
            it.segmentType == "meeting_segment" || it.label == "meeting"
        }

        // Extract distraction segments
        val distractionSegments = segments.filter {
            it.segmentType == "distraction_segment" || it.label == "distraction"
        }

        // Calculate focus bout metrics
        val focusDurations = focusSegments.map { segment ->
            calculateDurationMinutes(segment.startTime, segment.endTime)
        }.filter { it > 0 }

        val averageFocusBoutMinutes = if (focusDurations.isNotEmpty()) {
            focusDurations.average()
        } else 0.0

        val longestFocusBoutMinutes = focusDurations.maxOrNull() ?: 0
        val totalFocusMinutes = focusDurations.sum()

        // Calculate meeting metrics
        val meetingDurations = meetingSegments.map { segment ->
            calculateDurationMinutes(segment.startTime, segment.endTime)
        }.filter { it > 0 }
        val totalMeetingMinutes = meetingDurations.sum()

        // Calculate distraction metrics
        val distractionDurations = distractionSegments.map { segment ->
            calculateDurationMinutes(segment.startTime, segment.endTime)
        }.filter { it > 0 }
        val totalDistractionMinutes = distractionDurations.sum()

        // Calculate switching rate (approximate: count of distinct activity segments per hour)
        val activitySegments = segments.filter { it.segmentType == "activity_segment" }
        val totalHours = 14.0 // Assume ~14 waking hours if no segments
        val switchingRatePerHour = if (totalHours > 0) {
            activitySegments.size.toDouble() / totalHours
        } else 0.0

        // Meeting fragmentation score (0 = no meetings, 1 = meetings break up entire day)
        val meetingFragmentationScore = calculateMeetingFragmentation(segments, totalFocusMinutes + totalMeetingMinutes + totalDistractionMinutes)

        // Deep work ratio
        val deepWorkMinutes = focusSegments.filter { it.label == "deep_work" }.map {
            calculateDurationMinutes(it.startTime, it.endTime)
        }.sum()
        val deepWorkRatio = if (totalFocusMinutes > 0) {
            deepWorkMinutes.toDouble() / totalFocusMinutes
        } else 0.0

        // Focus session success rate (segments with confidence > 0.7)
        val successfulFocusSessions = focusSegments.count { it.confidence > 0.7 }
        val focusSessionSuccessRate = if (focusSegments.isNotEmpty()) {
            successfulFocusSessions.toDouble() / focusSegments.size
        } else 0.0

        // Distraction frequency
        val distractionFrequencyPerHour = if (totalHours > 0) {
            distractionSegments.size.toDouble() / totalHours
        } else 0.0

        val averageDistractionMinutes = if (distractionDurations.isNotEmpty()) {
            distractionDurations.average()
        } else 0.0

        return WorkPatternMetrics(
            averageFocusBoutMinutes = averageFocusBoutMinutes,
            switchingRatePerHour = switchingRatePerHour,
            meetingFragmentationScore = meetingFragmentationScore,
            deepWorkRatio = deepWorkRatio,
            focusSessionSuccessRate = focusSessionSuccessRate,
            distractionFrequencyPerHour = distractionFrequencyPerHour,
            averageDistractionMinutes = averageDistractionMinutes,
            longestFocusBoutMinutes = longestFocusBoutMinutes,
            totalFocusMinutes = totalFocusMinutes,
            totalMeetingMinutes = totalMeetingMinutes,
            totalDistractionMinutes = totalDistractionMinutes
        )
    }

    /**
     * Gets focus bouts for a time range.
     */
    suspend fun getFocusBouts(userId: String, from: String, to: String): List<FocusBout> {
        val segments = database.segmentDao().getSegmentsInRange(userId, from, to)

        return segments
            .filter { it.segmentType == "focus_segment" }
            .map { segment ->
                FocusBout(
                    startTime = segment.startTime,
                    endTime = segment.endTime,
                    durationMinutes = calculateDurationMinutes(segment.startTime, segment.endTime),
                    label = segment.label,
                    successScore = segment.confidence
                )
            }
            .sortedBy { it.startTime }
    }

    /**
     * Gets a rolling 7-day average of work pattern metrics.
     */
    suspend fun getWeeklyAverageMetrics(userId: String = "primary"): WorkPatternMetrics {
        val metrics = mutableListOf<WorkPatternMetrics>()

        // Get last 7 days of metrics
        for (daysAgo in 0..6) {
            val calendar = java.util.Calendar.getInstance()
            calendar.add(java.util.Calendar.DAY_OF_YEAR, -daysAgo)
            val dateStr = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US).format(calendar.time)

            val dailyMetrics = computeDailyMetrics(dateStr, userId)
            metrics.add(dailyMetrics)
        }

        // Average all metrics
        return WorkPatternMetrics(
            averageFocusBoutMinutes = metrics.map { it.averageFocusBoutMinutes }.average(),
            switchingRatePerHour = metrics.map { it.switchingRatePerHour }.average(),
            meetingFragmentationScore = metrics.map { it.meetingFragmentationScore }.average(),
            deepWorkRatio = metrics.map { it.deepWorkRatio }.average(),
            focusSessionSuccessRate = metrics.map { it.focusSessionSuccessRate }.average(),
            distractionFrequencyPerHour = metrics.map { it.distractionFrequencyPerHour }.average(),
            averageDistractionMinutes = metrics.map { it.averageDistractionMinutes }.average(),
            longestFocusBoutMinutes = metrics.map { it.longestFocusBoutMinutes }.maxOrNull() ?: 0,
            totalFocusMinutes = metrics.sumOf { it.totalFocusMinutes },
            totalMeetingMinutes = metrics.sumOf { it.totalMeetingMinutes },
            totalDistractionMinutes = metrics.sumOf { it.totalDistractionMinutes }
        )
    }

    /**
     * Calculates meeting fragmentation score.
     * Higher score = more fragmented work day.
     */
    private fun calculateMeetingFragmentation(segments: List<TimelineSegmentEntity>, totalMinutes: Int): Double {
        if (totalMinutes <= 0) return 0.0

        val meetingSegments = segments.filter { it.segmentType == "meeting_segment" || it.label == "meeting" }
        if (meetingSegments.isEmpty()) return 0.0

        // Count transitions between meeting and non-meeting segments
        val sortedSegments = segments.sortedBy { it.startTime }
        var transitions = 0
        var wasMeeting = false

        for (segment in sortedSegments) {
            val isMeeting = segment.segmentType == "meeting_segment" || segment.label == "meeting"
            if (wasMeeting && !isMeeting) transitions++
            wasMeeting = isMeeting
        }

        // Normalize: more transitions = more fragmentation
        return (transitions.toDouble() / (sortedSegments.size.coerceAtLeast(1))).coerceIn(0.0, 1.0)
    }

    /**
     * Calculates duration in minutes between two ISO timestamps.
     */
    private fun calculateDurationMinutes(start: String, end: String): Int {
        return try {
            val startTime = java.text.SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", java.util.Locale.US).parse(start)?.time ?: 0L
            val endTime = java.text.SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", java.util.Locale.US).parse(end)?.time ?: 0L
            ((endTime - startTime) / 60000).toInt().coerceAtLeast(0)
        } catch (e: Exception) {
            0
        }
    }
}
