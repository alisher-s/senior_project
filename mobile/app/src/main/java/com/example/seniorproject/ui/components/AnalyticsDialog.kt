package com.example.seniorproject.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import com.example.seniorproject.data.model.EventStatsResponseDTO
import com.example.seniorproject.ui.theme.GoldPrimary
import com.example.seniorproject.ui.theme.SurfaceDark
import com.example.seniorproject.ui.viewmodel.AnalyticsState
import com.example.seniorproject.ui.viewmodel.AnalyticsViewModel

@Composable
fun AnalyticsDialog(
    eventId: String,
    viewModel: AnalyticsViewModel,
    onDismiss: () -> Unit
) {
    val state = viewModel.analyticsState

    LaunchedEffect(eventId) {
        viewModel.fetchEventStats(eventId)
    }

    Dialog(onDismissRequest = onDismiss) {
        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            shape = RoundedCornerShape(28.dp),
            color = SurfaceDark,
            border = BorderStroke(1.dp, Color.White.copy(alpha = 0.1f))
        ) {
            Column(modifier = Modifier.padding(24.dp)) {
                Text(
                    "Event Analytics",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.ExtraBold,
                    color = Color.White
                )

                Spacer(modifier = Modifier.height(24.dp))

                when (state) {
                    is AnalyticsState.Loading -> {
                        Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator(color = GoldPrimary)
                        }
                    }
                    is AnalyticsState.Success -> {
                        val stats = state.stats
                        StatRow("Total Capacity", stats.totalCapacity.toString())
                        StatRow("Registered", stats.registeredCount.toString())
                        StatRow("Remaining", stats.remainingCapacity.toString())
                        
                        val fillPercentage = if (stats.totalCapacity > 0) {
                            (stats.registeredCount.toFloat() / stats.totalCapacity.toFloat())
                        } else 0f
                        
                        Spacer(modifier = Modifier.height(16.dp))
                        Text("Fill Rate", style = MaterialTheme.typography.labelLarge, color = GoldPrimary)
                        LinearProgressIndicator(
                            progress = { fillPercentage },
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(8.dp)
                                .padding(vertical = 4.dp),
                            color = GoldPrimary,
                            trackColor = Color.Gray.copy(alpha = 0.3f),
                        )
                        Text(
                            "${(fillPercentage * 100).toInt()}%",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color.White.copy(alpha = 0.6f)
                        )

                        if (stats.registrationTimeline.isNotEmpty()) {
                            Spacer(modifier = Modifier.height(24.dp))
                            Text("Registration Timeline", style = MaterialTheme.typography.labelLarge, color = GoldPrimary, fontWeight = FontWeight.Bold)
                            Spacer(modifier = Modifier.height(12.dp))
                            RegistrationChart(timeline = stats.registrationTimeline)
                        }
                    }
                    is AnalyticsState.Error -> {
                        Text(state.message, color = Color.Red)
                    }
                    else -> {}
                }

                Spacer(modifier = Modifier.height(32.dp))

                Button(
                    onClick = onDismiss,
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.buttonColors(containerColor = GoldPrimary, contentColor = Color.Black)
                ) {
                    Text("Close")
                }
            }
        }
    }
}

@Composable
fun StatRow(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(label, color = Color.White.copy(alpha = 0.7f))
        Text(value, color = Color.White, fontWeight = FontWeight.Bold, fontSize = 18.sp)
    }
}

@Composable
fun RegistrationChart(timeline: List<com.example.seniorproject.data.model.RegistrationHour>) {
    val displayTimeline = timeline.takeLast(10)
    val maxCount = displayTimeline.maxOfOrNull { it.count }?.coerceAtLeast(1) ?: 1
    
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 16.dp)
            .background(Color.Black.copy(alpha = 0.2f), RoundedCornerShape(12.dp))
            .padding(16.dp)
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(150.dp)
        ) {
            // Background Grid Lines
            Column(
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.SpaceBetween
            ) {
                repeat(4) {
                    HorizontalDivider(color = Color.White.copy(alpha = 0.05f), thickness = 1.dp)
                }
            }

            Row(
                modifier = Modifier.fillMaxSize(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.Bottom
            ) {
                // Y-Axis labels
                Column(
                    modifier = Modifier
                        .fillMaxHeight()
                        .padding(end = 4.dp),
                    verticalArrangement = Arrangement.SpaceBetween,
                    horizontalAlignment = Alignment.End
                ) {
                    Text(maxCount.toString(), style = MaterialTheme.typography.labelSmall, color = Color.Gray, fontSize = 9.sp)
                    Text((maxCount / 2).toString(), style = MaterialTheme.typography.labelSmall, color = Color.Gray, fontSize = 9.sp)
                    Text("0", style = MaterialTheme.typography.labelSmall, color = Color.Gray, fontSize = 9.sp)
                }

                // Bars
                displayTimeline.forEach { hour ->
                    val barHeight = (hour.count.toFloat() / maxCount.toFloat())
                    
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxHeight(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Bottom
                    ) {
                        if (hour.count > 0) {
                            Text(
                                text = hour.count.toString(),
                                style = MaterialTheme.typography.labelSmall,
                                color = GoldPrimary,
                                fontWeight = FontWeight.Bold,
                                fontSize = 10.sp,
                                modifier = Modifier.padding(bottom = 2.dp)
                            )
                        }
                        
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .fillMaxHeight(barHeight.coerceAtLeast(0.02f))
                                .background(
                                    brush = androidx.compose.ui.graphics.Brush.verticalGradient(
                                        colors = listOf(GoldPrimary, GoldPrimary.copy(alpha = 0.6f))
                                    ),
                                    shape = RoundedCornerShape(topStart = 4.dp, topEnd = 4.dp)
                                )
                                .border(0.5.dp, GoldPrimary.copy(alpha = 0.3f), RoundedCornerShape(topStart = 4.dp, topEnd = 4.dp))
                        )
                        
                        Spacer(modifier = Modifier.height(6.dp))
                        
                        val timeLabel = try {
                            hour.hour.split("T").last().take(5)
                        } catch (e: Exception) {
                            hour.hour.takeLast(5)
                        }
                        
                        Text(
                            text = timeLabel,
                            style = MaterialTheme.typography.labelSmall,
                            color = Color.White.copy(alpha = 0.5f),
                            fontSize = 8.sp,
                            maxLines = 1
                        )
                    }
                }
            }
        }
    }
}
