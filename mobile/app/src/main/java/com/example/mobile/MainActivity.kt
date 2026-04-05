package com.example.mobile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.lifecycle.ViewModelProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.livedata.observeAsState
import androidx.compose.foundation.isSystemInDarkTheme
import com.example.mobile.dashboard.DashboardViewModel
import com.example.mobile.ui.dashboard.DashboardScreen
import com.example.mobile.ui.theme.MobileTheme

class MainActivity : ComponentActivity() {

    private lateinit var viewModel: DashboardViewModel

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        
        viewModel = ViewModelProvider(this).get(DashboardViewModel::class.java)
        viewModel.startPolling("LOC-DEMO-001")

        setContent {
            val isDarkTheme by viewModel.isDarkTheme.observeAsState()
            MobileTheme(darkTheme = isDarkTheme ?: isSystemInDarkTheme()) {
                DashboardScreen(viewModel)
            }
        }
    }
}
