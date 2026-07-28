package com.andrometry

import android.app.Application
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.sync.queue.SyncQueue
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

/**
 * Application class with Hilt dependency injection.
 *
 * ScreenReceiver is declared in AndroidManifest.xml and gets Hilt injection
 * automatically via @AndroidEntryPoint. No manual registration needed.
 */
@HiltAndroidApp
class AndrometryApp : Application() {

    @Inject lateinit var database: AndrometryDatabase
    @Inject lateinit var syncQueue: SyncQueue

    override fun onCreate() {
        super.onCreate()

        // Schedule periodic sync — SyncWorker now uses @HiltWorker properly
        syncQueue.schedulePeriodicSync()
    }
}
