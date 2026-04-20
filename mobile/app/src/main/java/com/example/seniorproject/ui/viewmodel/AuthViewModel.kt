package com.example.seniorproject.ui.viewmodel

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.AuthResponseDTO
import com.example.seniorproject.data.model.LoginRequestDTO
import com.example.seniorproject.data.model.PatchMeRolesRequestDTO
import com.example.seniorproject.data.model.RegisterRequestDTO
import kotlinx.coroutines.launch
import org.json.JSONObject
import retrofit2.Response

sealed class AuthState {
    object Idle : AuthState()
    object Loading : AuthState()
    data class Success(val data: AuthResponseDTO) : AuthState()
    object RequestSuccess : AuthState()
    data class Error(val message: String) : AuthState()
}

class AuthViewModel : ViewModel() {
    var state by mutableStateOf<AuthState>(AuthState.Idle)
        private set

    fun register(email: String, password: String) {
        viewModelScope.launch {
            state = AuthState.Loading
            try {
                val response = RetrofitClient.authService.register(RegisterRequestDTO(email, password))
                if (response.isSuccessful && response.body() != null) {
                    state = AuthState.Success(response.body()!!)
                } else {
                    state = AuthState.Error(parseError(response))
                }
            } catch (e: Exception) {
                state = AuthState.Error("Connection error. Please check your internet.")
            }
        }
    }

    fun login(email: String, password: String) {
        viewModelScope.launch {
            state = AuthState.Loading
            try {
                val response = RetrofitClient.authService.login(LoginRequestDTO(email, password))
                if (response.isSuccessful && response.body() != null) {
                    state = AuthState.Success(response.body()!!)
                } else {
                    state = AuthState.Error(parseError(response))
                }
            } catch (e: Exception) {
                state = AuthState.Error("Connection error. Please check your internet.")
            }
        }
    }

    private fun parseError(response: Response<*>): String {
        return try {
            val errorBody = response.errorBody()?.string()
            val json = JSONObject(errorBody ?: "")
            
            val errorObj = json.optJSONObject("error")
            val code = errorObj?.optString("code") ?: json.optString("code", "")
            val message = errorObj?.optString("message") ?: json.optString("message", "")

            when (code) {
                "invalid_credentials" -> "Incorrect email or password."
                "email_exists" -> "An account with this email already exists."
                "email_not_allowed" -> "Please use your university email (@nu.edu.kz)."
                "organizer_request_forbidden" -> "Only students can request an organizer role."
                "organizer_already_active" -> "You already have the organizer role."
                "invalid_request" -> {
                    if (message.contains("password", ignoreCase = true)) {
                        "Password must be at least 8 characters long."
                    } else {
                        "Invalid request. Please check your input."
                    }
                }
                else -> {
                    if (message.isNotBlank()) message else throw Exception("Fallback to HTTP code")
                }
            }
        } catch (e: Exception) {
            when (response.code()) {
                401 -> "Incorrect email or password."
                403 -> "You don't have permission for this action."
                409 -> "Conflict occurred. Please try again."
                400 -> "Invalid details. Please check your input."
                500 -> "Server error. Please try again later."
                else -> "Something went wrong. Please try again."
            }
        }
    }

    fun requestOrganizerRole() {
        viewModelScope.launch {
            state = AuthState.Loading
            try {
                val response = RetrofitClient.authService.requestOrganizerRole(PatchMeRolesRequestDTO(listOf("organizer")))
                if (response.isSuccessful) {
                    state = AuthState.RequestSuccess
                } else {
                    state = AuthState.Error(parseError(response))
                }
            } catch (e: Exception) {
                state = AuthState.Error("Connection error. Please check your internet.")
            }
        }
    }

    fun resetState() {
        state = AuthState.Idle
    }
}
