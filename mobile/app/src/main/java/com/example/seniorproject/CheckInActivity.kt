package com.example.seniorproject

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.util.Size
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import com.example.seniorproject.ui.components.AppNotification
import com.example.seniorproject.ui.components.NotificationType
import com.example.seniorproject.ui.theme.SeniorProjectTheme
import com.example.seniorproject.ui.viewmodel.CheckInState
import com.example.seniorproject.ui.viewmodel.TicketingViewModel
import com.example.seniorproject.util.QRCodeAnalyzer
import java.util.concurrent.Executors

class CheckInActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            SeniorProjectTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    val viewModel: TicketingViewModel = viewModel()
                    val state = viewModel.checkInState
                    var showNotification by remember { mutableStateOf(false) }
                    var notificationMessage by remember { mutableStateOf("") }
                    var notificationType by remember { mutableStateOf(NotificationType.INFO) }

                    LaunchedEffect(state) {
                        when (state) {
                            is CheckInState.Success -> {
                                notificationMessage = "Check-in successful!"
                                notificationType = NotificationType.SUCCESS
                                showNotification = true
                                viewModel.resetStates()
                            }
                            is CheckInState.Error -> {
                                notificationMessage = state.message
                                notificationType = NotificationType.ERROR
                                showNotification = true
                                viewModel.resetStates()
                            }
                            else -> {}
                        }
                    }

                    Box(modifier = Modifier.fillMaxSize()) {
                        CheckInScreen(
                            isLoading = state is CheckInState.Loading,
                            onCheckIn = { qrHash ->
                                viewModel.useTicket(qrHash)
                            }
                        )

                        AppNotification(
                            message = notificationMessage,
                            type = notificationType,
                            visible = showNotification,
                            onDismiss = { showNotification = false }
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun CheckInScreen(
    isLoading: Boolean,
    onCheckIn: (String) -> Unit
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    var hasCamPermission by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.CAMERA
            ) == PackageManager.PERMISSION_GRANTED
        )
    }
    val launcher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission(),
        onResult = { granted ->
            hasCamPermission = granted
        }
    )
    LaunchedEffect(key1 = true) {
        launcher.launch(Manifest.permission.CAMERA)
    }

    var qrHash by remember { mutableStateOf("") }
    var isScanning by remember { mutableStateOf(true) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text("Staff Check-in", style = MaterialTheme.typography.headlineLarge)
        Spacer(modifier = Modifier.height(24.dp))

        if (hasCamPermission && isScanning) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
            ) {
                AndroidView(
                    factory = { context ->
                        val previewView = PreviewView(context)
                        val preview = Preview.Builder().build()
                        val selector = CameraSelector.Builder()
                            .requireLensFacing(CameraSelector.LENS_FACING_BACK)
                            .build()
                        preview.surfaceProvider = previewView.surfaceProvider
                        val imageAnalysis = ImageAnalysis.Builder()
                            .setTargetResolution(Size(1280, 720))
                            .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                            .build()
                        imageAnalysis.setAnalyzer(
                            ContextCompat.getMainExecutor(context),
                            QRCodeAnalyzer { result ->
                                qrHash = result
                                isScanning = false
                                onCheckIn(result)
                            }
                        )
                        try {
                            ProcessCameraProvider.getInstance(context).get().bindToLifecycle(
                                lifecycleOwner,
                                selector,
                                preview,
                                imageAnalysis
                            )
                        } catch (e: Exception) {
                            e.printStackTrace()
                        }
                        previewView
                    },
                    modifier = Modifier.fillMaxSize()
                )
            }
        } else {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
                contentAlignment = Alignment.Center
            ) {
                if (!hasCamPermission) {
                    Text("Camera permission is required to scan QR codes", color = Color.Red)
                } else if (!isScanning) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("QR Scanned!")
                        Text("Hash: $qrHash", style = MaterialTheme.typography.bodySmall)
                        Spacer(modifier = Modifier.height(16.dp))
                        Button(onClick = { isScanning = true; qrHash = "" }) {
                            Text("Scan Again")
                        }
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = qrHash,
            onValueChange = { qrHash = it },
            label = { Text("Enter QR Hash Manually") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            enabled = !isLoading
        )

        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = { if (qrHash.isNotBlank()) onCheckIn(qrHash) },
            modifier = Modifier.fillMaxWidth(),
            enabled = !isLoading && qrHash.isNotBlank()
        ) {
            Text("Verify Ticket")
        }

        if (isLoading) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth().padding(top = 16.dp))
        }
    }
}
