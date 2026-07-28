package com.andrometry.inference.classifier

import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.pm.ApplicationInfo
import com.andrometry.core.database.AppClassificationEntity
import com.andrometry.core.database.AndrometryDatabase
import javax.inject.Inject
import javax.inject.Singleton

/**
 * App classification engine that categorizes apps into semantic categories.
 *
 * Classification sources (priority order):
 * 1. User overrides
 * 2. Known package allowlist/denylist
 * 3. Android app category
 * 4. Heuristic label matching
 *
 * Categories:
 * productivity, communication, calendar, email, browser, social,
 * entertainment, news, finance, health, travel, shopping, maps, system, unknown
 */
@Singleton
class AppCategoryEngine @Inject constructor(
    private val database: AndrometryDatabase
) {
    data class AppClassification(
        val packageName: String,
        val category: String,
        val subcategory: String? = null,
        val alignment: String? = null, // aligned, neutral, misaligned
        val source: String // user_override, android_category, heuristic
    )

    // Default category mappings for well-known apps
    private val knownCategories = mapOf(
        // Productivity
        "com.notion.id" to AppClassification("com.notion.id", "productivity", "notes", "aligned", "known"),
        "com.todoist" to AppClassification("com.todoist", "productivity", "tasks", "aligned", "known"),
        "com.asana" to AppClassification("com.asana", "productivity", "tasks", "aligned", "known"),
        "com.trello" to AppClassification("com.trello", "productivity", "kanban", "aligned", "known"),
        "com.google.android.apps.docs" to AppClassification("com.google.android.apps.docs", "productivity", "documents", "aligned", "known"),
        "com.google.android.apps.docs.editors.sheets" to AppClassification("com.google.android.apps.docs.editors.sheets", "productivity", "spreadsheets", "aligned", "known"),
        "com.microsoft.office.word" to AppClassification("com.microsoft.office.word", "productivity", "documents", "aligned", "known"),
        "com.microsoft.office.excel" to AppClassification("com.microsoft.office.excel", "productivity", "spreadsheets", "aligned", "known"),
        "md.obsidian" to AppClassification("md.obsidian", "productivity", "notes", "aligned", "known"),
        "com.intellij.android" to AppClassification("com.intellij.android", "productivity", "development", "aligned", "known"),
        "com.termux" to AppClassification("com.termux", "productivity", "terminal", "aligned", "known"),

        // Communication
        "com.slack" to AppClassification("com.slack", "communication", "chat", "aligned", "known"),
        "com.microsoft.teams" to AppClassification("com.microsoft.teams", "communication", "meetings", "aligned", "known"),
        "com.discord" to AppClassification("com.discord", "communication", "chat", "neutral", "known"),
        "org.telegram.messenger" to AppClassification("org.telegram.messenger", "communication", "messaging", "neutral", "known"),
        "com.whatsapp" to AppClassification("com.whatsapp", "communication", "messaging", "neutral", "known"),
        "com.google.android.gm" to AppClassification("com.google.android.gm", "communication", "email", "aligned", "known"),
        "com.microsoft.office.outlook" to AppClassification("com.microsoft.office.outlook", "communication", "email", "aligned", "known"),
        "us.zoom.videomeetings" to AppClassification("us.zoom.videomeetings", "communication", "meetings", "aligned", "known"),
        "com.google.android.apps.meetings" to AppClassification("com.google.android.apps.meetings", "communication", "meetings", "aligned", "known"),

        // Browser
        "com.android.chrome" to AppClassification("com.android.chrome", "browser", "web", "neutral", "known"),
        "org.mozilla.firefox" to AppClassification("org.mozilla.firefox", "browser", "web", "neutral", "known"),
        "com.brave.browser" to AppClassification("com.brave.browser", "browser", "web", "neutral", "known"),
        "com.opera.browser" to AppClassification("com.opera.browser", "browser", "web", "neutral", "known"),

        // Social
        "com.twitter.android" to AppClassification("com.twitter.android", "social", "microblogging", "misaligned", "known"),
        "com.instagram.android" to AppClassification("com.instagram.android", "social", "photos", "misaligned", "known"),
        "com.facebook.katana" to AppClassification("com.facebook.katana", "social", "feed", "misaligned", "known"),
        "com.reddit.frontpage" to AppClassification("com.reddit.frontpage", "social", "forum", "misaligned", "known"),
        "com.zhiliaoapp.musically" to AppClassification("com.zhiliaoapp.musically", "social", "short_video", "misaligned", "known"),
        "com.linkedin.android" to AppClassification("com.linkedin.android", "social", "professional", "neutral", "known"),

        // Entertainment
        "com.google.android.youtube" to AppClassification("com.google.android.youtube", "entertainment", "video", "misaligned", "known"),
        "com.netflix.mediaclient" to AppClassification("com.netflix.mediaclient", "entertainment", "streaming", "misaligned", "known"),
        "com.spotify.music" to AppClassification("com.spotify.music", "entertainment", "music", "neutral", "known"),
        "tv.twitch.android.app" to AppClassification("tv.twitch.android.app", "entertainment", "streaming", "misaligned", "known"),
        "com.amazon.avod.thirdpartyclient" to AppClassification("com.amazon.avod.thirdpartyclient", "entertainment", "streaming", "misaligned", "known"),

        // System
        "com.android.settings" to AppClassification("com.android.settings", "system", "settings", "neutral", "known"),
        "com.android.systemui" to AppClassification("com.android.systemui", "system", "ui", "neutral", "known"),
    )

    /**
     * Classifies a package, checking user overrides first, then known mappings, then Android category.
     */
    suspend fun classify(context: Context, packageName: String): AppClassification {
        // Check user overrides from database
        val userOverride = database.appClassificationDao().get(packageName)
        if (userOverride != null && userOverride.source == "user_override") {
            return AppClassification(
                packageName = packageName,
                category = userOverride.category,
                subcategory = userOverride.subcategory,
                alignment = userOverride.alignment,
                source = "user_override"
            )
        }

        // Check known mappings
        knownCategories[packageName]?.let { return it }

        // Check Android's app category (API 26+)
        try {
            val pm = context.packageManager
            val appInfo = pm.getApplicationInfo(packageName, 0)
            val androidCategory = getCategoryFromAndroid(appInfo.category)
            if (androidCategory != null) {
                return AppClassification(
                    packageName = packageName,
                    category = androidCategory,
                    subcategory = null,
                    alignment = if (androidCategory in listOf("social", "entertainment")) "misaligned" else "neutral",
                    source = "android_category"
                )
            }
        } catch (e: Exception) {
            // Package not found
        }

        // Fallback to unknown
        return AppClassification(
            packageName = packageName,
            category = "unknown",
            subcategory = null,
            alignment = "neutral",
            source = "heuristic"
        )
    }

    /**
     * Save a user override classification.
     */
    suspend fun setUserOverride(
        packageName: String,
        category: String,
        subcategory: String? = null,
        alignment: String? = null
    ) {
        val now = java.text.SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", java.util.Locale.US).apply {
            timeZone = java.util.TimeZone.getTimeZone("UTC")
        }.format(java.util.Date())

        database.appClassificationDao().upsert(
            AppClassificationEntity(
                packageName = packageName,
                category = category,
                subcategory = subcategory,
                alignment = alignment,
                source = "user_override",
                updatedAt = now
            )
        )
    }

    private fun getCategoryFromAndroid(category: Int): String? {
        // Raw int values to avoid SDK version resolution issues
        return when (category) {
            0 -> null // CATEGORY_UNDEFINED
            1 -> "game"
            2 -> "social"
            3 -> "news"
            4 -> "maps"
            5 -> "entertainment"
            6 -> "education"
            7 -> "shopping"
            8 -> "business"
            9 -> "productivity"
            10 -> "travel"
            11 -> "finance"
            12 -> "health"
            13 -> "tools"
            else -> null
        }
    }
}
