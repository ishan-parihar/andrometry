package com.andrometry

import android.app.Application
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.sync.queue.SyncQueue
import com.andrometry.sync.queue.CollectorScheduler
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

/**
 * Application class with Hilt dependency injection.
 *
 * Wires all collector scheduling per architecture §8 tiered collection strategy.
 * ScreenReceiver is declared in AndroidManifest.xml and gets Hilt injection
 * automatically via @AndroidEntryPoint.
 */
@HiltAndroidApp
class AndrometryApp : Application() {

    @Inject lateinit var database: AndrometryDatabase
    @Inject lateinit var syncQueue: SyncQueue
    @Inject lateinit var collectorScheduler: CollectorScheduler

    override fun onCreate() {
        super.onCreate()

        // Schedule periodic sync — SyncWorker now uses @HiltWorker properly
        syncQueue.schedulePeriodicSync()

        // Schedule all collector workers per architecture §8 tiered collection
        collectorScheduler.scheduleAll(this)
    }
}
