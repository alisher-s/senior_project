package com.example.seniorproject.data.api

import android.content.Context
import android.content.SharedPreferences

class TokenManager(context: Context) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences("auth_prefs", Context.MODE_PRIVATE)

    fun saveTokens(accessToken: String, refreshToken: String, userId: String, role: String) {
        prefs.edit().apply {
            putString("access_token", accessToken)
            putString("refresh_token", refreshToken)
            putString("user_id", userId)
            putString("user_role", role)
            apply()
        }
    }

    fun getAccessToken(): String? = prefs.getString("access_token", null)

    fun getRefreshToken(): String? = prefs.getString("refresh_token", null)

    fun getUserId(): String? = prefs.getString("user_id", null)

    fun getUserRole(): String? = prefs.getString("user_role", null)

    fun clearTokens() {
        prefs.edit().clear().apply()
    }
}
