package com.andrometry.agents.interventions

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.andrometry.MainActivity
import com.andrometry.R
import com.andrometry.core.database.AndrometryDatabase
import com.google.gson.Gson
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Intervention Notification Service — §34 Intervention Delivery Layer
 *
 * Polls for pending interventions from the local database and shows
 * Android notifications for delivery to the user.
 *
 * Architecture spec §34.1: "Local Android notifications — highest priority"
 */
@Singleton
class InterventionNotificationService @Inject constructor(
    @ApplicationContext private val context: Context,
    private val database: AndrometryDatabase
) {
    companion object {
        const val CHANNEL_ID = "andrometry_interventions"
        const val NOTIFICATION_TAG = "intervention"
        private const val POLL_INTERVAL_MS = 30_000L // 30 seconds
    }

    private var pollingJob: Job? = null
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    /**
     * Start polling for pending interventions.
     */
    fun start() {
        createNotificationChannel()
        pollingJob = scope.launch {
            while (isActive) {
                try {
                    pollAndNotify()
                } catch (e: Exception) {
                    // Log but don't crash — continue polling
                }
                delay(POLL_INTERVAL_MS)
            }
        }
    }

    /**
     * Stop polling for pending interventions.
     */
    fun stop() {
        pollingJob?.cancel()
        pollingJob = null
    }

    /**
     * Poll for pending interventions and show notifications.
     */
    private suspend fun pollAndNotify() {
        val pending = database.interventionDao().getPending(10)
        for (intervention in pending) {
            // Skip if already delivered (status is 'delivered')
            if (intervention.status == "delivered") continue

            val payload = try {
                Gson().fromJson(intervention.payload, Map::class.java) as? Map<String, Any> ?: emptyMap()
            } catch (e: Exception) {
                emptyMap()
            }

            val title = (payload["title"] as? String) ?: intervention.interventionType
            val message = (payload["message"] as? String) ?: "New intervention from ${intervention.agentId}"
            val severity = (payload["severity"] as? String) ?: "medium"

            showNotification(
                id = intervention.interventionId.hashCode(),
                title = title,
                message = message,
                severity = severity
            )

            // Mark as delivered (§34.2 intervention lifecycle)
            database.interventionDao().updateStatus(intervention.interventionId, "delivered")
        }
    }

    /**
     * Show a notification for an intervention.
     */
    private fun showNotification(id: Int, title: String, message: String, severity: String) {
        // Create intent to open intervention inbox when notification is tapped
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("navigate_to", "interventions")
        }
        val pendingIntent = PendingIntent.getActivity(
            context, id, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        // Set notification priority based on severity
        val priority = when (severity) {
            "high" -> NotificationCompat.PRIORITY_HIGH
            "medium" -> NotificationCompat.PRIORITY_DEFAULT
            else -> NotificationCompat.PRIORITY_LOW
        }

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(message)
            .setPriority(priority)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        try {
            NotificationManagerCompat.from(context).notify("$NOTIFICATION_TAG:$id", id, notification)
        } catch (e: SecurityException) {
            // Notification permission not granted — silently ignore
        }
    }

    /**
     * Create the notification channel for interventions (Android 8+).
     */
    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "AI Interventions",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Notifications for AI agent interventions and suggestions"
            }
            val manager = context.getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }
}
