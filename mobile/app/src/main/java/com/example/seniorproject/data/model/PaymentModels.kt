package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class InitiatePaymentRequestDTO(
    @SerializedName("event_id") val eventId: String,
    val amount: Long,
    val currency: String
)

data class InitiatePaymentResponseDTO(
    @SerializedName("payment_id") val paymentId: String,
    @SerializedName("provider_ref") val providerRef: String,
    @SerializedName("provider_url") val providerUrl: String
)
