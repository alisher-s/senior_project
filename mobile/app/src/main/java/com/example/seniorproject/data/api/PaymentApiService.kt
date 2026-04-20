package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.InitiatePaymentRequestDTO
import com.example.seniorproject.data.model.InitiatePaymentResponseDTO
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface PaymentApiService {
    @POST("api/v1/payments/initiate")
    suspend fun initiatePayment(@Body request: InitiatePaymentRequestDTO): Response<InitiatePaymentResponseDTO>
}
