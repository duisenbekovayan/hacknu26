package com.example.mobile

import android.graphics.Color
import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.ViewModelProvider
import com.example.mobile.databinding.ActivityMainBinding
import com.example.mobile.dashboard.DashboardViewModel
import com.github.mikephil.charting.data.Entry
import com.github.mikephil.charting.data.LineData
import com.github.mikephil.charting.data.LineDataSet

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var viewModel: DashboardViewModel

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        setSupportActionBar(binding.toolbar)

        viewModel = ViewModelProvider(this).get(DashboardViewModel::class.java)
        
        setupChart()
        
        viewModel.telemetry.observe(this) { sample ->
            binding.textHealthIndex.text = String.format("%.0f", sample.healthIndex)
            binding.textSpeed.text = String.format("%.1f км/ч", sample.speedKmh)
            binding.textFuel.text = String.format("%.0f л", sample.fuelLevelL)
            
            val alertsText = sample.alerts?.joinToString("\n") { it.text } ?: getString(R.string.no_active_alerts)
            binding.textAlerts.text = if (alertsText.isEmpty()) getString(R.string.no_active_alerts) else alertsText
            
            val color = when (sample.healthGrade.lowercase()) {
                "crit" -> Color.RED
                "warn" -> Color.YELLOW
                else -> Color.GREEN
            }
            binding.cardHealth.setStrokeColor(color)
        }

        viewModel.history.observe(this) { history ->
            updateChart(history.reversed())
        }

        viewModel.startPolling("LOC-DEMO-001")
    }

    private fun setupChart() {
        binding.chartTrends.apply {
            description.isEnabled = false
            setTouchEnabled(true)
            setPinchZoom(true)
            xAxis.textColor = Color.WHITE
            axisLeft.textColor = Color.WHITE
            axisRight.isEnabled = false
            legend.textColor = Color.WHITE
            setBackgroundColor(Color.TRANSPARENT)
        }
    }

    private fun updateChart(history: List<com.example.mobile.data.TelemetrySample>) {
        val speedEntries = history.mapIndexed { index, sample ->
            Entry(index.toFloat(), sample.speedKmh.toFloat())
        }

        val speedSet = LineDataSet(speedEntries, getString(R.string.speed)).apply {
            color = Color.BLUE
            setDrawCircles(false)
            lineWidth = 2f
            mode = LineDataSet.Mode.CUBIC_BEZIER
        }

        binding.chartTrends.data = LineData(speedSet)
        binding.chartTrends.invalidate()
    }
}
