package com.andrometry.core.di

import android.content.Context
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.network.RelayClient
import com.andrometry.inference.habits.HabitEngine
import com.andrometry.inference.readiness.ReadinessEngine
import com.andrometry.inference.work_patterns.WorkPatternEngine
import com.andrometry.sync.queue.SyncQueue
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Named
import javax.inject.Singleton

/**
 * Hilt dependency injection module providing all app-level dependencies.
 *
 * Provides: Database, SequenceGenerator, RelayClient, SyncQueue,
 *           ReadinessEngine, WorkPatternEngine, HabitEngine
 */
@Module
@InstallIn(SingletonComponent::class)
object AppModule {

    @Provides
    @Named("relay_url")
    fun provideRelayUrl(): String {
        return System.getenv("ANDROMETRY_RELAY_URL") ?: "http://localhost:8420"
    }

    @Provides
    @Named("device_token")
    fun provideDeviceToken(): String {
        return System.getenv("ANDROMETRY_DEVICE_TOKEN") ?: ""
    }

    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): AndrometryDatabase {
        return AndrometryDatabase.create(context)
    }

    @Provides
    @Singleton
    fun provideSequenceGenerator(@ApplicationContext context: Context): SequenceGenerator {
        return SequenceGenerator(context)
    }

    @Provides
    @Singleton
    fun provideRelayClient(
        @Named("relay_url") relayUrl: String,
        @Named("device_token") deviceToken: String
    ): RelayClient {
        return RelayClient(relayUrl, deviceToken)
    }

    @Provides
    @Singleton
    fun provideSyncQueue(
        @ApplicationContext context: Context,
        database: AndrometryDatabase,
        relayClient: RelayClient
    ): SyncQueue {
        return SyncQueue(context, database, relayClient)
    }

    // Pure computation engines — no constructor dependencies beyond @Inject

    @Provides
    @Singleton
    fun provideReadinessEngine(): ReadinessEngine {
        return ReadinessEngine()
    }

    @Provides
    @Singleton
    fun provideWorkPatternEngine(database: AndrometryDatabase): WorkPatternEngine {
        return WorkPatternEngine(database)
    }

    @Provides
    @Singleton
    fun provideHabitEngine(database: AndrometryDatabase): HabitEngine {
        return HabitEngine(database)
    }
}
