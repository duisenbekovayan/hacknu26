package com.example.mobile.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val DarkColorScheme = darkColorScheme(
    primary = AccentDark,
    secondary = OkDark,
    tertiary = WarnDark,
    background = BgDark,
    surface = SurfaceDark,
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.Black,
    onBackground = TextDark,
    onSurface = TextDark,
    surfaceVariant = RingTrackDark,
    outline = BorderDark,
    error = BadDark,
    secondaryContainer = FuelDark,
    onSecondaryContainer = Color.White,
    outlineVariant = MutedDark
)

private val LightColorScheme = lightColorScheme(
    primary = AccentLight,
    secondary = OkLight,
    tertiary = WarnLight,
    background = BgLight,
    surface = SurfaceLight,
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.White,
    onBackground = TextLight,
    onSurface = TextLight,
    surfaceVariant = RingTrackLight,
    outline = BorderLight,
    error = BadLight,
    secondaryContainer = FuelLight,
    onSecondaryContainer = Color.White,
    outlineVariant = MutedLight
)

@Composable
fun MobileTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        content = content
    )
}
