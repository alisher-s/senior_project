package com.example.seniorproject

import android.app.Application
import com.example.seniorproject.data.api.TokenManager

class SeniorProjectApplication : Application() {
    companion object {
        lateinit var tokenManager: TokenManager
            private set
    }

    override fun onCreate() {
        super.onCreate()
        tokenManager = TokenManager(this)
    }
}
