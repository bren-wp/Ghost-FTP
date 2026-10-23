package com.ghostftp.android

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.text.SpannableString
import android.text.Spanned
import android.text.style.ForegroundColorSpan
import android.view.Gravity
import android.view.ViewGroup
import android.view.WindowManager
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView

/**
 * Native, bounded launcher frame matching the approved Ghost FTP splash
 * reference. No source screenshots, network calls, sample sites or passwords
 * are loaded. Android's own system splash is still the first launch surface.
 */
class SplashActivity : Activity() {
    private val handler = Handler(Looper.getMainLooper())
    private val enterApp = Runnable {
        if (!isFinishing && !isDestroyed) {
            startActivity(Intent(this, MainActivity::class.java))
            finish()
            overridePendingTransition(android.R.anim.fade_in, android.R.anim.fade_out)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(WindowManager.LayoutParams.FLAG_SECURE)
        window.statusBarColor = PremiumReferenceStyle.canvas
        window.navigationBarColor = PremiumReferenceStyle.canvas
        val frame = FrameLayout(this).apply {
            // Render the dark midnight gradient from the user's 16-screen
            // reference as native Android UI, not as a screenshot background.
            background = GradientDrawable(
                GradientDrawable.Orientation.TOP_BOTTOM,
                intArrayOf(
                    Color.rgb(4, 11, 21),
                    PremiumReferenceStyle.canvas,
                    Color.rgb(12, 19, 37)
                )
            )
        }
        val identity = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            addView(ImageView(this@SplashActivity).apply {
                setImageResource(R.drawable.ic_ghost_premium_splash)
                scaleType = ImageView.ScaleType.FIT_CENTER
                contentDescription = "Ghost FTP"
            }, LinearLayout.LayoutParams(dp(142), dp(142)))
            addView(TextView(this@SplashActivity).apply {
                val name = SpannableString("Ghost FTP")
                name.setSpan(
                    ForegroundColorSpan(PremiumReferenceStyle.purpleLight),
                    6, name.length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE
                )
                text = name
                setTextColor(Color.WHITE)
                textSize = 32f
                typeface = Typeface.DEFAULT_BOLD
                gravity = Gravity.CENTER
                includeFontPadding = false
            }, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, dp(49)
            ))
            addView(TextView(this@SplashActivity).apply {
                text = "Secure. Powerful. Everywhere."
                setTextColor(PremiumReferenceStyle.secondary)
                textSize = 14f
                gravity = Gravity.CENTER
                includeFontPadding = false
            }, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, dp(28)
            ))
        }
        frame.addView(identity, FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT,
            Gravity.CENTER
        ).apply { bottomMargin = dp(90) })

        // Indeterminate animation is honest about startup work; never display
        // a fabricated percent or progress tied to no actual operation.
        val footer = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER
            addView(ProgressBar(this@SplashActivity, null,
                android.R.attr.progressBarStyleHorizontal).apply {
                isIndeterminate = true
                indeterminateTintList = android.content.res.ColorStateList.valueOf(
                    PremiumReferenceStyle.purple
                )
                progressBackgroundTintList = android.content.res.ColorStateList.valueOf(
                    PremiumReferenceStyle.raised
                )
                contentDescription = "Opening Ghost FTP"
            }, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, dp(6)
            ))
            addView(TextView(this@SplashActivity).apply {
                text = "Initializing…"
                setTextColor(PremiumReferenceStyle.secondary)
                textSize = 12f
                gravity = Gravity.CENTER
                setPadding(0, dp(16), 0, 0)
            })
        }
        frame.addView(footer, FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT,
            Gravity.BOTTOM
        ).apply {
            leftMargin = dp(45)
            rightMargin = dp(45)
            bottomMargin = dp(95)
        })
        setContentView(frame)
        // This is the visible brand handoff, not an artificial blocking
        // loading task. It is bounded to 450 ms and only runs on cold launch.
        val debugBuild = applicationInfo.flags and
            android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE != 0
        // Instrumentation can hold the real splash view for exact 480×960
        // capture. Release/preview packages ignore this test-only flag.
        if (!(debugBuild && intent?.getBooleanExtra("ghostftp.qa.hold_splash", false) == true)) {
            handler.postDelayed(enterApp, 450L)
        }
    }

    override fun onDestroy() {
        handler.removeCallbacks(enterApp)
        super.onDestroy()
    }

    private fun dp(v: Int): Int = PremiumReferenceStyle.dp(this, v)
}
