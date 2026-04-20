package com.example.seniorproject.data.api

import com.example.seniorproject.SeniorProjectApplication
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

object RetrofitClient {
    private const val BASE_URL = "http://10.0.2.2:8080/"

    fun getEventImageUrl(path: String?): String? {
        if (path.isNullOrBlank()) return null
        if (path.startsWith("http")) {
            // Replace localhost with 10.0.2.2 for emulator
            return path.replace("localhost", "10.0.2.2")
        }
        return "${BASE_URL}api/v1/static/$path"
    }

    private val logging = HttpLoggingInterceptor().apply {
        level = HttpLoggingInterceptor.Level.BODY
    }

    private val httpClient = OkHttpClient.Builder()
        .addInterceptor(logging)
        .addInterceptor { chain ->
            val original = chain.request()
            val requestBuilder = original.newBuilder()
            
            // Don't add auth header for login and register
            if (!original.url.encodedPath.contains("auth/login") && 
                !original.url.encodedPath.contains("auth/register")) {
                SeniorProjectApplication.tokenManager.getAccessToken()?.let { token ->
                    requestBuilder.addHeader("Authorization", "Bearer $token")
                }
            }

            chain.proceed(requestBuilder.build())
        }
        .build()

    private val retrofit: Retrofit by lazy {
        Retrofit.Builder()
            .baseUrl(BASE_URL)
            .addConverterFactory(GsonConverterFactory.create())
            .client(httpClient)
            .build()
    }

    val authService: AuthApiService by lazy {
        retrofit.create(AuthApiService::class.java)
    }

    val eventsService: EventsApiService by lazy {
        retrofit.create(EventsApiService::class.java)
    }

    val adminService: AdminApiService by lazy {
        retrofit.create(AdminApiService::class.java)
    }

    val ticketingService: TicketingApiService by lazy {
        retrofit.create(TicketingApiService::class.java)
    }

    val paymentService: PaymentApiService by lazy {
        retrofit.create(PaymentApiService::class.java)
    }

    val analyticsService: AnalyticsApiService by lazy {
        retrofit.create(AnalyticsApiService::class.java)
    }
}
