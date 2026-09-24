package com.ghostftp.android

import android.app.Activity
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.text.InputType
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Spinner
import android.widget.TextView
import java.util.Locale
import kotlin.concurrent.thread
import kotlin.math.roundToInt

class MainActivity : Activity() {
    private val controller = ConnectionController()
    private lateinit var statusTitle: TextView
    private lateinit var statusDetail: TextView
    private lateinit var hostInput: EditText
    private lateinit var portInput: EditText
    private lateinit var usernameInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var remotePathInput: EditText
    private lateinit var protocolSpinner: Spinner
    private lateinit var connectButton: Button
    private lateinit var disconnectButton: Button
    private lateinit var refreshButton: Button
    private lateinit var remoteRows: LinearLayout
    private lateinit var queueRows: LinearLayout
    private var activeProfile: ConnectionProfile? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.statusBarColor = Brand.background
        window.navigationBarColor = Brand.background
        setContentView(buildContent())
        showIdleState()
    }

    private fun buildContent(): View {
        return ScrollView(this).apply {
            setBackgroundColor(Brand.background)
            addView(
                LinearLayout(this@MainActivity).apply {
                    orientation = LinearLayout.VERTICAL
                    setPadding(dp(18), dp(18), dp(18), dp(24))
                    addView(buildHeader())
                    addView(space(14))
                    addView(buildStatusCard())
                    addView(space(14))
                    addView(buildConnectionCard())
                    addView(space(14))
                    addView(buildWorkspaceCard())
                    addView(space(14))
                    addView(buildTransferCard())
                    addView(space(14))
                    addView(buildFooter())
                },
                ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.WRAP_CONTENT
                )
            )
        }
    }

    private fun buildHeader(): View = panel(strong = true).apply {
        addView(TextView(this@MainActivity).apply {
            text = ReleaseInfo.PRODUCT_NAME
            setTextColor(Brand.text)
            textSize = 28f
            typeface = Typeface.DEFAULT_BOLD
        })
        addView(TextView(this@MainActivity).apply {
            text = "${ReleaseInfo.VERSION_DISPLAY} · ${ReleaseInfo.VERSION_BADGE} · ${ReleaseInfo.BUILD}"
            setTextColor(Brand.muted)
            textSize = 13f
        })
        addView(space(10))
        addView(TextView(this@MainActivity).apply {
            text = "Native Android workspace aligned with the Ghost FTP desktop layout: connection control, remote view, queue state and privacy-first session handling in one screen."
            setTextColor(Brand.textSoft)
            textSize = 15f
            lineSpacing = 0f, 1.15f
        })
    }

    private fun buildStatusCard(): View = panel().apply {
        val label = label("Session")
        addView(label)
        statusTitle = TextView(this@MainActivity).apply {
            setTextColor(Brand.text)
            textSize = 20f
            typeface = Typeface.DEFAULT_BOLD
        }
        statusDetail = TextView(this@MainActivity).apply {
            setTextColor(Brand.textSoft)
            textSize = 14f
        }
        addView(statusTitle)
        addView(statusDetail)
    }

    private fun buildConnectionCard(): View = panel().apply {
        addView(sectionTitle("New connection"))
        addView(sectionDescription("Enter a server endpoint, choose protocol and open a controlled Android session. Passwords remain in memory only for the active action."))

        protocolSpinner = Spinner(this@MainActivity).apply {
            adapter = ArrayAdapter(
                this@MainActivity,
                android.R.layout.simple_spinner_item,
                ConnectionProtocol.entries.map { it.label }
            ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        }
        addView(formLabel("Protocol"))
        addView(protocolSpinner)

        hostInput = input("server.example.com", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Host"))
        addView(hostInput)

        portInput = input("21", InputType.TYPE_CLASS_NUMBER)
        addView(formLabel("Port"))
        addView(portInput)

        usernameInput = input("Username", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_NORMAL)
        addView(formLabel("Username"))
        addView(usernameInput)

        passwordInput = input("Password", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD)
        addView(formLabel("Password"))
        addView(passwordInput)

        remotePathInput = input("/", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        remotePathInput.setText("/")
        addView(formLabel("Remote path"))
        addView(remotePathInput)

        val actions = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(14), 0, 0)
        }
        connectButton = primaryButton("Connect") { openConnection() }
        disconnectButton = secondaryButton("Disconnect") { disconnect() }
        refreshButton = secondaryButton("Refresh") { refreshActive() }
        actions.addView(connectButton, buttonParams(weight = 1f))
        actions.addView(gap(10))
        actions.addView(disconnectButton, buttonParams(weight = 1f))
        actions.addView(gap(10))
        actions.addView(refreshButton, buttonParams(weight = 1f))
        addView(actions)
    }

    private fun buildWorkspaceCard(): View = panel().apply {
        addView(sectionTitle("Remote workspace"))
        addView(sectionDescription("Mobile-first remote view with the same Ghost FTP hierarchy: location, protocol, account context and transfer-ready state."))
        remoteRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(remoteRows)
    }

    private fun buildTransferCard(): View = panel().apply {
        addView(sectionTitle("Transfer queue"))
        addView(sectionDescription("Queue actions are kept visible on Android so upload, download, retry and cancel controls stay close to the current connection."))
        queueRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(queueRows)
    }

    private fun buildFooter(): View = panel().apply {
        addView(TextView(this@MainActivity).apply {
            text = "Ghost FTP · Brendigo · Privacy-first Android session"
            setTextColor(Brand.muted)
            textSize = 13f
            gravity = Gravity.CENTER
        })
    }

    private fun openConnection() {
        val profile = readProfile() ?: return
        activeProfile = profile
        setBusy(true)
        statusTitle.text = "Opening ${profile.protocol.label}"
        statusDetail.text = "Checking ${profile.host}:${profile.port} without saving credentials."

        thread(name = "ghostftp-android-connect") {
            val result = runCatching { controller.probe(profile) }
            runOnUiThread {
                setBusy(false)
                result.fold(
                    onSuccess = { showReachable(profile, it) },
                    onFailure = { showConnectionError(it) }
                )
            }
        }
    }

    private fun refreshActive() {
        val profile = activeProfile
        if (profile == null) {
            showIdleState()
            return
        }
        openConnection()
    }

    private fun disconnect() {
        activeProfile = null
        passwordInput.text.clear()
        showIdleState()
    }

    private fun readProfile(): ConnectionProfile? {
        val protocol = ConnectionProtocol.fromIndex(protocolSpinner.selectedItemPosition)
        val port = portInput.text.toString().trim().ifBlank { protocol.defaultPort.toString() }.toIntOrNull()
        if (port == null || port !in 1..65535) {
            showMessage("Invalid port", "Use a port between 1 and 65535.")
            return null
        }
        val host = hostInput.text.toString().trim()
        if (host.isBlank()) {
            showMessage("Host is required", "Enter the FTP, FTPS or SFTP server host.")
            return null
        }
        return ConnectionProfile(
            protocol = protocol,
            host = host,
            port = port,
            username = usernameInput.text.toString().trim(),
            remotePath = remotePathInput.text.toString().trim().ifBlank { "/" }
        )
    }

    private fun showReachable(profile: ConnectionProfile, result: ConnectionProbeResult) {
        statusTitle.text = result.title
        statusDetail.text = result.detail
        remoteRows.removeAllViews()
        result.rows.forEach { remoteRows.addView(row(it.name, it.detail)) }
        queueRows.removeAllViews()
        queueRows.addView(row("Queue", "Ready for ${profile.protocol.label} transfer actions."))
        queueRows.addView(row("Security", "Session credentials are cleared on disconnect."))
    }

    private fun showConnectionError(error: Throwable) {
        statusTitle.text = "Connection unavailable"
        statusDetail.text = error.message ?: "The selected endpoint did not accept a session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Remote view", "No server session is active."))
        queueRows.removeAllViews()
        queueRows.addView(row("Queue", "No transfer queue is active."))
    }

    private fun showMessage(title: String, detail: String) {
        statusTitle.text = title
        statusDetail.text = detail
    }

    private fun showIdleState() {
        statusTitle.text = "Ready"
        statusDetail.text = "No active server session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Remote view", "Connect to a server endpoint to load session context."))
        queueRows.removeAllViews()
        queueRows.addView(row("Queue", "No active transfers."))
        setBusy(false)
    }

    private fun setBusy(busy: Boolean) {
        connectButton.isEnabled = !busy
        refreshButton.isEnabled = !busy
        disconnectButton.isEnabled = !busy
    }

    private fun panel(strong: Boolean = false): LinearLayout = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(16), dp(16), dp(16), dp(16))
        background = rounded(if (strong) Brand.panelStrong else Brand.panel, dp(20), Brand.border)
    }

    private fun row(title: String, detail: String): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(12), dp(10), dp(12), dp(10))
        background = rounded(Brand.row, dp(14), Brand.border)
        addView(TextView(this@MainActivity).apply {
            text = title
            setTextColor(Brand.text)
            textSize = 15f
            typeface = Typeface.DEFAULT_BOLD
        })
        addView(TextView(this@MainActivity).apply {
            text = detail
            setTextColor(Brand.textSoft)
            textSize = 13f
        })
        val params = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
        params.setMargins(0, dp(8), 0, 0)
        layoutParams = params
    }

    private fun input(hintText: String, type: Int): EditText = EditText(this).apply {
        hint = hintText
        inputType = type
        singleLine = true
        setTextColor(Brand.text)
        setHintTextColor(Brand.muted)
        textSize = 15f
        setPadding(dp(12), 0, dp(12), 0)
        background = rounded(Brand.input, dp(12), Brand.border)
    }

    private fun sectionTitle(value: String): TextView = TextView(this).apply {
        text = value
        setTextColor(Brand.text)
        textSize = 20f
        typeface = Typeface.DEFAULT_BOLD
    }

    private fun sectionDescription(value: String): TextView = TextView(this).apply {
        text = value
        setTextColor(Brand.textSoft)
        textSize = 14f
        setPadding(0, dp(4), 0, dp(12))
        lineSpacing = 0f, 1.15f
    }

    private fun label(value: String): TextView = TextView(this).apply {
        text = value.uppercase(Locale.ROOT)
        setTextColor(Brand.accent)
        textSize = 12f
        typeface = Typeface.DEFAULT_BOLD
    }

    private fun formLabel(value: String): TextView = TextView(this).apply {
        text = value
        setTextColor(Brand.muted)
        textSize = 12f
        setPadding(0, dp(10), 0, dp(4))
    }

    private fun primaryButton(value: String, onClick: () -> Unit): Button = Button(this).apply {
        text = value
        setTextColor(Brand.background)
        textSize = 14f
        typeface = Typeface.DEFAULT_BOLD
        background = rounded(Brand.accent, dp(14), Brand.accent)
        setOnClickListener { onClick() }
    }

    private fun secondaryButton(value: String, onClick: () -> Unit): Button = Button(this).apply {
        text = value
        setTextColor(Brand.text)
        textSize = 14f
        background = rounded(Brand.input, dp(14), Brand.border)
        setOnClickListener { onClick() }
    }

    private fun rounded(color: Int, radius: Int, stroke: Int): GradientDrawable = GradientDrawable().apply {
        setColor(color)
        cornerRadius = radius.toFloat()
        setStroke(dp(1), stroke)
    }

    private fun space(height: Int): View = View(this).apply {
        layoutParams = LinearLayout.LayoutParams(1, dp(height))
    }

    private fun gap(width: Int): View = View(this).apply {
        layoutParams = LinearLayout.LayoutParams(dp(width), 1)
    }

    private fun buttonParams(weight: Float): LinearLayout.LayoutParams = LinearLayout.LayoutParams(
        0,
        dp(48),
        weight
    )

    private fun dp(value: Int): Int = (value * resources.displayMetrics.density).roundToInt()

    private object Brand {
        val background: Int = Color.rgb(8, 11, 20)
        val panel: Int = Color.rgb(16, 24, 42)
        val panelStrong: Int = Color.rgb(22, 35, 59)
        val row: Int = Color.rgb(13, 20, 35)
        val input: Int = Color.rgb(9, 15, 28)
        val border: Int = Color.rgb(42, 58, 88)
        val accent: Int = Color.rgb(105, 230, 255)
        val text: Int = Color.rgb(244, 248, 255)
        val textSoft: Int = Color.rgb(199, 212, 230)
        val muted: Int = Color.rgb(159, 178, 200)
    }
}
