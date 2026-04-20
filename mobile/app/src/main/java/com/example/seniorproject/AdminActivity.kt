package com.example.seniorproject

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Event
import androidx.compose.material.icons.filled.RadioButtonChecked
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.ui.draw.clip
import com.example.seniorproject.data.model.UserDTO
import com.example.seniorproject.ui.components.AppNotification
import com.example.seniorproject.ui.components.NotificationType
import com.example.seniorproject.ui.theme.DarkBackground
import com.example.seniorproject.ui.theme.GoldPrimary
import com.example.seniorproject.ui.theme.SurfaceDark
import com.example.seniorproject.ui.theme.TextGray
import com.example.seniorproject.ui.theme.TextWhite
import com.example.seniorproject.ui.theme.SeniorProjectTheme
import androidx.compose.ui.text.style.TextOverflow
import com.example.seniorproject.ui.viewmodel.AdminState
import com.example.seniorproject.ui.viewmodel.AdminViewModel

class AdminActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            SeniorProjectTheme(darkTheme = true) {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = DarkBackground
                ) {
                    val viewModel: AdminViewModel = viewModel()
                    val state = viewModel.adminState
                    var showNotification by remember { mutableStateOf(false) }
                    var notificationMessage by remember { mutableStateOf("") }
                    var notificationType by remember { mutableStateOf(NotificationType.INFO) }

                    LaunchedEffect(state) {
                        when (state) {
                            is AdminState.Success -> {
                                notificationMessage = "Action successful"
                                notificationType = NotificationType.SUCCESS
                                showNotification = true
                                viewModel.resetState()
                            }
                            is AdminState.Error -> {
                                notificationMessage = state.message
                                notificationType = NotificationType.ERROR
                                showNotification = true
                                viewModel.resetState()
                            }
                            else -> {}
                        }
                    }

                    Box(modifier = Modifier.fillMaxSize()) {
                        AdminScreen(
                            state = state,
                            onModerate = { eventId, approve, reason ->
                                viewModel.moderateEvent(eventId, approve, reason) {
                                    // The screen will refresh due to LaunchedEffect(selectedStatus) 
                                    // if we just trigger a state change or if we call listAdminEvents explicitly.
                                    // Since we want to refresh the current view:
                                    viewModel.listAdminEvents(status = "pending") // default to pending or we can pass status
                                }
                            },
                            onSetRole = { userId, role ->
                                viewModel.setUserRole(userId, role)
                            },
                            onSearchUsers = { query ->
                                viewModel.listUsers(query)
                            },
                            onLoadEvents = { status ->
                                viewModel.listAdminEvents(status = status)
                            },
                            onBack = { finish() }
                        )

                        AppNotification(
                            message = notificationMessage,
                            type = notificationType,
                            visible = showNotification,
                            onDismiss = { showNotification = false }
                        )
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AdminScreen(
    state: AdminState,
    onModerate: (String, Boolean, String?) -> Unit,
    onSetRole: (String, String) -> Unit,
    onSearchUsers: (String) -> Unit,
    onLoadEvents: (String) -> Unit,
    onBack: () -> Unit
) {
    var eventId by remember { mutableStateOf("") }
    var userId by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("organizer") }
    var reason by remember { mutableStateOf("") }
    var userQuery by remember { mutableStateOf("") }
    var selectedStatus by remember { mutableStateOf("pending") }

    val isLoading = state is AdminState.Loading
    val users = if (state is AdminState.UsersLoaded) state.users else emptyList()
    val allEvents = if (state is AdminState.EventsLoaded) state.events else emptyList()

    LaunchedEffect(selectedStatus) {
        onLoadEvents(selectedStatus)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { 
                    Text(
                        "Admin Control Center", 
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.ExtraBold,
                        color = GoldPrimary
                    ) 
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack, 
                            contentDescription = "Back",
                            tint = TextWhite
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = DarkBackground
                )
            )
        },
        containerColor = DarkBackground
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            // Event Moderation Section
            Text(
                "Event Moderation",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = TextWhite,
                modifier = Modifier.padding(bottom = 4.dp)
            )

            // Status Selector
            val statuses = listOf("pending", "approved", "rejected")
            SingleChoiceSegmentedButtonRow(
                modifier = Modifier.fillMaxWidth(),
            ) {
                statuses.forEachIndexed { index, status ->
                    SegmentedButton(
                        shape = SegmentedButtonDefaults.itemShape(index = index, count = statuses.size),
                        onClick = { selectedStatus = status },
                        selected = selectedStatus == status,
                        colors = SegmentedButtonDefaults.colors(
                            activeContainerColor = GoldPrimary,
                            activeContentColor = Color.Black,
                            inactiveContainerColor = SurfaceDark,
                            inactiveContentColor = TextGray
                        )
                    ) {
                        Text(status.replaceFirstChar { it.uppercase() }, style = MaterialTheme.typography.labelSmall)
                    }
                }
            }

            if (isLoading && allEvents.isEmpty()) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally), color = GoldPrimary)
            } else if (allEvents.isEmpty()) {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = SurfaceDark)
                ) {
                    Box(modifier = Modifier.padding(32.dp).fillMaxWidth(), contentAlignment = Alignment.Center) {
                        Text("No $selectedStatus events", color = TextGray)
                    }
                }
            } else {
                allEvents.forEach { event ->
                    AdminEventCard(
                        event = event,
                        isSelected = eventId == event.id,
                        onClick = { eventId = event.id }
                    )
                }
            }

            if (eventId.isNotBlank()) {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = SurfaceDark),
                    border = BorderStroke(1.dp, GoldPrimary.copy(alpha = 0.5f))
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Moderation Action", style = MaterialTheme.typography.labelLarge, color = GoldPrimary)
                        Spacer(modifier = Modifier.height(12.dp))
                        OutlinedTextField(
                            value = reason,
                            onValueChange = { reason = it },
                            label = { Text("Moderation Reason (Optional)") },
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(12.dp),
                            colors = OutlinedTextFieldDefaults.colors(
                                focusedBorderColor = GoldPrimary,
                                unfocusedBorderColor = Color.Gray,
                                focusedLabelColor = GoldPrimary,
                                focusedTextColor = TextWhite,
                                unfocusedTextColor = TextWhite
                            )
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            Button(
                                onClick = { onModerate(eventId, true, reason) },
                                enabled = !isLoading,
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(8.dp),
                                colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF4CAF50))
                            ) { 
                                Text("Approve", fontWeight = FontWeight.Bold, color = Color.White) 
                            }
                            Button(
                                onClick = { onModerate(eventId, false, reason) },
                                enabled = !isLoading,
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(8.dp),
                                colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFF44336))
                            ) { 
                                Text("Reject", fontWeight = FontWeight.Bold, color = Color.White) 
                            }
                        }
                    }
                }
            }

            HorizontalDivider(color = Color.Gray.copy(alpha = 0.3f), modifier = Modifier.padding(vertical = 8.dp))

            // User Management Section
            Text(
                "User Management",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = TextWhite
            )

            OutlinedTextField(
                value = userQuery,
                onValueChange = { 
                    userQuery = it
                    if (it.length >= 2) onSearchUsers(it)
                },
                label = { Text("Search by Email") },
                placeholder = { Text("e.g. example@nu.edu.kz") },
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(12.dp),
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null, tint = GoldPrimary) },
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = GoldPrimary,
                    unfocusedBorderColor = Color.Gray,
                    focusedLabelColor = GoldPrimary,
                    focusedTextColor = TextWhite,
                    unfocusedTextColor = TextWhite
                )
            )

            if (users.isNotEmpty()) {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    users.forEach { user ->
                        UserListItem(
                            user = user,
                            isSelected = userId == user.id,
                            onSelect = { 
                                userId = user.id
                                // We don't change userQuery here so user can still see their search
                            }
                        )
                    }
                }
            }

            if (userId.isNotBlank()) {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = SurfaceDark),
                    border = BorderStroke(1.dp, GoldPrimary.copy(alpha = 0.5f))
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        val selectedUser = users.find { it.id == userId }
                        Text(
                            "Set Role for: ${selectedUser?.email ?: userId}",
                            style = MaterialTheme.typography.labelLarge,
                            color = GoldPrimary
                        )
                        Spacer(modifier = Modifier.height(12.dp))
                        
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            listOf("student", "organizer", "admin").forEach { r ->
                                FilterChip(
                                    selected = role == r,
                                    onClick = { role = r },
                                    label = { Text(r.replaceFirstChar { it.uppercase() }) },
                                    colors = FilterChipDefaults.filterChipColors(
                                        selectedContainerColor = GoldPrimary,
                                        selectedLabelColor = Color.Black,
                                        labelColor = TextGray
                                    )
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(12.dp))

                        Button(
                            onClick = { onSetRole(userId, role) },
                            enabled = !isLoading,
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = GoldPrimary)
                        ) {
                            Text("Update User Role", fontWeight = FontWeight.Bold, color = Color.Black)
                        }
                    }
                }
            }

            if (isLoading) {
                LinearProgressIndicator(
                    modifier = Modifier.fillMaxWidth(),
                    color = GoldPrimary,
                    trackColor = GoldPrimary.copy(alpha = 0.1f)
                )
            }
        }
    }
}

@Composable
fun AdminEventCard(
    event: com.example.seniorproject.data.model.AdminEventDTO,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (isSelected) SurfaceDark.copy(alpha = 0.8f) else SurfaceDark
        ),
        border = if (isSelected) BorderStroke(2.dp, GoldPrimary) else null,
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(60.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(GoldPrimary.copy(alpha = 0.1f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(Icons.Default.Event, contentDescription = null, tint = GoldPrimary, modifier = Modifier.size(32.dp))
            }
            
            Spacer(modifier = Modifier.width(16.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    event.title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = TextWhite,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    event.startsAt,
                    style = MaterialTheme.typography.bodySmall,
                    color = TextGray
                )
                Spacer(modifier = Modifier.height(4.dp))
                StatusBadge(event.moderationStatus)
            }
            
            if (isSelected) {
                Icon(Icons.Default.CheckCircle, contentDescription = null, tint = GoldPrimary)
            } else {
                Icon(Icons.AutoMirrored.Filled.ArrowForward, contentDescription = null, tint = TextGray, modifier = Modifier.size(20.dp))
            }
        }
    }
}

@Composable
fun StatusBadge(status: String) {
    val (color, bgColor) = when (status.lowercase()) {
        "approved" -> Color(0xFF4CAF50) to Color(0xFF4CAF50).copy(alpha = 0.1f)
        "rejected" -> Color(0xFFF44336) to Color(0xFFF44336).copy(alpha = 0.1f)
        else -> Color(0xFFFF9800) to Color(0xFFFF9800).copy(alpha = 0.1f)
    }

    Surface(
        color = bgColor,
        shape = RoundedCornerShape(8.dp),
    ) {
        Text(
            text = status.uppercase(),
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            color = color
        )
    }
}

@Composable
fun UserListItem(
    user: UserDTO,
    isSelected: Boolean,
    onSelect: () -> Unit
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onSelect() },
        color = if (isSelected) GoldPrimary.copy(alpha = 0.1f) else Color.Transparent,
        shape = RoundedCornerShape(12.dp),
        border = if (isSelected) BorderStroke(1.dp, GoldPrimary) else BorderStroke(1.dp, Color.Gray.copy(alpha = 0.2f))
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    user.email,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Bold,
                    color = TextWhite
                )
                Text(
                    "Current Role: ${user.role.uppercase()}",
                    style = MaterialTheme.typography.bodySmall,
                    color = TextGray
                )
            }
            if (isSelected) {
                Icon(Icons.Default.RadioButtonChecked, contentDescription = null, tint = GoldPrimary)
            } else {
                Icon(Icons.Default.RadioButtonUnchecked, contentDescription = null, tint = TextGray)
            }
        }
    }
}
