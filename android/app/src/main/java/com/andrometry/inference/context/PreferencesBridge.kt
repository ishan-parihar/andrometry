package com.andrometry.inference.context

import com.andrometry.core.network.AndrometryApi
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Preferences Bridge — fetches user-configurable alignment weights from the server
 * and applies them to ContextInferenceEngine.
 *
 * This bridges the server-side /v1/preferences endpoint with the Android inference engine,
 * allowing users to customize what "productive" means to them.
 *
 * Thread-safe via Mutex for concurrent coroutine access.
 */
@Singleton
class PreferencesBridge @Inject constructor(
    private val api: AndrometryApi
) {
    private val mutex = Mutex()
    private var cachedWeights: Map<String, Double> = emptyMap()
    private var lastFetchTime: Long = 0L

    /** Refresh interval: fetch from server every 5 minutes */
    private val REFRESH_INTERVAL_MS = 5 * 60 * 1000L

    /** Default alignment weights (matches ContextInferenceEngine.calculateAlignmentProbability) */
    val defaultWeights = mapOf(
        "baseline" to 0.5,
        "focus_bonus" to 0.3,
        "productivity_weight" to 0.15,
        "education_weight" to 0.15,
        "email_weight" to 0.15,
        "calendar_weight" to 0.15,
        "meeting_bonus" to 0.1,
        "social_penalty" to -0.2,
        "entertainment_penalty" to -0.2
    )

    /**
     * Fetches alignment weights from the server (thread-safe).
     * Returns cached weights if recent, otherwise fetches fresh data.
     */
    suspend fun getAlignmentWeights(): Map<String, Double> {
        val now = System.currentTimeMillis()
        if (now - lastFetchTime < REFRESH_INTERVAL_MS && cachedWeights.isNotEmpty()) {
            return cachedWeights
        }

        return mutex.withLock {
            // Double-check after acquiring lock
            if (now - lastFetchTime < REFRESH_INTERVAL_MS && cachedWeights.isNotEmpty()) {
                return@withLock cachedWeights
            }

            try {
                val response = api.getPreferences("alignment")
                if (response.isSuccessful) {
                    val body = response.body()
                    val prefs = body?.get("preferences") as? Map<*, *>
                    val alignmentPrefs = prefs?.get("alignment") as? Map<*, *>

                    if (alignmentPrefs != null) {
                        val weights = mutableMapOf<String, Double>()
                        weights.putAll(defaultWeights)
                        for ((key, value) in alignmentPrefs) {
                            if (key is String && value is String) {
                                value.toDoubleOrNull()?.let { weights[key] = it }
                            }
                        }
                        cachedWeights = weights
                        lastFetchTime = now
                        return@withLock weights
                    }
                }
            } catch (e: Exception) {
                // Network error — use defaults
            }

            cachedWeights = defaultWeights
            lastFetchTime = now
            defaultWeights
        }
    }

    /**
     * Applies server-configured alignment weights to the alignment probability calculation.
     *
     * @param focusActive Whether focus mode is active
     * @param currentAppCategory Current foreground app category
     * @param isInMeeting Whether user is in a calendar meeting
     * @return Adjusted alignment probability (0.0-1.0)
     */
    suspend fun calculateAlignmentProbability(
        focusActive: Boolean,
        currentAppCategory: String,
        isInMeeting: Boolean
    ): Double {
        val weights = getAlignmentWeights()
        val baseline = weights["baseline"] ?: 0.5
        var alignment = baseline

        if (focusActive) {
            alignment += weights["focus_bonus"] ?: 0.3
        }

        val categoryWeight = when (currentAppCategory) {
            "productivity" -> weights["productivity_weight"] ?: 0.15
            "education" -> weights["education_weight"] ?: 0.15
            "email" -> weights["email_weight"] ?: 0.15
            "calendar" -> weights["calendar_weight"] ?: 0.15
            "social" -> weights["social_penalty"] ?: -0.2
            "entertainment" -> weights["entertainment_penalty"] ?: -0.2
            else -> 0.0
        }
        alignment += categoryWeight

        if (isInMeeting) {
            alignment += weights["meeting_bonus"] ?: 0.1
        }

        return alignment.coerceIn(0.0, 1.0)
    }

}
