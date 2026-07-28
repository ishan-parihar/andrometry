package com.andrometry.inference.habits

import com.andrometry.core.database.AndrometryDatabase
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Habit Engine — implements §38 Habit Engine.
 *
 * Tracks habit adherence, computes streaks, and calculates habit strength
 * based on consistency over time.
 *
 * §38.1 — Habit definition:
 * Each habit has:
 * - Type (morning_routine, exercise, reading, etc.)
 * - Target (what metric indicates completion)
 * - Schedule (days of week, time windows)
 * - Evidence sources (which collectors can verify completion)
 * - Status (active, paused, completed, abandoned)
 *
 * §38.2 — Habit features:
 * - Streak tracking (consecutive days completed)
 * - Habit strength (0-1, based on consistency over time)
 * - Completion tracking (daily check-ins)
 * - Exception rules (weekends, travel days)
 * - Schedule adherence
 *
 * §38.3 — Habit strength:
 * Calculated using exponential moving average of daily completion.
 * Strength increases with consistent completion and decays with misses.
 * Formula: strength_new = strength_old * decay + completion * (1 - decay)
 * Where decay = 0.85 (retains 85% of previous strength)
 */
@Singleton
class HabitEngine @Inject constructor(
    private val database: AndrometryDatabase
) {

    data class Habit(
        val habitId: String,
        val habitType: String,
        val title: String,
        val target: Map<String, Any>,
        val schedule: Map<String, Any>,
        val evidenceSources: List<String>,
        val status: String = "active",
        val streakDays: Int = 0,
        val habitStrength: Double = 0.0,
        val createdAt: String,
        val updatedAt: String
    )

    data class HabitCompletion(
        val habitId: String,
        val date: String,
        val completed: Boolean,
        val evidence: Map<String, Any>,
        val timestamp: String
    )

    data class HabitStats(
        val habitId: String,
        val habitType: String,
        val currentStreak: Int,
        val longestStreak: Int,
        val completionRate: Double,  // Last 30 days
        val habitStrength: Double,
        val totalCompletions: Int,
        val totalDays: Int,
        val lastCompletedDate: String?,
        val trend: String  // "improving", "stable", "declining"
    )

    // Exponential moving average decay factor (§38.3)
    private val STRENGTH_DECAY = 0.85

    /**
     * Records a habit completion for a given date.
     */
    suspend fun recordCompletion(
        habitId: String,
        date: String,
        completed: Boolean,
        evidence: Map<String, Any> = emptyMap()
    ): HabitCompletion {
        val now = getCurrentTimestamp()

        val completion = HabitCompletion(
            habitId = habitId,
            date = date,
            completed = completed,
            evidence = evidence,
            timestamp = now
        )

        // Store completion event
        val event = com.andrometry.core.database.EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = 0, // Will be assigned by sequence generator
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "habit_updated",
            collector = "habit_engine",
            source = "user_input",
            payload = Gson().toJson(completion)
        )
        database.eventDao().insert(event)

        // Update habit stats
        updateHabitStats(habitId, completed, date)

        return completion
    }

    /**
     * Updates habit statistics after a completion.
     */
    private suspend fun updateHabitStats(habitId: String, completed: Boolean, date: String) {
        // Get habit from database
        val habits = database.habitDao().getAll()
        val habit = habits.find { it.habitId == habitId } ?: return

        // Calculate new streak
        val newStreak = if (completed) {
            habit.streakDays + 1
        } else {
            0 // Reset streak on miss
        }

        // Calculate new habit strength using EMA (§38.3)
        val completionValue = if (completed) 1.0 else 0.0
        val newStrength = habit.habitStrength * STRENGTH_DECAY + completionValue * (1 - STRENGTH_DECAY)

        // Update habit in database
        val updatedHabit = habit.copy(
            streakDays = newStreak,
            habitStrength = newStrength,
            updatedAt = date
        )
        database.habitDao().upsert(updatedHabit)
    }

    /**
     * Gets comprehensive statistics for a habit.
     */
    suspend fun getHabitStats(habitId: String, daysToAnalyze: Int = 30): HabitStats? {
        val habits = database.habitDao().getAll()
        val habit = habits.find { it.habitId == habitId } ?: return null

        // Get completion events for this habit
        val events = database.eventDao().getByType("habit_updated", daysToAnalyze * 10)
        val completions = events.mapNotNull { event ->
            try {
                Gson().fromJson(event.payload, HabitCompletion::class.java)
            } catch (e: Exception) {
                null
            }
        }.filter { it.habitId == habitId }

        val completedDays = completions.filter { it.completed }.map { it.date }.toSet()
        val totalDays = daysToAnalyze
        val completionRate = if (totalDays > 0) completedDays.size.toDouble() / totalDays else 0.0

        // Calculate longest streak from history
        val sortedDates = completedDays.sorted()
        var longestStreak = 0
        var currentStreak = 0
        var lastDate: Date? = null

        val dateFormat = SimpleDateFormat("yyyy-MM-dd", Locale.US)
        for (dateStr in sortedDates) {
            val date = dateFormat.parse(dateStr) ?: continue
            if (lastDate != null) {
                val diffDays = ((date.time - lastDate.time) / (24 * 60 * 60 * 1000)).toInt()
                if (diffDays == 1) {
                    currentStreak++
                } else {
                    currentStreak = 1
                }
            } else {
                currentStreak = 1
            }
            longestStreak = maxOf(longestStreak, currentStreak)
            lastDate = date
        }

        // Determine trend (compare last 7 days vs previous 7 days)
        val recentCompletions = completions.filter {
            val compDate = dateFormat.parse(it.date) ?: return@filter false
            val daysDiff = ((Date().time - compDate.time) / (24 * 60 * 60 * 1000)).toInt()
            daysDiff <= 7 && it.completed
        }.size

        val previousCompletions = completions.filter {
            val compDate = dateFormat.parse(it.date) ?: return@filter false
            val daysDiff = ((Date().time - compDate.time) / (24 * 60 * 60 * 1000)).toInt()
            daysDiff in 8..14 && it.completed
        }.size

        val trend = when {
            recentCompletions > previousCompletions -> "improving"
            recentCompletions < previousCompletions -> "declining"
            else -> "stable"
        }

        return HabitStats(
            habitId = habitId,
            habitType = habit.habitType,
            currentStreak = habit.streakDays,
            longestStreak = longestStreak,
            completionRate = completionRate,
            habitStrength = habit.habitStrength,
            totalCompletions = completedDays.size,
            totalDays = totalDays,
            lastCompletedDate = completedDays.maxOrNull(),
            trend = trend
        )
    }

    /**
     * Checks if a habit should be completed today based on its schedule.
     */
    suspend fun isScheduledForToday(habitId: String): Boolean {
        val habits = database.habitDao().getAll()
        val habit = habits.find { it.habitId == habitId } ?: return false

        val today = SimpleDateFormat("EEEE", Locale.US).format(Date()).lowercase()
        val scheduleDays = habit.schedule["days"] as? List<*> ?: return false

        return today in scheduleDays
    }

    /**
     * Gets all habits with their current stats.
     */
    suspend fun getAllHabitStats(): List<HabitStats> {
        val habits = database.habitDao().getAll()
        return habits.mapNotNull { getHabitStats(it.habitId) }
    }

    private fun getCurrentTimestamp(): String {
        return SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())
    }
}
