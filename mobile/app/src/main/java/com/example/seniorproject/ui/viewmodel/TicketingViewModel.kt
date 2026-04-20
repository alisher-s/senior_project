package com.example.seniorproject.ui.viewmodel

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.InitiatePaymentRequestDTO
import com.example.seniorproject.data.model.InitiatePaymentResponseDTO
import com.example.seniorproject.data.model.RegisterTicketRequestDTO
import com.example.seniorproject.data.model.RegisterTicketResponseDTO
import com.example.seniorproject.data.model.TicketDTO
import com.example.seniorproject.data.model.UseTicketRequestDTO
import kotlinx.coroutines.launch

sealed class TicketingState {
    object Idle : TicketingState()
    object Loading : TicketingState()
    data class Success(val ticket: RegisterTicketResponseDTO) : TicketingState()
    data class PaymentInitiated(val response: InitiatePaymentResponseDTO) : TicketingState()
    data class MyTicketsSuccess(val tickets: List<TicketDTO>) : TicketingState()
    data class Error(val message: String) : TicketingState()
}

sealed class CheckInState {
    object Idle : CheckInState()
    object Loading : CheckInState()
    object Success : CheckInState()
    data class Error(val message: String) : CheckInState()
}

class TicketingViewModel : ViewModel() {
    var ticketingState by mutableStateOf<TicketingState>(TicketingState.Idle)
        private set

    var checkInState by mutableStateOf<CheckInState>(CheckInState.Idle)
        private set

    fun registerTicket(eventId: String) {
        viewModelScope.launch {
            ticketingState = TicketingState.Loading
            try {
                val response = RetrofitClient.ticketingService.registerTicket(RegisterTicketRequestDTO(eventId))
                if (response.isSuccessful && response.body() != null) {
                    ticketingState = TicketingState.Success(response.body()!!)
                } else {
                    val errorBody = response.errorBody()?.string()
                    ticketingState = TicketingState.Error(errorBody ?: "Registration failed")
                }
            } catch (e: Exception) {
                ticketingState = TicketingState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun completeRegistration(eventId: String) {
        viewModelScope.launch {
            ticketingState = TicketingState.Loading
            try {
                val response = RetrofitClient.ticketingService.registerTicket(RegisterTicketRequestDTO(eventId))
                if (response.isSuccessful && response.body() != null) {
                    ticketingState = TicketingState.Success(response.body()!!)
                } else {
                    val errorBody = response.errorBody()?.string()
                    ticketingState = TicketingState.Error(errorBody ?: "Registration failed")
                }
            } catch (e: Exception) {
                ticketingState = TicketingState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun fetchMyTickets() {
        viewModelScope.launch {
            ticketingState = TicketingState.Loading
            try {
                val response = RetrofitClient.ticketingService.getMyTickets()
                if (response.isSuccessful && response.body() != null) {
                    ticketingState = TicketingState.MyTicketsSuccess(response.body()!!.tickets)
                } else {
                    val errorBody = response.errorBody()?.string()
                    ticketingState = TicketingState.Error(errorBody ?: "Failed to fetch tickets")
                }
            } catch (e: Exception) {
                ticketingState = TicketingState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun cancelTicket(ticketId: String) {
        viewModelScope.launch {
            try {
                val response = RetrofitClient.ticketingService.cancelTicket(ticketId)
                if (response.isSuccessful) {
                    fetchMyTickets() // Refresh the list
                } else {
                    val errorBody = response.errorBody()?.string()
                    // Don't overwrite the entire state if we just failed a cancellation, 
                    // maybe show a toast instead? But for now, we'll keep the Error state.
                    ticketingState = TicketingState.Error(errorBody ?: "Cancellation failed")
                }
            } catch (e: Exception) {
                ticketingState = TicketingState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun useTicket(qrHash: String) {
        viewModelScope.launch {
            checkInState = CheckInState.Loading
            try {
                val response = RetrofitClient.ticketingService.useTicket(UseTicketRequestDTO(qrHash))
                if (response.isSuccessful) {
                    checkInState = CheckInState.Success
                } else {
                    val errorBody = response.errorBody()?.string()
                    checkInState = CheckInState.Error(errorBody ?: "Check-in failed: ${response.message()}")
                }
            } catch (e: Exception) {
                checkInState = CheckInState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }

    fun resetStates() {
        ticketingState = TicketingState.Idle
        checkInState = CheckInState.Idle
    }
}
