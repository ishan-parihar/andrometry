package com.andrometry.core.di

import android.content.Context
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.network.RelayClient
import com.andrometry.sync.queue.SyncQueue
import com.andrometry.agents.interventions.InterventionNotificationService
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Named
import javax.inject.Singleton

/**
 * Hilt dependency injection module providing app-level dependencies.
 *
 * ReadinessEngine, WorkPatternEngine, and HabitEngine are NOT provided here
 * because they have @Inject constructors and @Singleton — Hilt auto-provides them.
 *
 * Only classes that need manual construction (database creation, @Named bindings)
 * are provided here.
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
}
