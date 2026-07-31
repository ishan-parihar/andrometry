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
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.InterventionEntity
import com.google.gson.Gson
import kotlinx.coroutines.launch

/**
 * Intervention Inbox Screen — §6.4.6
 *
 * Shows proposed interventions from AI agents for user review.
 * Fetches real interventions from the local database.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InterventionInboxScreen(database: AndrometryDatabase) {
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
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        val pending = database.interventionDao().getPending(20)
        interventions = pending.map { entity ->
            val payload = try {
                Gson().fromJson(entity.payload, Map::class.java) as? Map<String, Any> ?: emptyMap()
            } catch (e: Exception) {
                emptyMap()
            }

            Intervention(
                id = entity.interventionId,
                agentId = entity.agentId,
                type = entity.interventionType,
                title = (payload["title"] as? String) ?: entity.interventionType,
                message = (payload["message"] as? String) ?: "No message",
                severity = (payload["severity"] as? String) ?: "low",
                status = entity.status,
                timestamp = entity.createdAt.ifEmpty { entity.timestamp }
            )
        }
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
                            "From: ${intervention.agentId} · ${intervention.status}",
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
                                onClick = {
                                    scope.launch {
                                        database.interventionDao().updateStatus(intervention.id, "accepted")
                                        interventions = interventions.filter { it.id != intervention.id }
                                    }
                                },
                                modifier = Modifier.weight(1f)
                            ) { Text("Accept") }
                            OutlinedButton(
                                onClick = {
                                    scope.launch {
                                        database.interventionDao().updateStatus(intervention.id, "dismissed")
                                        interventions = interventions.filter { it.id != intervention.id }
                                    }
                                },
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
