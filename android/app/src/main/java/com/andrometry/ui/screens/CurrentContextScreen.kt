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
 * Current Context Screen — §6.4.5
 *
 * Shows the live context snapshot that agents poll, with activity,
 * place, focus state, distraction probability, and data quality.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CurrentContextScreen() {
    var context by remember { mutableStateOf<Map<String, Any>>(emptyMap()) }

    LaunchedEffect(Unit) {
        context = mapOf(
            "primary_activity" to "deep_work",
            "activity_confidence" to 0.81,
            "place" to "office",
            "place_confidence" to 0.92,
            "movement" to "still",
            "foreground_app_category" to "productivity",
            "screen_state" to "on",
            "focus_active" to true,
            "focus_minutes_elapsed" to 37,
            "distraction_probability" to 0.18,
            "fatigue_probability" to 0.31,
            "alignment_probability" to 0.74,
            "data_completeness" to 0.87,
            "staleness_seconds" to 42
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Current Context") },
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
            // Activity card
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Activity", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(
                        context["primary_activity"] as String,
                        fontSize = 24.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary
                    )
                    Text(
                        "Confidence: ${(context["activity_confidence"] as Double * 100).toInt()}%",
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            // Location card
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Location", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(
                        context["place"] as String,
                        fontSize = 24.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary
                    )
                    Text(
                        "Confidence: ${(context["place_confidence"] as Double * 100).toInt()}%",
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            // Focus card
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Focus", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Active")
                        Text(
                            if (context["focus_active"] as Boolean) "YES" else "NO",
                            color = if (context["focus_active"] as Boolean) Color(0xFF4CAF50) else MaterialTheme.colorScheme.error,
                            fontWeight = FontWeight.Bold
                        )
                    }
                    Text("Elapsed: ${context["focus_minutes_elapsed"]} minutes")
                }
            }

            // Probabilities card
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Probabilities", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Spacer(modifier = Modifier.height(8.dp))
                    ProbabilityRow("Distraction", context["distraction_probability"] as Double)
                    ProbabilityRow("Fatigue", context["fatigue_probability"] as Double)
                    ProbabilityRow("Alignment", context["alignment_probability"] as Double)
                }
            }

            // Data quality card
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Data Quality", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Completeness")
                        Text("${(context["data_completeness"] as Double * 100).toInt()}%")
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Staleness")
                        Text("${context["staleness_seconds"]}s")
                    }
                }
            }
        }
    }
}

@Composable
private fun ProbabilityRow(label: String, value: Double) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(label)
        LinearProgressIndicator(
            progress = { value.toFloat() },
            modifier = Modifier.width(120.dp),
            color = when {
                value > 0.6 -> Color(0xFFF44336)
                value > 0.3 -> Color(0xFFFF9800)
                else -> Color(0xFF4CAF50)
            }
        )
        Text("${(value * 100).toInt()}%", fontSize = 14.sp)
    }
}
