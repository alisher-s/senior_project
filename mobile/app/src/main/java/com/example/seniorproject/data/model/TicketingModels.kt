package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class RegisterTicketRequestDTO(
    @SerializedName("event_id") val eventID: String
)

data class RegisterTicketResponseDTO(
    @SerializedName("ticket_id") val ticketID: String,
    @SerializedName("event_id") val eventID: String,
    @SerializedName("user_id") val userID: String,
    val status: String,
    @SerializedName("qr_png_base64") val qrPNGBase64: String,
    @SerializedName("qr_hash_hex") val qrHashHex: String
)

data class CancelTicketResponseDTO(
    @SerializedName("ticket_id") val ticketID: String,
    @SerializedName("event_id") val eventID: String,
    @SerializedName("user_id") val userID: String,
    val status: String
)

data class UseTicketRequestDTO(
    @SerializedName("qr_hash_hex") val qrHashHex: String
)

data class UseTicketResponseDTO(
    @SerializedName("ticket_id") val ticketID: String,
    @SerializedName("event_id") val eventID: String,
    @SerializedName("user_id") val userID: String,
    val status: String
)

data class TicketDTO(
    @SerializedName("ticket_id") val id: String,
    @SerializedName("event_id") val eventID: String,
    @SerializedName("user_id") val userID: String? = null,
    val status: String,
    @SerializedName("qr_hash_hex") val qrHashHex: String,
    @SerializedName("event_title") val eventTitle: String? = null,
    @SerializedName("event_date") val eventDate: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    val event: EventDTO? = null
)

data class MyTicketsResponseDTO(
    val tickets: List<TicketDTO>
)
