package com.andrometry.inference.readiness

import javax.inject.Inject
import javax.inject.Singleton

/**
 * Readiness and Fatigue Model — implements §40.
 *
 * A pure computation engine that estimates user readiness and fatigue.
 * No database access needed — this is a stateless model.
 *
 * §40.1 — Inputs: sleep, time awake, late-night screen, HR, activity, workload
 * §40.2 — Outputs: readiness_score, fatigue_probability, recommended_focus_intensity, suggested_break
 * §40.3 — Usage: agents time interventions, suggest task difficulty, recommend breaks
 */
@Singleton
class ReadinessEngine @Inject constructor() {

    data class ReadinessInput(
        val sleepDurationMinutes: Int = 0,
        val sleepRegularityScore: Double = 0.5,
        val hoursSinceWake: Double = 0.0,
        val lateNightScreenMinutes: Int = 0,
        val heartRateAvg: Int = 0,
        val heartRateVariability: Double = 0.0,
        val stepsToday: Int = 0,
        val focusMinutesToday: Int = 0,
        val meetingMinutesToday: Int = 0,
        val age: Int = 30
    )

    data class ReadinessOutput(
        val readinessScore: Double,
        val fatigueProbability: Double,
        val recommendedFocusIntensity: String,
        val suggestedBreak: Boolean,
        val sleepDebtMinutes: Int,
        val cognitiveLoad: String,
        val bestNextAction: String
    )

    fun computeReadiness(input: ReadinessInput): ReadinessOutput {
        val idealSleepMinutes = 480
        val sleepDebtMinutes = (idealSleepMinutes - input.sleepDurationMinutes).coerceAtLeast(0)
        val sleepQuality = calculateSleepQuality(input)
        val timeAwakeFactor = calculateTimeAwakeFactor(input.hoursSinceWake)
        val lateNightImpact = calculateLateNightImpact(input.lateNightScreenMinutes)
        val activityBoost = calculateActivityBoost(input.stepsToday)
        val workloadFactor = calculateWorkloadFactor(input.focusMinutesToday, input.meetingMinutesToday)
        val heartRateFactor = if (input.heartRateAvg > 0) {
            calculateHeartRateFactor(input.heartRateAvg, input.heartRateVariability)
        } else {
            0.5
        }

        var readinessScore = 0.0
        readinessScore += sleepQuality * 0.30
        readinessScore += timeAwakeFactor * 0.25
        readinessScore += activityBoost * 0.10
        readinessScore -= lateNightImpact * 0.10
        readinessScore += heartRateFactor * 0.10
        readinessScore += workloadFactor * 0.15
        readinessScore = readinessScore.coerceIn(0.0, 1.0)

        val fatigueProbability = (1.0 - readinessScore + lateNightImpact * 0.3).coerceIn(0.0, 1.0)

        val cognitiveLoad = when {
            input.meetingMinutesToday > 120 -> "high"
            input.focusMinutesToday > 180 -> "medium"
            else -> "low"
        }

        val recommendedFocusIntensity = when {
            readinessScore > 0.7 && sleepDebtMinutes < 60 -> "high"
            readinessScore > 0.4 && sleepDebtMinutes < 120 -> "medium"
            else -> "low"
        }

        val suggestedBreak = fatigueProbability > 0.6 ||
                            input.focusMinutesToday > 120 ||
                            input.meetingMinutesToday > 90

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

    private fun calculateSleepQuality(input: ReadinessInput): Double {
        if (input.sleepDurationMinutes <= 0) return 0.0
        val durationScore = when {
            input.sleepDurationMinutes in 420..540 -> 1.0
            input.sleepDurationMinutes in 360..419 -> 0.7
            input.sleepDurationMinutes in 300..359 -> 0.4
            input.sleepDurationMinutes < 300 -> 0.2
            else -> 0.6
        }
        val regularityBonus = input.sleepRegularityScore * 0.3
        return (durationScore * 0.7 + regularityBonus).coerceIn(0.0, 1.0)
    }

    private fun calculateTimeAwakeFactor(hoursSinceWake: Double): Double {
        return when {
            hoursSinceWake <= 2 -> 0.6
            hoursSinceWake in 2.0..6.0 -> 0.9
            hoursSinceWake in 6.0..10.0 -> 0.7
            hoursSinceWake in 10.0..14.0 -> 0.5
            hoursSinceWake > 14 -> 0.3
            else -> 0.5
        }
    }

    private fun calculateLateNightImpact(lateNightScreenMinutes: Int): Double {
        return when {
            lateNightScreenMinutes == 0 -> 0.0
            lateNightScreenMinutes <= 30 -> 0.1
            lateNightScreenMinutes <= 60 -> 0.2
            lateNightScreenMinutes <= 120 -> 0.4
            else -> 0.6
        }
    }

    private fun calculateActivityBoost(stepsToday: Int): Double {
        return when {
            stepsToday >= 10000 -> 0.8
            stepsToday >= 7000 -> 0.6
            stepsToday >= 4000 -> 0.4
            stepsToday >= 1000 -> 0.2
            else -> 0.0
        }
    }

    private fun calculateWorkloadFactor(focusMinutes: Int, meetingMinutes: Int): Double {
        val totalLoadMinutes = focusMinutes + meetingMinutes
        return when {
            totalLoadMinutes > 360 -> -0.3
            totalLoadMinutes > 240 -> -0.1
            totalLoadMinutes > 120 -> 0.1
            else -> 0.2
        }
    }

    private fun calculateHeartRateFactor(heartRateAvg: Int, hrv: Double): Double {
        val hrScore = when {
            heartRateAvg < 60 -> 0.8
            heartRateAvg < 70 -> 0.6
            heartRateAvg < 80 -> 0.4
            heartRateAvg < 90 -> 0.2
            else -> 0.0
        }
        val hrvBonus = when {
            hrv > 100 -> 0.2
            hrv > 60 -> 0.1
            hrv > 30 -> 0.0
            else -> -0.1
        }
        return (hrScore + hrvBonus).coerceIn(0.0, 1.0)
    }
}
