package com.example.mobile.ui.dashboard

import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Info
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.livedata.observeAsState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.nativeCanvas
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalConfiguration
import com.example.mobile.dashboard.DashboardViewModel
import com.example.mobile.data.TelemetrySample
import com.example.mobile.data.AIResponse
import com.example.mobile.data.Alert
import com.github.mikephil.charting.charts.LineChart
import com.github.mikephil.charting.data.Entry
import com.github.mikephil.charting.data.LineData
import com.github.mikephil.charting.data.LineDataSet
import com.github.mikephil.charting.formatter.ValueFormatter
import kotlinx.coroutines.delay
import java.text.SimpleDateFormat
import java.util.Date
import java.util.TimeZone
import java.util.Locale

@Composable
fun SectionHeader(title: String) {
    Text(
        text = title,
        color = MaterialTheme.colorScheme.onBackground,
        fontSize = 17.sp,
        fontWeight = FontWeight.Bold,
        modifier = Modifier.padding(bottom = 12.dp)
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(viewModel: DashboardViewModel) {
    val telemetryLive by viewModel.telemetry.observeAsState()
    val history by viewModel.history.observeAsState(emptyList())
    val aiResponse by viewModel.aiResponse.observeAsState()
    val isAILoading by viewModel.isAILoading.observeAsState(false)
    val isDarkTheme by viewModel.isDarkTheme.observeAsState()

    var viewMode by remember { mutableStateOf("live") }
    var scrubIndex by remember { mutableFloatStateOf(0f) }
    var isPlaying by remember { mutableStateOf(false) }

    val trackScrollState = rememberScrollState()
    var trackViewportWidth by remember { mutableIntStateOf(0) }
    val density = LocalDensity.current
    val totalWidthDp = 5000.dp
    val horizontalPaddingDp = 80.dp

    val effectiveTelemetry = if (viewMode == "replay" && history.isNotEmpty()) {
        val idx = scrubIndex.toInt().coerceIn(0, history.lastIndex)
        history[idx]
    } else {
        telemetryLive
    }

    val formattedTime = remember(effectiveTelemetry?.ts) {
        val ts = effectiveTelemetry?.ts ?: ""
        if (ts.isEmpty()) "--.--.---- --:--:--"
        else {
            try {
                val inputFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.ROOT).apply {
                    timeZone = TimeZone.getTimeZone("UTC")
                }
                val outputFormat = SimpleDateFormat("dd.MM.yyyy HH:mm:ss", Locale.getDefault())
                
                val cleanTs = ts.replace("Z", "")
                val date = inputFormat.parse(cleanTs)
                date?.let { outputFormat.format(it) } ?: ts
            } catch (e: Exception) {
                ts
            }
        }
    }

    LaunchedEffect(effectiveTelemetry?.mileageKm, trackViewportWidth) {
        if (trackViewportWidth > 0 && effectiveTelemetry != null) {
            val mileage = effectiveTelemetry.mileageKm.toFloat()
            val loopKm = 1500f
            val currentKm = ((mileage % loopKm) + loopKm) % loopKm
            
            val totalWidthPx = with(density) { totalWidthDp.toPx() }
            val paddingPx = with(density) { horizontalPaddingDp.toPx() }
            val contentWidthPx = totalWidthPx - 2 * paddingPx
            
            val posX = (currentKm / loopKm) * contentWidthPx + paddingPx
            val targetScroll = (posX - trackViewportWidth / 2).coerceIn(0f, totalWidthPx - trackViewportWidth)
            
            trackScrollState.animateScrollTo(targetScroll.toInt())
        }
    }

    LaunchedEffect(isPlaying, viewMode, history.size) {
        if (isPlaying && viewMode == "replay" && history.isNotEmpty()) {
            while (isPlaying) {
                delay(1000)
                if (scrubIndex < history.size - 1) {
                    scrubIndex += 1
                } else {
                    isPlaying = false
                }
            }
        }
    }


    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(24.dp)
                                .background(MaterialTheme.colorScheme.primary, RoundedCornerShape(4.dp)),
                            contentAlignment = Alignment.Center
                        ) {
                            Text("◆", color = Color.White, fontWeight = FontWeight.Bold)
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                        Column {
                            Text("Цифровой двойник локомотива", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onBackground)
                            Text("Телеметрия в реальном времени · индекс здоровья", fontSize = 11.sp, color = MaterialTheme.colorScheme.outlineVariant)
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.toggleTheme() }) {
                        Text(if (isDarkTheme == true) "🌙" else "☀️", fontSize = 20.sp)
                    }
                    Column(horizontalAlignment = Alignment.End, modifier = Modifier.padding(end = 16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("Поезд", fontSize = 10.sp, color = MaterialTheme.colorScheme.outlineVariant)
                            Spacer(modifier = Modifier.width(8.dp))
                            Surface(color = Color.Black, shape = RoundedCornerShape(4.dp), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                                Text(effectiveTelemetry?.trainID ?: "---", color = Color.White, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp))
                            }
                            Spacer(modifier = Modifier.width(8.dp))
                            Text("Тяга", fontSize = 10.sp, color = MaterialTheme.colorScheme.outlineVariant)
                            Spacer(modifier = Modifier.width(4.dp))
                            Surface(color = if (viewMode == "live") Color(0xFF1B4721) else MaterialTheme.colorScheme.surfaceVariant, shape = RoundedCornerShape(4.dp)) {
                                Text(if (viewMode == "live") "онлайн" else "replay", color = if (viewMode == "live") Color.White else MaterialTheme.colorScheme.primary, fontSize = 10.sp, fontWeight = FontWeight.Bold, modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp))
                            }
                        }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            item {
                AIAssistantBanner(
                    isAILoading = isAILoading,
                    onExplainClick = { viewModel.analyzeWithAI("explain", effectiveTelemetry) },
                    onActionsClick = { viewModel.analyzeWithAI("actions", effectiveTelemetry) }
                ) 
            }

            if (aiResponse != null) {
                item {
                    AIResponsePanel(aiResponse!!, onDismiss = { viewModel.clearAIResponse() })
                }
            }

            item {
                ReplayControls(
                    viewMode = viewMode,
                    onViewModeChange = { 
                        viewMode = it
                        if (it == "live") isPlaying = false
                    },
                    scrubIndex = scrubIndex,
                    onScrubIndexChange = { 
                        scrubIndex = it
                        isPlaying = false 
                    },
                    isPlaying = isPlaying,
                    onIsPlayingChange = { isPlaying = it },
                    historySize = history.size,
                    currentTime = formattedTime
                )
            }

            item {
                val config = LocalConfiguration.current
                val isTablet = config.screenWidthDp >= 600

                if (isTablet) {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        Column(modifier = Modifier.weight(1.6f)) {
                            Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                                Column(modifier = Modifier.width(140.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                    HealthGauge(effectiveTelemetry?.healthIndex ?: 0.0, effectiveTelemetry?.healthGrade ?: "--")
                                    Spacer(modifier = Modifier.height(12.dp))
                                    HealthStatusText(effectiveTelemetry)
                                    Spacer(modifier = Modifier.height(16.dp))
                                    HealthFactorsList(effectiveTelemetry)
                                }

                                Column(modifier = Modifier.weight(1f)) {
                                    SectionHeader("Параметры")
                                    ParametersGrid(effectiveTelemetry, columns = 2)
                                }
                            }
                        }

                        Column(modifier = Modifier.weight(1f)) {
                            SectionHeader("Алерты")
                            AlertsSection(effectiveTelemetry?.alerts ?: emptyList())
                            Spacer(modifier = Modifier.height(16.dp))
                            SectionHeader("Рекомендации")
                            val recs = remember(effectiveTelemetry) { getRecommendations(effectiveTelemetry) }
                            RecommendationsSection(recs)
                        }
                    }
                } else {
                    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                            Column(modifier = Modifier.width(130.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                HealthGauge(effectiveTelemetry?.healthIndex ?: 0.0, effectiveTelemetry?.healthGrade ?: "--", size = 130.dp)
                                Spacer(modifier = Modifier.height(8.dp))
                                HealthStatusText(effectiveTelemetry)
                            }
                            Column(modifier = Modifier.weight(1f)) {
                                SectionHeader("Основные параметры")
                                ParametersGrid(effectiveTelemetry, columns = 2)
                            }
                        }
                        
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                            Column(modifier = Modifier.weight(1f)) {
                                SectionHeader("Алерты")
                                AlertsSection(effectiveTelemetry?.alerts ?: emptyList(), height = 140.dp)
                            }
                            Column(modifier = Modifier.weight(1f)) {
                                SectionHeader("Рекомендации")
                                val recs = remember(effectiveTelemetry) { getRecommendations(effectiveTelemetry) }
                                RecommendationsSection(recs, height = 140.dp)
                            }
                        }
                    }
                }
            }

            item {
                SectionHeader("ТЭД (6 осей)")
                TractionMotorsGrid(effectiveTelemetry?.tractionMotorTempC ?: emptyList())
            }

            item {
                TrendsSection(history)
            }

            item {
                SectionHeader("Участок пути")
                TrackPathSection(effectiveTelemetry, trackScrollState, totalWidthDp, horizontalPaddingDp) { width ->
                    trackViewportWidth = width
                }
            }
            
            item { Spacer(modifier = Modifier.height(32.dp)) }
        }
    }
}

@Composable
fun ReplayControls(
    viewMode: String,
    onViewModeChange: (String) -> Unit,
    scrubIndex: Float,
    onScrubIndexChange: (Float) -> Unit,
    isPlaying: Boolean,
    onIsPlayingChange: (Boolean) -> Unit,
    historySize: Int,
    currentTime: String
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Text("Режим:", fontSize = 11.sp, color = MaterialTheme.colorScheme.outlineVariant)
                Spacer(modifier = Modifier.width(8.dp))
                
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.clickable { onViewModeChange("live") }) {
                    RadioButton(
                        selected = viewMode == "live",
                        onClick = { onViewModeChange("live") },
                        colors = RadioButtonDefaults.colors(selectedColor = MaterialTheme.colorScheme.primary)
                    )
                    Text("Эфир", fontSize = 11.sp, color = if (viewMode == "live") MaterialTheme.colorScheme.onBackground else MaterialTheme.colorScheme.outlineVariant)
                }
                
                Spacer(modifier = Modifier.width(8.dp))
                
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.clickable { onViewModeChange("replay") }) {
                    RadioButton(
                        selected = viewMode == "replay",
                        onClick = { onViewModeChange("replay") },
                        colors = RadioButtonDefaults.colors(selectedColor = MaterialTheme.colorScheme.primary)
                    )
                    Text("Replay", fontSize = 11.sp, color = if (viewMode == "replay") MaterialTheme.colorScheme.onBackground else MaterialTheme.colorScheme.outlineVariant)
                }

                if (viewMode == "replay") {
                    Spacer(modifier = Modifier.weight(1f))
                    IconButton(
                        onClick = { onIsPlayingChange(!isPlaying) },
                        modifier = Modifier.size(32.dp)
                    ) {
                        Text(
                            if (isPlaying) "⏸" else "▶",
                            fontSize = 14.sp,
                            color = MaterialTheme.colorScheme.primary
                        )
                    }
                }
            }
            
            if (viewMode == "replay") {
                Spacer(modifier = Modifier.height(8.dp))
                Slider(
                    value = scrubIndex,
                    onValueChange = onScrubIndexChange,
                    valueRange = 0f..historySize.coerceAtLeast(1).minus(1).toFloat(),
                    colors = SliderDefaults.colors(thumbColor = MaterialTheme.colorScheme.primary, activeTrackColor = MaterialTheme.colorScheme.primary)
                )
                Text(currentTime, fontSize = 10.sp, color = MaterialTheme.colorScheme.onBackground, textAlign = TextAlign.End, modifier = Modifier.fillMaxWidth())
            }
        }
    }
}

@Composable
fun AIAssistantBanner(
    isAILoading: Boolean,
    onExplainClick: () -> Unit,
    onActionsClick: () -> Unit
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text("ИИ-помощник машиниста", color = MaterialTheme.colorScheme.onBackground, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                val statusText = if (isAILoading) "Запрос к модели..." else "ИИ готов (on-demand)."
                Text(statusText, color = if (isAILoading) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant, fontSize = 11.sp)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = onExplainClick,
                    enabled = !isAILoading,
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
                    shape = RoundedCornerShape(6.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
                    modifier = Modifier.height(36.dp)
                ) {
                    if (isAILoading) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), color = Color.White, strokeWidth = 2.dp)
                    } else {
                        Text("Объяснить", fontSize = 12.sp, color = Color.White)
                    }
                }
                OutlinedButton(
                    onClick = onActionsClick,
                    enabled = !isAILoading,
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.onBackground),
                    shape = RoundedCornerShape(6.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
                    modifier = Modifier.height(36.dp)
                ) {
                    Text("Что делать?", fontSize = 12.sp)
                }
            }
        }
    }
}

@Composable
fun AIResponsePanel(response: AIResponse, onDismiss: () -> Unit) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(10.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.primary),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                val sevColor = when (response.severity) {
                    "critical" -> MaterialTheme.colorScheme.error
                    "warning" -> MaterialTheme.colorScheme.tertiary
                    else -> MaterialTheme.colorScheme.secondary
                }
                val sevBg = sevColor.copy(alpha = 0.15f)
                
                Surface(color = sevBg, shape = RoundedCornerShape(6.dp)) {
                    Text(
                        text = when(response.severity) {
                            "critical" -> "КРИТИЧНО"
                            "warning" -> "ВНИМАНИЕ"
                            else -> "НОРМА"
                        },
                        color = sevColor,
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
                Spacer(modifier = Modifier.weight(1f))
                IconButton(onClick = onDismiss, modifier = Modifier.size(24.dp)) {
                    Icon(Icons.Default.Info, contentDescription = "Close", tint = MaterialTheme.colorScheme.outlineVariant, modifier = Modifier.size(16.dp))
                }
            }
            Spacer(modifier = Modifier.height(12.dp))
            Text(response.summary, color = MaterialTheme.colorScheme.onBackground, fontSize = 15.sp, lineHeight = 20.sp)
            
            Row(modifier = Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                if (response.probableCauses.isNotEmpty()) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text("Вероятные причины", color = MaterialTheme.colorScheme.outlineVariant, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                        Spacer(modifier = Modifier.height(4.dp))
                        response.probableCauses.forEach { cause ->
                            Text("• $cause", color = MaterialTheme.colorScheme.onBackground, fontSize = 13.sp, lineHeight = 18.sp)
                        }
                    }
                }
                
                if (response.recommendations.isNotEmpty()) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text("Рекомендации", color = MaterialTheme.colorScheme.outlineVariant, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                        Spacer(modifier = Modifier.height(4.dp))
                        response.recommendations.forEach { rec ->
                            Text("• $rec", color = MaterialTheme.colorScheme.onBackground, fontSize = 13.sp, lineHeight = 18.sp)
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun HealthStatusText(telemetry: TelemetrySample?) {
    val healthIndex = telemetry?.healthIndex ?: 0.0
    val statusText = when {
        healthIndex >= 85 -> "Норма"
        healthIndex >= 60 -> "Внимание"
        else -> "Критично"
    }
    Text(
        text = "$statusText - ${String.format(Locale.ROOT, "%.1f", healthIndex)}",
        fontSize = 14.sp,
        color = MaterialTheme.colorScheme.onBackground,
        fontWeight = FontWeight.Bold,
        textAlign = TextAlign.Center
    )
}

@Composable
fun HealthFactorsList(telemetry: TelemetrySample?) {
    val factors = telemetry?.healthTopFactors ?: emptyList()
    if (factors.isEmpty()) {
        Text("Нет активных штрафов", fontSize = 12.sp, color = MaterialTheme.colorScheme.outlineVariant, textAlign = TextAlign.Center)
    } else {
        factors.forEach { factor ->
            Row(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(factor.factor, fontSize = 12.sp, color = MaterialTheme.colorScheme.onBackground, modifier = Modifier.weight(1f), maxLines = 1)
                Text("-${String.format(Locale.ROOT, "%.1f", factor.penalty)}", fontSize = 12.sp, color = MaterialTheme.colorScheme.outlineVariant)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outline, thickness = 0.5.dp)
        }
    }
}

@Composable
fun HealthGauge(healthIndex: Double, grade: String, size: androidx.compose.ui.unit.Dp = 140.dp) {
    val gaugeColor = when {
        healthIndex >= 85 -> MaterialTheme.colorScheme.secondary
        healthIndex >= 60 -> MaterialTheme.colorScheme.tertiary
        else -> MaterialTheme.colorScheme.error
    }
    
    val ringTrackColor = MaterialTheme.colorScheme.surfaceVariant

    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Box(contentAlignment = Alignment.Center, modifier = Modifier.size(size)) {
            Canvas(modifier = Modifier.size(size)) {
                drawArc(
                    color = ringTrackColor,
                    startAngle = 0f,
                    sweepAngle = 360f,
                    useCenter = false,
                    style = Stroke(width = 12.dp.toPx())
                )
                drawArc(
                    color = gaugeColor,
                    startAngle = -90f,
                    sweepAngle = (healthIndex / 100f * 360f).toFloat(),
                    useCenter = false,
                    style = Stroke(width = 12.dp.toPx(), cap = StrokeCap.Round)
                )
            }
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = String.format(Locale.ROOT, "%.0f", healthIndex),
                    fontSize = if (size < 140.dp) 24.sp else 28.sp,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onBackground
                )
                Text(
                    text = grade,
                    fontSize = if (size < 140.dp) 14.sp else 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.outlineVariant
                )
            }
        }
    }
}

@Composable
fun ParametersGrid(telemetry: TelemetrySample?, columns: Int = 2) {
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

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        items.chunked(columns).forEach { rowItems ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                rowItems.forEach { (label, value) ->
                    ParameterItem(label, value, modifier = Modifier.weight(1f))
                }
            }
        }
    }
}

@Composable
fun ParameterItem(label: String, value: String, modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = modifier
    ) {
        Column(modifier = Modifier.padding(10.dp)) {
            Text(label, fontSize = 11.sp, color = MaterialTheme.colorScheme.outlineVariant, fontWeight = FontWeight.Bold, letterSpacing = 0.5.sp, maxLines = 1)
            Text(value, fontSize = 18.sp, color = MaterialTheme.colorScheme.onBackground, fontWeight = FontWeight.ExtraBold)
        }
    }
}

fun filterHistoryByMinutes(history: List<TelemetrySample>, minutes: Int): List<TelemetrySample> {
    if (minutes <= 0 || history.isEmpty()) return history
    val sdf = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.ROOT).apply {
        timeZone = TimeZone.getTimeZone("UTC")
    }
    
    val latestTs = history.last().ts
    val latestDate = try {
        sdf.parse(latestTs.replace("T", " ").replace("Z", ""))
    } catch (e: Exception) {
        null
    } ?: return history.takeLast(minutes * 60)

    val threshold = latestDate.time - (minutes.toLong() * 60 * 1000)
    
    val firstIdx = history.indexOfFirst { sample ->
        val date = try {
            sdf.parse(sample.ts.replace("T", " ").replace("Z", ""))
        } catch (e: Exception) {
            null
        }
        date != null && date.time >= threshold
    }
    
    return if (firstIdx == -1) emptyList() else history.subList(firstIdx, history.size)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TrendsSection(history: List<TelemetrySample>) {
    var timeWindow by remember { mutableStateOf(15) }

    val filteredHistory = remember(history, timeWindow) {
        filterHistoryByMinutes(history, timeWindow)
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Тренды",
                color = MaterialTheme.colorScheme.onBackground,
                fontSize = 17.sp,
                fontWeight = FontWeight.Bold
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                listOf(5, 10, 15).forEach { mins ->
                    val selected = timeWindow == mins
                    Surface(
                        onClick = { timeWindow = mins },
                        color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                        shape = RoundedCornerShape(4.dp),
                        border = BorderStroke(1.dp, if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline),
                        modifier = Modifier.height(24.dp)
                    ) {
                        Box(contentAlignment = Alignment.Center, modifier = Modifier.padding(horizontal = 8.dp)) {
                            Text("${mins}м", fontSize = 10.sp, fontWeight = FontWeight.Bold, color = if (selected) Color.White else MaterialTheme.colorScheme.onSurface)
                        }
                    }
                }
            }
        }

        Surface(
            color = MaterialTheme.colorScheme.surface,
            shape = RoundedCornerShape(8.dp),
            border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
            modifier = Modifier.fillMaxWidth()
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                TrendChart("Скорость (км/ч)", filteredHistory, { it.speedKmh.toFloat() }, MaterialTheme.colorScheme.primary)
                TrendChart("Температура ОЖ (°C)", filteredHistory, { it.coolantTempC.toFloat() }, MaterialTheme.colorScheme.tertiary)
                TrendChart("Индекс здоровья", filteredHistory, { it.healthIndex.toFloat() }, MaterialTheme.colorScheme.secondary)
                TrendChart("Топливо (л)", filteredHistory, { it.fuelLevelL.toFloat() }, MaterialTheme.colorScheme.secondaryContainer)
            }
        }
    }
}

@Composable
fun TrendChart(label: String, data: List<TelemetrySample>, selector: (TelemetrySample) -> Float, color: Color) {
    val gridColor = MaterialTheme.colorScheme.outline.toArgb()
    val textColor = MaterialTheme.colorScheme.outlineVariant.toArgb()

    Column {
        Text(label, fontSize = 10.sp, color = MaterialTheme.colorScheme.outlineVariant, modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        AndroidView(
            factory = { context ->
                LineChart(context).apply {
                    description.isEnabled = false
                    xAxis.textColor = textColor
                    xAxis.setDrawGridLines(true)
                    xAxis.gridColor = gridColor
                    xAxis.position = com.github.mikephil.charting.components.XAxis.XAxisPosition.BOTTOM
                    xAxis.granularity = 1f
                    xAxis.labelCount = 4
                    axisLeft.textColor = textColor
                    axisLeft.setDrawGridLines(true)
                    axisLeft.gridColor = gridColor
                    axisRight.isEnabled = false
                    legend.isEnabled = false
                    setTouchEnabled(true)
                    isDragEnabled = true
                    setScaleEnabled(true)
                    setPinchZoom(true)
                    setBackgroundColor(android.graphics.Color.TRANSPARENT)
                }
            },
            update = { chart ->
                val displayData = if (data.size > 1000) data.takeLast(1000) else data
                val entries = displayData.mapIndexed { i, sample -> Entry(i.toFloat(), selector(sample)) }
                
                chart.xAxis.valueFormatter = object : ValueFormatter() {
                    private val inputFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.ROOT).apply {
                        timeZone = TimeZone.getTimeZone("UTC")
                    }
                    private val outputFormat = SimpleDateFormat("HH:mm:ss", Locale.getDefault())

                    override fun getFormattedValue(value: Float): String {
                        val idx = value.toInt()
                        if (idx >= 0 && idx < displayData.size) {
                            val ts = displayData[idx].ts
                            return try {
                                val date = inputFormat.parse(ts.replace("T", " ").replace("Z", ""))
                                date?.let { outputFormat.format(it) } ?: ts
                            } catch (e: Exception) {
                                if (ts.length >= 19) ts.substring(11, 19) else ts
                            }
                        }
                        return ""
                    }
                }

                chart.data = LineData(LineDataSet(entries, "").apply {
                    this.color = color.toArgb()
                    setDrawCircles(false)
                    setDrawValues(false)
                    lineWidth = 2f
                    mode = LineDataSet.Mode.CUBIC_BEZIER
                    setDrawFilled(true)
                    fillAlpha = 20
                    fillColor = color.toArgb()
                })
                chart.invalidate()
            },
            modifier = Modifier.fillMaxWidth().height(120.dp)
        )
    }
}

@Composable
fun TrackPathSection(
    telemetry: TelemetrySample?,
    scrollState: ScrollState,
    totalWidth: androidx.compose.ui.unit.Dp,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onViewportSizeChanged: (Int) -> Unit
) {
    val loopKm = 1500f
    
    val settlements = listOf(
        0f to "Астана-1", 200f to "Караганды-Сорт.", 245f to "Караганды-Пасс.", 369f to "Жарык",
        433f to "Акадыр", 575f to "Мойынты", 673f to "Сары-Шаган", 801f to "Шыганак",
        995f to "Шу", 1115f to "Турксиб", 1230f to "Тараз", 1342f to "Боранды",
        1395f to "Тюлькубас", 1462f to "Манкент", 1500f to "Шымкент"
    )
    
    val segments = listOf(
        TrackSegment(0f, 200f, 90),
        TrackSegment(200f, 245f, 85),
        TrackSegment(245f, 369f, 90),
        TrackSegment(369f, 433f, 90),
        TrackSegment(433f, 575f, 95),
        TrackSegment(575f, 673f, 90),
        TrackSegment(673f, 801f, 90),
        TrackSegment(801f, 995f, 95, "Длинный перегон"),
        TrackSegment(995f, 1115f, 90),
        TrackSegment(1115f, 1230f, 90),
        TrackSegment(1230f, 1342f, 85),
        TrackSegment(1342f, 1395f, 80),
        TrackSegment(1395f, 1462f, 85),
        TrackSegment(1462f, 1500f, 70, "Подход к Шымкенту")
    )

    val mileage = (telemetry?.mileageKm ?: 0.0).toFloat()
    val currentKm = ((mileage % loopKm) + loopKm) % loopKm
    val currentSegment = segments.find { currentKm >= it.from && currentKm < it.to }
    val vMax = currentSegment?.vMax ?: 90
    
    val prevSettlement = settlements.lastOrNull { it.first <= currentKm } ?: settlements.first()
    val nextSettlement = settlements.find { it.first > currentKm } ?: settlements.first()
    
    val speed = (telemetry?.speedKmh ?: 0.0).toFloat()

    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Участок пути", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = MaterialTheme.colorScheme.onBackground)
                Spacer(modifier = Modifier.width(12.dp))
                Text("• схема • НП • ограничения", fontSize = 12.sp, color = MaterialTheme.colorScheme.outlineVariant)
            }
            
            Spacer(modifier = Modifier.height(12.dp))
            
            Text(
                "Маршрут Астана-1 — Шымкент (станции по расписанию на 4–5 апр.). Километраж по пути пропорционален времени в пути от отправления с Астаны; длина линии на схеме ~1500 км (ориентир по главному ходу). Положение — mileage_km по модулю длины; координаты — линейная интерполяция Астана → Шымкент.",
                fontSize = 10.sp,
                lineHeight = 14.sp,
                color = MaterialTheme.colorScheme.outlineVariant
            )

            Spacer(modifier = Modifier.height(16.dp))

            val surfaceColor = MaterialTheme.colorScheme.surface
            val onSurfaceColor = MaterialTheme.colorScheme.onSurface
            val outlineColor = MaterialTheme.colorScheme.outline
            val outlineVariantColor = MaterialTheme.colorScheme.outlineVariant
            val primaryColor = MaterialTheme.colorScheme.primary

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(140.dp)
                    .background(surfaceColor, RoundedCornerShape(4.dp))
                    .border(1.dp, outlineColor, RoundedCornerShape(4.dp))
                    .onSizeChanged { onViewportSizeChanged(it.width) }
                    .horizontalScroll(scrollState)
            ) {
                Canvas(modifier = Modifier.width(totalWidth).fillMaxHeight().padding(horizontal = horizontalPadding)) {
                    val width = size.width
                    val height = size.height
                    val railY = height * 0.45f

                    segments.forEach { seg ->
                        val x0 = (seg.from / loopKm) * width
                        val x1 = (seg.to / loopKm) * width
                        drawRect(
                            color = outlineColor.copy(alpha = 0.1f),
                            topLeft = Offset(x0, railY - 10.dp.toPx()),
                            size = Size(x1 - x0, 20.dp.toPx())
                        )
                    }

                    val railColor = onSurfaceColor.copy(alpha = 0.4f)
                    drawLine(railColor, Offset(0f, railY - 3.dp.toPx()), Offset(width, railY - 3.dp.toPx()), 1.dp.toPx())
                    drawLine(railColor, Offset(0f, railY + 3.dp.toPx()), Offset(width, railY + 3.dp.toPx()), 1.dp.toPx())

                    settlements.forEachIndexed { index, (km, name) ->
                        val x = (km / loopKm) * width
                        drawCircle(onSurfaceColor, 3.dp.toPx(), Offset(x, railY))
                        
                        drawContext.canvas.nativeCanvas.apply {
                            val paint = android.graphics.Paint().apply {
                                color = onSurfaceColor.toArgb()
                                textSize = 24f
                                isFakeBoldText = true
                                textAlign = android.graphics.Paint.Align.CENTER
                            }
                            val yOffset = -35.dp.toPx()
                            drawText(name, x, railY + yOffset, paint)
                            
                            paint.textSize = 16f
                            paint.isFakeBoldText = false
                            paint.color = outlineVariantColor.toArgb()
                            drawText("${km.toInt()} км", x, railY + yOffset + 18f, paint)
                        }
                        
                        if (index < settlements.size - 1) {
                            val nextKm = settlements[index + 1].first
                            val dist = (nextKm - km).toInt()
                            val midX = ((km + nextKm) / 2 / loopKm) * width
                            drawContext.canvas.nativeCanvas.apply {
                                val paint = android.graphics.Paint().apply {
                                    color = onSurfaceColor.toArgb()
                                    textSize = 22f
                                    isFakeBoldText = true
                                    textAlign = android.graphics.Paint.Align.CENTER
                                }
                                drawText("$dist км", midX, railY + 35.dp.toPx(), paint)
                            }
                        }
                    }

                    val posX = (currentKm / loopKm) * width
                    drawCircle(primaryColor, 8.dp.toPx(), Offset(posX, railY))
                    drawCircle(surfaceColor, 3.dp.toPx(), Offset(posX, railY))
                }
            }

            Spacer(modifier = Modifier.height(16.dp))
            
            val lat = telemetry?.lat ?: 0.0
            val lon = telemetry?.lon ?: 0.0
            Text(
                text = "${String.format(Locale.ROOT, "%.5f", lat)}°, ${String.format(Locale.ROOT, "%.5f", lon)}° · путь ${String.format(Locale.ROOT, "%.2f", currentKm)} км (цикл ${loopKm.toInt()} км) · Vmax $vMax · ход ${speed.toInt()} км/ч",
                fontSize = 14.sp,
                fontWeight = FontWeight.ExtraBold,
                color = MaterialTheme.colorScheme.onBackground
            )
            
            Text(
                text = "между ${prevSettlement.second} и ${nextSettlement.second}. ${(nextSettlement.first - prevSettlement.first).toInt()} км · Vmax $vMax км/ч",
                fontSize = 12.sp,
                color = MaterialTheme.colorScheme.outlineVariant,
                modifier = Modifier.padding(top = 4.dp)
            )

            Spacer(modifier = Modifier.height(12.dp))
            Text("Ограничения по перегонам (Vmax, условно):", fontSize = 11.sp, color = MaterialTheme.colorScheme.outlineVariant)
            Spacer(modifier = Modifier.height(8.dp))

    VmaxTagsGrid(segments, currentKm)

            Spacer(modifier = Modifier.height(12.dp))
            
            Spacer(modifier = Modifier.height(8.dp))
        }
    }
}

data class TrackSegment(
    val from: Float,
    val to: Float,
    val vMax: Int,
    val label: String? = null
)

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun VmaxTagsGrid(segments: List<TrackSegment>, currentKm: Float) {
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        segments.forEach { seg ->
            val isActive = currentKm >= seg.from && currentKm < seg.to
            
            Surface(
                color = if (isActive) MaterialTheme.colorScheme.onSurface.copy(alpha = 0.05f) else Color.Transparent,
                shape = RoundedCornerShape(6.dp),
                border = BorderStroke(
                    width = if (isActive) 1.5.dp else 1.dp,
                    color = if (isActive) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outline.copy(alpha = 0.3f)
                )
            ) {
                Text(
                    text = "${seg.from.toInt()}–${seg.to.toInt()} км · Vmax ${seg.vMax}${if (seg.label != null) " · " + seg.label else ""}",
                    fontSize = 11.sp,
                    fontWeight = if (isActive) FontWeight.Bold else FontWeight.Normal,
                    color = MaterialTheme.colorScheme.onBackground,
                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp)
                )
            }
        }
    }
}

@Composable
fun LegendItem(text: String, color: Color) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(modifier = Modifier.size(12.dp, 4.dp).background(color, RoundedCornerShape(1.dp)))
        Spacer(modifier = Modifier.width(6.dp))
        Text(text, fontSize = 10.sp, color = MaterialTheme.colorScheme.outlineVariant)
    }
}


@Composable
fun TractionMotorsGrid(temps: List<Double>) {
    val config = LocalConfiguration.current
    val cols = if (config.screenWidthDp >= 600) 6 else 3
    
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            (0..5).chunked(cols).forEach { rowIndices ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    rowIndices.forEach { i ->
                        val temp = temps.getOrNull(i) ?: 0.0
                        Surface(
                            modifier = Modifier.weight(1f),
                            color = when {
                                temp > 115 -> MaterialTheme.colorScheme.error.copy(alpha = 0.1f)
                                temp > 105 -> MaterialTheme.colorScheme.tertiary.copy(alpha = 0.1f)
                                else -> MaterialTheme.colorScheme.surface
                            },
                            shape = RoundedCornerShape(8.dp),
                            border = BorderStroke(1.dp, when {
                                temp > 115 -> MaterialTheme.colorScheme.error
                                temp > 105 -> MaterialTheme.colorScheme.tertiary
                                else -> MaterialTheme.colorScheme.outline
                            })
                        ) {
                            Column(modifier = Modifier.padding(vertical = 8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                Text("ТЭД ${i + 1}", fontSize = 10.sp, color = MaterialTheme.colorScheme.outlineVariant)
                                Text("${temp.toInt()}°", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onBackground)
                            }
                        }
                    }
                    if (rowIndices.size < cols) {
                        repeat(cols - rowIndices.size) {
                            Spacer(modifier = Modifier.weight(1f))
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun AlertsSection(alerts: List<Alert>, height: androidx.compose.ui.unit.Dp = 160.dp) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth().height(height)
    ) {
        if (alerts.isEmpty()) {
            Box(contentAlignment = Alignment.Center) {
                Text("нет активных", color = MaterialTheme.colorScheme.outlineVariant, fontSize = 12.sp)
            }
        } else {
            LazyColumn(modifier = Modifier.padding(8.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                items(alerts) { alert ->
                    val borderColor = when (alert.severity.lowercase()) {
                        "crit" -> MaterialTheme.colorScheme.error
                        "warn" -> MaterialTheme.colorScheme.tertiary
                        else -> MaterialTheme.colorScheme.primary
                    }
                    Surface(
                        color = borderColor.copy(alpha = 0.08f),
                        shape = RoundedCornerShape(4.dp),
                        border = BorderStroke(0.5.dp, MaterialTheme.colorScheme.outline),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(modifier = Modifier.height(IntrinsicSize.Min)) {
                            Box(modifier = Modifier.width(3.dp).fillMaxHeight().background(borderColor))
                            Column(modifier = Modifier.padding(8.dp)) {
                                Text(text = (if (alert.code.isNotEmpty()) "[${alert.code}] " else "") + alert.text, color = MaterialTheme.colorScheme.onBackground, fontSize = 11.sp, lineHeight = 14.sp)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun RecommendationsSection(recommendations: List<String>, height: androidx.compose.ui.unit.Dp = 160.dp) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        modifier = Modifier.fillMaxWidth().height(height)
    ) {
        if (recommendations.isEmpty()) {
            Box(contentAlignment = Alignment.Center) {
                Text("Появятся после анализа", color = MaterialTheme.colorScheme.outlineVariant, fontSize = 12.sp)
            }
        } else {
            LazyColumn(modifier = Modifier.padding(12.dp)) {
                items(recommendations) { rec ->
                    Row(modifier = Modifier.padding(vertical = 3.dp)) {
                        Text("•", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold, fontSize = 12.sp)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(rec, color = MaterialTheme.colorScheme.onBackground, fontSize = 11.sp, lineHeight = 15.sp)
                    }
                }
            }
        }
    }
}

fun getRecommendations(telemetry: TelemetrySample?): List<String> {
    val recs = mutableListOf<String>()
    if (telemetry == null) return emptyList()

    val factorActions = listOf(
        Regex("перегрев ОЖ", RegexOption.IGNORE_CASE) to "Снизить нагрузку двигателя; проверить охлаждение и уровень ОЖ.",
        Regex("низкое давление масла", RegexOption.IGNORE_CASE) to "Не наращивать тягу; проверить уровень и давление масла по регламенту.",
        Regex("напряжение АКБ", RegexOption.IGNORE_CASE) to "Проверить заряд АКБ и цепи генератора/выпрямителя.",
        Regex("ТЭД\\d+", RegexOption.IGNORE_CASE) to "Проверить вентиляцию и нагрузку на тяговые двигатели; избегать длительной работы на пределе.",
        Regex("низкое давление ГР", RegexOption.IGNORE_CASE) to "Проверить питание сжатым воздухом и тормозную магистраль.",
        Regex("^алерт\\s+", RegexOption.IGNORE_CASE) to "Свериться с кодом алерта в инструкции; при необходимости снизить ход."
    )

    telemetry.healthTopFactors?.forEach { f ->
        factorActions.forEach { (regex, action) ->
            if (regex.containsMatchIn(f.factor)) recs.add(action)
        }
    }
    
    telemetry.alerts?.forEach { a ->
        val sev = a.severity.lowercase()
        val code = a.code
        if (sev == "crit") {
            recs.add("Критический сигнал $code: быть готовым к остановке или снижению хода по регламенту.")
        } else if (sev == "warn") {
            recs.add("Предупреждение $code: проверить узел до усугубления; зафиксировать событие.")
        }
    }
    
    if (recs.isEmpty()) {
        recs.add(if (telemetry.healthIndex >= 85) "Состояние в норме; продолжайте мониторинг ключевых параметров." else "Удерживайте параметры из списка факторов под контролем; при падении индекса — снижать нагрузку.")
    }
    return recs.distinct()
}
