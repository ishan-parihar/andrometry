package com.andrometry.ui.screens

import android.content.Context
import android.content.Intent
import android.widget.Toast
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import java.io.File
import java.text.SimpleDateFormat
import java.util.*

/**
 * Export/Backup Controls Screen — §6.4.8
 *
 * Provides JSON export, cloud backup, and restore functionality.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ExportBackupScreen() {
    val context = LocalContext.current
    var lastExport by remember { mutableStateOf("Never") }
    var backupEnabled by remember { mutableStateOf(false) }
    var exportFormat by remember { mutableStateOf("JSON") }
    var showRestoreDialog by remember { mutableStateOf(false) }
    var isExporting by remember { mutableStateOf(false) }

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
                onClick = {
                    isExporting = true
                    // Export data and generate file
                    val exportFile = exportDataToFile(context, exportFormat)
                    lastExport = SimpleDateFormat("MMM d, yyyy HH:mm", Locale.US).format(Date())
                    isExporting = false

                    if (exportFile != null) {
                        Toast.makeText(context, "Exported to ${exportFile.name}", Toast.LENGTH_SHORT).show()
                    } else {
                        Toast.makeText(context, "Export failed", Toast.LENGTH_SHORT).show()
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = !isExporting
            ) {
                Text(if (isExporting) "Exporting..." else "Export All Data")
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
                onClick = { showRestoreDialog = true },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Restore from Backup")
            }

            OutlinedButton(
                onClick = {
                    // Share data via Android share intent
                    val exportFile = exportDataToFile(context, "JSON")
                    if (exportFile != null) {
                        shareFile(context, exportFile)
                    } else {
                        Toast.makeText(context, "Export first before sharing", Toast.LENGTH_SHORT).show()
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Share Data")
            }
        }

        // Restore confirmation dialog
        if (showRestoreDialog) {
            AlertDialog(
                onDismissRequest = { showRestoreDialog = false },
                title = { Text("Restore from Backup") },
                text = { Text("This will restore your data from the latest cloud backup. Existing local data will be replaced. Continue?") },
                confirmButton = {
                    TextButton(
                        onClick = {
                            showRestoreDialog = false
                            Toast.makeText(context, "Cloud restore not yet implemented", Toast.LENGTH_LONG).show()
                        }
                    ) {
                        Text("Restore")
                    }
                },
                dismissButton = {
                    TextButton(onClick = { showRestoreDialog = false }) {
                        Text("Cancel")
                    }
                }
            )
        }
    }
}

/**
 * Exports data to a file in the specified format.
 */
private fun exportDataToFile(context: Context, format: String): File? {
    return try {
        val timestamp = SimpleDateFormat("yyyyMMdd_HHmmss", Locale.US).format(Date())
        val fileName = "andrometry_export_${timestamp}.${format.lowercase()}"
        val exportDir = File(context.filesDir, "exports")
        if (!exportDir.exists()) exportDir.mkdirs()

        val exportFile = File(exportDir, fileName)

        // Generate export content based on format
        val content = when (format) {
            "JSON" -> generateJsonExport()
            "CSV" -> generateCsvExport()
            else -> generateJsonExport()
        }

        exportFile.writeText(content)
        exportFile
    } catch (e: Exception) {
        null
    }
}

/**
 * Generates JSON export of all data.
 */
private fun generateJsonExport(): String {
    // Placeholder: In production, query all databases and serialize
    return """
    {
        "export_version": "1.0",
        "exported_at": "${SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).format(Date())}",
        "events": [],
        "segments": [],
        "summaries": {
            "five_minute": [],
            "hourly": [],
            "daily": [],
            "weekly": []
        },
        "habits": [],
        "baselines": [],
        "goals": []
    }
    """.trimIndent()
}

/**
 * Generates CSV export of key data.
 */
private fun generateCsvExport(): String {
    return "type,id,timestamp,data\n"
}

/**
 * Shares a file via Android share intent.
 */
private fun shareFile(context: Context, file: File) {
    val uri = FileProvider.getUriForFile(
        context,
        "${context.packageName}.fileprovider",
        file
    )

    val shareIntent = Intent(Intent.ACTION_SEND).apply {
        type = "application/octet-stream"
        putExtra(Intent.EXTRA_STREAM, uri)
        putExtra(Intent.EXTRA_SUBJECT, "Andrometry Data Export")
        putExtra(Intent.EXTRA_TEXT, "Here is my Andrometry data export.")
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }

    val chooserIntent = Intent.createChooser(shareIntent, "Share Exported Data")
    chooserIntent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    context.startActivity(chooserIntent)
}
