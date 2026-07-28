package com.andrometry.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Collector Status Screen — §6.4.2
 *
 * Shows the health status of all collectors, last update time,
 * error counts, and degraded state.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CollectorStatusScreen() {
    data class CollectorStatus(
        val name: String,
        val active: Boolean,
        val lastUpdateMs: Long,
        val errorCount: Int,
        val degraded: Boolean
    )

    var collectors by remember { mutableStateOf<List<CollectorStatus>>(emptyList()) }

    LaunchedEffect(Unit) {
        collectors = listOf(
            CollectorStatus("Screen", true, System.currentTimeMillis() - 60000, 0, false),
            CollectorStatus("Usage Stats", true, System.currentTimeMillis() - 300000, 0, false),
            CollectorStatus("Notifications", true, System.currentTimeMillis() - 120000, 1, false),
            CollectorStatus("Calendar", true, System.currentTimeMillis() - 900000, 0, false),
            CollectorStatus("Location", true, System.currentTimeMillis() - 600000, 0, false),
            CollectorStatus("Health Connect", false, 0, 3, true),
            CollectorStatus("Communication", true, System.currentTimeMillis() - 180000, 0, false),
            CollectorStatus("Device State", true, System.currentTimeMillis() - 30000, 0, false)
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Collector Status") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            item {
                Text(
                    "Collector Health",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(bottom = 16.dp)
                )
            }

            items(collectors) { collector ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(
                        containerColor = when {
                            collector.degraded -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.3f)
                            !collector.active -> MaterialTheme.colorScheme.surfaceVariant
                            else -> MaterialTheme.colorScheme.surfaceVariant
                        }
                    )
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = collector.name,
                                fontWeight = FontWeight.Medium,
                                modifier = Modifier.weight(1f)
                            )
                            StatusBadge(
                                active = collector.active,
                                degraded = collector.degraded
                            )
                        }
                        if (collector.errorCount > 0) {
                            Text(
                                text = "${collector.errorCount} errors",
                                color = MaterialTheme.colorScheme.error,
                                fontSize = 12.sp,
                                modifier = Modifier.padding(top = 4.dp)
                            )
                        }
                        if (collector.lastUpdateMs > 0) {
                            val minutesAgo = (System.currentTimeMillis() - collector.lastUpdateMs) / 60000
                            Text(
                                text = "Last update: ${minutesAgo}m ago",
                                fontSize = 12.sp,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(top = 2.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StatusBadge(active: Boolean, degraded: Boolean) {
    val (text, color) = when {
        degraded -> "Degraded" to MaterialTheme.colorScheme.error
        active -> "Active" to Color(0xFF4CAF50)
        else -> "Inactive" to MaterialTheme.colorScheme.onSurfaceVariant
    }
    Text(
        text = text,
        fontSize = 12.sp,
        color = color,
        fontWeight = FontWeight.Medium
    )
}
