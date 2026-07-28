package com.andrometry.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Sync Status Screen — §6.4.3
 *
 * Shows sync queue depth, last successful sync, pending events,
 * and sync failure count.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SyncStatusScreen() {
    var queueDepth by remember { mutableIntStateOf(12) }
    var lastSyncMinutesAgo by remember { mutableIntStateOf(7) }
    var syncFailureCount by remember { mutableIntStateOf(0) }
    var isSyncing by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Sync Status") },
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
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                "Sync Queue",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )

            // Queue depth
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Pending Events", fontWeight = FontWeight.Medium)
                    Text(
                        "$queueDepth",
                        fontSize = 32.sp,
                        fontWeight = FontWeight.Bold,
                        color = if (queueDepth > 0) MaterialTheme.colorScheme.primary
                               else MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        "Events waiting to be synced to relay server",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            // Last sync
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Last Successful Sync", fontWeight = FontWeight.Medium)
                    Text(
                        "${lastSyncMinutesAgo}m ago",
                        fontSize = 24.sp,
                        fontWeight = FontWeight.Bold,
                        color = if (lastSyncMinutesAgo < 15) Color(0xFF4CAF50)
                               else if (lastSyncMinutesAgo < 60) MaterialTheme.colorScheme.primary
                               else MaterialTheme.colorScheme.error
                    )
                }
            }

            // Sync failures
            if (syncFailureCount > 0) {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.errorContainer
                    )
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            "Sync Failures",
                            fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onErrorContainer
                        )
                        Text(
                            "$syncFailureCount consecutive failures",
                            color = MaterialTheme.colorScheme.onErrorContainer,
                            modifier = Modifier.padding(top = 4.dp)
                        )
                    }
                }
            }

            // Sync now button
            Button(
                onClick = { isSyncing = true },
                modifier = Modifier.fillMaxWidth(),
                enabled = !isSyncing && queueDepth > 0
            ) {
                Text(if (isSyncing) "Syncing..." else "Sync Now")
            }
        }
    }
}
