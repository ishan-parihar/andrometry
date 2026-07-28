package com.andrometry

import android.app.Application
import android.content.Intent
import android.content.IntentFilter
import com.andrometry.collectors.screen.ScreenCollector
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.sync.queue.SyncQueue
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class AndrometryApp : Application() {

    @Inject lateinit var database: AndrometryDatabase
    @Inject lateinit var syncQueue: SyncQueue

    private val screenCollector = ScreenCollector()

    override fun onCreate() {
        super.onCreate()

        // Register screen broadcast receiver
        val filter = ScreenCollector.createIntentFilter()
        registerReceiver(screenCollector, filter)

        // Schedule periodic sync
        syncQueue.schedulePeriodicSync()
    }

    override fun onTerminate() {
        super.onTerminate()
        try {
            unregisterReceiver(screenCollector)
        } catch (e: Exception) {
            // Receiver may not be registered
        }
    }
}
