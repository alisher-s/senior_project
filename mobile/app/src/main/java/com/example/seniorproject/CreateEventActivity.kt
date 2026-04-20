package com.example.seniorproject

import android.net.Uri
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarToday
import androidx.compose.material.icons.filled.CloudUpload
import androidx.compose.material.icons.filled.Image
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.selection.SelectionContainer
import coil.compose.rememberAsyncImagePainter
import java.text.SimpleDateFormat
import java.util.*
import androidx.lifecycle.viewmodel.compose.viewModel
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.EventDTO
import com.example.seniorproject.ui.components.AppNotification
import com.example.seniorproject.ui.components.NotificationType
import com.example.seniorproject.ui.theme.SeniorProjectTheme
import com.example.seniorproject.ui.viewmodel.CreateEventState
import com.example.seniorproject.ui.viewmodel.EventsViewModel

class CreateEventActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            SeniorProjectTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    val eventIdToEdit = intent.getStringExtra("EVENT_ID")
                    val isEditMode = eventIdToEdit != null

                    val viewModel: EventsViewModel = viewModel()
                    val state = viewModel.createEventState

                    var showSuccessDialog by remember { mutableStateOf<String?>(null) }
                    var showNotification by remember { mutableStateOf(false) }
                    var notificationMessage by remember { mutableStateOf("") }
                    var notificationType by remember { mutableStateOf(NotificationType.INFO) }

                    var eventToEdit by remember { mutableStateOf<com.example.seniorproject.data.model.EventDTO?>(null) }

                    LaunchedEffect(eventIdToEdit) {
                        if (eventIdToEdit != null) {
                            try {
                                val response = RetrofitClient.eventsService.getEvent(eventIdToEdit)
                                if (response.isSuccessful) {
                                    eventToEdit = response.body()
                                }
                            } catch (e: Exception) {
                                notificationMessage = "Failed to load event for editing"
                                notificationType = NotificationType.ERROR
                                showNotification = true
                            }
                        }
                    }

                    LaunchedEffect(state) {
                        if (state is CreateEventState.Success) {
                            showSuccessDialog = state.event.id
                            viewModel.resetCreateState()
                        } else if (state is CreateEventState.Error) {
                            notificationMessage = state.message
                            notificationType = NotificationType.ERROR
                            showNotification = true
                            viewModel.resetCreateState()
                        }
                    }

                    Box(modifier = Modifier.fillMaxSize()) {
                        if (showSuccessDialog != null) {
                            AlertDialog(
                                onDismissRequest = { 
                                    showSuccessDialog = null
                                    finish() 
                                },
                                title = { Text(if (isEditMode) "Event Updated" else "Event Created") },
                                text = { 
                                    Column {
                                        Text(if (isEditMode) "Your event has been updated successfully." else "Your event has been submitted for moderation. Please provide this ID to an admin for approval:")
                                        if (!isEditMode) {
                                            Spacer(modifier = Modifier.height(8.dp))
                                            androidx.compose.foundation.text.selection.SelectionContainer {
                                                Text(
                                                    text = showSuccessDialog!!,
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.primary
                                                )
                                            }
                                        }
                                    }
                                },
                                confirmButton = {
                                    Button(onClick = { 
                                        showSuccessDialog = null
                                        finish() 
                                    }) { Text("Done") }
                                }
                            )
                        }

                        if (isEditMode && eventToEdit == null) {
                            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                CircularProgressIndicator()
                            }
                        } else {
                            CreateEventScreen(
                                isLoading = state is CreateEventState.Loading,
                                initialEvent = eventToEdit,
                                onEventCreated = { title, description, startsAt, capacity, location, endAt, coverImageUri ->
                                    if (isEditMode) {
                                        viewModel.updateEvent(
                                            eventId = eventIdToEdit!!,
                                            title = title,
                                            description = description,
                                            startsAt = startsAt,
                                            capacity = capacity,
                                            location = location,
                                            endAt = endAt,
                                            coverImageUri = coverImageUri,
                                            existingCoverUrl = eventToEdit?.coverImageURL,
                                            context = this@CreateEventActivity,
                                            isMine = true
                                        )
                                    } else {
                                        viewModel.createEvent(
                                            title = title,
                                            description = description,
                                            startsAt = startsAt,
                                            capacity = capacity,
                                            location = location,
                                            endAt = endAt,
                                            coverImageUri = coverImageUri,
                                            context = this@CreateEventActivity,
                                            isMine = true
                                        )
                                    }
                                },
                                onBack = { finish() }
                            )
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
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateEventScreen(
    isLoading: Boolean,
    initialEvent: com.example.seniorproject.data.model.EventDTO? = null,
    onEventCreated: (String, String, String, Int, String?, String?, Uri?) -> Unit,
    onBack: () -> Unit
) {
    var title by remember { mutableStateOf(initialEvent?.title ?: "Tech Conference 2024") }
    var startsAt by remember { mutableStateOf(initialEvent?.startsAt ?: "") }
    var displayStartsAt by remember { 
        mutableStateOf(
            if (initialEvent != null) {
                try {
                    val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
                    sdf.timeZone = TimeZone.getTimeZone("UTC")
                    val date = sdf.parse(initialEvent.startsAt)
                    val displaySdf = SimpleDateFormat("MMM dd, yyyy HH:mm", Locale.getDefault())
                    displaySdf.format(date!!)
                } catch (e: Exception) { initialEvent.startsAt }
            } else ""
        ) 
    }
    var endAt by remember { mutableStateOf(initialEvent?.endAt ?: "") }
    var displayEndAt by remember { 
        mutableStateOf(
            if (initialEvent?.endAt != null) {
                try {
                    val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
                    sdf.timeZone = TimeZone.getTimeZone("UTC")
                    val date = sdf.parse(initialEvent.endAt!!)
                    val displaySdf = SimpleDateFormat("MMM dd, yyyy HH:mm", Locale.getDefault())
                    displaySdf.format(date!!)
                } catch (e: Exception) { initialEvent.endAt!! }
            } else ""
        ) 
    }
    var location by remember { mutableStateOf(initialEvent?.location ?: "Main Auditorium") }
    var description by remember { mutableStateOf(initialEvent?.description ?: "A grand gathering of tech enthusiasts to discuss the future of Android development.") }
    var capacity by remember { mutableStateOf(initialEvent?.capacityTotal?.toString() ?: "100") }
    var coverImageUri by remember { mutableStateOf<Uri?>(null) }
    
    val galleryLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent()
    ) { uri: Uri? ->
        coverImageUri = uri
    }
    
    val isEditMode = initialEvent != null
    
    // Date & Time Picker state
    val context = LocalContext.current
    val calendar = remember { Calendar.getInstance() }
    var showDatePicker by remember { mutableStateOf(false) }
    var showTimePicker by remember { mutableStateOf(false) }
    var pickingForStart by remember { mutableStateOf(true) }

    val datePickerState = rememberDatePickerState()
    val timePickerState = rememberTimePickerState(
        initialHour = calendar.get(Calendar.HOUR_OF_DAY),
        initialMinute = calendar.get(Calendar.MINUTE)
    )

    if (showDatePicker) {
        DatePickerDialog(
            onDismissRequest = { showDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    datePickerState.selectedDateMillis?.let {
                        calendar.timeInMillis = it
                        showDatePicker = false
                        showTimePicker = true
                    }
                }) { Text("Next") }
            },
            dismissButton = {
                TextButton(onClick = { showDatePicker = false }) { Text("Cancel") }
            }
        ) {
            DatePicker(state = datePickerState)
        }
    }

    if (showTimePicker) {
        AlertDialog(
            onDismissRequest = { showTimePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    calendar.set(Calendar.HOUR_OF_DAY, timePickerState.hour)
                    calendar.set(Calendar.MINUTE, timePickerState.minute)
                    
                    val displaySdf = SimpleDateFormat("MMM dd, yyyy HH:mm", Locale.getDefault())
                    val formattedDisplay = displaySdf.format(calendar.time)
                    
                    val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
                    sdf.timeZone = TimeZone.getTimeZone("UTC")
                    val formattedIso = sdf.format(calendar.time)

                    if (pickingForStart) {
                        displayStartsAt = formattedDisplay
                        startsAt = formattedIso
                    } else {
                        displayEndAt = formattedDisplay
                        endAt = formattedIso
                    }
                    
                    showTimePicker = false
                }) { Text("OK") }
            },
            dismissButton = {
                TextButton(onClick = { showTimePicker = false }) { Text("Cancel") }
            },
            text = {
                TimePicker(state = timePickerState)
            }
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
            .verticalScroll(rememberScrollState())
    ) {
        Text(
            text = if (isEditMode) "Edit Event" else "Create Event",
            style = MaterialTheme.typography.headlineLarge,
            modifier = Modifier.padding(bottom = 24.dp)
        )

        OutlinedTextField(
            value = title,
            onValueChange = { title = it },
            label = { Text("Event Title") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = location,
            onValueChange = { location = it },
            label = { Text("Location") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            placeholder = { Text("Where will the event take place?") }
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Event Cover Image",
            style = MaterialTheme.typography.titleSmall,
            modifier = Modifier.padding(bottom = 8.dp)
        )

        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(180.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(MaterialTheme.colorScheme.surfaceVariant)
                .border(
                    width = 1.dp,
                    color = MaterialTheme.colorScheme.outline,
                    shape = RoundedCornerShape(12.dp)
                )
                .clickable { galleryLauncher.launch("image/*") },
            contentAlignment = Alignment.Center
        ) {
            if (coverImageUri != null) {
                Image(
                    painter = rememberAsyncImagePainter(coverImageUri),
                    contentDescription = "Cover preview",
                    modifier = Modifier.fillMaxSize(),
                    contentScale = ContentScale.Crop
                )
                Surface(
                    color = Color.Black.copy(alpha = 0.6f),
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .fillMaxWidth()
                ) {
                    Text(
                        text = "Tap to change image",
                        color = Color.White,
                        style = MaterialTheme.typography.labelSmall,
                        modifier = Modifier.padding(8.dp),
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center
                    )
                }
            } else {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Icon(
                        imageVector = Icons.Default.CloudUpload,
                        contentDescription = null,
                        modifier = Modifier.size(48.dp),
                        tint = MaterialTheme.colorScheme.primary
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "Upload Cover Image",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Row(modifier = Modifier.fillMaxWidth()) {
            Box(
                modifier = Modifier
                    .weight(1f)
                    .clickable { 
                        pickingForStart = true
                        showDatePicker = true 
                    }
            ) {
                OutlinedTextField(
                    value = displayStartsAt,
                    onValueChange = { },
                    label = { Text("Starts At") },
                    modifier = Modifier.fillMaxWidth(),
                    readOnly = true,
                    enabled = false,
                    trailingIcon = {
                        Icon(Icons.Default.CalendarToday, contentDescription = "Select Start Date")
                    },
                    placeholder = { Text("Start date") },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        disabledTextColor = MaterialTheme.colorScheme.onSurface,
                        disabledBorderColor = MaterialTheme.colorScheme.outline,
                        disabledLabelColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        disabledTrailingIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        disabledPlaceholderColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            Box(
                modifier = Modifier
                    .weight(1f)
                    .clickable { 
                        pickingForStart = false
                        showDatePicker = true 
                    }
            ) {
                OutlinedTextField(
                    value = displayEndAt,
                    onValueChange = { },
                    label = { Text("Ends At") },
                    modifier = Modifier.fillMaxWidth(),
                    readOnly = true,
                    enabled = false,
                    trailingIcon = {
                        Icon(Icons.Default.CalendarToday, contentDescription = "Select End Date")
                    },
                    placeholder = { Text("End date") },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        disabledTextColor = MaterialTheme.colorScheme.onSurface,
                        disabledBorderColor = MaterialTheme.colorScheme.outline,
                        disabledLabelColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        disabledTrailingIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        disabledPlaceholderColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = capacity,
            onValueChange = { capacity = it },
            label = { Text("Capacity") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            enabled = !isLoading
        )

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = description,
            onValueChange = { description = it },
            label = { Text("Description") },
            modifier = Modifier
                .fillMaxWidth()
                .height(150.dp),
            singleLine = false,
            enabled = !isLoading
        )

        Spacer(modifier = Modifier.height(32.dp))

        if (isLoading) {
            CircularProgressIndicator(modifier = Modifier.fillMaxWidth().padding(16.dp))
        } else {
            Button(
                onClick = {
                    val cap = capacity.toIntOrNull() ?: 0
                    onEventCreated(
                        title, 
                        description, 
                        startsAt, 
                        cap, 
                        location.ifBlank { null }, 
                        endAt.ifBlank { null }, 
                        coverImageUri
                    )
                },
                modifier = Modifier.fillMaxWidth(),
                shape = MaterialTheme.shapes.medium
            ) {
                Text(if (isEditMode) "Update Event" else "Create Event")
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        TextButton(
            onClick = onBack,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Cancel")
        }
    }
}

