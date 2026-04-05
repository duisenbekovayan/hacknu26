package com.example.mobile.ui.dashboard

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Info
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.unit.IntSize
import com.example.mobile.dashboard.DashboardViewModel
import com.example.mobile.data.TelemetrySample
import com.example.mobile.data.AIResponse
import com.github.mikephil.charting.charts.LineChart
import com.github.mikephil.charting.data.Entry
import com.github.mikephil.charting.data.LineData
import com.github.mikephil.charting.data.LineDataSet
import java.util.Locale

// Design Colors
val DarkBg = Color(0xFF0D1117)
val CardBg = Color(0xFF161B22)
val BorderColor = Color(0xFF30363D)
val AccentGreen = Color(0xFF3FB950)
val AccentBlue = Color(0xFF2F81F7)
val TextSecondary = Color(0xFF8B949E)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(viewModel: DashboardViewModel) {
    val telemetry by viewModel.telemetry.observeAsState()
    val history by viewModel.history.observeAsState(emptyList())
    val aiResponse by viewModel.aiResponse.observeAsState()
    val isAILoading by viewModel.isAILoading.observeAsState(false)

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(24.dp)
                                .background(AccentBlue, RoundedCornerShape(4.dp)),
                            contentAlignment = Alignment.Center
                        ) {
                            Text("◆", color = Color.White, fontWeight = FontWeight.Bold)
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                        Column {
                            Text("Цифровой двойник локомотива", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = Color.White)
                            Text("Телеметрия в реальном времени · индекс здоровья", fontSize = 11.sp, color = TextSecondary)
                        }
                    }
                },
                actions = {
                    Column(horizontalAlignment = Alignment.End, modifier = Modifier.padding(end = 16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("Поезд", fontSize = 10.sp, color = TextSecondary)
                            Spacer(modifier = Modifier.width(8.dp))
                            Surface(color = Color.Black, shape = RoundedCornerShape(4.dp), border = BorderStroke(1.dp, BorderColor)) {
                                Text(telemetry?.trainID ?: "---", color = Color.White, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp))
                            }
                            Spacer(modifier = Modifier.width(8.dp))
                            Text("Тяга", fontSize = 10.sp, color = TextSecondary)
                            Spacer(modifier = Modifier.width(4.dp))
                            Surface(color = Color(0xFF1B4721), shape = RoundedCornerShape(4.dp)) {
                                Text("онлайн", color = AccentGreen, fontSize = 10.sp, fontWeight = FontWeight.Bold, modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp))
                            }
                        }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = DarkBg)
            )
        },
        containerColor = DarkBg
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // AI Assistant
            item { 
                AIAssistantBanner(
                    isAILoading = isAILoading,
                    onExplainClick = { viewModel.analyzeWithAI("") },
                    onActionsClick = { viewModel.analyzeWithAI("actions") }
                ) 
            }

            // AI Result Panel (New)
            aiResponse?.let { response ->
                item {
                    AIResponsePanel(response) { viewModel.clearAIResponse() }
                }
            }

            // Main Content Area (Sidebar + Grid)
            item {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    // Left sidebar-like column
                    Column(modifier = Modifier.width(140.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                        HealthGauge(telemetry?.healthIndex ?: 0.0, telemetry?.healthGrade ?: "--")
                        Spacer(modifier = Modifier.height(16.dp))
                        val statusText = when {
                            (telemetry?.healthIndex ?: 0.0) >= 85 -> "Норма"
                            (telemetry?.healthIndex ?: 0.0) >= 60 -> "Внимание"
                            else -> "Критично"
                        }
                        Text("$statusText - индекс ${String.format(Locale.ROOT, "%.1f", telemetry?.healthIndex ?: 0.0)}", fontSize = 11.sp, color = TextSecondary)
                        Spacer(modifier = Modifier.height(24.dp))
                        
                        val factors = telemetry?.healthTopFactors ?: emptyList()
                        if (factors.isEmpty()) {
                            Text("Нет активных штрафов", fontSize = 11.sp, color = TextSecondary, textAlign = TextAlign.Center)
                        } else {
                            factors.forEach { factor ->
                                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                    Text(factor.factor, fontSize = 10.sp, color = TextSecondary, modifier = Modifier.weight(1f))
                                    Text("-${String.format(Locale.ROOT, "%.1f", factor.penalty)}", fontSize = 10.sp, color = TextSecondary)
                                }
                            }
                        }
                    }

                    // Parameters Grid
                    Column(modifier = Modifier.weight(1f)) {
                        SectionHeader("Параметры")
                        ParametersGrid(telemetry)
                    }
                }
            }

            // Trends
            item {
                SectionHeader("Тренды")
                TrendsSection(history)
            }

            // Track Path
            item {
                SectionHeader("Участок пути")
                TrackPathSection(telemetry)
            }

            // Traction Motors
            item {
                SectionHeader("ТЭД (6 осей)")
                TractionMotorsGrid(telemetry?.tractionMotorTempC ?: emptyList())
            }

            // Alerts and Recommendations
            item {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    Column(modifier = Modifier.weight(1f)) {
                        SectionHeader("Алерты")
                        AlertsSection(telemetry?.alerts?.map { (if (it.code.isNotEmpty()) "[${it.code}] " else "") + it.text } ?: emptyList())
                    }
                    Column(modifier = Modifier.weight(1f)) {
                        SectionHeader("Рекомендации действий")
                        RecommendationsSection(telemetry)
                    }
                }
            }
            
            item { Spacer(modifier = Modifier.height(32.dp)) }
        }
    }
}

@Composable
fun SectionHeader(title: String) {
    Text(
        text = title.uppercase(),
        color = TextSecondary,
        fontSize = 11.sp,
        fontWeight = FontWeight.Bold,
        modifier = Modifier.padding(bottom = 8.dp)
    )
}

@Composable
fun AIAssistantBanner(
    isAILoading: Boolean,
    onExplainClick: () -> Unit,
    onActionsClick: () -> Unit
) {
    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text("ИИ-помощник машиниста", color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                val statusText = if (isAILoading) "Запрос к модели..." else "ИИ готов (on-demand)."
                Text(statusText, color = if (isAILoading) AccentBlue else TextSecondary, fontSize = 11.sp)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = onExplainClick,
                    enabled = !isAILoading,
                    colors = ButtonDefaults.buttonColors(containerColor = AccentBlue),
                    shape = RoundedCornerShape(4.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
                    modifier = Modifier.height(32.dp)
                ) {
                    if (isAILoading) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), color = Color.White, strokeWidth = 2.dp)
                    } else {
                        Text("Объяснить состояние", fontSize = 12.sp, color = Color.White)
                    }
                }
                OutlinedButton(
                    onClick = onActionsClick,
                    enabled = !isAILoading,
                    border = BorderStroke(1.dp, BorderColor),
                    shape = RoundedCornerShape(4.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
                    modifier = Modifier.height(32.dp)
                ) {
                    Text("Что делать?", fontSize = 12.sp, color = TextSecondary)
                }
            }
        }
    }
}

@Composable
fun AIResponsePanel(response: AIResponse, onDismiss: () -> Unit) {
    Surface(
        color = Color(0xFF1C2128),
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, if (response.severity == "critical") Color(0xFFF85149) else BorderColor),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                val sevColor = when (response.severity) {
                    "critical" -> Color(0xFFF85149)
                    "warning" -> Color(0xFFD29922)
                    else -> AccentGreen
                }
                Box(modifier = Modifier.size(8.dp).background(sevColor, CircleShape))
                Spacer(modifier = Modifier.width(8.dp))
                Text(response.severity.uppercase(), color = sevColor, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.weight(1f))
                IconButton(onClick = onDismiss, modifier = Modifier.size(24.dp)) {
                    Icon(Icons.Default.Info, contentDescription = "Close", tint = TextSecondary, modifier = Modifier.size(16.dp))
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(response.summary, color = Color.White, fontSize = 14.sp)
            
            if (response.probableCauses.isNotEmpty()) {
                Spacer(modifier = Modifier.height(12.dp))
                Text("Вероятные причины:", color = TextSecondary, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                response.probableCauses.forEach { cause ->
                    Text("• $cause", color = Color.White, fontSize = 12.sp)
                }
            }
            
            if (response.recommendations.isNotEmpty()) {
                Spacer(modifier = Modifier.height(12.dp))
                Text("Рекомендации:", color = TextSecondary, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                response.recommendations.forEach { rec ->
                    Text("• $rec", color = AccentGreen, fontSize = 12.sp)
                }
            }
        }
    }
}

@Composable
fun HealthGauge(healthIndex: Double, grade: String) {
    val gaugeColor = when {
        healthIndex >= 85 -> AccentGreen
        healthIndex >= 60 -> Color(0xFFD29922)
        else -> Color(0xFFF85149)
    }
    Box(contentAlignment = Alignment.Center, modifier = Modifier.size(120.dp)) {
        Canvas(modifier = Modifier.fillMaxSize()) {
            drawArc(Color(0xFF1C2128), 0f, 360f, false, style = Stroke(12.dp.toPx(), cap = StrokeCap.Round))
            drawArc(gaugeColor, -90f, (healthIndex / 100f * 360f).toFloat(), false, style = Stroke(12.dp.toPx(), cap = StrokeCap.Round))
        }
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(String.format(Locale.ROOT, "%.0f", healthIndex), fontSize = 32.sp, fontWeight = FontWeight.Black, color = Color.White)
            Text(grade, fontSize = 14.sp, color = TextSecondary)
        }
    }
}

@Composable
fun ParametersGrid(telemetry: TelemetrySample?) {
    val items = listOf(
        "СКОРОСТЬ" to (String.format(Locale.ROOT, "%.1f км/ч", telemetry?.speedKmh ?: 0.0)),
        "ТОПЛИВО" to (String.format(Locale.ROOT, "%.0f л", telemetry?.fuelLevelL ?: 0.0)),
        "РАСХОД" to (String.format(Locale.ROOT, "%.1f л/ч", telemetry?.fuelRateLph ?: 0.0)),
        "ТОРМ. МАГИСТРАЛЬ" to (String.format(Locale.ROOT, "%.2f бар", telemetry?.brakePipePressureBar ?: 0.0)),
        "ГЛАВНЫЙ РЕЗЕРВУАР" to (String.format(Locale.ROOT, "%.2f бар", telemetry?.mainReservoirBar ?: 0.0)),
        "ДАВЛ. МАСЛА" to (String.format(Locale.ROOT, "%.2f бар", telemetry?.engineOilPressureBar ?: 0.0)),
        "ОЖ" to (String.format(Locale.ROOT, "%.1f °C", telemetry?.coolantTempC ?: 0.0)),
        "МАСЛО ДВИГ." to (String.format(Locale.ROOT, "%.1f °C", telemetry?.engineOilTempC ?: 0.0)),
        "АКБ" to (String.format(Locale.ROOT, "%.1f В", telemetry?.batteryVoltageV ?: 0.0)),
        "ТОК ТЯГИ" to ("${telemetry?.tractionCurrentA ?: "--"} А"),
        "КОНТАКТНАЯ СЕТЬ" to ("${telemetry?.lineVoltageV ?: "--"} В"),
        "ПРОБЕГ (уч.)" to (String.format(Locale.ROOT, "%.3f км", telemetry?.mileageKm ?: 0.0))
    )

    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor)
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            items.chunked(6).forEach { rowItems ->
                Row(modifier = Modifier.fillMaxWidth()) {
                    rowItems.forEach { (label, value) ->
                        ParameterItem(label, value, modifier = Modifier.weight(1f))
                    }
                }
                if (items.indexOf(rowItems.last()) < items.size - 1) {
                    Spacer(modifier = Modifier.height(12.dp))
                }
            }
        }
    }
}

@Composable
fun ParameterItem(label: String, value: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier.padding(4.dp)) {
        Text(label, fontSize = 9.sp, color = TextSecondary, fontWeight = FontWeight.Bold)
        Text(value, fontSize = 15.sp, color = Color.White, fontWeight = FontWeight.Bold)
    }
}

@Composable
fun TrendsSection(history: List<TelemetrySample>) {
    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Режим:", fontSize = 11.sp, color = TextSecondary)
                Spacer(modifier = Modifier.width(8.dp))
                RadioButton(selected = true, onClick = {}, colors = RadioButtonDefaults.colors(selectedColor = AccentBlue))
                Text("Эфир", fontSize = 11.sp, color = Color.White)
                Spacer(modifier = Modifier.width(12.dp))
                RadioButton(selected = false, onClick = {}, colors = RadioButtonDefaults.colors(unselectedColor = TextSecondary))
                Text("Replay", fontSize = 11.sp, color = TextSecondary)
                Spacer(modifier = Modifier.weight(1f))
                Surface(color = DarkBg, shape = RoundedCornerShape(4.dp), border = BorderStroke(1.dp, BorderColor)) {
                    Text("15 мин", fontSize = 11.sp, color = Color.White, modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp))
                }
            }
            Spacer(modifier = Modifier.height(16.dp))
            TrendChart("Скорость", history.map { it.speedKmh.toFloat() }, AccentBlue)
            Spacer(modifier = Modifier.height(16.dp))
            TrendChart("Температура ОЖ", history.map { it.coolantTempC.toFloat() }, Color(0xFFD29922))
        }
    }
}

@Composable
fun TrendChart(label: String, data: List<Float>, color: Color) {
    Column {
        Text(label, fontSize = 10.sp, color = TextSecondary, modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        AndroidView(
            factory = { context ->
                LineChart(context).apply {
                    description.isEnabled = false
                    xAxis.textColor = android.graphics.Color.GRAY
                    xAxis.setDrawGridLines(true)
                    xAxis.gridColor = android.graphics.Color.parseColor("#30363D")
                    axisLeft.textColor = android.graphics.Color.GRAY
                    axisLeft.setDrawGridLines(true)
                    axisLeft.gridColor = android.graphics.Color.parseColor("#30363D")
                    axisRight.isEnabled = false
                    legend.isEnabled = false
                }
            },
            update = { chart ->
                val entries = data.takeLast(50).mapIndexed { i, v -> Entry(i.toFloat(), v) }
                chart.data = LineData(LineDataSet(entries, "").apply {
                    this.color = color.hashCode()
                    setDrawCircles(false)
                    setDrawValues(false)
                    lineWidth = 2f
                    mode = LineDataSet.Mode.CUBIC_BEZIER
                })
                chart.invalidate()
            },
            modifier = Modifier.fillMaxWidth().height(100.dp)
        )
    }
}

@Composable
fun TrackPathSection(telemetry: TelemetrySample?) {
    var size by remember { mutableStateOf(IntSize.Zero) }

    val loopKm = 1650f
    val settlements = listOf(
        0f to "Астана-1",
        220f to "Караганды-Сорт.",
        270f to "Караганды-Пасс.",
        406f to "Жарык",
        476f to "Акадыр",
        632f to "Мойынты",
        740f to "Сары-Шаган",
        881f to "Шыганак",
        1095f to "Шу",
        1227f to "Турксиб",
        1353f to "Тараз",
        1476f to "Боранды",
        1535f to "Тюлькубас",
        1608f to "Манкент",
        1650f to "Шымкент"
    )
    
    val segments = listOf(
        Triple(0f, 220f, 90),
        Triple(220f, 270f, 85),
        Triple(270f, 406f, 90),
        Triple(406f, 476f, 90),
        Triple(476f, 632f, 95),
        Triple(632f, 740f, 90),
        Triple(740f, 881f, 90),
        Triple(881f, 1095f, 95),
        Triple(1095f, 1227f, 90),
        Triple(1227f, 1353f, 90),
        Triple(1353f, 1476f, 85),
        Triple(1476f, 1535f, 80),
        Triple(1535f, 1608f, 85),
        Triple(1608f, 1650f, 70)
    )

    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Маршрут Астана-1 — Шымкент. Состояние путей на участке.", fontSize = 11.sp, color = TextSecondary)
            Spacer(modifier = Modifier.height(12.dp))

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(60.dp)
                    .onGloballyPositioned { size = it.size }
            ) {
                Canvas(modifier = Modifier.fillMaxSize()) {
                    val width = size.width.toFloat()
                    val height = size.height.toFloat()
                    val railY = height * 0.6f

                    // Draw speed segments
                    segments.forEach { (from, to, vMax) ->
                        val x0 = (from / loopKm) * width
                        val x1 = (to / loopKm) * width
                        val color = when {
                            vMax >= 85 -> Color(0x593FB950)
                            vMax >= 60 -> Color(0x5958A6FF)
                            vMax >= 45 -> Color(0x59D29922)
                            else -> Color(0x47F85149)
                        }
                        drawRect(color = color, topLeft = androidx.compose.ui.geometry.Offset(x0, railY - 10.dp.toPx()), size = androidx.compose.ui.geometry.Size(x1 - x0, 20.dp.toPx()))
                    }

                    // Draw rail lines
                    drawLine(Color(0xFF30363D), androidx.compose.ui.geometry.Offset(0f, railY - 2.dp.toPx()), androidx.compose.ui.geometry.Offset(width, railY - 2.dp.toPx()), 1.dp.toPx())
                    drawLine(Color(0xFF30363D), androidx.compose.ui.geometry.Offset(0f, railY + 2.dp.toPx()), androidx.compose.ui.geometry.Offset(width, railY + 2.dp.toPx()), 1.dp.toPx())

                    // Draw stations
                    settlements.forEach { (km, name) ->
                        val x = (km / loopKm) * width
                        drawCircle(Color.White, 3.dp.toPx(), androidx.compose.ui.geometry.Offset(x, railY))
                    }

                    // Indicator for current position
                    val mileage = (telemetry?.mileageKm ?: 0.0).toFloat()
                    val currentKm = ((mileage % loopKm) + loopKm) % loopKm
                    val positionX = (currentKm / loopKm) * width

                    val spd = (telemetry?.speedKmh ?: 0.0).toFloat()
                    val currentSegment = segments.find { currentKm >= it.first && currentKm < it.second }
                    val vMax = currentSegment?.third ?: 90
                    val isOver = spd > vMax + 2

                    drawCircle(
                        color = if (isOver) Color(0xFFF85149) else Color.White,
                        radius = 6.dp.toPx(),
                        center = androidx.compose.ui.geometry.Offset(positionX, railY)
                    )
                }
            }

            Spacer(modifier = Modifier.height(8.dp))
            val mileage = (telemetry?.mileageKm ?: 0.0).toFloat()
            val currentKm = ((mileage % loopKm) + loopKm) % loopKm
            val currentSegment = segments.find { currentKm >= it.first && currentKm < it.second }
            val vMax = currentSegment?.third ?: 90
            
            Text(
                "${telemetry?.lat ?: "--"}, ${telemetry?.lon ?: "--"} · путь ${String.format(Locale.ROOT, "%.2f", currentKm)} км · Vmax $vMax · ход ${String.format(Locale.ROOT, "%.1f", telemetry?.speedKmh ?: 0.0)} км/ч",
                fontSize = 12.sp,
                color = if ( (telemetry?.speedKmh ?: 0.0) > vMax + 2) Color(0xFFF85149) else Color.White,
                fontWeight = FontWeight.Bold
            )
        }
    }
}

@Composable
fun TractionMotorsGrid(temps: List<Double>) {
    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(modifier = Modifier.padding(12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            (0..5).forEach { i ->
                val temp = temps.getOrNull(i) ?: 0.0
                Column(modifier = Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("ТЭД ${i + 1}", fontSize = 9.sp, color = TextSecondary)
                    Text("${temp.toInt()}°", fontSize = 14.sp, fontWeight = FontWeight.Bold, color = Color.White)
                }
            }
        }
    }
}

@Composable
fun AlertsSection(alerts: List<String>) {
    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth().height(100.dp)
    ) {
        if (alerts.isEmpty()) {
            Box(contentAlignment = Alignment.Center) {
                Text("нет активных", color = TextSecondary, fontSize = 12.sp)
            }
        } else {
            LazyColumn(modifier = Modifier.padding(8.dp)) {
                items(alerts) { alert ->
                    Text("• $alert", color = Color.White, fontSize = 12.sp, modifier = Modifier.padding(vertical = 2.dp))
                }
            }
        }
    }
}

@Composable
fun RecommendationsSection(telemetry: TelemetrySample?) {
    Surface(
        color = CardBg,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, BorderColor),
        modifier = Modifier.fillMaxWidth().height(100.dp)
    ) {
        val recommendations = mutableListOf<String>()
        val hi = telemetry?.healthIndex ?: 100.0
        
        telemetry?.healthTopFactors?.forEach { f ->
            val name = f.factor
            when {
                name.contains("перегрев ОЖ", ignoreCase = true) -> recommendations.add("Снизить нагрузку двигателя; проверить охлаждение и уровень ОЖ.")
                name.contains("низкое давление масла", ignoreCase = true) -> recommendations.add("Не наращивать тягу; проверить уровень и давление масла по регламенту.")
                name.contains("напряжение АКБ", ignoreCase = true) -> recommendations.add("Проверить заряд АКБ и цепи генератора/выпрямителя.")
                name.contains("ТЭД", ignoreCase = true) -> recommendations.add("Проверить вентиляцию и нагрузку на тяговые двигатели.")
                name.contains("низкое давление ГР", ignoreCase = true) -> recommendations.add("Проверить питание сжатым воздухом и тормозную магистраль.")
            }
        }
        
        telemetry?.alerts?.forEach { a ->
            when (a.severity.lowercase()) {
                "crit" -> recommendations.add("Критический сигнал ${a.code}: быть готовым к остановке или снижению хода.")
                "warn" -> recommendations.add("Предупреждение ${a.code}: проверить узел до усугубления.")
            }
        }

        if (recommendations.isEmpty()) {
            if (hi >= 85) {
                recommendations.add("Состояние в норме; продолжайте мониторинг ключевых параметров.")
            } else {
                recommendations.add("Удерживайте параметры из списка факторов под контролем.")
            }
        }

        LazyColumn(modifier = Modifier.padding(12.dp)) {
            items(recommendations.distinct()) { rec ->
                Text("• $rec", color = TextSecondary, fontSize = 12.sp, modifier = Modifier.padding(vertical = 2.dp))
            }
        }
    }
}

