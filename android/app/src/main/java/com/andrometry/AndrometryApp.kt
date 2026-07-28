package com.andrometry

import android.app.Application
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.di.AppModule
import com.andrometry.sync.queue.SyncQueue
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

/**
 * Application class with Hilt dependency injection.
 *
 * Fixed: ScreenCollector now uses @AndroidEntryPoint, so we register it
 * via a delegating receiver wrapper instead of direct instantiation.
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
