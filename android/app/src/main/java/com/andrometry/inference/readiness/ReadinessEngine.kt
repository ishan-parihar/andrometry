package com.andrometry.inference.readiness

import com.andrometry.core.database.AndrometryDatabase
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Readiness and Fatigue Model — implements §40.
 *
 * A lightweight model that estimates user readiness and fatigue based on:
 * - Sleep duration and regularity
 * - Time since last sleep
 * - Late-night screen usage
 * - Heart rate trends
 * - Activity level
 * - Current workload (focus minutes, meeting load)
 *
 * §40.1 — Inputs:
 * - sleep_duration_minutes
 * - sleep_regularity_score
 * - hours_since_wake
 * - late_night_screen_minutes (previous day)
 * - heart_rate_avg
 * - heart_rate_variability
 * - steps_today
 * - focus_minutes_today
 * - meeting_minutes_today
 *
 * §40.2 — Outputs:
 * - readiness_score (0.0 to 1.0)
 * - fatigue_probability (0.0 to 1.0)
 * - recommended_focus_intensity (low, medium, high)
 * - suggested_break (boolean)
 *
 * §40.3 — Usage:
 * Agents use readiness scores to:
 * - Time interventions appropriately
 * - Suggest task difficulty levels
 * - Recommend breaks
 * - Adjust notification frequency
 */
@Singleton
class ReadinessEngine @Inject constructor(
    private val database: AndrometryDatabase
) {

    data class ReadinessInput(
        val sleepDurationMinutes: Int = 0,
        val sleepRegularityScore: Double = 0.5,  // 0-1, how consistent is sleep schedule
        val hoursSinceWake: Double = 0.0,
        val lateNightScreenMinutes: Int = 0,      // Screen time after 10pm previous day
        val heartRateAvg: Int = 0,                 // Resting HR average
        val heartRateVariability: Double = 0.0,    // HRV in ms (higher = better)
        val stepsToday: Int = 0,
        val focusMinutesToday: Int = 0,
        val meetingMinutesToday: Int = 0,
        val age: Int = 30                          // Default adult age
    )

    data class ReadinessOutput(
        val readinessScore: Double,           // 0.0 to 1.0
        val fatigueProbability: Double,       // 0.0 to 1.0
        val recommendedFocusIntensity: String, // "low", "medium", "high"
        val suggestedBreak: Boolean,
        val sleepDebtMinutes: Int,
        val cognitiveLoad: String,            // "low", "medium", "high"
        val bestNextAction: String            // "deep_work", "shallow_work", "break", "exercise"
    )

    /**
     * Computes readiness score based on §40 model.
     */
    fun computeReadiness(input: ReadinessInput): ReadinessOutput {
        // Sleep debt calculation (§40.1)
        val idealSleepMinutes = 480 // 8 hours
        val sleepDebtMinutes = (idealSleepMinutes - input.sleepDurationMinutes).coerceAtLeast(0)

        // Sleep quality factor (0-1, higher is better)
        val sleepQuality = calculateSleepQuality(input)

        // Time awake factor (alertness drops after 14+ hours)
        val timeAwakeFactor = calculateTimeAwakeFactor(input.hoursSinceWake)

        // Late night screen impact
        val lateNightImpact = calculateLateNightImpact(input.lateNightScreenMinutes)

        // Physical activity boost
        val activityBoost = calculateActivityBoost(input.stepsToday)

        // Workload factor (high meeting load reduces cognitive readiness)
        val workloadFactor = calculateWorkloadFactor(input.focusMinutesToday, input.meetingMinutesToday)

        // Heart rate indicators (if available)
        val heartRateFactor = if (input.heartRateAvg > 0) {
            calculateHeartRateFactor(input.heartRateAvg, input.heartRateVariability)
        } else {
            0.5 // Neutral if no HR data
        }

        // Composite readiness score
        var readinessScore = 0.0
        readinessScore += sleepQuality * 0.30          // Sleep is most important
        readinessScore += timeAwakeFactor * 0.25       // Time awake matters
        readinessScore += activityBoost * 0.10         // Movement helps
        readinessScore -= lateNightImpact * 0.10       // Late night hurts
        readinessScore += heartRateFactor * 0.10       // HR indicators
        readinessScore += workloadFactor * 0.15        // Current workload

        readinessScore = readinessScore.coerceIn(0.0, 1.0)

        // Fatigue probability (inverse of readiness with adjustments)
        val fatigueProbability = (1.0 - readinessScore + lateNightImpact * 0.3).coerceIn(0.0, 1.0)

        // Cognitive load assessment
        val cognitiveLoad = when {
            input.meetingMinutesToday > 120 -> "high"
            input.focusMinutesToday > 180 -> "medium"
            else -> "low"
        }

        // Recommended focus intensity
        val recommendedFocusIntensity = when {
            readinessScore > 0.7 && sleepDebtMinutes < 60 -> "high"
            readinessScore > 0.4 && sleepDebtMinutes < 120 -> "medium"
            else -> "low"
        }

        // Break suggestion
        val suggestedBreak = fatigueProbability > 0.6 ||
                            input.focusMinutesToday > 120 ||
                            input.meetingMinutesToday > 90

        // Best next action recommendation
        val bestNextAction = when {
            fatigueProbability > 0.7 -> "break"
            readinessScore > 0.7 && input.focusMinutesToday < 120 -> "deep_work"
            input.meetingMinutesToday > 60 -> "shallow_work"
            input.stepsToday < 3000 -> "exercise"
            else -> "shallow_work"
        }

        return ReadinessOutput(
            readinessScore = readinessScore,
            fatigueProbability = fatigueProbability,
            recommendedFocusIntensity = recommendedFocusIntensity,
            suggestedBreak = suggestedBreak,
            sleepDebtMinutes = sleepDebtMinutes,
            cognitiveLoad = cognitiveLoad,
            bestNextAction = bestNextAction
        )
    }

    /**
     * Computes sleep quality score (0-1).
     */
    private fun calculateSleepQuality(input: ReadinessInput): Double {
        if (input.sleepDurationMinutes <= 0) return 0.0

        // Duration quality (optimal: 7-9 hours)
        val durationScore = when {
            input.sleepDurationMinutes in 420..540 -> 1.0  // 7-9 hours
            input.sleepDurationMinutes in 360..419 -> 0.7  // 6-7 hours
            input.sleepDurationMinutes in 300..359 -> 0.4  // 5-6 hours
            input.sleepDurationMinutes < 300 -> 0.2        // < 5 hours
            else -> 0.6                                     // > 9 hours
        }

        // Regularity bonus
        val regularityBonus = input.sleepRegularityScore * 0.3

        return (durationScore * 0.7 + regularityBonus).coerceIn(0.0, 1.0)
    }

    /**
     * Calculates time awake factor (alertness decreases over time).
     */
    private fun calculateTimeAwakeFactor(hoursSinceWake: Double): Double {
        return when {
            hoursSinceWake <= 2 -> 0.6   // Just woke up, groggy
            hoursSinceWake in 2.0..6.0 -> 0.9  // Peak alertness
            hoursSinceWake in 6.0..10.0 -> 0.7 // Afternoon dip
            hoursSinceWake in 10.0..14.0 -> 0.5 // Evening
            hoursSinceWake > 14 -> 0.3   // Very late
            else -> 0.5
        }
    }

    /**
     * Calculates late night screen impact.
     */
    private fun calculateLateNightImpact(lateNightScreenMinutes: Int): Double {
        return when {
            lateNightScreenMinutes == 0 -> 0.0
            lateNightScreenMinutes <= 30 -> 0.1
            lateNightScreenMinutes <= 60 -> 0.2
            lateNightScreenMinutes <= 120 -> 0.4
            else -> 0.6
        }
    }

    /**
     * Calculates activity boost from steps.
     */
    private fun calculateActivityBoost(stepsToday: Int): Double {
        return when {
            stepsToday >= 10000 -> 0.8
            stepsToday >= 7000 -> 0.6
            stepsToday >= 4000 -> 0.4
            stepsToday >= 1000 -> 0.2
            else -> 0.0
        }
    }

    /**
     * Calculates workload factor (high load reduces readiness).
     */
    private fun calculateWorkloadFactor(focusMinutes: Int, meetingMinutes: Int): Double {
        val totalLoadMinutes = focusMinutes + meetingMinutes
        return when {
            totalLoadMinutes > 360 -> -0.3  // > 6 hours of work
            totalLoadMinutes > 240 -> -0.1  // 4-6 hours
            totalLoadMinutes > 120 -> 0.1   // 2-4 hours
            else -> 0.2                     // Light day
        }
    }

    /**
     * Calculates heart rate factor.
     */
    private fun calculateHeartRateFactor(heartRateAvg: Int, hrv: Double): Double {
        // Lower resting HR generally indicates better fitness/recovery
        val hrScore = when {
            heartRateAvg < 60 -> 0.8   // Athlete-level
            heartRateAvg < 70 -> 0.6   // Good
            heartRateAvg < 80 -> 0.4   // Average
            heartRateAvg < 90 -> 0.2   // Above average
            else -> 0.0                // High
        }

        // HRV bonus (higher is better)
        val hrvBonus = when {
            hrv > 100 -> 0.2
            hrv > 60 -> 0.1
            hrv > 30 -> 0.0
            else -> -0.1
        }

        return (hrScore + hrvBonus).coerceIn(0.0, 1.0)
    }

    /**
     * Provides a quick readiness assessment for context snapshots.
     */
    fun quickAssessment(
        sleepMinutes: Int,
        hoursSinceWake: Double,
        focusMinutes: Int
    ): Triple<Double, Double, String> {
        val input = ReadinessInput(
            sleepDurationMinutes = sleepMinutes,
            hoursSinceWake = hoursSinceWake,
            focusMinutesToday = focusMinutes
        )
        val output = computeReadiness(input)
        return Triple(output.readinessScore, output.fatigueProbability, output.recommendedFocusIntensity)
    }
}
