package com.andrometry.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Export/Backup Controls Screen — §6.4.8
 *
 * Provides JSON export, cloud backup, and restore functionality.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ExportBackupScreen() {
    var lastExport by remember { mutableStateOf("Never") }
    var backupEnabled by remember { mutableStateOf(false) }
    var exportFormat by remember { mutableStateOf("JSON") }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Export & Backup") },
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
                "Export Data",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )

            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Export Format", fontWeight = FontWeight.Medium)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        FilterChip(
                            selected = exportFormat == "JSON",
                            onClick = { exportFormat = "JSON" },
                            label = { Text("JSON") }
                        )
                        FilterChip(
                            selected = exportFormat == "CSV",
                            onClick = { exportFormat = "CSV" },
                            label = { Text("CSV") }
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        "Last export: $lastExport",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            Button(
                onClick = { lastExport = "Just now" },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Export All Data")
            }

            HorizontalDivider()

            Text(
                "Cloud Backup",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )

            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text("Automatic Backup", fontWeight = FontWeight.Medium)
                            Text(
                                "Back up data to cloud storage daily",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                        Switch(
                            checked = backupEnabled,
                            onCheckedChange = { backupEnabled = it }
                        )
                    }
                }
            }

            OutlinedButton(
                onClick = { /* Restore from backup */ },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Restore from Backup")
            }

            OutlinedButton(
                onClick = { /* Share data */ },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Share Data")
            }
        }
    }
}
