package com.example.seniorproject

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.seniorproject.ui.theme.SeniorProjectTheme

class QueueActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            SeniorProjectTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    QueueScreen(
                        onBackToEvents = {
                            val intent = Intent(this, EventsActivity::class.java)
                            intent.flags = Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP
                            startActivity(intent)
                            finish()
                        }
                    )
                }
            }
        }
    }
}

@Composable
fun QueueScreen(onBackToEvents: () -> Unit) {
    val yourPosition = 24
    val totalInQueue = 125

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            text = "You are in the queue!",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(48.dp))

        Box(contentAlignment = Alignment.Center) {
            CircularProgressIndicator(
                progress = { yourPosition.toFloat() / totalInQueue.toFloat() },
                modifier = Modifier.size(200.dp),
                strokeWidth = 12.dp,
                color = MaterialTheme.colorScheme.primary,
                trackColor = MaterialTheme.colorScheme.surfaceVariant,
            )
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = getOrdinalString(yourPosition),
                    style = MaterialTheme.typography.headlineLarge.copy(fontSize = 48.sp),
                    fontWeight = FontWeight.ExtraBold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "Position",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }

        Spacer(modifier = Modifier.height(48.dp))

        Text(
            text = "Wait status: $yourPosition / $totalInQueue",
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.onSurface
        )

        Text(
            text = "Estimated wait: ~15 mins",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.secondary,
            modifier = Modifier.padding(top = 8.dp)
        )

        Spacer(modifier = Modifier.height(64.dp))

        Button(
            onClick = onBackToEvents,
            modifier = Modifier.fillMaxWidth(),
            shape = MaterialTheme.shapes.medium
        ) {
            Text("Go to Events")
        }
    }
}

private fun getOrdinalString(number: Int): String {
    return when {
        number % 100 in 11..13 -> "${number}th"
        number % 10 == 1 -> "${number}st"
        number % 10 == 2 -> "${number}nd"
        number % 10 == 3 -> "${number}rd"
        else -> "${number}th"
    }
}

