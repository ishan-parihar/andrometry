package com.andrometry.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.andrometry.core.config.SensitiveModeManager
import kotlinx.coroutines.launch

/**
 * Privacy Screen — §6.4.7
 *
 * Shows sensitive mode toggle, redaction settings, and data wipe controls.
 * Uses SensitiveModeManager via Hilt DI for persistent state.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PrivacyScreen(
    sensitiveModeManager: SensitiveModeManager
) {
    val scope = rememberCoroutineScope()
    val isSensitiveMode by sensitiveModeManager.isSensitiveModeActive.collectAsState(initial = false)
    val redactLoc by sensitiveModeManager.redactLocation.collectAsState(initial = false)
    val redactApp by sensitiveModeManager.redactAppUsage.collectAsState(initial = false)
    val redactCal by sensitiveModeManager.redactCalendar.collectAsState(initial = true)

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Privacy Settings") },
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
                "Privacy Controls",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )

            // Sensitive mode
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text("Sensitive Mode", fontWeight = FontWeight.Medium)
                            Text(
                                "Pauses all collection and sync. No data leaves the device.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                        Switch(
                            checked = isSensitiveMode,
                            onCheckedChange = { enabled ->
                                scope.launch {
                                    if (enabled) sensitiveModeManager.enableSensitiveMode()
                                    else sensitiveModeManager.disableSensitiveMode()
                                }
                            }
                        )
                    }
                }
            }

            // Redaction settings
            Text(
                "Data Redaction",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Medium,
                modifier = Modifier.padding(top = 8.dp)
            )

            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    RedactionToggle(
                        label = "Redact Location",
                        description = "Replace exact coordinates with semantic places",
                        checked = redactLoc,
                        onCheckedChange = { enabled ->
                            scope.launch { sensitiveModeManager.setRedactLocation(enabled) }
                        }
                    )
                    HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
                    RedactionToggle(
                        label = "Redact App Names",
                        description = "Show categories instead of specific app names",
                        checked = redactApp,
                        onCheckedChange = { enabled ->
                            scope.launch { sensitiveModeManager.setRedactAppUsage(enabled) }
                        }
                    )
                    HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
                    RedactionToggle(
                        label = "Redact Calendar Titles",
                        description = "Hide meeting titles in context and summaries",
                        checked = redactCal,
                        onCheckedChange = { enabled ->
                            scope.launch { sensitiveModeManager.setRedactCalendar(enabled) }
                        }
                    )
                }
            }

            // Data management
            Text(
                "Data Management",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Medium,
                modifier = Modifier.padding(top = 8.dp)
            )

            OutlinedButton(
                onClick = { /* Wipe local data */ },
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.error
                )
            ) {
                Text("Wipe All Local Data")
            }

            OutlinedButton(
                onClick = { /* Export data */ },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Export Data (JSON)")
            }
        }
    }
}

@Composable
private fun RedactionToggle(
    label: String,
    description: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(label, fontWeight = FontWeight.Medium)
            Text(
                description,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange
        )
    }
}
