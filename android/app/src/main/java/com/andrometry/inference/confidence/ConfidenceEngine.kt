package com.andrometry.inference.confidence

import javax.inject.Inject
import javax.inject.Singleton

/**
 * Confidence scoring engine — evaluates inference quality.
 * Every context inference includes confidence scores.
 *
 * Confidence factors:
 * - Signal freshness
 * - Signal agreement
 * - Permission availability
 * - Historical accuracy
 * - Data completeness
 * - Timezone stability
 * - Location confidence
 * - Calendar availability
 */
@Singleton
class ConfidenceEngine @Inject constructor() {

    data class ConfidenceFactors(
        val signalFreshness: Double = 0.0,
        val signalAgreement: Double = 0.0,
        val permissionAvailability: Double = 0.0,
        val dataCompleteness: Double = 0.0,
        val historicalAccuracy: Double = 0.5 // default 50% if no history
    )

    data class ConfidenceResult(
        val activityConfidence: Double,
        val placeConfidence: Double,
        val focusConfidence: Double,
        val overallConfidence: Double,
        val evidence: Map<String, Any>,
        val missing: List<String>
    )

    /**
     * Calculates confidence for a context inference.
     */
    fun calculateConfidence(
        factors: ConfidenceFactors,
        activity: String,
        place: String,
        focusActive: Boolean,
        availableSignals: List<String>,
        missingSignals: List<String>
    ): ConfidenceResult {
        val baseConfidence = calculateBaseConfidence(factors)
        val activityConfidence = adjustForActivity(baseConfidence, activity, factors)
        val placeConfidence = adjustForPlace(baseConfidence, place, factors)
        val focusConfidence = if (focusActive) (baseConfidence * 1.1).coerceAtMost(1.0) else baseConfidence

        val overallConfidence = (activityConfidence + placeConfidence + focusConfidence) / 3.0

        val evidence = mapOf(
            "signal_freshness" to factors.signalFreshness,
            "signal_agreement" to factors.signalAgreement,
            "permission_availability" to factors.permissionAvailability,
            "data_completeness" to factors.dataCompleteness,
            "available_signals" to availableSignals,
            "missing_signals" to missingSignals
        )

        return ConfidenceResult(
            activityConfidence = activityConfidence,
            placeConfidence = placeConfidence,
            focusConfidence = focusConfidence,
            overallConfidence = overallConfidence,
            evidence = evidence,
            missing = missingSignals
        )
    }

    private fun calculateBaseConfidence(factors: ConfidenceFactors): Double {
        return (factors.signalFreshness * 0.3 +
                factors.signalAgreement * 0.25 +
                factors.permissionAvailability * 0.2 +
                factors.dataCompleteness * 0.15 +
                factors.historicalAccuracy * 0.1)
    }

    private fun adjustForActivity(base: Double, activity: String, factors: ConfidenceFactors): Double {
        // High-confidence activities get a boost
        val activityBoost = when (activity) {
            "deep_work", "sleep" -> 0.1
            "meeting", "exercise" -> 0.05
            "unknown" -> -0.2
            else -> 0.0
        }
        return (base + activityBoost).coerceIn(0.0, 1.0)
    }

    private fun adjustForPlace(base: Double, place: String, factors: ConfidenceFactors): Double {
        val placeBoost = when (place) {
            "home", "office" -> 0.1
            "unknown" -> -0.15
            else -> 0.0
        }
        return (base + placeBoost).coerceIn(0.0, 1.0)
    }

    /**
     * Calculates signal freshness based on age of most recent signal.
     */
    fun calculateSignalFreshness(lastSignalAgeSeconds: Int): Double {
        return when {
            lastSignalAgeSeconds < 60 -> 1.0
            lastSignalAgeSeconds < 300 -> 0.8
            lastSignalAgeSeconds < 900 -> 0.6
            lastSignalAgeSeconds < 3600 -> 0.4
            else -> 0.2
        }
    }

    /**
     * Calculates signal agreement based on consistency across signals.
     */
    fun calculateSignalAgreement(signals: Map<String, String>): Double {
        if (signals.isEmpty()) return 0.0

        val dominantValue = signals.values.groupingBy { it }.eachCount().maxByOrNull { it.value }?.value ?: 0
        return dominantValue.toDouble() / signals.size
    }

    /**
     * Calculates data completeness based on available collectors.
     */
    fun calculateDataCompleteness(
        expectedCollectors: List<String>,
        activeCollectors: List<String>
    ): Double {
        if (expectedCollectors.isEmpty()) return 1.0
        return activeCollectors.size.toDouble() / expectedCollectors.size
    }
}
