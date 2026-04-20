package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.AuthResponseDTO
import com.example.seniorproject.data.model.LoginRequestDTO
import com.example.seniorproject.data.model.MeRolesResponseDTO
import com.example.seniorproject.data.model.PatchMeRolesRequestDTO
import com.example.seniorproject.data.model.RefreshRequestDTO
import com.example.seniorproject.data.model.RegisterRequestDTO
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.PATCH
import retrofit2.http.POST

interface AuthApiService {
    @POST("api/v1/auth/register")
    suspend fun register(@Body request: RegisterRequestDTO): Response<AuthResponseDTO>

    @POST("api/v1/auth/login")
    suspend fun login(@Body request: LoginRequestDTO): Response<AuthResponseDTO>

    @POST("api/v1/auth/refresh")
    suspend fun refresh(@Body request: RefreshRequestDTO): Response<AuthResponseDTO>

    @PATCH("api/v1/auth/me/roles")
    suspend fun requestOrganizerRole(@Body request: PatchMeRolesRequestDTO): Response<MeRolesResponseDTO>
}
