package com.andrometry.collectors.notifications

import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject

/**
 * NotificationListenerService that captures notification metadata.
 * Stores only metadata (package, category, timestamp) — never content.
 *
 * Tier 0 collection — event-driven, low battery impact.
 * Produces: notification_event events
 *
 * Includes debounce: rapid notifications from the same package within 1 second
 * are collapsed to avoid flooding the DB.
 */
@AndroidEntryPoint
class NotificationCollector : NotificationListenerService() {

    @Inject lateinit var sequenceGenerator: SequenceGenerator
    @Inject lateinit var database: AndrometryDatabase

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    // Debounce: last notification per package with timestamp
    // Evicted every 5 minutes to prevent unbounded growth
    private val lastNotificationByPackage = mutableMapOf<String, Long>()
    private var lastEvictionTime = System.currentTimeMillis()
    private val DEBOUNCE_MS = 1000L // 1 second
    private val EVICT_INTERVAL_MS = 5 * 60 * 1000L // 5 minutes
    private val MAX_DEBOUNCE_ENTRIES = 100

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        scope.launch {
            emitNotificationEvent(sbn, posted = true)
        }
    }

    override fun onNotificationRemoved(sbn: StatusBarNotification) {
        scope.launch {
            emitNotificationEvent(sbn, posted = false)
        }
    }

    private suspend fun emitNotificationEvent(sbn: StatusBarNotification, posted: Boolean) {
        // Evict stale debounce entries periodically
        val now = System.currentTimeMillis()
        if (now - lastEvictionTime > EVICT_INTERVAL_MS) {
            lastNotificationByPackage.entries.removeAll { now - it.value > EVICT_INTERVAL_MS }
            lastEvictionTime = now
        }

        // Debounce: skip if same package within 1 second
        val lastTime = lastNotificationByPackage[sbn.packageName] ?: 0
        if (now - lastTime < DEBOUNCE_MS) {
            return
        }
        if (lastNotificationByPackage.size < MAX_DEBOUNCE_ENTRIES) {
            lastNotificationByPackage[sbn.packageName] = now
        }

        val timestamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val payload = mapOf(
            "package" to sbn.packageName,
            "category" to categorizePackage(sbn.packageName),
            "posted" to posted,
            "removed" to !posted,
            "timestamp" to sbn.postTime
        )

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = timestamp,
            timezone = TimeZone.getDefault().id,
            eventType = "notification_event",
            collector = "notification_listener",
            source = "broadcast",
            payload = com.google.gson.Gson().toJson(payload)
        )

        database.eventDao().insert(event)
    }

    private fun categorizePackage(packageName: String): String {
        return when {
            packageName.contains("slack") || packageName.contains("teams") ||
            packageName.contains("discord") || packageName.contains("telegram") ||
            packageName.contains("whatsapp") || packageName.contains("signal") -> "communication"
            packageName.contains("gmail") || packageName.contains("outlook") ||
            packageName.contains("mail") -> "email"
            packageName.contains("calendar") -> "calendar"
            packageName.contains("twitter") || packageName.contains("instagram") ||
            packageName.contains("facebook") || packageName.contains("reddit") -> "social"
            packageName.contains("news") -> "news"
            else -> "other"
        }
    }
}
