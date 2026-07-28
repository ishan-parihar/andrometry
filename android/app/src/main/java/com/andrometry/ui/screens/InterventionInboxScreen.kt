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
 * Intervention Inbox Screen — §6.4.6
 *
 * Shows proposed interventions from AI agents for user review.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InterventionInboxScreen() {
    data class Intervention(
        val id: String,
        val agentId: String,
        val type: String,
        val title: String,
        val message: String,
        val severity: String,
        val status: String,
        val timestamp: String
    )

    var interventions by remember { mutableStateOf<List<Intervention>>(emptyList()) }

    LaunchedEffect(Unit) {
        interventions = listOf(
            Intervention(
                id = "int-1", agentId = "coach", type = "nudge",
                title = "Take a break", message = "You've been in deep work for 2 hours. A 10-minute break can restore focus.",
                severity = "low", status = "pending", timestamp = "14:22"
            ),
            Intervention(
                id = "int-2", agentId = "coach", type = "suggestion",
                title = "Review daily goals", message = "You completed 3 of 5 goals today. Consider prioritizing the remaining ones.",
                severity = "medium", status = "pending", timestamp = "15:00"
            )
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Intervention Inbox") },
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
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                "Proposed Interventions",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.padding(bottom = 8.dp)
            )

            if (interventions.isEmpty()) {
                Card(modifier = Modifier.fillMaxWidth()) {
                    Text(
                        "No pending interventions.",
                        modifier = Modifier.padding(16.dp),
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            interventions.forEach { intervention ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(intervention.title, fontWeight = FontWeight.Bold, fontSize = 16.sp)
                            SeverityBadge(intervention.severity)
                        }
                        Text(
                            intervention.message,
                            modifier = Modifier.padding(top = 8.dp),
                            style = MaterialTheme.typography.bodyMedium
                        )
                        Text(
                            "From: ${intervention.agentId} · ${intervention.timestamp}",
                            modifier = Modifier.padding(top = 8.dp),
                            fontSize = 12.sp,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 12.dp),
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Button(
                                onClick = { /* Accept */ },
                                modifier = Modifier.weight(1f)
                            ) { Text("Accept") }
                            OutlinedButton(
                                onClick = { /* Dismiss */ },
                                modifier = Modifier.weight(1f)
                            ) { Text("Dismiss") }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SeverityBadge(severity: String) {
    val color = when (severity) {
        "high" -> Color(0xFFF44336)
        "medium" -> Color(0xFFFF9800)
        else -> Color(0xFF4CAF50)
    }
    Text(
        text = severity.uppercase(),
        fontSize = 11.sp,
        fontWeight = FontWeight.Bold,
        color = color
    )
}
