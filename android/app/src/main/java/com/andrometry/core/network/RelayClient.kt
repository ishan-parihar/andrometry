package com.andrometry.core.network

import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import retrofit2.http.*
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Named
import javax.inject.Singleton

// ============================================================
// API interface
// ============================================================

data class BatchRequest(
    val device_id: String,
    val batch_id: String,
    val client_version: String,
    val schema_version: Int,
    val sequence_start: Long,
    val sequence_end: Long,
    val events: List<Map<String, Any?>>,
    val segments: List<Map<String, Any?>>,
    val summaries: List<Map<String, Any?>>,
    val context: Map<String, Any?>?,
    val device_state: Map<String, Any?>?
)

data class BatchResponse(
    val accepted_events: Int,
    val rejected_events: Int,
    val accepted_segments: Int,
    val accepted_summaries: Int,
    val missing_sequences: List<Long>,
    val pending_interventions: Int,
    val server_time: String
)

data class ContextResponse(
    val timestamp: String,
    val timezone: String,
    val device_id: String,
    val etag: String,
    val context: Map<String, Any?>?
)

data class TimelineResponse(
    val timeline: List<Map<String, Any?>>
)

interface AndrometryApi {

    @POST("v1/batch")
    suspend fun postBatch(
        @Header("Authorization") token: String,
        @Body batch: BatchRequest
    ): BatchResponse

    @GET("v1/context/current")
    suspend fun getCurrentContext(
        @Header("Authorization") token: String,
        @Header("If-None-Match") etag: String? = null
    ): ContextResponse

    @GET("v1/timeline")
    suspend fun getTimeline(
        @Header("Authorization") token: String,
        @Query("from") from: String,
        @Query("to") to: String,
        @Query("segment_type") segmentType: String? = null,
        @Query("limit") limit: Int = 200
    ): TimelineResponse

    @GET("v1/briefing/current")
    suspend fun getCurrentBriefing(
        @Header("Authorization") token: String
    ): Map<String, Any?>

    @GET("v1/preferences")
    suspend fun getPreferences(
        @Header("Authorization") token: String,
        @Query("category") category: String? = null
    ): Map<String, Any?>

    @GET("v1/summary/daily")
    suspend fun getDailySummary(
        @Header("Authorization") token: String,
        @Query("date") date: String
    ): Map<String, Any?>

    @GET("v1/device/interventions")
    suspend fun getPendingInterventions(
        @Header("Authorization") token: String
    ): Map<String, Any?>

    @POST("v1/feedback")
    suspend fun postFeedback(
        @Header("Authorization") token: String,
        @Body feedback: Map<String, Any?>
    ): Map<String, String>
}

// ============================================================
// Client
// ============================================================

@Singleton
class RelayClient @Inject constructor(
    @Named("relay_url") private val relayUrl: String,
    @Named("device_token") private val deviceToken: String
) {
    private val api: AndrometryApi by lazy {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        val client = OkHttpClient.Builder()
            .addInterceptor { chain ->
                val request = chain.request().newBuilder()
                    .addHeader("Content-Type", "application/json")
                    .build()
                chain.proceed(request)
            }
            .addInterceptor(logging)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()

        Retrofit.Builder()
            .baseUrl(relayUrl.trimEnd('/') + "/")
            .client(client)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
            .create(AndrometryApi::class.java)
    }

    private val authHeader: String get() = "Bearer $deviceToken"

    suspend fun postBatch(batch: BatchRequest): BatchResponse {
        return api.postBatch(authHeader, batch)
    }

    suspend fun getCurrentContext(): ContextResponse? {
        return try {
            api.getCurrentContext(authHeader)
        } catch (e: Exception) {
            null
        }
    }

    suspend fun getTimeline(from: String, to: String, segmentType: String? = null): TimelineResponse? {
        return try {
            api.getTimeline(authHeader, from, to, segmentType)
        } catch (e: Exception) {
            null
        }
    }

    suspend fun getDailySummary(date: String): Map<String, Any?>? {
        return try {
            api.getDailySummary(authHeader, date)
        } catch (e: Exception) {
            null
        }
    }

    suspend fun getPendingInterventions(): Map<String, Any?>? {
        return try {
            api.getPendingInterventions(authHeader)
        } catch (e: Exception) {
            null
        }
    }
}
