package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.CancelTicketResponseDTO
import com.example.seniorproject.data.model.MyTicketsResponseDTO
import com.example.seniorproject.data.model.RegisterTicketRequestDTO
import com.example.seniorproject.data.model.RegisterTicketResponseDTO
import com.example.seniorproject.data.model.TicketDTO
import com.example.seniorproject.data.model.UseTicketRequestDTO
import com.example.seniorproject.data.model.UseTicketResponseDTO
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface TicketingApiService {
    @POST("api/v1/tickets/register")
    suspend fun registerTicket(@Body request: RegisterTicketRequestDTO): Response<RegisterTicketResponseDTO>

    @POST("api/v1/tickets/{id}/cancel")
    suspend fun cancelTicket(@Path("id") id: String): Response<CancelTicketResponseDTO>

    @POST("api/v1/tickets/use")
    suspend fun useTicket(@Body request: UseTicketRequestDTO): Response<UseTicketResponseDTO>

    @GET("api/v1/tickets/my")
    suspend fun getMyTickets(): Response<MyTicketsResponseDTO>
}
