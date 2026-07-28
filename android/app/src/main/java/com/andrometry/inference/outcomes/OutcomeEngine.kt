package com.andrometry.inference.outcomes

import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.OutcomeEntity
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Outcome Engine — implements §35 Outcome Attribution.
 *
 * Tracks whether agent interventions actually worked.
 *
 * §35.1 — Outcome types:
 * - accepted: user acknowledged the intervention
 * - dismissed: user explicitly dismissed
 * - ignored: user took no action
 * - completed: user completed the suggested action
 * - behavior_change: measurable behavior change observed
 * - metric_improved: target metric improved
 * - self_reported_helpful: user reported helpful
 * - no_measurable_effect: no observable change
 * - negative_effect: intervention caused harm
 *
 * §35.3 — Effectiveness scoring:
 * effectiveness_score =
 *   0.25 * accepted +
 *   0.35 * behavior_change +
 *   0.25 * metric_improvement +
 *   0.15 * user_helpfulness
 */
@Singleton
class OutcomeEngine @Inject constructor(
    private val database: AndrometryDatabase
) {

    data class Outcome(
        val outcomeId: String,
        val interventionId: String,
        val timestamp: String,
        val outcomeType: String,
        val evidence: Map<String, Any> = emptyMap(),
        val selfReport: Map<String, Any> = emptyMap(),
        val effectivenessScore: Double? = null
    )

    data class OutcomeStats(
        val interventionId: String,
        val totalOutcomes: Int,
        val acceptedCount: Int,
        val behaviorChangeCount: Int,
        val metricImprovedCount: Int,
        val helpfulCount: Int,
        val negativeCount: Int,
        val averageEffectiveness: Double?,
        val latestOutcomeType: String?
    )

    companion object {
        // §35.3 Effectiveness scoring weights
        private const val WEIGHT_ACCEPTED = 0.25
        private const val WEIGHT_BEHAVIOR_CHANGE = 0.35
        private const val WEIGHT_METRIC_IMPROVEMENT = 0.25
        private const val WEIGHT_USER_HELPFULNESS = 0.15

        // Valid outcome types per §35.1
        val VALID_OUTCOME_TYPES = setOf(
            "accepted", "dismissed", "ignored", "completed",
            "behavior_change", "metric_improved", "self_reported_helpful",
            "no_measurable_effect", "negative_effect"
        )
    }

    /**
     * Records an outcome for an intervention (§35.2).
     */
    suspend fun recordOutcome(
        interventionId: String,
        outcomeType: String,
        evidence: Map<String, Any> = emptyMap(),
        selfReport: Map<String, Any> = emptyMap()
    ): Outcome {
        require(outcomeType in VALID_OUTCOME_TYPES) {
            "Invalid outcome type: $outcomeType. Must be one of: $VALID_OUTCOME_TYPES"
        }

        val now = getCurrentTimestamp()
        val outcomeId = UUID.randomUUID().toString()

        // Calculate effectiveness score
        val effectivenessScore = calculateEffectiveness(
            outcomeType = outcomeType,
            selfReport = selfReport
        )

        val outcome = Outcome(
            outcomeId = outcomeId,
            interventionId = interventionId,
            timestamp = now,
            outcomeType = outcomeType,
            evidence = evidence,
            selfReport = selfReport,
            effectivenessScore = effectivenessScore
        )

        // Persist to database
        val entity = OutcomeEntity(
            outcomeId = outcomeId,
            interventionId = interventionId,
            timestamp = now,
            outcomeType = outcomeType,
            effectivenessScore = effectivenessScore,
            payload = Gson().toJson(outcome),
            syncState = "pending"
        )
        database.outcomeDao().upsert(entity)

        // Also record as event for audit trail
        val event = com.andrometry.core.database.EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = 0,
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "outcome_recorded",
            collector = "outcome_engine",
            source = "agent",
            payload = Gson().toJson(outcome)
        )
        database.eventDao().insert(event)

        return outcome
    }

    /**
     * Calculates effectiveness score per §35.3 formula.
     */
    private fun calculateEffectiveness(
        outcomeType: String,
        selfReport: Map<String, Any>
    ): Double {
        var score = 0.0

        // Accepted component (§35.3)
        val accepted = when (outcomeType) {
            "accepted", "completed", "behavior_change", "metric_improved" -> 1.0
            "dismissed" -> 0.0
            "ignored" -> 0.0
            "self_reported_helpful" -> 0.5
            "no_measurable_effect" -> 0.3
            "negative_effect" -> 0.0
            else -> 0.0
        }
        score += WEIGHT_ACCEPTED * accepted

        // Behavior change component (§35.3)
        val behaviorChange = when (outcomeType) {
            "behavior_change" -> 1.0
            "metric_improved" -> 0.8
            "completed" -> 0.6
            "accepted" -> 0.3
            "self_reported_helpful" -> 0.4
            "no_measurable_effect" -> 0.0
            "negative_effect" -> -0.5
            else -> 0.0
        }
        score += WEIGHT_BEHAVIOR_CHANGE * behaviorChange

        // Metric improvement component (§35.3)
        val metricImprovement = when (outcomeType) {
            "metric_improved" -> 1.0
            "behavior_change" -> 0.7
            "completed" -> 0.5
            "accepted" -> 0.2
            "self_reported_helpful" -> 0.3
            "no_measurable_effect" -> 0.0
            "negative_effect" -> -0.3
            else -> 0.0
        }
        score += WEIGHT_METRIC_IMPROVEMENT * metricImprovement

        // User helpfulness component from self-report (§35.3)
        val helpfulRating = selfReport["rating"] as? Number
        val helpfulBoolean = selfReport["helpful"] as? Boolean
        val userHelpfulness = when {
            helpfulRating != null -> (helpfulRating.toDouble() / 5.0).coerceIn(0.0, 1.0)
            helpfulBoolean == true -> 0.8
            helpfulBoolean == false -> 0.2
            outcomeType == "self_reported_helpful" -> 0.8
            else -> 0.5 // neutral if no self-report
        }
        score += WEIGHT_USER_HELPFULNESS * userHelpfulness

        return score.coerceIn(0.0, 1.0)
    }

    /**
     * Gets all outcomes for a specific intervention.
     */
    suspend fun getOutcomesForIntervention(interventionId: String): List<Outcome> {
        val entities = database.outcomeDao().getByIntervention(interventionId)
        return entities.map { entity ->
            Outcome(
                outcomeId = entity.outcomeId,
                interventionId = entity.interventionId,
                timestamp = entity.timestamp,
                outcomeType = entity.outcomeType,
                effectivenessScore = entity.effectivenessScore
            )
        }
    }

    /**
     * Gets statistics for an intervention's outcomes.
     */
    suspend fun getOutcomeStats(interventionId: String): OutcomeStats {
        val outcomes = getOutcomesForIntervention(interventionId)

        return OutcomeStats(
            interventionId = interventionId,
            totalOutcomes = outcomes.size,
            acceptedCount = outcomes.count { it.outcomeType in setOf("accepted", "completed", "behavior_change", "metric_improved") },
            behaviorChangeCount = outcomes.count { it.outcomeType == "behavior_change" },
            metricImprovedCount = outcomes.count { it.outcomeType == "metric_improved" },
            helpfulCount = outcomes.count { it.outcomeType == "self_reported_helpful" || (it.selfReport["helpful"] == true) },
            negativeCount = outcomes.count { it.outcomeType == "negative_effect" },
            averageEffectiveness = outcomes.mapNotNull { it.effectivenessScore }.average().takeIf { it.isNaN().not() },
            latestOutcomeType = outcomes.maxByOrNull { it.timestamp }?.outcomeType
        )
    }

    /**
     * Gets recent outcomes across all interventions.
     */
    suspend fun getRecentOutcomes(limit: Int = 50): List<Outcome> {
        val entities = database.outcomeDao().getRecent(limit)
        return entities.map { entity ->
            Outcome(
                outcomeId = entity.outcomeId,
                interventionId = entity.interventionId,
                timestamp = entity.timestamp,
                outcomeType = entity.outcomeType,
                effectivenessScore = entity.effectivenessScore
            )
        }
    }

    /**
     * Records user self-report for an intervention (§35.2 self_report field).
     */
    suspend fun recordSelfReport(
        interventionId: String,
        helpful: Boolean,
        rating: Int? = null,
        comment: String? = null
    ): Outcome {
        val selfReport = mutableMapOf<String, Any>(
            "helpful" to helpful
        )
        if (rating != null) selfReport["rating"] = rating.coerceIn(1, 5)
        if (comment != null) selfReport["comment"] = comment

        return recordOutcome(
            interventionId = interventionId,
            outcomeType = "self_reported_helpful",
            selfReport = selfReport
        )
    }

    /**
     * Gets the average effectiveness across all outcomes (for calibration).
     */
    suspend fun getOverallEffectiveness(): Double? {
        val outcomes = getRecentOutcomes(200)
        return outcomes.mapNotNull { it.effectivenessScore }
            .average()
            .takeIf { it.isNaN().not() }
    }

    private fun getCurrentTimestamp(): String {
        return SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())
    }
}
