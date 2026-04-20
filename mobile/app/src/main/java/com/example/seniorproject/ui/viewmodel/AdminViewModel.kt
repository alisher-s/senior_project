package com.example.seniorproject.ui.viewmodel

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.ModerateEventRequestDTO
import com.example.seniorproject.data.model.SetUserRoleRequestDTO
import com.example.seniorproject.data.model.UserDTO
import kotlinx.coroutines.launch

sealed class AdminState {
    object Idle : AdminState()
    object Loading : AdminState()
    object Success : AdminState()
    data class UsersLoaded(val users: List<UserDTO>) : AdminState()
    data class EventsLoaded(val events: List<com.example.seniorproject.data.model.AdminEventDTO>) : AdminState()
    data class Error(val message: String) : AdminState()
}

class AdminViewModel : ViewModel() {
    var adminState by mutableStateOf<AdminState>(AdminState.Idle)
        private set

    fun moderateEvent(eventId: String, approve: Boolean, reason: String?, onComplete: () -> Unit) {
        viewModelScope.launch {
            adminState = AdminState.Loading
            try {
                val action = if (approve) "approve" else "reject"
                val response = RetrofitClient.adminService.moderateEvent(eventId, ModerateEventRequestDTO(action, reason))
                if (response.isSuccessful) {
                    adminState = AdminState.Success
                    onComplete()
                } else {
                    adminState = AdminState.Error("Moderation failed: ${response.message()}")
                }
            } catch (e: Exception) {
                adminState = AdminState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun setUserRole(userId: String, role: String) {
        viewModelScope.launch {
            adminState = AdminState.Loading
            try {
                val response = RetrofitClient.adminService.setUserRole(userId, SetUserRoleRequestDTO(role))
                if (response.isSuccessful) {
                    adminState = AdminState.Success
                } else {
                    adminState = AdminState.Error("Role update failed: ${response.message()}")
                }
            } catch (e: Exception) {
                adminState = AdminState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun listUsers(query: String? = null) {
        viewModelScope.launch {
            adminState = AdminState.Loading
            try {
                val response = RetrofitClient.adminService.listUsers(query)
                if (response.isSuccessful) {
                    adminState = AdminState.UsersLoaded(response.body()?.items ?: emptyList())
                } else {
                    adminState = AdminState.Error("Failed to fetch users: ${response.message()}")
                }
            } catch (e: Exception) {
                adminState = AdminState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun listAdminEvents(status: String = "pending", query: String? = null) {
        viewModelScope.launch {
            adminState = AdminState.Loading
            try {
                val response = RetrofitClient.adminService.listEvents(moderationStatus = status, query = query)
                if (response.isSuccessful) {
                    adminState = AdminState.EventsLoaded(response.body()?.items ?: emptyList())
                } else {
                    adminState = AdminState.Error("Failed to fetch events: ${response.message()}")
                }
            } catch (e: Exception) {
                adminState = AdminState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun resetState() {
        adminState = AdminState.Idle
    }
}
