package com.example.seniorproject

import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.viewmodel.compose.viewModel
import com.example.seniorproject.data.api.TokenManager
import com.example.seniorproject.data.api.RetrofitClient
import androidx.compose.foundation.Image
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.example.seniorproject.data.model.EventDTO
import com.example.seniorproject.data.model.InitiatePaymentResponseDTO
import com.example.seniorproject.data.model.TicketDTO
import com.example.seniorproject.ui.components.AnalyticsDialog
import com.example.seniorproject.ui.components.AppNotification
import com.example.seniorproject.ui.components.NotificationType
import com.example.seniorproject.ui.theme.*
import kotlinx.coroutines.delay
import com.example.seniorproject.ui.viewmodel.AnalyticsViewModel
import com.example.seniorproject.ui.viewmodel.AuthState
import com.example.seniorproject.ui.viewmodel.AuthViewModel
import com.example.seniorproject.ui.viewmodel.EventsState
import com.example.seniorproject.ui.viewmodel.EventsViewModel
import com.example.seniorproject.ui.viewmodel.TicketingState
import com.example.seniorproject.ui.viewmodel.TicketingViewModel
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter

class EventsActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val tokenManager = TokenManager(this)
        
        setContent {
            SeniorProjectTheme(darkTheme = true, dynamicColor = false) {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = DarkBackground
                ) {
                    MainScreen(tokenManager)
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(tokenManager: TokenManager) {
    val context = LocalContext.current
    val eventsViewModel: EventsViewModel = viewModel(key = "all_events")
    val ticketingViewModel: TicketingViewModel = viewModel()
    val authViewModel: AuthViewModel = viewModel()
    val analyticsViewModel: AnalyticsViewModel = viewModel()
    val myEventsViewModel: EventsViewModel = viewModel(key = "my_events")
    val eventsState = eventsViewModel.eventsState
    val myEventsState = myEventsViewModel.eventsState
    val ticketingState = ticketingViewModel.ticketingState
    val authState = authViewModel.state
    val userRole = remember { tokenManager.getUserRole() ?: "student" }
    var selectedTab by remember { mutableStateOf(0) }

    var showNotification by remember { mutableStateOf(false) }
    var notificationMessage by remember { mutableStateOf("") }
    var notificationType by remember { mutableStateOf(NotificationType.INFO) }

    val isStaff = userRole == "organizer" || userRole == "admin"

    LaunchedEffect(Unit) {
        eventsViewModel.fetchEvents(moderationStatus = "approved")
        if (isStaff) {
            myEventsViewModel.fetchEvents(isMine = true)
        }
    }

    LaunchedEffect(ticketingState) {
        when (ticketingState) {
            is TicketingState.Success -> {
                notificationMessage = "Ticket successfully registered!"
                notificationType = NotificationType.SUCCESS
                showNotification = true
                selectedTab = if (isStaff) 2 else 1
                ticketingViewModel.resetStates()
            }
            is TicketingState.Error -> {
                notificationMessage = ticketingState.message
                notificationType = NotificationType.ERROR
                showNotification = true
                // Removed resetStates() to prevent infinite refresh loop in MyTicketsScreenContent
            }
            else -> {}
        }
    }

    LaunchedEffect(authState) {
        if (authState is AuthState.RequestSuccess) {
            notificationMessage = "Request submitted! An admin will review it."
            notificationType = NotificationType.SUCCESS
            showNotification = true
            authViewModel.resetState()
        } else if (authState is AuthState.Error) {
            notificationMessage = authState.message
            notificationType = NotificationType.ERROR
            showNotification = true
            authViewModel.resetState()
        }
    }

    LaunchedEffect(selectedTab) {
        if (selectedTab == 1 && isStaff) {
            myEventsViewModel.fetchEvents(isMine = true)
        }
        val ticketsTabIndex = if (isStaff) 2 else 1
        if (selectedTab == ticketsTabIndex) {
            ticketingViewModel.fetchMyTickets()
        }
    }

    Scaffold(
        topBar = {
            CenterAlignedTopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Surface(
                            modifier = Modifier.size(32.dp),
                            shape = RoundedCornerShape(8.dp),
                            color = GoldPrimary
                        ) {
                            Box(contentAlignment = Alignment.Center) {
                                Text("NU", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.ExtraBold, color = Color.Black)
                            }
                        }
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = when (selectedTab) {
                                0 -> "Discover Events"
                                1 -> if (isStaff) "My Events" else "My Tickets"
                                2 -> if (isStaff) "My Tickets" else "Profile"
                                3 -> "Profile"
                                else -> "Profile"
                            },
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = Color.White
                        )
                    }
                },
                actions = {
                    if (userRole == "admin") {
                        IconButton(onClick = { context.startActivity(Intent(context, AdminActivity::class.java)) }) {
                            Icon(Icons.Default.AdminPanelSettings, contentDescription = "Admin Panel", tint = GoldPrimary)
                        }
                    }
                },
                colors = TopAppBarDefaults.centerAlignedTopAppBarColors(
                    containerColor = DarkBackground,
                    titleContentColor = Color.White
                )
            )
        },
        bottomBar = {
            NavigationBar(
                containerColor = SurfaceDark,
                tonalElevation = 8.dp
            ) {
                NavigationBarItem(
                    icon = { Icon(Icons.Default.Dashboard, contentDescription = null) },
                    label = { Text("Explore") },
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = GoldPrimary,
                        selectedTextColor = GoldPrimary,
                        unselectedIconColor = Color.Gray,
                        unselectedTextColor = Color.Gray,
                        indicatorColor = GoldPrimary.copy(alpha = 0.1f)
                    )
                )
                if (userRole == "organizer" || userRole == "admin") {
                    NavigationBarItem(
                        icon = { Icon(Icons.Default.Event, contentDescription = null) },
                        label = { Text("My Events") },
                        selected = selectedTab == 1,
                        onClick = { selectedTab = 1 },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = GoldPrimary,
                            selectedTextColor = GoldPrimary,
                            unselectedIconColor = Color.Gray,
                            unselectedTextColor = Color.Gray,
                            indicatorColor = GoldPrimary.copy(alpha = 0.1f)
                        )
                    )
                }
                NavigationBarItem(
                    icon = { Icon(Icons.Default.ConfirmationNumber, contentDescription = null) },
                    label = { Text("Tickets") },
                    selected = if (isStaff) selectedTab == 2 else selectedTab == 1,
                    onClick = { selectedTab = if (isStaff) 2 else 1 },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = GoldPrimary,
                        selectedTextColor = GoldPrimary,
                        unselectedIconColor = Color.Gray,
                        unselectedTextColor = Color.Gray,
                        indicatorColor = GoldPrimary.copy(alpha = 0.1f)
                    )
                )
                NavigationBarItem(
                    icon = { Icon(Icons.Default.Person, contentDescription = null) },
                    label = { Text("Account") },
                    selected = if (isStaff) selectedTab == 3 else selectedTab == 2,
                    onClick = { selectedTab = if (isStaff) 3 else 2 },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = GoldPrimary,
                        selectedTextColor = GoldPrimary,
                        unselectedIconColor = Color.Gray,
                        unselectedTextColor = Color.Gray,
                        indicatorColor = GoldPrimary.copy(alpha = 0.1f)
                    )
                )
            }
        },
        floatingActionButton = {
            if (selectedTab == 0 && (userRole == "organizer" || userRole == "admin")) {
                LargeFloatingActionButton(
                    onClick = { context.startActivity(Intent(context, CreateEventActivity::class.java)) },
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                    shape = RoundedCornerShape(16.dp)
                ) {
                    Icon(Icons.Default.Add, contentDescription = "Create Event")
                }
            }
        }
    ) { paddingValues ->
        Box(modifier = Modifier.padding(paddingValues)) {
            when (selectedTab) {
                0 -> EventsScreenContent(
                    state = eventsState,
                    isBooking = ticketingState is TicketingState.Loading,
                    onRetry = { eventsViewModel.fetchEvents(moderationStatus = "approved") },
                    onBookTicket = { eventId -> 
                        ticketingViewModel.registerTicket(eventId) 
                    },
                    tokenManager = tokenManager,
                    analyticsViewModel = analyticsViewModel,
                    showStatus = false,
                    isMyEvents = false
                )
                1 -> if (isStaff) {
                    EventsScreenContent(
                        state = myEventsState,
                        isBooking = false,
                        onRetry = { 
                            myEventsViewModel.fetchEvents(isMine = true)
                        },
                        onBookTicket = { },
                        tokenManager = tokenManager,
                        analyticsViewModel = analyticsViewModel,
                        showStatus = true,
                        isMyEvents = true
                    )
                } else {
                    MyTicketsScreenContent()
                }
                2 -> if (isStaff) {
                    MyTicketsScreenContent()
                } else {
                    ProfileScreen(tokenManager, authViewModel, context)
                }
                3 -> if (isStaff) {
                    ProfileScreen(tokenManager, authViewModel, context)
                }
            }

            AppNotification(
                message = notificationMessage,
                type = notificationType,
                visible = showNotification,
                onDismiss = { showNotification = false }
            )
        }
    }
}

@Composable
fun PaymentSimulationDialog(
    response: InitiatePaymentResponseDTO,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit
) {
    Dialog(onDismissRequest = onDismiss) {
        Surface(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            shape = RoundedCornerShape(28.dp),
            color = SurfaceDark,
            border = BorderStroke(1.dp, GoldPrimary.copy(alpha = 0.2f))
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Icon(
                    Icons.Default.Payments,
                    contentDescription = null,
                    modifier = Modifier.size(64.dp),
                    tint = GoldPrimary
                )
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    "Secure Payment",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = Color.White
                )
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    "You are being redirected to our secure payment provider to complete the transaction.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.Gray,
                    textAlign = TextAlign.Center
                )
                Spacer(modifier = Modifier.height(24.dp))
                
                Surface(
                    color = Color.Black.copy(alpha = 0.3f),
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(horizontalArrangement = Arrangement.SpaceBetween, modifier = Modifier.fillMaxWidth()) {
                            Text("Payment ID", color = Color.Gray, style = MaterialTheme.typography.labelSmall)
                            Text(response.paymentId.take(8), color = Color.White, style = MaterialTheme.typography.labelSmall)
                        }
                        Spacer(modifier = Modifier.height(4.dp))
                        Row(horizontalArrangement = Arrangement.SpaceBetween, modifier = Modifier.fillMaxWidth()) {
                            Text("Amount", color = Color.Gray, style = MaterialTheme.typography.labelSmall)
                            Text("1,000.00 KZT", color = GoldPrimary, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
                
                Spacer(modifier = Modifier.height(24.dp))
                
                Button(
                    onClick = onConfirm,
                    modifier = Modifier.fillMaxWidth().height(50.dp),
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = GoldPrimary, contentColor = Color.Black)
                ) {
                    Text("Pay Now", fontWeight = FontWeight.Bold)
                }
                
                TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                    Text("Cancel", color = Color.Gray)
                }
            }
        }
    }
}

@Composable
fun EventsScreenContent(
    state: EventsState,
    isBooking: Boolean,
    onRetry: () -> Unit,
    onBookTicket: (String) -> Unit,
    tokenManager: TokenManager,
    analyticsViewModel: AnalyticsViewModel,
    showStatus: Boolean = false,
    isMyEvents: Boolean = false
) {
    val context = LocalContext.current
    var selectedEvent by remember { mutableStateOf<EventDTO?>(null) }
    var selectedEventForAnalytics by remember { mutableStateOf<String?>(null) }

    if (selectedEventForAnalytics != null) {
        AnalyticsDialog(
            eventId = selectedEventForAnalytics!!,
            viewModel = analyticsViewModel,
            onDismiss = { selectedEventForAnalytics = null }
        )
    }

    Column(modifier = Modifier.fillMaxSize()) {
        if (isBooking) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        }

        when (state) {
            is EventsState.Loading -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            is EventsState.Error -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(24.dp)) {
                        Icon(Icons.Default.ErrorOutline, contentDescription = null, modifier = Modifier.size(64.dp), tint = MaterialTheme.colorScheme.error)
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(text = state.message, color = MaterialTheme.colorScheme.error, textAlign = androidx.compose.ui.text.style.TextAlign.Center)
                        Spacer(modifier = Modifier.height(16.dp))
                        Button(onClick = onRetry) { Text("Try Again") }
                    }
                }
            }
            is EventsState.Success -> {
                LazyVerticalGrid(
                    columns = GridCells.Fixed(2),
                    contentPadding = PaddingValues(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize()
                ) {
                    items(state.events) { event ->
                        EventCard(event = event, showStatus = showStatus, onClick = { selectedEvent = event })
                    }
                }
            }
            else -> {}
        }
    }

    selectedEvent?.let { event ->
        EventDetailsDialog(
            event = event,
            onDismiss = { selectedEvent = null },
            onBookTicket = {
                onBookTicket(event.id)
                selectedEvent = null
            },
            currentUserId = tokenManager.getUserId(),
            onViewAnalytics = {
                selectedEventForAnalytics = event.id
                selectedEvent = null
            },
            isMyEvents = isMyEvents
        )
    }
}

@Composable
fun EventCard(event: EventDTO, showStatus: Boolean = false, onClick: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = SurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.05f)),
        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp)
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(120.dp),
                contentAlignment = Alignment.Center
            ) {
                if (!event.coverImageURL.isNullOrBlank()) {
                    AsyncImage(
                        model = ImageRequest.Builder(LocalContext.current)
                            .data(RetrofitClient.getEventImageUrl(event.coverImageURL))
                            .crossfade(true)
                            .build(),
                        contentDescription = "Event Cover",
                        modifier = Modifier.fillMaxSize(),
                        contentScale = androidx.compose.ui.layout.ContentScale.Crop
                    )
                } else {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .background(
                                Brush.verticalGradient(
                                    colors = listOf(GoldPrimary.copy(alpha = 0.15f), Color.Transparent)
                                )
                            ),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            imageVector = Icons.Default.EventAvailable,
                            contentDescription = null,
                            modifier = Modifier.size(48.dp),
                            tint = GoldPrimary
                        )
                    }
                }
            }
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = event.title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.ExtraBold,
                    color = Color.White,
                    maxLines = 1
                )
                Spacer(modifier = Modifier.height(8.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.CalendarToday, contentDescription = null, modifier = Modifier.size(14.dp), tint = Color.Gray)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        text = event.startsAt.take(10) + (if (event.location != null) " • ${event.location}" else ""),
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.Gray,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
                if (showStatus) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        val statusColor = when (event.moderationStatus.lowercase()) {
                            "approved" -> Color(0xFF4CAF50)
                            "pending" -> Color(0xFFFF9800)
                            "rejected" -> Color(0xFFF44336)
                            else -> Color.Gray
                        }
                        Box(
                            modifier = Modifier
                                .size(8.dp)
                                .clip(CircleShape)
                                .background(statusColor)
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = event.moderationStatus.uppercase(),
                            style = MaterialTheme.typography.labelSmall,
                            color = statusColor,
                            fontWeight = FontWeight.Bold
                        )
                    }
                }
                Spacer(modifier = Modifier.height(12.dp))
                Surface(
                    color = GoldPrimary.copy(alpha = 0.1f),
                    shape = RoundedCornerShape(8.dp)
                ) {
                    Text(
                        text = "${event.capacityAvailable} tickets left",
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        style = MaterialTheme.typography.labelSmall,
                        color = GoldPrimary,
                        fontWeight = FontWeight.Bold
                    )
                }
            }
        }
    }
}

@Composable
fun MyTicketsScreenContent() {
    val ticketingViewModel: TicketingViewModel = viewModel()
    val state = ticketingViewModel.ticketingState
    var ticketToCancel by remember { mutableStateOf<TicketDTO?>(null) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(DarkBackground)
    ) {
        Text(
            text = "Your Active Tickets",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            color = Color.White,
            modifier = Modifier.padding(horizontal = 24.dp, vertical = 16.dp)
        )

        when (state) {
            is TicketingState.Loading -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = GoldPrimary)
                }
            }
            is TicketingState.MyTicketsSuccess -> {
                if (state.tickets.isEmpty()) {
                    EmptyTicketsState()
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(16.dp)
                    ) {
                        items(state.tickets) { ticket ->
                            TicketItem(
                                ticket = ticket,
                                onCancel = { ticketToCancel = ticket }
                            )
                        }
                    }
                }
            }
            is TicketingState.Error -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Default.ErrorOutline, contentDescription = null, tint = Color.Red, modifier = Modifier.size(48.dp))
                        Text(state.message, color = Color.White, textAlign = TextAlign.Center)
                        Button(onClick = { ticketingViewModel.fetchMyTickets() }) { Text("Retry") }
                    }
                }
            }
            else -> {
                // Initial load
                LaunchedEffect(Unit) {
                    ticketingViewModel.fetchMyTickets()
                }
            }
        }
    }

    ticketToCancel?.let { ticket ->
        AlertDialog(
            onDismissRequest = { ticketToCancel = null },
            title = { Text("Cancel Ticket") },
            text = { Text("Are you sure you want to cancel your ticket for '${ticket.eventTitle}'? This action cannot be undone.") },
            confirmButton = {
                Button(
                    onClick = {
                        ticketingViewModel.cancelTicket(ticket.id)
                        ticketToCancel = null
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = Color.Red)
                ) {
                    Text("Confirm Cancel")
                }
            },
            dismissButton = {
                TextButton(onClick = { ticketToCancel = null }) {
                    Text("No, Keep it")
                }
            },
            containerColor = SurfaceDark,
            titleContentColor = Color.White,
            textContentColor = Color.Gray
        )
    }
}

@Composable
fun TicketItem(ticket: TicketDTO, onCancel: () -> Unit) {
    var showQR by remember { mutableStateOf(false) }

    Surface(
        modifier = Modifier
            .fillMaxWidth(),
        shape = RoundedCornerShape(24.dp),
        color = SurfaceDark,
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.05f))
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Event Icon/Image
                Box(
                    modifier = Modifier
                        .size(64.dp)
                        .clip(RoundedCornerShape(16.dp))
                        .background(GoldPrimary.copy(alpha = 0.1f)),
                    contentAlignment = Alignment.Center
                ) {
                    val imageUrl = ticket.event?.coverImageURL ?: ""
                    if (imageUrl.isNotBlank()) {
                        AsyncImage(
                            model = ImageRequest.Builder(LocalContext.current)
                                .data(RetrofitClient.getEventImageUrl(imageUrl))
                                .crossfade(true)
                                .build(),
                            contentDescription = "Event Cover",
                            modifier = Modifier.fillMaxSize(),
                            contentScale = androidx.compose.ui.layout.ContentScale.Crop
                        )
                    } else {
                        Icon(
                            imageVector = Icons.Default.ConfirmationNumber,
                            contentDescription = null,
                            tint = GoldPrimary,
                            modifier = Modifier.size(32.dp)
                        )
                    }
                }

                Spacer(modifier = Modifier.width(16.dp))

                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = ticket.eventTitle ?: "Event Ticket",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.ExtraBold,
                        color = Color.White,
                        maxLines = 1
                    )
                    
                    if (ticket.eventDate != null) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                Icons.Default.CalendarToday, 
                                contentDescription = null, 
                                modifier = Modifier.size(12.dp), 
                                tint = Color.Gray
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text(
                                text = ticket.eventDate.take(10),
                                style = MaterialTheme.typography.bodySmall,
                                color = Color.Gray
                            )
                        }
                    } else {
                        Text(
                            text = "ID: ${ticket.id.take(8)}",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color.Gray
                        )
                    }

                    Spacer(modifier = Modifier.height(4.dp))
                    
                    Surface(
                        color = when(ticket.status) {
                            "used" -> Color.Gray.copy(alpha = 0.1f)
                            "cancelled" -> Color.Red.copy(alpha = 0.1f)
                            else -> Color(0xFF4CAF50).copy(alpha = 0.1f)
                        },
                        shape = RoundedCornerShape(4.dp)
                    ) {
                        Text(
                            text = ticket.status.uppercase(),
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                            style = MaterialTheme.typography.labelSmall,
                            color = when(ticket.status) {
                                "used" -> Color.Gray
                                "cancelled" -> Color.Red
                                else -> Color(0xFF4CAF50)
                            },
                            fontWeight = FontWeight.Bold
                        )
                    }
                }

                IconButton(onClick = { if (ticket.status == "active") showQR = true }) {
                    Icon(
                        Icons.Default.QrCode,
                        contentDescription = "Show QR",
                        tint = if (ticket.status == "active") GoldPrimary else Color.Gray,
                        modifier = Modifier.size(32.dp)
                    )
                }
            }

            if (ticket.status == "active") {
                Spacer(modifier = Modifier.height(12.dp))
                HorizontalDivider(color = Color.White.copy(alpha = 0.05f))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End
                ) {
                    TextButton(
                        onClick = onCancel,
                        colors = ButtonDefaults.textButtonColors(contentColor = Color.Red.copy(alpha = 0.7f))
                    ) {
                        Icon(Icons.Default.Cancel, contentDescription = null, modifier = Modifier.size(16.dp))
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Cancel Ticket", style = MaterialTheme.typography.labelLarge)
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = { showQR = true },
                        colors = ButtonDefaults.buttonColors(containerColor = GoldPrimary, contentColor = Color.Black),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.height(36.dp)
                    ) {
                        Text("View QR", fontWeight = FontWeight.Bold)
                    }
                }
            }
        }
    }

    if (showQR) {
        TicketQRDialog(ticket = ticket, onDismiss = { showQR = false })
    }
}

@Composable
fun TicketQRDialog(ticket: TicketDTO, onDismiss: () -> Unit) {
    Dialog(onDismissRequest = onDismiss) {
        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            shape = RoundedCornerShape(28.dp),
            color = SurfaceDark
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Text(
                    "Your Entry Ticket",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = Color.White
                )
                
                Spacer(modifier = Modifier.height(24.dp))

                val qrBitmap = remember(ticket.qrHashHex) {
                    generateQRCode(ticket.qrHashHex)
                }

                if (qrBitmap != null) {
                    Surface(
                        modifier = Modifier
                            .size(240.dp)
                            .padding(8.dp),
                        shape = RoundedCornerShape(16.dp),
                        color = Color.White
                    ) {
                        Image(
                            bitmap = qrBitmap.asImageBitmap(),
                            contentDescription = "Ticket QR Code",
                            modifier = Modifier.fillMaxSize()
                        )
                    }
                }

                Spacer(modifier = Modifier.height(24.dp))

                Text(
                    "Show this QR code at the entrance to check-in for the event.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.Gray,
                    textAlign = TextAlign.Center
                )

                Spacer(modifier = Modifier.height(32.dp))

                Button(
                    onClick = onDismiss,
                    modifier = Modifier.fillMaxWidth().height(50.dp),
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = GoldPrimary, contentColor = Color.Black)
                ) {
                    Text("Done", fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
fun EmptyTicketsState() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(32.dp)) {
            Icon(
                Icons.Default.ConfirmationNumber,
                contentDescription = null,
                modifier = Modifier.size(80.dp),
                tint = Color.White.copy(alpha = 0.1f)
            )
            Spacer(modifier = Modifier.height(24.dp))
            Text(
                "No Tickets Yet",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = Color.White
            )
            Text(
                "Your registered tickets will appear here once you book an event.",
                style = MaterialTheme.typography.bodyMedium,
                color = Color.Gray,
                textAlign = TextAlign.Center
            )
        }
    }
}

@Composable
fun ProfileScreen(
    tokenManager: TokenManager,
    authViewModel: AuthViewModel,
    context: android.content.Context
) {
    val userRole = tokenManager.getUserRole() ?: "student"
    val userId = tokenManager.getUserId() ?: "Unknown"
    val isLoading = authViewModel.state is AuthState.Loading

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Surface(
            modifier = Modifier.size(100.dp),
            shape = RoundedCornerShape(32.dp),
            color = MaterialTheme.colorScheme.primaryContainer
        ) {
            Icon(
                Icons.Default.Person, 
                contentDescription = null, 
                modifier = Modifier.padding(24.dp).fillMaxSize(),
                tint = MaterialTheme.colorScheme.primary
            )
        }
        
        Spacer(modifier = Modifier.height(24.dp))
        
        Text(
            text = "My Account", 
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )
        
        Card(
            modifier = Modifier.fillMaxWidth().padding(top = 24.dp),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f))
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Role", style = MaterialTheme.typography.labelMedium, color = Color.Gray)
                Text(userRole.replaceFirstChar { it.uppercase() }, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Bold)
                
                HorizontalDivider(modifier = Modifier.padding(vertical = 12.dp), color = Color.LightGray.copy(alpha = 0.5f))
                
                Text("User ID", style = MaterialTheme.typography.labelMedium, color = Color.Gray)
                SelectionContainer {
                    Text(userId, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                }
            }
        }
        
        Spacer(modifier = Modifier.height(32.dp))

        if (userRole == "student") {
            Button(
                onClick = { authViewModel.requestOrganizerRole() },
                modifier = Modifier.fillMaxWidth().height(56.dp),
                shape = RoundedCornerShape(12.dp),
                enabled = !isLoading,
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary)
            ) {
                if (isLoading) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), color = Color.White, strokeWidth = 2.dp)
                } else {
                    Icon(Icons.Default.VerifiedUser, contentDescription = null)
                    Spacer(modifier = Modifier.width(12.dp))
                    Text("Become Organizer")
                }
            }
            Text(
                "Request organizer role to create your own events.",
                style = MaterialTheme.typography.labelSmall,
                color = Color.Gray,
                modifier = Modifier.padding(top = 8.dp)
            )
            Spacer(modifier = Modifier.height(16.dp))
        }

        if (userRole == "admin" || userRole == "organizer") {
            Button(
                onClick = { context.startActivity(Intent(context, CheckInActivity::class.java)) },
                modifier = Modifier.fillMaxWidth().height(56.dp),
                shape = RoundedCornerShape(12.dp)
            ) {
                Icon(Icons.Default.QrCodeScanner, contentDescription = null)
                Spacer(modifier = Modifier.width(12.dp))
                Text("Ticket Check-in")
            }
            Spacer(modifier = Modifier.height(16.dp))
        }
        
        OutlinedButton(
            onClick = {
                tokenManager.clearTokens()
                context.startActivity(Intent(context, LoginActivity::class.java))
                (context as ComponentActivity).finish()
            },
            modifier = Modifier.fillMaxWidth().height(56.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error)
        ) {
            Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = null)
            Spacer(modifier = Modifier.width(12.dp))
            Text("Logout")
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EventDetailsDialog(
    event: EventDTO,
    onDismiss: () -> Unit,
    onBookTicket: () -> Unit,
    currentUserId: String? = null,
    onViewAnalytics: () -> Unit = {},
    isMyEvents: Boolean = false
) {
    val isOrganizer = (event.organizerId != null && currentUserId != null && event.organizerId == currentUserId) || isMyEvents

    Dialog(onDismissRequest = onDismiss) {
        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            shape = RoundedCornerShape(28.dp),
            color = SurfaceDark,
            border = BorderStroke(1.dp, Color.White.copy(alpha = 0.1f))
        ) {
            Column(
                modifier = Modifier
                    .padding(24.dp)
                    .verticalScroll(rememberScrollState())
            ) {
                if (!event.coverImageURL.isNullOrBlank()) {
                    AsyncImage(
                        model = ImageRequest.Builder(LocalContext.current)
                            .data(RetrofitClient.getEventImageUrl(event.coverImageURL))
                            .crossfade(true)
                            .build(),
                        contentDescription = "Event Cover",
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(180.dp)
                            .clip(RoundedCornerShape(16.dp)),
                        contentScale = androidx.compose.ui.layout.ContentScale.Crop
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                }

                Text(
                    event.title,
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.ExtraBold,
                    color = Color.White
                )
                
                Spacer(modifier = Modifier.height(12.dp))
                
                SelectionContainer {
                    Text(
                        "Event ID: ${event.id}",
                        style = MaterialTheme.typography.labelSmall,
                        color = GoldPrimary
                    )
                }

                Spacer(modifier = Modifier.height(8.dp))

                val statusColor = when (event.moderationStatus.lowercase()) {
                    "approved" -> Color(0xFF4CAF50)
                    "pending" -> Color(0xFFFF9800)
                    "rejected" -> Color(0xFFF44336)
                    else -> Color.Gray
                }
                Surface(
                    color = statusColor.copy(alpha = 0.1f),
                    shape = RoundedCornerShape(8.dp),
                    border = BorderStroke(1.dp, statusColor.copy(alpha = 0.2f))
                ) {
                    Text(
                        text = "Status: ${event.moderationStatus.uppercase()}",
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        style = MaterialTheme.typography.labelSmall,
                        color = statusColor,
                        fontWeight = FontWeight.Bold
                    )
                }
                
                Spacer(modifier = Modifier.height(24.dp))
                
                Text(
                    "Description",
                    style = MaterialTheme.typography.labelLarge,
                    color = GoldPrimary,
                    fontWeight = FontWeight.Bold
                )
                Text(
                    event.description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.White.copy(alpha = 0.8f),
                    lineHeight = 22.sp
                )
                
                Spacer(modifier = Modifier.height(24.dp))
                
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.Schedule, contentDescription = null, modifier = Modifier.size(20.dp), tint = GoldPrimary)
                    Spacer(modifier = Modifier.width(12.dp))
                    Text(
                        event.startsAt.take(16).replace("T", " ") + (event.endAt?.let { " - " + it.take(16).replace("T", " ") } ?: ""),
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White
                    )
                }
                
                if (event.location != null) {
                    Spacer(modifier = Modifier.height(12.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.LocationOn, contentDescription = null, modifier = Modifier.size(20.dp), tint = GoldPrimary)
                        Spacer(modifier = Modifier.width(12.dp))
                        Text(
                            event.location,
                            style = MaterialTheme.typography.bodyMedium,
                            color = Color.White
                        )
                    }
                }
                
                Spacer(modifier = Modifier.height(12.dp))
                
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.Groups, contentDescription = null, modifier = Modifier.size(20.dp), tint = GoldPrimary)
                    Spacer(modifier = Modifier.width(12.dp))
                    Text(
                        "Capacity: ${event.capacityTotal} (Available: ${event.capacityAvailable})",
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White
                    )
                }
                
                Spacer(modifier = Modifier.height(40.dp))
                
                if (isOrganizer) {
                    Button(
                        onClick = onViewAnalytics,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(56.dp),
                        shape = RoundedCornerShape(12.dp),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = GoldPrimary,
                            contentColor = Color.Black
                        )
                    ) {
                        Icon(Icons.Default.BarChart, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("View Analytics", fontWeight = FontWeight.Bold)
                    }
                    Spacer(modifier = Modifier.height(16.dp))
                }

                Button(
                    onClick = onBookTicket,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(56.dp),
                    shape = RoundedCornerShape(12.dp),
                    enabled = event.capacityAvailable > 0 && !isOrganizer,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = GoldPrimary,
                        contentColor = Color.Black,
                        disabledContainerColor = Color.Gray
                    )
                ) {
                    Text(
                        when {
                            isOrganizer -> "Organizing this"
                            event.capacityAvailable > 0 -> "Get Ticket Now"
                            else -> "Sold Out"
                        },
                        fontWeight = FontWeight.Bold,
                        fontSize = 16.sp
                    )
                }

                TextButton(
                    onClick = onDismiss,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 8.dp)
                ) {
                    Text("Maybe later", color = Color.Gray)
                }
            }
        }
    }
}

private fun generateQRCode(content: String): Bitmap? {
    return try {
        val writer = QRCodeWriter()
        val bitMatrix = writer.encode(content, BarcodeFormat.QR_CODE, 512, 512)
        val width = bitMatrix.width
        val height = bitMatrix.height
        val bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.RGB_565)
        for (x in 0 until width) {
            for (y in 0 until height) {
                bitmap.setPixel(x, y, if (bitMatrix.get(x, y)) android.graphics.Color.BLACK else android.graphics.Color.WHITE)
            }
        }
        bitmap
    } catch (e: Exception) {
        null
    }
}
