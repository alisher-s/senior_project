package com.example.seniorproject.ui.components

import androidx.compose.animation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay

enum class NotificationType {
    SUCCESS, ERROR, INFO, WARNING
}

@Composable
fun AppNotification(
    message: String,
    type: NotificationType,
    visible: Boolean,
    onDismiss: () -> Unit
) {
    LaunchedEffect(visible) {
        if (visible) {
            delay(4000)
            onDismiss()
        }
    }

    AnimatedVisibility(
        visible = visible,
        enter = slideInVertically(initialOffsetY = { -it }) + fadeIn(),
        exit = slideOutVertically(targetOffsetY = { -it }) + fadeOut(),
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp)
            .statusBarsPadding()
    ) {
        val containerColor = when (type) {
            NotificationType.SUCCESS -> Color(0xFF1B3B2E) // Dark Green
            NotificationType.ERROR -> Color(0xFF421C1C)   // Dark Red
            NotificationType.INFO -> Color(0xFF1E2B3C)    // Dark Navy
            NotificationType.WARNING -> Color(0xFF42341C) // Dark Orange/Gold
        }
        val contentColor = when (type) {
            NotificationType.SUCCESS -> Color(0xFF81C784)
            NotificationType.ERROR -> Color(0xFFE57373)
            NotificationType.INFO -> Color(0xFF64B5F6)
            NotificationType.WARNING -> Color(0xFFFFB74D)
        }
        val icon = when (type) {
            NotificationType.SUCCESS -> Icons.Default.CheckCircle
            NotificationType.ERROR -> Icons.Default.Error
            NotificationType.INFO -> Icons.Default.Info
            NotificationType.WARNING -> Icons.Default.Warning
        }

        Surface(
            color = containerColor,
            contentColor = contentColor,
            shape = RoundedCornerShape(16.dp),
            shadowElevation = 8.dp,
            modifier = Modifier.fillMaxWidth()
        ) {
            Row(
                modifier = Modifier
                    .padding(16.dp)
                    .fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    modifier = Modifier.size(24.dp)
                )
                Spacer(modifier = Modifier.width(12.dp))
                Text(
                    text = message,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    lineHeight = 20.sp
                )
            }
        }
    }
}
