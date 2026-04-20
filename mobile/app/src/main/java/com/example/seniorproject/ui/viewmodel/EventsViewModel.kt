package com.example.seniorproject.ui.viewmodel

import android.content.Context
import android.net.Uri
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.CreateEventRequestDTO
import com.example.seniorproject.data.model.EventDTO
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.asRequestBody
import java.io.File
import java.io.FileOutputStream
import java.io.InputStream

sealed class EventsState {
    object Idle : EventsState()
    object Loading : EventsState()
    data class Success(val events: List<EventDTO>) : EventsState()
    data class Error(val message: String) : EventsState()
}

sealed class CreateEventState {
    object Idle : CreateEventState()
    object Loading : CreateEventState()
    data class Success(val event: EventDTO) : CreateEventState()
    data class Error(val message: String) : CreateEventState()
}

class EventsViewModel : ViewModel() {
    var eventsState by mutableStateOf<EventsState>(EventsState.Idle)
        private set

    var createEventState by mutableStateOf<CreateEventState>(CreateEventState.Idle)
        private set

    fun fetchEvents(
        query: String? = null,
        organizerId: String? = null,
        moderationStatus: String? = null,
        isMine: Boolean = false
    ) {
        viewModelScope.launch {
            eventsState = EventsState.Loading
            try {
                val response = if (isMine) {
                    RetrofitClient.eventsService.listMyEvents()
                } else {
                    RetrofitClient.eventsService.listEvents(
                        query = query,
                        organizerId = organizerId,
                        moderationStatus = moderationStatus
                    )
                }

                if (response.isSuccessful && response.body() != null) {
                    var items = response.body()!!.items
                    
                    // Client-side filtering as fallback only for the general list
                    if (!isMine) {
                        if (organizerId != null) {
                            items = items.filter { it.organizerId == organizerId }
                        }
                        if (moderationStatus != null) {
                            items = items.filter { it.moderationStatus == moderationStatus }
                        }
                    }
                    
                    eventsState = EventsState.Success(items)
                } else {
                    eventsState = EventsState.Error("Failed to fetch events: ${response.message()}")
                }
            } catch (e: Exception) {
                eventsState = EventsState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun createEvent(
        title: String,
        description: String,
        startsAt: String,
        capacity: Int,
        location: String? = null,
        endAt: String? = null,
        coverImageUri: Uri? = null,
        context: Context? = null,
        isMine: Boolean = false
    ) {
        viewModelScope.launch {
            createEventState = CreateEventState.Loading
            try {
                val request = CreateEventRequestDTO(
                    title = title,
                    description = description,
                    startsAt = startsAt,
                    location = location,
                    endAt = endAt,
                    capacityTotal = capacity,
                    coverImageURL = null // URL is set after upload
                )
                val response = RetrofitClient.eventsService.createEvent(request)
                if (response.isSuccessful && response.body() != null) {
                    val createdEvent = response.body()!!
                    
                    if (coverImageUri != null && context != null) {
                        uploadImageInternal(createdEvent, coverImageUri, context, isMine)
                    } else {
                        createEventState = CreateEventState.Success(createdEvent)
                        fetchEvents(isMine = isMine) // Refresh list
                    }
                } else {
                    val errorBody = response.errorBody()?.string()
                    createEventState = CreateEventState.Error(errorBody ?: "Failed to create event: ${response.message()}")
                }
            } catch (e: Exception) {
                createEventState = CreateEventState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun updateEvent(
        eventId: String,
        title: String,
        description: String,
        startsAt: String,
        capacity: Int,
        location: String? = null,
        endAt: String? = null,
        coverImageUri: Uri? = null,
        existingCoverUrl: String? = null,
        context: Context? = null,
        isMine: Boolean = false
    ) {
        viewModelScope.launch {
            createEventState = CreateEventState.Loading
            try {
                // Ensure we only send the filename, not the full URL if present
                val cleanedCoverUrl = existingCoverUrl?.let { url ->
                    if (url.startsWith("http")) {
                        url.substringAfterLast("/")
                    } else {
                        url
                    }
                }

                val request = com.example.seniorproject.data.model.UpdateEventRequestDTO(
                    id = eventId,
                    title = title,
                    description = description,
                    startsAt = startsAt,
                    location = location,
                    endAt = endAt?.ifBlank { null },
                    capacityTotal = capacity,
                    coverImageURL = cleanedCoverUrl
                )
                val response = RetrofitClient.eventsService.updateEvent(eventId, request)
                if (response.isSuccessful && response.body() != null) {
                    val updatedEvent = response.body()!!
                    
                    if (coverImageUri != null && context != null) {
                        uploadImageInternal(updatedEvent, coverImageUri, context, isMine)
                    } else {
                        createEventState = CreateEventState.Success(updatedEvent)
                        fetchEvents(isMine = isMine) // Refresh list
                    }
                } else {
                    val errorBody = response.errorBody()?.string()
                    createEventState = CreateEventState.Error(errorBody ?: "Failed to update event: ${response.message()}")
                }
            } catch (e: Exception) {
                createEventState = CreateEventState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    private suspend fun uploadImageInternal(
        event: EventDTO,
        uri: Uri,
        context: Context,
        isMine: Boolean = false
    ) {
        try {
            val file = uriToFile(uri, context)
            val requestFile = file.asRequestBody(
                context.contentResolver.getType(uri)?.toMediaTypeOrNull()
            )
            val body = MultipartBody.Part.createFormData("image", file.name, requestFile)

            val uploadResponse = RetrofitClient.eventsService.uploadCoverImage(event.id, body)
            if (uploadResponse.isSuccessful) {
                // Try to fetch the updated event, but don't fail if it's 404 (likely pending moderation)
                val eventResponse = RetrofitClient.eventsService.getEvent(event.id)
                if (eventResponse.isSuccessful && eventResponse.body() != null) {
                    createEventState = CreateEventState.Success(eventResponse.body()!!)
                } else {
                    // Fallback to the original event object if refresh fails (common for pending events)
                    createEventState = CreateEventState.Success(event)
                }
                fetchEvents(isMine = isMine)
            } else {
                createEventState = CreateEventState.Error("Event created, but image upload failed: ${uploadResponse.message()}")
            }
            file.delete()
        } catch (e: Exception) {
            createEventState = CreateEventState.Error("Event created, but image upload failed: ${e.message}")
        }
    }

    private fun uriToFile(uri: Uri, context: Context): File {
        val inputStream: InputStream? = context.contentResolver.openInputStream(uri)
        val file = File(context.cacheDir, "temp_image_${System.currentTimeMillis()}")
        val outputStream = FileOutputStream(file)
        inputStream?.copyTo(outputStream)
        inputStream?.close()
        outputStream.close()
        return file
    }

    fun resetCreateState() {
        createEventState = CreateEventState.Idle
    }
}
