package com.example.seniorproject.ui.viewmodel

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.seniorproject.data.api.RetrofitClient
import com.example.seniorproject.data.model.EventStatsResponseDTO
import kotlinx.coroutines.launch

sealed class AnalyticsState {
    object Idle : AnalyticsState()
    object Loading : AnalyticsState()
    data class Success(val stats: EventStatsResponseDTO) : AnalyticsState()
    data class Error(val message: String) : AnalyticsState()
}

class AnalyticsViewModel : ViewModel() {
    var analyticsState by mutableStateOf<AnalyticsState>(AnalyticsState.Idle)
        private set

    fun fetchEventStats(eventId: String? = null) {
        viewModelScope.launch {
            analyticsState = AnalyticsState.Loading
            try {
                val response = RetrofitClient.analyticsService.getEventStats(eventId)
                if (response.isSuccessful && response.body() != null) {
                    analyticsState = AnalyticsState.Success(response.body()!!)
                } else {
                    val errorBody = response.errorBody()?.string()
                    analyticsState = AnalyticsState.Error(errorBody ?: "Failed to fetch analytics")
                }
            } catch (e: Exception) {
                analyticsState = AnalyticsState.Error(e.message ?: "An unknown error occurred")
            }
        }
    }
}
