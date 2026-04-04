package com.example.mobile.ui.dashboard

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.livedata.observeAsState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import com.example.mobile.dashboard.DashboardViewModel
import com.example.mobile.data.TelemetrySample
import com.github.mikephil.charting.charts.LineChart
import com.github.mikephil.charting.data.Entry
import com.github.mikephil.charting.data.LineData
import com.github.mikephil.charting.data.LineDataSet
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(viewModel: DashboardViewModel) {
    val telemetry by viewModel.telemetry.observeAsState()
    val history by viewModel.history.observeAsState(emptyList())

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Цифровой двойник локомотива", fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Color.White)
                        Text("Телеметрия в реальном времени", fontSize = 12.sp, color = Color.Gray)
                    }
                },
                actions = {
                    StatusBadge(telemetry?.trainID ?: "---")
                    Spacer(modifier = Modifier.width(16.dp))
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = Color(0xFF0A0A0A))
            )
        },
        containerColor = Color(0xFF0A0A0A)
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // HERO SECTION: Health and Speed
            item {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    HealthGauge(telemetry, modifier = Modifier.weight(1f))
                    PrimaryMetricCard(
                        label = "СКОРОСТЬ",
                        value = telemetry?.speedKmh?.let { String.format(Locale.ROOT, "%.1f", it) } ?: "--",
                        unit = "км/ч",
                        modifier = Modifier.weight(1.2f)
                    )
                }
            }

            // CRITICAL ALERTS
            item { ActiveAlertsCard(telemetry?.alerts?.map { it.text } ?: emptyList()) }

            // BRAKE SYSTEM
            item {
                ParameterSection("Тормозная система") {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        ParameterItemLarge("Торм. Магистраль", "${telemetry?.brakePipePressureBar ?: "--"}", "бар", Modifier.weight(1f))
                        ParameterItemLarge("Главный Резервуар", "${telemetry?.mainReservoirBar ?: "--"}", "бар", Modifier.weight(1f))
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    SimpleGraphCard("ДАВЛЕНИЕ В МАГИСТРАЛИ (бар)", history.map { it.brakePipePressureBar.toFloat() }, Color(0xFF64B5F6), Modifier.fillMaxWidth().height(80.dp))
                }
            }

            // POWER PLANT
            item {
                ParameterSection("Силовая установка") {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        ParameterItemLarge("Топливо", "${telemetry?.fuelLevelL?.toInt() ?: "--"}", "л", Modifier.weight(1f))
                        ParameterItemLarge("ОЖ", "${telemetry?.coolantTempC ?: "--"}", "°C", Modifier.weight(1f))
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    SimpleGraphCard("ТЕМПЕРАТУРА ОЖ (°C)", history.map { it.coolantTempC.toFloat() }, Color(0xFFFF9800), Modifier.fillMaxWidth().height(80.dp))
                }
            }

            // ELECTRICAL SYSTEM
            item {
                ParameterSection("Электрооборудование") {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        SimpleGraphCard("ТЯГА (A)", history.map { it.tractionCurrentA.toFloat() }, Color.Yellow, Modifier.weight(1f))
                        SimpleGraphCard("НАПРЯЖЕНИЕ (В)", history.map { it.lineVoltageV.toFloat() }, Color.Cyan, Modifier.weight(1f))
                    }
                }
            }

            // TRACTION MOTORS
            item {
                ParameterSection("Тяговые двигатели (ТЭД)") {
                    TractionMotorsGridLarge(telemetry?.tractionMotorTempC ?: emptyList())
                }
            }

            // MAIN TRENDS
            item {
                TrendsSectionLarge("ИСТОРИЯ СКОРОСТИ", history.map { it.speedKmh.toFloat() }, Color.Cyan)
            }
            
            item { Spacer(modifier = Modifier.height(24.dp)) }
        }
    }
}

@Composable
fun StatusBadge(trainId: String) {
    Surface(color = Color(0xFF1E1E1E), shape = RoundedCornerShape(8.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)) {
            Box(modifier = Modifier.size(8.dp).background(Color(0xFF69F0AE), CircleShape))
            Spacer(modifier = Modifier.width(8.dp))
            Text(trainId, color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
fun HealthGauge(sample: TelemetrySample?, modifier: Modifier = Modifier) {
    val healthValue = sample?.healthIndex ?: 0.0
    val color = when (sample?.healthGrade?.lowercase(Locale.ROOT)) {
        "crit" -> Color(0xFFFF5252)
        "warn" -> Color(0xFFFFD740)
        else -> Color(0xFF69F0AE)
    }

    Card(modifier = modifier, colors = CardDefaults.cardColors(containerColor = Color(0xFF1E1E1E)), shape = RoundedCornerShape(16.dp)) {
        Column(modifier = Modifier.padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Text("ЗДОРОВЬЕ", fontSize = 10.sp, color = Color.Gray, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(8.dp))
            Box(contentAlignment = Alignment.Center, modifier = Modifier.size(80.dp)) {
                Canvas(modifier = Modifier.fillMaxSize()) {
                    drawArc(Color(0xFF2C2C2C), 0f, 360f, false, style = Stroke(8.dp.toPx(), cap = StrokeCap.Round))
                    drawArc(color, -90f, (healthValue / 100f * 360f).toFloat(), false, style = Stroke(8.dp.toPx(), cap = StrokeCap.Round))
                }
                Text(String.format(Locale.ROOT, "%.0f", healthValue), fontSize = 28.sp, fontWeight = FontWeight.ExtraBold, color = color)
            }
        }
    }
}

@Composable
fun PrimaryMetricCard(label: String, value: String, unit: String, modifier: Modifier = Modifier) {
    Card(modifier = modifier, colors = CardDefaults.cardColors(containerColor = Color(0xFF1E1E1E)), shape = RoundedCornerShape(16.dp)) {
        Column(modifier = Modifier.padding(16.dp).fillMaxHeight(), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
            Text(label, fontSize = 10.sp, color = Color.Gray, fontWeight = FontWeight.Bold)
            Row(verticalAlignment = Alignment.Bottom) {
                Text(value, fontSize = 48.sp, fontWeight = FontWeight.Black, color = Color.White)
                Text(unit, fontSize = 14.sp, color = Color.Gray, modifier = Modifier.padding(bottom = 10.dp, start = 4.dp))
            }
        }
    }
}

@Composable
fun SimpleGraphCard(label: String, data: List<Float>, color: Color, modifier: Modifier = Modifier) {
    Card(modifier = modifier, colors = CardDefaults.cardColors(containerColor = Color(0xFF1E1E1E)), shape = RoundedCornerShape(12.dp)) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(label, fontSize = 9.sp, color = Color.Gray, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(8.dp))
            AndroidView(
                factory = { context ->
                    LineChart(context).apply {
                        description.isEnabled = false
                        setTouchEnabled(false)
                        xAxis.isEnabled = false
                        axisLeft.isEnabled = false
                        axisRight.isEnabled = false
                        legend.isEnabled = false
                        setDrawGridBackground(false)
                        setViewPortOffsets(0f, 0f, 0f, 0f)
                    }
                },
                update = { chart ->
                    val entries = data.takeLast(25).mapIndexed { i, v -> Entry(i.toFloat(), v) }
                    chart.data = LineData(LineDataSet(entries, "").apply {
                        this.color = color.hashCode()
                        setDrawCircles(false)
                        setDrawValues(false)
                        lineWidth = 2f
                        mode = LineDataSet.Mode.CUBIC_BEZIER
                        setDrawFilled(true)
                        fillColor = color.hashCode()
                        fillAlpha = 30
                    })
                    chart.invalidate()
                },
                modifier = Modifier.fillMaxWidth().height(40.dp)
            )
        }
    }
}

@Composable
fun ParameterSection(title: String, content: @Composable () -> Unit) {
    Column {
        Text(title, fontSize = 12.sp, color = Color.Gray, fontWeight = FontWeight.Bold, modifier = Modifier.padding(start = 4.dp, bottom = 4.dp))
        content()
    }
}

@Composable
fun ParameterItemLarge(label: String, value: String, unit: String, modifier: Modifier = Modifier) {
    Surface(modifier = modifier, color = Color(0xFF1E1E1E), shape = RoundedCornerShape(12.dp)) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(label, fontSize = 9.sp, color = Color.LightGray)
            Row(verticalAlignment = Alignment.Bottom) {
                Text(value, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Color.White)
                Spacer(modifier = Modifier.width(4.dp))
                Text(unit, fontSize = 11.sp, color = Color.Gray, modifier = Modifier.padding(bottom = 3.dp))
            }
        }
    }
}

@Composable
fun TractionMotorsGridLarge(temps: List<Double>) {
    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        temps.chunked(3).forEach { group ->
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                group.forEachIndexed { index, temp ->
                    val isHot = temp > 100
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        color = if (isHot) Color(0x33FF5252) else Color(0xFF1E1E1E),
                        shape = RoundedCornerShape(8.dp),
                        border = if (isHot) BorderStroke(1.dp, Color.Red) else null
                    ) {
                        Row(modifier = Modifier.padding(8.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                            Text("ТЭД", fontSize = 10.sp, color = Color.Gray)
                            Text("${temp.toInt()}°", fontSize = 13.sp, fontWeight = FontWeight.Bold, color = if (isHot) Color.Red else Color.White)
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun ActiveAlertsCard(alerts: List<String>) {
    if (alerts.isNotEmpty()) {
        Card(modifier = Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = Color(0xFFFF5252).copy(alpha = 0.1f)), border = BorderStroke(1.dp, Color.Red), shape = RoundedCornerShape(12.dp)) {
            Column(modifier = Modifier.padding(12.dp)) {
                Text("⚠️ ВНИМАНИЕ: ${alerts.size} СОБЫТИЙ", color = Color.Red, fontWeight = FontWeight.Black, fontSize = 12.sp)
                alerts.take(2).forEach { alert -> Text("• $alert", color = Color.White, fontSize = 13.sp, fontWeight = FontWeight.Bold) }
            }
        }
    }
}

@Composable
fun TrendsSectionLarge(label: String, data: List<Float>, color: Color) {
    Card(modifier = Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = Color(0xFF1E1E1E)), shape = RoundedCornerShape(16.dp)) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(label, fontSize = 11.sp, color = Color.Gray, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(12.dp))
            AndroidView(
                factory = { context ->
                    LineChart(context).apply {
                        description.isEnabled = false
                        xAxis.textColor = android.graphics.Color.GRAY
                        axisLeft.textColor = android.graphics.Color.WHITE
                        axisRight.isEnabled = false
                        legend.isEnabled = false
                        setBackgroundColor(android.graphics.Color.TRANSPARENT)
                    }
                },
                update = { chart ->
                    val entries = data.takeLast(40).mapIndexed { i, v -> Entry(i.toFloat(), v) }
                    chart.data = LineData(LineDataSet(entries, "").apply {
                        this.color = color.hashCode()
                        lineWidth = 2.5f
                        setDrawCircles(false)
                        setDrawValues(false)
                        mode = LineDataSet.Mode.CUBIC_BEZIER
                        setDrawFilled(true)
                        fillAlpha = 40
                    })
                    chart.invalidate()
                },
                modifier = Modifier.fillMaxWidth().height(120.dp)
            )
        }
    }
}
