package com.ghostftp.android

import android.content.Context
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.view.Gravity
import android.view.View
import android.widget.LinearLayout
import android.widget.TextView
import kotlin.math.roundToInt

/**
 * Native Android source primitives aligned with the most recent user-supplied
 * 75-screen GhostFTP-Premium-Brand-UI(8).zip reference (2026-10-25).
 *
 * This is a style system, not image-backed UI or synthetic user content.
 * Screens display only real profile, filesystem and transfer information.
 */
internal object PremiumReferenceStyle {
    val canvas = Color.rgb(7, 14, 26)
    val panel = Color.rgb(13, 25, 43)
    val raised = Color.rgb(17, 31, 52)
    val border = Color.rgb(37, 58, 83)
    val purple = Color.rgb(56, 171, 255)
    val purpleLight = Color.rgb(126, 222, 255)
    val selected = Color.rgb(19, 45, 82)
    val text = Color.rgb(234, 246, 255)
    val secondary = Color.rgb(171, 196, 220)
    val subtle = Color.rgb(117, 145, 178)
    val green = Color.rgb(36, 212, 143)
    val yellow = Color.rgb(255, 202, 66)

    fun dp(context: Context, v: Int): Int =
        (context.resources.displayMetrics.density * v).roundToInt()

    fun surface(context: Context, color: Int = panel, radius: Int = 10, outline: Int = border): GradientDrawable =
        GradientDrawable().apply {
            setColor(color)
            cornerRadius = dp(context, radius).toFloat()
            setStroke(dp(context, 1), outline)
        }

    fun actionBackground(context: Context): GradientDrawable =
        GradientDrawable(GradientDrawable.Orientation.LEFT_RIGHT, intArrayOf(purple, purpleLight)).apply {
            cornerRadius = dp(context, 8).toFloat()
        }

    fun caption(context: Context, value: String, size: Float = 13f, muted: Boolean = false) =
        TextView(context).apply {
            text = value
            setTextColor(if (muted) PremiumReferenceStyle.secondary else PremiumReferenceStyle.text)
            textSize = size
            includeFontPadding = false
            gravity = Gravity.CENTER_VERTICAL
        }

    fun sectionHeader(context: Context, title: String, onBack: (() -> Unit)? = null,
                      trailing: String? = null, onTrailing: (() -> Unit)? = null): LinearLayout =
        LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(context, 14), 0, dp(context, 12), 0)
            minimumHeight = dp(context, 52)
            if (onBack != null) {
                addView(caption(context, "‹", 30f).apply {
                    gravity = Gravity.CENTER
                    contentDescription = context.getString(R.string.ui_header_back)
                    isClickable = true
                    isFocusable = true
                    setOnClickListener { onBack() }
                }, LinearLayout.LayoutParams(dp(context, 48), dp(context, 48)))
            }
            addView(caption(context, title, 18f).apply {
                typeface = Typeface.DEFAULT_BOLD
                contentDescription = title
            }, LinearLayout.LayoutParams(0, dp(context, 48), 1f))
            if (trailing != null && onTrailing != null) {
                addView(caption(context, trailing, 24f).apply {
                    setTextColor(purpleLight)
                    gravity = Gravity.CENTER
                    contentDescription = context.getString(R.string.ui_header_add)
                    background = surface(context, selected, 8, border)
                    isFocusable = true
                    isClickable = true
                    setOnClickListener { onTrailing() }
                }, LinearLayout.LayoutParams(dp(context, 48), dp(context, 48)))
            }
        }

    fun panelColumn(context: Context, padding: Int = 12): LinearLayout =
        LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(context, padding), dp(context, padding),
                dp(context, padding), dp(context, padding))
            background = surface(context)
        }

    fun action(context: Context, label: String, primary: Boolean = false,
               onClick: () -> Unit): TextView =
        caption(context, label, 13f).apply {
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            minimumHeight = dp(context, 48)
            background = if (primary) actionBackground(context) else surface(context, raised, 8)
            contentDescription = label
            isFocusable = true
            isClickable = true
            setOnClickListener { onClick() }
        }

    /** Protocol-colored server tile from Android reference frame #3. */
    fun serverIcon(context: Context, protocol: ConnectionProtocol): TextView {
        val tint = when (protocol) {
            ConnectionProtocol.SFTP -> Color.rgb(51, 166, 255)
            ConnectionProtocol.EXPLICIT_FTPS -> Color.rgb(239, 84, 170)
            ConnectionProtocol.FTP -> Color.rgb(255, 139, 78)
        }
        return caption(context, "▦", 25f).apply {
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            setTextColor(tint)
            background = GradientDrawable().apply {
                setColor(Color.argb(42, Color.red(tint), Color.green(tint), Color.blue(tint)))
                cornerRadius = dp(context, 9).toFloat()
                setStroke(dp(context, 1), Color.argb(110,
                    Color.red(tint), Color.green(tint), Color.blue(tint)))
            }
            contentDescription = context.getString(R.string.ui_protocol_server, protocol.label)
        }
    }

    fun spacer(context: Context, height: Int): View = View(context).apply {
        layoutParams = LinearLayout.LayoutParams(1, dp(context, height))
    }
}
