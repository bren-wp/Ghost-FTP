package com.ghostftp.android

/**
 * Android layouts are chosen from the physical device's smallest width,
 * not the current orientation width: a phone remains a phone in landscape.
 * Content width governs when two independent browser panes can fit.
 *
 * Values are logical dp and deliberately independent of the emulator's px.
 */
internal object AndroidResponsiveLayout {
    const val TABLET_SMALLEST_WIDTH_DP = 600
    const val MIN_DUAL_PANE_WIDTH_DP = 420

    fun usesPhoneNavigation(smallestWidthDp: Int): Boolean =
        smallestWidthDp < TABLET_SMALLEST_WIDTH_DP

    fun canShowDualFilePanes(widthDp: Int): Boolean =
        widthDp >= MIN_DUAL_PANE_WIDTH_DP

    fun compactFileActions(widthDp: Int, fontScale: Float): Boolean =
        widthDp < 380 || fontScale >= 1.3f && widthDp < 450

    fun drawerWidthDp(widthDp: Int): Int =
        (widthDp - 56).coerceIn(192, 280).coerceAtMost(widthDp.coerceAtLeast(1))

    fun filePaneMinHeightDp(heightDp: Int): Int =
        (heightDp - 180).coerceIn(220, 720)

    fun navigationItemHeightDp(fontScale: Float, tablet: Boolean): Int {
        val minimum = if (tablet) 52 else 48
        return if (fontScale >= 1.3f) minimum + 12 else minimum
    }
}
