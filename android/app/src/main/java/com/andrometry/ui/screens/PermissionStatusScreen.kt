package com.andrometry.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Error
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Permission Status Screen — §6.4.1
 *
 * Shows the current state of all collector permissions so users
 * can diagnose missing data sources.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PermissionStatusScreen() {
    var permissionState by remember { mutableStateOf<Map<String, Boolean>>(emptyMap()) }

    // Simulated permission states — in production, inject PermissionCollector
    LaunchedEffect(Unit) {
        permissionState = mapOf(
            "Usage Stats" to true,
            "Notification Listener" to true,
            "Calendar" to true,
            "Fine Location" to true,
            "Background Location" to false,
            "Health Connect" to true,
            "Health: Steps" to true,
            "Health: Sleep" to true,
            "Health: Heart Rate" to false,
            "Battery Optimization" to true
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Permission Status") },
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
                    "Collector Permissions",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(bottom = 8.dp)
                )
                Text(
                    "Missing permissions explain missing data in context snapshots.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(bottom = 16.dp)
                )
            }

            items(permissionState.entries.toList()) { (name, granted) ->
                PermissionItem(name = name, granted = granted)
            }

            item {
                Spacer(modifier = Modifier.height(16.dp))
                val grantedCount = permissionState.values.count { it }
                val totalCount = permissionState.size
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(
                        containerColor = if (grantedCount == totalCount)
                            MaterialTheme.colorScheme.primaryContainer
                        else
                            MaterialTheme.colorScheme.errorContainer
                    )
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            "Data Completeness: ${grantedCount}/${totalCount} permissions granted",
                            fontWeight = FontWeight.Medium
                        )
                        if (grantedCount < totalCount) {
                            Text(
                                "Some data sources will be unavailable. Grant missing permissions for full context.",
                                style = MaterialTheme.typography.bodySmall,
                                modifier = Modifier.padding(top = 4.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PermissionItem(name: String, granted: Boolean) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = if (granted) Icons.Default.CheckCircle else Icons.Default.Error,
                contentDescription = if (granted) "Granted" else "Denied",
                tint = if (granted) Color(0xFF4CAF50) else MaterialTheme.colorScheme.error,
                modifier = Modifier.size(24.dp)
            )
            Spacer(modifier = Modifier.width(12.dp))
            Text(
                text = name,
                modifier = Modifier.weight(1f),
                fontWeight = FontWeight.Medium
            )
            Text(
                text = if (granted) "Granted" else "Denied",
                color = if (granted) Color(0xFF4CAF50) else MaterialTheme.colorScheme.error,
                fontSize = 14.sp
            )
        }
    }
}
