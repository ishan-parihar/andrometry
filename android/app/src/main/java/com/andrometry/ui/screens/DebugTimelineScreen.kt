package com.andrometry.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Debug Timeline Screen — §6.4.4
 *
 * Shows the raw timeline segments for debugging and verification.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DebugTimelineScreen() {
    data class TimelineEntry(
        val segmentType: String,
        val label: String,
        val startTime: String,
        val endTime: String,
        val confidence: Double,
        val layer: String
    )

    var entries by remember { mutableStateOf<List<TimelineEntry>>(emptyList()) }

    LaunchedEffect(Unit) {
        entries = listOf(
            TimelineEntry("place_visit", "home", "07:12", "07:57", 0.95, "place"),
            TimelineEntry("commute_segment", "commute", "07:57", "08:22", 0.88, "activity"),
            TimelineEntry("place_visit", "office", "08:22", "12:02", 0.93, "place"),
            TimelineEntry("focus_segment", "deep_work", "08:22", "11:22", 0.85, "focus"),
            TimelineEntry("meeting_segment", "meeting", "11:22", "12:07", 0.88, "calendar"),
            TimelineEntry("break_segment", "break", "12:07", "12:47", 0.90, "activity"),
            TimelineEntry("focus_segment", "deep_work", "12:47", "15:17", 0.78, "focus"),
            TimelineEntry("distraction_segment", "social", "14:17", "14:25", 0.80, "activity")
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Debug Timeline") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                "Today's Timeline",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.padding(bottom = 8.dp)
            )

            entries.forEach { entry ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Text(entry.label, fontWeight = FontWeight.Bold, fontSize = 16.sp)
                            Text(
                                entry.layer,
                                fontSize = 12.sp,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }
                        Text(
                            "${entry.startTime} → ${entry.endTime}",
                            fontSize = 14.sp,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Text(
                            "${entry.segmentType} | confidence: ${(entry.confidence * 100).toInt()}%",
                            fontSize = 12.sp,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }
        }
    }
}
