package com.andrometry

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.andrometry.ui.screens.*
import com.andrometry.ui.theme.AndrometryTheme
import com.andrometry.core.database.AndrometryDatabase
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

/**
 * Main Activity — the app entry point.
 * Uses Jetpack Compose Navigation with a bottom navigation bar
 * to wire all §6.4 UI screens.
 */
@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    @Inject
    lateinit var database: AndrometryDatabase

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            AndrometryTheme {
                AndrometryApp(database)
            }
        }
    }
}

sealed class Screen(val route: String, val label: String, val icon: ImageVector) {
    object Context : Screen("context", "Context", Icons.Default.Info)
    object Collectors : Screen("collectors", "Collectors", Icons.Default.List)
    object Permissions : Screen("permissions", "Permissions", Icons.Default.Lock)
    object Sync : Screen("sync", "Sync", Icons.Default.CloudUpload)
    object Timeline : Screen("timeline", "Timeline", Icons.Default.Timeline)
    object Inbox : Screen("inbox", "Inbox", Icons.Default.Inbox)
    object Privacy : Screen("privacy", "Privacy", Icons.Default.Security)
    object Export : Screen("export", "Export", Icons.Default.FileDownload)
}

val bottomNavItems = listOf(
    Screen.Context,
    Screen.Collectors,
    Screen.Timeline,
    Screen.Inbox,
    Screen.Privacy
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AndrometryApp(database: AndrometryDatabase) {
    val navController = rememberNavController()
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = navBackStackEntry?.destination?.route

    Scaffold(
        topBar = {
            var menuExpanded by remember { mutableStateOf(false) }
            TopAppBar(
                title = { Text("Andrometry", fontWeight = FontWeight.Bold) },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                ),
                actions = {
                    Box {
                        IconButton(onClick = { menuExpanded = true }) {
                            Icon(Icons.Default.MoreVert, contentDescription = "More")
                        }
                        DropdownMenu(
                            expanded = menuExpanded,
                            onDismissRequest = { menuExpanded = false }
                        ) {
                            DropdownMenuItem(
                                text = { Text("Permissions") },
                                onClick = {
                                    menuExpanded = false
                                    navController.navigate(Screen.Permissions.route) {
                                        launchSingleTop = true
                                    }
                                },
                                leadingIcon = { Icon(Screen.Permissions.icon, null) }
                            )
                            DropdownMenuItem(
                                text = { Text("Sync") },
                                onClick = {
                                    menuExpanded = false
                                    navController.navigate(Screen.Sync.route) {
                                        launchSingleTop = true
                                    }
                                },
                                leadingIcon = { Icon(Screen.Sync.icon, null) }
                            )
                            DropdownMenuItem(
                                text = { Text("Export") },
                                onClick = {
                                    menuExpanded = false
                                    navController.navigate(Screen.Export.route) {
                                        launchSingleTop = true
                                    }
                                },
                                leadingIcon = { Icon(Screen.Export.icon, null) }
                            )
                        }
                    }
                }
            )
        },
        bottomBar = {
            NavigationBar {
                bottomNavItems.forEach { screen ->
                    NavigationBarItem(
                        icon = { Icon(screen.icon, contentDescription = screen.label) },
                        label = { Text(screen.label) },
                        selected = currentRoute == screen.route,
                        onClick = {
                            navController.navigate(screen.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    )
                }
            }
        }
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = Screen.Context.route,
            modifier = Modifier.padding(padding)
        ) {
            composable(Screen.Context.route) { CurrentContextScreen() }
            composable(Screen.Collectors.route) { CollectorStatusScreen() }
            composable(Screen.Permissions.route) { PermissionStatusScreen() }
            composable(Screen.Sync.route) { SyncStatusScreen() }
            composable(Screen.Timeline.route) { DebugTimelineScreen() }
            composable(Screen.Inbox.route) { InterventionInboxScreen() }
            composable(Screen.Privacy.route) { PrivacyScreen() }
            composable(Screen.Export.route) { ExportBackupScreen(database) }
        }
    }
}
