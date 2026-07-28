package com.andrometry.core.database

import android.content.Context
import androidx.room.*
import kotlinx.coroutines.flow.Flow

// ============================================================
// Entities
// ============================================================

@Entity(tableName = "events")
data class EventEntity(
    @PrimaryKey val eventId: String,
    val deviceId: String,
    val userId: String = "primary",
    val sequence: Long,
    val timestamp: String,
    val timezone: String,
    val eventType: String,
    val collector: String,
    val source: String,
    val schemaVersion: Int = 1,
    val payload: String,
    val syncState: String = "pending",
    val createdAt: String = "datetime('now')"
)

@Entity(tableName = "current_context")
data class CurrentContextEntity(
    @PrimaryKey val id: Int = 1,
    val timestamp: String,
    val timezone: String,
    val context: String,
    val syncState: String = "pending"
)

@Entity(tableName = "timeline_segments")
data class TimelineSegmentEntity(
    @PrimaryKey val segmentId: String,
    val segmentType: String,
    val layer: String,
    val startTime: String,
    val endTime: String,
    val label: String,
    val confidence: Double,
    val version: Int = 1,
    val validFrom: String,
    val validTo: String? = null,
    val ingestedAt: String,
    val revisedAt: String? = null,
    val revisionReason: String? = null,
    val evidence: String = "{}",
    val payload: String = "{}",
    val syncState: String = "pending"
)

@Entity(tableName = "five_minute_summary")
data class FiveMinuteSummaryEntity(
    @PrimaryKey val startTime: String,
    val endTime: String,
    val summary: String,
    val syncState: String = "pending"
)

@Entity(tableName = "hourly_summary")
data class HourlySummaryEntity(
    @PrimaryKey val startTime: String,
    val endTime: String,
    val summary: String,
    val syncState: String = "pending"
)

@Entity(tableName = "daily_summary")
data class DailySummaryEntity(
    @PrimaryKey val date: String,
    val timezone: String,
    val summary: String,
    val syncState: String = "pending"
)

@Entity(tableName = "weekly_summary")
data class WeeklySummaryEntity(
    @PrimaryKey val week: String,
    val timezone: String,
    val summary: String,
    val syncState: String = "pending"
)

@Entity(tableName = "agent_observations")
data class AgentObservationEntity(
    @PrimaryKey val observationId: String,
    val timestamp: String,
    val agentId: String,
    val observationType: String,
    val severity: String,
    val payload: String,
    val delivered: Boolean = false,
    val createdAt: String = "datetime('now')"
)

@Entity(tableName = "interventions")
data class InterventionEntity(
    @PrimaryKey val interventionId: String,
    val timestamp: String,
    val agentId: String,
    val interventionType: String,
    val status: String = "proposed",
    val payload: String,
    val syncState: String = "pending"
)

@Entity(tableName = "user_feedback")
data class UserFeedbackEntity(
    @PrimaryKey val feedbackId: String,
    val timestamp: String,
    val targetType: String,
    val targetId: String,
    val feedbackType: String,
    val payload: String,
    val syncState: String = "pending"
)

@Entity(tableName = "goals")
data class GoalEntity(
    @PrimaryKey val goalId: String,
    val updatedAt: String,
    val goalType: String,
    val status: String,
    val payload: String,
    val syncState: String = "pending"
)

@Entity(tableName = "habits")
data class HabitEntity(
    @PrimaryKey val habitId: String,
    val habitType: String,
    val target: String,
    val schedule: String,
    val evidenceSources: String,
    val status: String = "active",
    val streakDays: Int = 0,
    val habitStrength: Double? = null,
    val createdAt: String = "datetime('now')",
    val updatedAt: String = "datetime('now')",
    val syncState: String = "pending"
)

@Entity(tableName = "baselines")
data class BaselineEntity(
    @PrimaryKey val baselineId: String,
    val metricName: String,
    val windowDays: Int = 7,
    val value: Double,
    val computedAt: String,
    val syncState: String = "pending"
)

@Entity(tableName = "app_classifications")
data class AppClassificationEntity(
    @PrimaryKey val packageName: String,
    val category: String,
    val subcategory: String? = null,
    val alignment: String? = null,
    val source: String,
    val updatedAt: String,
    val syncState: String = "pending"
)

@Entity(tableName = "permission_state")
data class PermissionStateEntity(
    @PrimaryKey val id: Int = 1,
    val updatedAt: String,
    val state: String,
    val syncState: String = "pending"
)

@Entity(tableName = "outcomes")
data class OutcomeEntity(
    @PrimaryKey val outcomeId: String,
    val interventionId: String,
    val timestamp: String,
    val outcomeType: String,
    val effectivenessScore: Double? = null,
    val payload: String,
    val syncState: String = "pending"
)

// ============================================================
// DAOs
// ============================================================

@Dao
interface EventDao {
    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insert(event: EventEntity): Long

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insertAll(events: List<EventEntity>): List<Long>

    @Query("SELECT * FROM events WHERE syncState = 'pending' ORDER BY sequence ASC")
    suspend fun getPendingEvents(): List<EventEntity>

    @Query("UPDATE events SET syncState = 'uploaded' WHERE eventId IN (:ids)")
    suspend fun markUploaded(ids: List<String>)

    @Query("SELECT * FROM events WHERE timestamp >= :from AND timestamp <= :to ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getEventsInRange(from: String, to: String, limit: Int = 200): List<EventEntity>

    @Query("DELETE FROM events WHERE createdAt < datetime('now', '-14 days')")
    suspend fun pruneOldEvents()

    @Query("SELECT MAX(sequence) FROM events WHERE deviceId = :deviceId")
    suspend fun getLastSequence(deviceId: String): Long?

    @Query("SELECT * FROM events WHERE eventType = :eventType ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getByType(eventType: String, limit: Int = 50): List<EventEntity>
}

@Dao
interface ContextDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(context: CurrentContextEntity)

    @Query("SELECT * FROM current_context WHERE id = 1")
    suspend fun get(): CurrentContextEntity?

    @Query("SELECT * FROM current_context WHERE id = 1")
    fun observe(): Flow<CurrentContextEntity?>
}

@Dao
interface SegmentDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(segment: TimelineSegmentEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertAll(segments: List<TimelineSegmentEntity>)

    @Query("SELECT * FROM timeline_segments WHERE startTime >= :from AND startTime <= :to ORDER BY startTime ASC LIMIT :limit")
    suspend fun getSegmentsInRange(from: String, to: String, limit: Int = 200): List<TimelineSegmentEntity>

    @Query("SELECT * FROM timeline_segments WHERE startTime >= :from AND startTime <= :to AND segmentType = :type ORDER BY startTime ASC")
    suspend fun getSegmentsByType(from: String, to: String, type: String): List<TimelineSegmentEntity>

    @Query("SELECT * FROM timeline_segments WHERE segmentId = :segmentId")
    suspend fun getById(segmentId: String): TimelineSegmentEntity?

    @Query("SELECT * FROM timeline_segments WHERE syncState = 'pending' ORDER BY startTime ASC")
    suspend fun getPendingSegments(): List<TimelineSegmentEntity>

    @Query("UPDATE timeline_segments SET syncState = 'uploaded' WHERE segmentId IN (:ids)")
    suspend fun markUploaded(ids: List<String>)
}

@Dao
interface FiveMinuteSummaryDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(summary: FiveMinuteSummaryEntity)

    @Query("SELECT * FROM five_minute_summary WHERE startTime = :startTime")
    suspend fun get(startTime: String): FiveMinuteSummaryEntity?

    @Query("SELECT * FROM five_minute_summary WHERE startTime >= :from AND startTime <= :to ORDER BY startTime ASC")
    suspend fun getInRange(from: String, to: String): List<FiveMinuteSummaryEntity>

    @Query("SELECT * FROM five_minute_summary ORDER BY startTime DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 100): List<FiveMinuteSummaryEntity>
}

@Dao
interface HourlySummaryDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(summary: HourlySummaryEntity)

    @Query("SELECT * FROM hourly_summary WHERE startTime = :startTime")
    suspend fun get(startTime: String): HourlySummaryEntity?

    @Query("SELECT * FROM hourly_summary WHERE startTime >= :from AND startTime <= :to ORDER BY startTime ASC")
    suspend fun getInRange(from: String, to: String): List<HourlySummaryEntity>

    @Query("SELECT * FROM hourly_summary ORDER BY startTime DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 48): List<HourlySummaryEntity>
}

@Dao
interface DailySummaryDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(summary: DailySummaryEntity)

    @Query("SELECT * FROM daily_summary WHERE date = :date")
    suspend fun get(date: String): DailySummaryEntity?

    @Query("SELECT * FROM daily_summary WHERE date = :date")

    @Query("SELECT * FROM daily_summary ORDER BY date DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 7): List<DailySummaryEntity>
}

@Dao
interface WeeklySummaryDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(summary: WeeklySummaryEntity)

    @Query("SELECT * FROM weekly_summary WHERE week = :week")
    suspend fun get(week: String): WeeklySummaryEntity?

    @Query("SELECT * FROM weekly_summary ORDER BY week DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 12): List<WeeklySummaryEntity>
}

@Dao
interface ObservationDao {
    @Insert
    suspend fun insert(observation: AgentObservationEntity)

    @Query("SELECT * FROM agent_observations ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 50): List<AgentObservationEntity>

    @Query("SELECT * FROM agent_observations WHERE delivered = 0")
    suspend fun getUndelivered(): List<AgentObservationEntity>
}

@Dao
interface InterventionDao {
    @Insert
    suspend fun insert(intervention: InterventionEntity)

    @Query("SELECT * FROM interventions WHERE status = 'approved_by_policy' OR status = 'delivered' ORDER BY createdAt DESC LIMIT :limit")
    suspend fun getPending(limit: Int = 10): List<InterventionEntity>

    @Query("SELECT * FROM interventions ORDER BY createdAt DESC LIMIT :limit")
    suspend fun getAll(limit: Int = 50): List<InterventionEntity>

    @Query("UPDATE interventions SET status = :status WHERE interventionId = :id")
    suspend fun updateStatus(id: String, status: String)
}

@Dao
interface FeedbackDao {
    @Insert
    suspend fun insert(feedback: UserFeedbackEntity)

    @Query("SELECT * FROM user_feedback WHERE targetType = :targetType AND targetId = :targetId")
    suspend fun getForTarget(targetType: String, targetId: String): List<UserFeedbackEntity>
}

@Dao
interface GoalDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(goal: GoalEntity)

    @Query("SELECT * FROM goals WHERE status = 'active'")
    suspend fun getActive(): List<GoalEntity>

    @Query("UPDATE goals SET status = :status, updatedAt = datetime('now') WHERE goalId = :id")
    suspend fun updateStatus(id: String, status: String)
}

@Dao
interface HabitDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(habit: HabitEntity)

    @Query("SELECT * FROM habits")
    suspend fun getAll(): List<HabitEntity>

    @Query("SELECT * FROM habits WHERE status = 'active'")
    suspend fun getActive(): List<HabitEntity>

    @Query("UPDATE habits SET streakDays = :streak, updatedAt = datetime('now') WHERE habitId = :id")
    suspend fun updateStreak(id: String, streak: Int)
}

@Dao
interface AppClassificationDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(classification: AppClassificationEntity)

    @Query("SELECT * FROM app_classifications WHERE packageName = :packageName")
    suspend fun get(packageName: String): AppClassificationEntity?

    @Query("SELECT * FROM app_classifications")
    suspend fun getAll(): List<AppClassificationEntity>
}

@Dao
interface BaselineDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(baseline: BaselineEntity)

    @Query("SELECT * FROM baselines WHERE metricName = :metricName LIMIT 1")
    suspend fun getByName(metricName: String): BaselineEntity?

    @Query("SELECT * FROM baselines")
    suspend fun getAll(): List<BaselineEntity>
}

@Dao
interface OutcomeDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(outcome: OutcomeEntity)

    @Query("SELECT * FROM outcomes WHERE interventionId = :interventionId")
    suspend fun getByIntervention(interventionId: String): List<OutcomeEntity>

    @Query("SELECT * FROM outcomes ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getRecent(limit: Int = 50): List<OutcomeEntity>
}

@Dao
interface PermissionStateDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(state: PermissionStateEntity)

    @Query("SELECT * FROM permission_state WHERE id = 1")
    suspend fun get(): PermissionStateEntity?
}

// ============================================================
// Database
// ============================================================

@Database(
    entities = [
        EventEntity::class,
        CurrentContextEntity::class,
        TimelineSegmentEntity::class,
        FiveMinuteSummaryEntity::class,
        HourlySummaryEntity::class,
        DailySummaryEntity::class,
        WeeklySummaryEntity::class,
        AgentObservationEntity::class,
        InterventionEntity::class,
        UserFeedbackEntity::class,
        GoalEntity::class,
        HabitEntity::class,
        BaselineEntity::class,
        AppClassificationEntity::class,
        PermissionStateEntity::class,
        OutcomeEntity::class,
    ],
    version = 2,
    exportSchema = false
)
@TypeConverters(Converters::class)
abstract class AndrometryDatabase : RoomDatabase() {
    abstract fun eventDao(): EventDao
    abstract fun contextDao(): ContextDao
    abstract fun segmentDao(): SegmentDao
    abstract fun fiveMinuteSummaryDao(): FiveMinuteSummaryDao
    abstract fun hourlySummaryDao(): HourlySummaryDao
    abstract fun dailySummaryDao(): DailySummaryDao
    abstract fun weeklySummaryDao(): WeeklySummaryDao
    abstract fun baselineDao(): BaselineDao
    abstract fun outcomeDao(): OutcomeDao
    abstract fun observationDao(): ObservationDao
    abstract fun interventionDao(): InterventionDao
    abstract fun feedbackDao(): FeedbackDao
    abstract fun goalDao(): GoalDao
    abstract fun habitDao(): HabitDao
    abstract fun appClassificationDao(): AppClassificationDao
    abstract fun permissionStateDao(): PermissionStateDao

    companion object {
        const val DB_NAME = "andrometry.db"

        fun create(context: Context): AndrometryDatabase {
            return Room.databaseBuilder(
                context.applicationContext,
                AndrometryDatabase::class.java,
                DB_NAME
            )
            .fallbackToDestructiveMigration()
            .build()
        }
    }
}

class Converters {
    // Add type converters if needed for complex types
}
