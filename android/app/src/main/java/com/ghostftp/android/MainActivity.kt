package com.ghostftp.android

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.LinearGradient
import android.graphics.Paint
import android.graphics.Path
import android.graphics.RectF
import android.graphics.Shader
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.OpenableColumns
import android.text.InputType
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.WindowInsets
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Spinner
import android.widget.TextView
import java.io.File
import java.util.Locale
import kotlin.concurrent.thread
import kotlin.math.roundToInt

class MainActivity : Activity() {
    private enum class Workspace(val labelRes: Int) {
        FILES(R.string.workspace_files),
        SITES(R.string.workspace_sites),
        TRANSFERS(R.string.workspace_transfers),
        SETTINGS(R.string.workspace_settings),
        ABOUT(R.string.workspace_about)
    }

    private val controller = ConnectionController()
    private val remoteActionButtons = mutableListOf<Button>()
    private val busySensitiveLocalButtons = mutableListOf<Button>()
    private val sessionDisconnectButtons = mutableListOf<Button>()
    private lateinit var statusTitle: TextView
    private lateinit var statusDetail: TextView
    private lateinit var hostInput: EditText
    private lateinit var portInput: EditText
    private lateinit var usernameInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var hostKeyFingerprintInput: EditText
    private lateinit var sftpFingerprintGroup: View
    private lateinit var remotePathInput: EditText
    private lateinit var transferRemotePathInput: EditText
    private lateinit var uploadRemoteNameInput: EditText
    private lateinit var mkdirNameInput: EditText
    private lateinit var renameRemoteNameInput: EditText
    private lateinit var uploadSelectionText: TextView
    private lateinit var transferStateText: TextView
    private lateinit var protocolSpinner: Spinner
    private lateinit var protocolSecurityText: TextView
    private lateinit var connectButton: Button
    private lateinit var disconnectButton: Button
    private lateinit var refreshButton: Button
    private lateinit var remoteRows: LinearLayout
    private lateinit var activityRows: LinearLayout
    private lateinit var contentScroll: ScrollView
    private lateinit var workspaceContainer: LinearLayout
    private lateinit var workspaceTitle: TextView
    private lateinit var navigationRail: View
    private lateinit var navigationScrim: View
    private lateinit var navigationToggleButton: Button
    private var navigationOpen = false
    private lateinit var confirmationPanel: LinearLayout
    private lateinit var confirmationTitle: TextView
    private lateinit var confirmationDetail: TextView
    private lateinit var confirmationAction: Button
    private var pendingConfirmation: (() -> Unit)? = null
    private val workspaceNavButtons = mutableMapOf<Workspace, TextView>()
    private var activeWorkspace = Workspace.FILES
    private lateinit var sitesSection: View
    private lateinit var filesSection: View
    private lateinit var transfersSection: View
    private lateinit var settingsSection: View
    private lateinit var aboutSection: View
    private var activeProfile: ConnectionProfile? = null
    private var selectedUploadUri: Uri? = null
    private var selectedUploadDisplayName: String = ""
    private var lastCompletedTransferPath: String = ""
    private var operationGeneration: Long = 0
    private var operationInFlight = false
    private var activeCancellation: OperationCancellation? = null
    private var selectedProtocol = ConnectionProtocol.FTP

    @Volatile
    private var activityClosing = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        activityClosing = false
        val content = buildContent()
        if (Build.VERSION.SDK_INT >= 35) {
            content.setOnApplyWindowInsetsListener { view, insets ->
                val bars = insets.getInsets(WindowInsets.Type.systemBars())
                view.setPadding(bars.left, bars.top, bars.right, bars.bottom)
                insets
            }
        }
        setContentView(content)
        if (Build.VERSION.SDK_INT >= 35) content.requestApplyInsets()
        showIdleState()
        if (savedInstanceState != null) restoreUiState(savedInstanceState)
        installProtocolSelectionBehavior(seedDefaultPort = savedInstanceState == null)
    }

    override fun onSaveInstanceState(outState: Bundle) {
        if (::protocolSpinner.isInitialized) outState.putInt(STATE_PROTOCOL, protocolSpinner.selectedItemPosition)
        if (::hostInput.isInitialized) outState.putString(STATE_HOST, hostInput.text.toString())
        if (::portInput.isInitialized) outState.putString(STATE_PORT, portInput.text.toString())
        if (::usernameInput.isInitialized) outState.putString(STATE_USERNAME, usernameInput.text.toString())
        if (::hostKeyFingerprintInput.isInitialized) outState.putString(STATE_FINGERPRINT, hostKeyFingerprintInput.text.toString())
        if (::remotePathInput.isInitialized) outState.putString(STATE_REMOTE_PATH, remotePathInput.text.toString())
        if (::transferRemotePathInput.isInitialized) outState.putString(STATE_TRANSFER_REMOTE_PATH, transferRemotePathInput.text.toString())
        if (::uploadRemoteNameInput.isInitialized) outState.putString(STATE_UPLOAD_REMOTE_NAME, uploadRemoteNameInput.text.toString())
        if (::mkdirNameInput.isInitialized) outState.putString(STATE_MKDIR_NAME, mkdirNameInput.text.toString())
        if (::renameRemoteNameInput.isInitialized) outState.putString(STATE_RENAME_REMOTE_NAME, renameRemoteNameInput.text.toString())
        outState.putString(STATE_UPLOAD_URI, selectedUploadUri?.toString())
        outState.putString(STATE_UPLOAD_DISPLAY_NAME, selectedUploadDisplayName)
        outState.putString(STATE_LAST_COMPLETED_PATH, lastCompletedTransferPath)
        outState.putString(STATE_WORKSPACE, activeWorkspace.name)
        outState.putBoolean(STATE_NAVIGATION_OPEN, navigationOpen)
        // Intentionally never persist passwordInput or activeProfile: a recreated
        // Activity must require a fresh authenticated connection.
        super.onSaveInstanceState(outState)
    }

    override fun onDestroy() {
        activityClosing = true
        operationGeneration += 1
        operationInFlight = false
        activeCancellation?.cancel()
        activeCancellation = null
        pendingConfirmation = null
        selectedUploadUri = null
        activeProfile = null
        if (::passwordInput.isInitialized) passwordInput.text.clear()
        super.onDestroy()
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != PICK_UPLOAD_REQUEST || resultCode != RESULT_OK || !uiReady()) return
        val uri = data?.data ?: return
        if (data.flags and Intent.FLAG_GRANT_READ_URI_PERMISSION != 0) {
            runCatching {
                contentResolver.takePersistableUriPermission(
                    uri,
                    Intent.FLAG_GRANT_READ_URI_PERMISSION
                )
            }
        }
        if (!canReadUploadUri(uri)) {
            selectedUploadUri = null
            selectedUploadDisplayName = ""
            showMessage(
                "Selected file unavailable",
                "Android did not grant readable access to the selected document. Choose the file again."
            )
            appendActivity("Upload selection rejected", "Selected Android document is not readable.")
            return
        }
        selectedUploadUri = uri
        selectedUploadDisplayName = displayNameFor(uri)
        setWorkspace(Workspace.TRANSFERS)
        uploadSelectionText.text = "Selected local file: $selectedUploadDisplayName"
        if (uploadRemoteNameInput.text.toString().isBlank()) {
            uploadRemoteNameInput.setText(selectedUploadDisplayName)
        }
        transferStateText.text = "Upload file selected: $selectedUploadDisplayName"
        appendActivity("Upload", "Selected $selectedUploadDisplayName from Android document storage.")
    }

    private fun buildContent(): View {
        sitesSection = buildConnectionCard()
        filesSection = buildFilesCard()
        transfersSection = buildTransfersCard()
        settingsSection = buildSettingsCard()
        aboutSection = buildAboutCard()

        workspaceContainer = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(filesSection)
            addView(sitesSection)
            addView(transfersSection)
            addView(settingsSection)
            addView(aboutSection)
        }

        val compactNavigation = resources.configuration.screenWidthDp < 600
        navigationOpen = !compactNavigation
        val railWidth = if (compactNavigation) dp(152) else dp(184)
        navigationRail = buildNavigationRail().apply {
            visibility = if (navigationOpen) View.VISIBLE else View.GONE
        }
        navigationScrim = View(this).apply {
            setBackgroundColor(Color.argb(156, 0, 0, 0))
            visibility = View.GONE
            isClickable = true
            isFocusable = true
            contentDescription = "Close navigation menu overlay"
            setOnClickListener { setNavigationOpen(false) }
        }

        contentScroll = ScrollView(this).apply {
            setBackgroundColor(Brand.background)
            isFillViewport = true
        }
        val outerPadding = if (resources.configuration.screenWidthDp < 480) dp(10) else dp(14)
        val content = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(outerPadding, dp(14), outerPadding, dp(22))
        }
        content.addView(buildHeader())
        content.addView(space(10))
        content.addView(buildStatusCard())
        content.addView(space(10))
        content.addView(buildInlineConfirmationCard())
        content.addView(space(10))
        content.addView(workspaceContainer)
        content.addView(space(10))
        content.addView(buildFooter())

        contentScroll.addView(
            content,
            ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        )
        val root: View = if (compactNavigation) {
            FrameLayout(this).apply {
                setBackgroundColor(Brand.background)
                addView(
                    contentScroll,
                    FrameLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                )
                addView(
                    navigationScrim,
                    FrameLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                )
                addView(
                    navigationRail,
                    FrameLayout.LayoutParams(
                        railWidth,
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        Gravity.START
                    )
                )
            }
        } else {
            LinearLayout(this).apply {
                orientation = LinearLayout.HORIZONTAL
                setBackgroundColor(Brand.background)
                addView(
                    navigationRail,
                    LinearLayout.LayoutParams(
                        railWidth,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                )
                addView(
                    contentScroll,
                    LinearLayout.LayoutParams(
                        0,
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        1f
                    )
                )
            }
        }
        setWorkspace(activeWorkspace, announce = false)
        setNavigationOpen(navigationOpen, announce = false)
        return root
    }

    private fun buildNavigationRail(): View = ScrollView(this).apply {
        isFillViewport = true
        isVerticalScrollBarEnabled = false
        importantForAccessibility = View.IMPORTANT_FOR_ACCESSIBILITY_NO
        background = rounded(Brand.panelStrong, 0, Brand.border)

        val rail = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            setPadding(dp(8), dp(14), dp(8), dp(14))

            addView(GhostMarkView(this@MainActivity), LinearLayout.LayoutParams(dp(42), dp(42)))
            addView(space(8))
            addView(TextView(this@MainActivity).apply {
                text = "Ghost FTP"
                setTextColor(Brand.text)
                textSize = 14f
                typeface = Typeface.DEFAULT_BOLD
                gravity = Gravity.CENTER
            })
            addView(TextView(this@MainActivity).apply {
                text = ReleaseInfo.VERSION_DISPLAY
                setTextColor(Brand.muted)
                textSize = 11f
                gravity = Gravity.CENTER
            })
            addView(space(18))

            Workspace.entries.forEach { workspace ->
                addView(
                    workspaceNavItem(workspace),
                    LinearLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        dp(if (resources.configuration.screenWidthDp >= 600) 50 else 56)
                    )
                )
                addView(space(6))
            }

            addView(space(12))
            addView(TextView(this@MainActivity).apply {
                text = "Private\nsession"
                setTextColor(Brand.muted)
                textSize = 10f
                gravity = Gravity.CENTER
                setLineSpacing(0f, 1.08f)
            })
        }
        addView(
            rail,
            ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        )
    }

    private fun buildHeader(): View = panel(strong = true).apply {
        val titleRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        navigationToggleButton = secondaryButton("☰") {
            setNavigationOpen(!navigationOpen)
        }.apply {
            contentDescription = if (navigationOpen) "Close navigation menu" else "Open navigation menu"
            textSize = 20f
            minWidth = 0
            minimumWidth = 0
        }
        titleRow.addView(navigationToggleButton, LinearLayout.LayoutParams(dp(48), dp(44)))
        titleRow.addView(gap(8))
        titleRow.addView(GhostMarkView(this@MainActivity), LinearLayout.LayoutParams(dp(44), dp(44)))
        titleRow.addView(gap(10))
        titleRow.addView(TextView(this@MainActivity).apply {
            text = "Ghost FTP"
            setTextColor(Brand.text)
            textSize = 24f
            typeface = Typeface.DEFAULT_BOLD
        }, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
        titleRow.addView(badge(ReleaseInfo.VERSION_BADGE))
        addView(titleRow)

        workspaceTitle = TextView(this@MainActivity).apply {
            text = activeWorkspace.label
            setTextColor(Brand.textSoft)
            textSize = 13f
            typeface = Typeface.DEFAULT_BOLD
            setPadding(0, dp(8), 0, 0)
        }
        addView(workspaceTitle)

        addView(space(12))
        val toolbar = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        toolbar.addView(trackRemoteAction(toolbarButton(getString(R.string.action_refresh)) { refreshActive() }), buttonParams(weight = 1f))
        toolbar.addView(gap(8))
        toolbar.addView(trackRemoteAction(toolbarButton(getString(R.string.action_upload)) { uploadOrPickFile() }), buttonParams(weight = 1f))
        toolbar.addView(gap(8))
        toolbar.addView(trackRemoteAction(toolbarButton(getString(R.string.action_download)) { downloadRemoteFile() }), buttonParams(weight = 1f))
        addView(toolbar)

        val toolbarMore = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(8), 0, 0)
        }
        toolbarMore.addView(trackRemoteAction(toolbarButton(getString(R.string.action_new_folder)) { createRemoteFolder() }), buttonParams(weight = 1f))
        toolbarMore.addView(gap(8))
        toolbarMore.addView(trackRemoteAction(toolbarButton(getString(R.string.action_rename)) { renameRemoteEntry() }), buttonParams(weight = 1f))
        toolbarMore.addView(gap(8))
        toolbarMore.addView(trackRemoteAction(toolbarButton(getString(R.string.action_delete), destructive = true) { deleteRemoteFile() }), buttonParams(weight = 1f))
        addView(toolbarMore)
    }

    private fun buildStatusCard(): View = panel().apply {
        addView(label("Session"))
        statusTitle = TextView(this@MainActivity).apply {
            setTextColor(Brand.text)
            textSize = 20f
            typeface = Typeface.DEFAULT_BOLD
        }
        statusDetail = TextView(this@MainActivity).apply {
            setTextColor(Brand.textSoft)
            textSize = 14f
            setLineSpacing(0f, 1.14f)
        }
        addView(statusTitle)
        addView(statusDetail)
    }

    private fun buildInlineConfirmationCard(): View = panel(strong = true).apply {
        visibility = View.GONE
        contentDescription = "Inline action confirmation"

        confirmationTitle = TextView(this@MainActivity).apply {
            setTextColor(Brand.text)
            textSize = 17f
            typeface = Typeface.DEFAULT_BOLD
        }
        confirmationDetail = TextView(this@MainActivity).apply {
            setTextColor(Brand.textSoft)
            textSize = 13f
            setLineSpacing(0f, 1.14f)
            setPadding(0, dp(6), 0, dp(12))
        }
        addView(confirmationTitle)
        addView(confirmationDetail)

        val actions = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        actions.addView(secondaryButton(getString(R.string.action_cancel)) {
            clearInlineConfirmation()
        }.apply {
            contentDescription = "Cancel inline confirmation"
        }, buttonParams(weight = 1f))
        actions.addView(gap(8))
        confirmationAction = primaryButton(getString(R.string.action_confirm)) {
            val action = pendingConfirmation
            clearInlineConfirmation()
            if (!closingOrDestroyed()) action?.invoke()
        }.apply {
            contentDescription = "Confirm inline action"
        }
        actions.addView(confirmationAction, buttonParams(weight = 1f))
        addView(actions)
    }

    private fun requestInlineConfirmation(
        title: String,
        detail: String,
        confirmLabel: String,
        onConfirm: () -> Unit
    ) {
        if (closingOrDestroyed() || !::confirmationPanel.isInitialized) return
        pendingConfirmation = onConfirm
        confirmationTitle.text = title
        confirmationDetail.text = detail
        confirmationAction.text = confirmLabel
        confirmationAction.contentDescription = "$confirmLabel inline action"
        confirmationPanel.visibility = View.VISIBLE
        confirmationPanel.isFocusable = true
        confirmationPanel.requestFocus()
        confirmationPanel.announceForAccessibility("$title $detail")
    }

    private fun clearInlineConfirmation() {
        pendingConfirmation = null
        if (::confirmationPanel.isInitialized) {
            confirmationPanel.visibility = View.GONE
            confirmationPanel.isFocusable = false
        }
    }

    private fun buildConnectionCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_sites)))
        addView(sectionDescription("Connect to FTP, explicit FTPS or SFTP. Passwords stay in memory for the active session and are cleared on disconnect."))

        protocolSpinner = Spinner(this@MainActivity).apply {
            contentDescription = "Connection protocol"
            adapter = ArrayAdapter(
                this@MainActivity,
                android.R.layout.simple_spinner_item,
                ConnectionProtocol.entries.map { it.label }
            ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        }
        addView(formLabel("Protocol"))
        addView(protocolSpinner)
        protocolSecurityText = TextView(this@MainActivity).apply {
            textSize = 12f
            setPadding(0, dp(6), 0, dp(4))
        }
        addView(protocolSecurityText)

        hostInput = input("Host", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Host"))
        addView(hostInput)

        portInput = input("21", InputType.TYPE_CLASS_NUMBER)
        addView(formLabel("Port"))
        addView(portInput)

        usernameInput = input("Username", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_NORMAL)
        addView(formLabel("Username"))
        addView(usernameInput)

        passwordInput = input("Password", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD).apply {
            // Never let Activity view-state persistence retain a session password.
            isSaveEnabled = false
        }
        addView(formLabel("Password"))
        addView(passwordInput)

        hostKeyFingerprintInput = input("SHA256 fingerprint for SFTP", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_NORMAL)
        sftpFingerprintGroup = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            addView(formLabel("SFTP host key fingerprint"))
            addView(hostKeyFingerprintInput)
        }
        addView(sftpFingerprintGroup)

        remotePathInput = input("/", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        remotePathInput.setText("/")
        addView(formLabel("Remote path"))
        addView(remotePathInput)

        val actions = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(14), 0, 0)
        }
        connectButton = primaryButton(getString(R.string.action_connect)) { openConnection() }.apply {
            contentDescription = "Connect to server"
        }
        disconnectButton = secondaryButton(getString(R.string.action_disconnect)) { disconnect() }.apply {
            contentDescription = "Disconnect from server"
        }
        refreshButton = secondaryButton(getString(R.string.action_refresh)) { refreshActive() }.apply {
            contentDescription = "Refresh current session"
        }
        actions.addView(connectButton, buttonParams(weight = 1f))
        actions.addView(gap(8))
        actions.addView(disconnectButton, buttonParams(weight = 1f))
        actions.addView(gap(8))
        actions.addView(refreshButton, buttonParams(weight = 1f))
        addView(actions)
    }

    private fun installProtocolSelectionBehavior(seedDefaultPort: Boolean) {
        selectedProtocol = ConnectionProtocol.fromIndex(protocolSpinner.selectedItemPosition)
        updateProtocolSpecificFields(selectedProtocol)

        if (seedDefaultPort && portInput.text.toString().isBlank()) {
            portInput.setText(selectedProtocol.defaultPort.toString())
        }

        protocolSpinner.onItemSelectedListener = object : AdapterView.OnItemSelectedListener {
            override fun onItemSelected(
                parent: AdapterView<*>?,
                view: View?,
                position: Int,
                id: Long
            ) {
                val nextProtocol = ConnectionProtocol.fromIndex(position)
                val currentPort = portInput.text.toString().trim()
                if (currentPort.isBlank() || currentPort == selectedProtocol.defaultPort.toString()) {
                    portInput.setText(nextProtocol.defaultPort.toString())
                }
                selectedProtocol = nextProtocol
                updateProtocolSpecificFields(nextProtocol)
            }

            override fun onNothingSelected(parent: AdapterView<*>?) = Unit
        }
    }

    private fun updateProtocolSpecificFields(protocol: ConnectionProtocol) {
        if (::sftpFingerprintGroup.isInitialized) {
            sftpFingerprintGroup.visibility =
                if (protocol == ConnectionProtocol.SFTP) View.VISIBLE else View.GONE
        }
        if (::protocolSecurityText.isInitialized) {
            val (message, color) = when (protocol) {
                ConnectionProtocol.FTP ->
                    "FTP sends credentials and file data without transport encryption. Prefer explicit FTPS or SFTP when the server supports it." to Brand.danger
                ConnectionProtocol.EXPLICIT_FTPS ->
                    "Explicit FTPS encrypts credentials and file data with TLS and validates the server hostname." to Brand.textSoft
                ConnectionProtocol.SFTP ->
                    "SFTP encrypts the session and requires strict SSH host-key verification." to Brand.textSoft
            }
            protocolSecurityText.text = message
            protocolSecurityText.setTextColor(color)
            protocolSecurityText.contentDescription = message
        }
    }

    private fun buildFilesCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_files)))
        addView(sectionDescription("Open folders, select files, then use the toolbar actions aligned with Ghost FTP desktop."))
        remoteRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(remoteRows)
    }

    private fun buildTransfersCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_transfers)))
        addView(sectionDescription("Manage the selected remote entry, upload target, rename target and remote folder action for the active session."))

        transferRemotePathInput = input("/remote/file-or-folder", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Remote file path"))
        addView(transferRemotePathInput)

        uploadRemoteNameInput = input("Remote upload file name", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Upload target name or path"))
        addView(uploadRemoteNameInput)

        mkdirNameInput = input("New remote folder", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Folder name or path"))
        addView(mkdirNameInput)

        renameRemoteNameInput = input("New remote name or path", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Rename target name or path"))
        addView(renameRemoteNameInput)

        uploadSelectionText = TextView(this@MainActivity).apply {
            text = "No local upload file selected."
            setTextColor(Brand.textSoft)
            textSize = 13f
            setPadding(0, dp(10), 0, dp(4))
        }
        addView(uploadSelectionText)

        transferStateText = TextView(this@MainActivity).apply {
            text = "No transfer started."
            setTextColor(Brand.muted)
            textSize = 13f
            setPadding(0, dp(4), 0, dp(4))
        }
        addView(transferStateText)

        val uploadRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, 0)
        }
        uploadRow.addView(trackBusySensitiveLocalAction(secondaryButton(getString(R.string.action_pick_file)) { selectUploadFile() }.apply {
            contentDescription = "Pick upload file"
        }), buttonParams(weight = 1f))
        uploadRow.addView(gap(8))
        uploadRow.addView(trackRemoteAction(secondaryButton(getString(R.string.action_upload)) { uploadSelectedFile() }.apply {
            contentDescription = "Upload selected file"
        }), buttonParams(weight = 1f))
        addView(uploadRow)

        activityRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        addView(activityRows)
    }

    private fun buildSettingsCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_settings)))
        addView(sectionDescription("Working session and privacy controls aligned with Ghost FTP desktop safety rules."))
        addView(row("Privacy", "No required tracking, analytics or telemetry."))
        addView(row("Credentials", "Session passwords stay in memory and are cleared on disconnect or Activity destruction."))
        addView(row("Connection safety", "Remote mutations are guarded and SFTP requires strict host-key verification."))

        val firstRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        firstRow.addView(secondaryButton("Clear Activity") { clearActivityLog() }.apply {
            contentDescription = "Clear activity log"
        }, buttonParams(weight = 1f))
        firstRow.addView(gap(8))
        firstRow.addView(trackBusySensitiveLocalAction(secondaryButton("Reset Transfers") { resetTransferFields() }.apply {
            contentDescription = "Reset transfer fields"
        }), buttonParams(weight = 1f))
        addView(firstRow)

        val secondRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(8), 0, 0)
        }
        secondRow.addView(trackBusySensitiveLocalAction(secondaryButton("Reset Connection") { resetConnectionForm() }.apply {
            contentDescription = "Reset connection form"
        }), buttonParams(weight = 1f))
        secondRow.addView(gap(8))
        secondRow.addView(trackSessionDisconnect(secondaryButton(getString(R.string.action_disconnect)) { disconnect() }.apply {
            contentDescription = "Settings disconnect session"
        }), buttonParams(weight = 1f))
        addView(secondRow)
    }

    private fun buildAboutCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_about)))
        addView(sectionDescription("Ghost FTP ${ReleaseInfo.VERSION_DISPLAY} · Build ${ReleaseInfo.BUILD}"))
        addView(row("Product", "Ghost FTP by Brendigo"))
        addView(row("Protocols", "FTP · Explicit FTPS · SFTP"))
        addView(officialLinkRow("Support", "Guides, troubleshooting and product support.", SUPPORT_URL))
        addView(officialLinkRow("Documentation", "Official Ghost FTP documentation.", DOCUMENTATION_URL))
        addView(officialLinkRow("Privacy", "Read the official privacy information.", PRIVACY_URL))
        addView(officialLinkRow("Terms of use / EULA", "Read the canonical Ghost FTP software licence terms.", EULA_URL))
        addView(officialLinkRow("Official website", "Ghost FTP product website.", WEBSITE_URL))
    }

    private fun setWorkspace(workspace: Workspace, announce: Boolean = true) {
        activeWorkspace = workspace
        val targets = mapOf(
            Workspace.FILES to filesSection,
            Workspace.SITES to sitesSection,
            Workspace.TRANSFERS to transfersSection,
            Workspace.SETTINGS to settingsSection,
            Workspace.ABOUT to aboutSection
        )
        targets.forEach { (key, view) ->
            view.visibility = if (key == workspace) View.VISIBLE else View.GONE
        }
        workspaceNavButtons.forEach { (key, view) ->
            styleWorkspaceNavItem(view, selected = key == workspace)
        }
        if (::workspaceTitle.isInitialized) workspaceTitle.text = getString(workspace.labelRes)
        if (::workspaceContainer.isInitialized) {
            workspaceContainer.contentDescription = getString(R.string.workspace_content, getString(workspace.labelRes))
        }
        if (::contentScroll.isInitialized) {
            contentScroll.post {
                val workspaceTop = if (::workspaceContainer.isInitialized) {
                    workspaceContainer.top
                } else {
                    0
                }
                contentScroll.scrollTo(0, workspaceTop)
                if (announce && ::workspaceContainer.isInitialized) {
                    workspaceContainer.announceForAccessibility(getString(R.string.workspace_announce, getString(workspace.labelRes)))
                }
            }
        }
        if (announce && resources.configuration.screenWidthDp < 600) {
            setNavigationOpen(false, announce = false)
        }
    }

    private fun setNavigationOpen(open: Boolean, announce: Boolean = true) {
        navigationOpen = open
        val compactNavigation = resources.configuration.screenWidthDp < 600
        if (::navigationRail.isInitialized) {
            navigationRail.visibility = if (open) View.VISIBLE else View.GONE
        }
        if (::navigationScrim.isInitialized) {
            navigationScrim.visibility =
                if (compactNavigation && open) View.VISIBLE else View.GONE
            navigationScrim.isFocusable = compactNavigation && open
        }
        if (::navigationToggleButton.isInitialized) {
            navigationToggleButton.text = if (open) "←" else "☰"
            navigationToggleButton.contentDescription =
                if (open) "Close navigation menu" else "Open navigation menu"
            if (announce) {
                navigationToggleButton.announceForAccessibility(
                    if (open) "Navigation menu opened" else "Navigation menu closed"
                )
            }
        }
    }

    private fun buildFooter(): View = panel().apply {
        addView(TextView(this@MainActivity).apply {
            text = "Ghost FTP · Brendigo · Private session"
            setTextColor(Brand.muted)
            textSize = 13f
            gravity = Gravity.CENTER
        })
    }

    private fun openConnection() {
        val profile = readProfile() ?: return
        openConnection(profile)
    }

    private fun openConnection(profile: ConnectionProfile) {
        if (operationInFlight) return
        val generation = ++operationGeneration
        val cancellation = OperationCancellation()
        activeCancellation = cancellation
        operationInFlight = true
        setBusy(true)
        statusTitle.text = "Opening ${profile.protocol.label}"
        statusDetail.text = "Loading ${profile.remotePath} from ${profile.host}:${profile.port}."

        thread(name = "ghostftp-android-connect") {
            val result = runCatching { controller.listRemote(profile, cancellation) }
            safeUi {
                if (generation != operationGeneration) return@safeUi
                activeCancellation = null
                operationInFlight = false
                result.fold(
                    onSuccess = {
                        activeProfile = profile
                        showReachable(profile, it)
                        setWorkspace(Workspace.FILES)
                    },
                    onFailure = {
                        activeProfile = null
                        showConnectionError(it)
                    }
                )
                setBusy(false)
            }
        }
    }

    private fun refreshActive() {
        val profile = activeProfile
        if (profile == null) {
            showIdleState()
            return
        }
        val nextPath = remotePathInput.text.toString().trim().ifBlank { profile.remotePath }
        val refreshedProfile = profile.copy(remotePath = nextPath)
        activeProfile = refreshedProfile
        openConnection(refreshedProfile)
    }

    private fun disconnect() {
        operationGeneration += 1
        operationInFlight = false
        activeCancellation?.cancel()
        activeCancellation = null
        activeProfile = null
        selectedUploadUri = null
        selectedUploadDisplayName = ""
        lastCompletedTransferPath = ""
        if (::passwordInput.isInitialized) passwordInput.text.clear()
        if (::uploadSelectionText.isInitialized) uploadSelectionText.text = "No local upload file selected."
        if (::transferStateText.isInitialized) transferStateText.text = "No transfer started."
        showIdleState()
    }

    private fun clearActivityLog() {
        if (::activityRows.isInitialized) activityRows.removeAllViews()
        showMessage("Activity cleared", "Activity log cleared.")
    }

    private fun resetTransferFields() {
        selectedUploadUri = null
        selectedUploadDisplayName = ""
        lastCompletedTransferPath = ""
        if (::transferRemotePathInput.isInitialized) transferRemotePathInput.text.clear()
        if (::uploadRemoteNameInput.isInitialized) uploadRemoteNameInput.text.clear()
        if (::mkdirNameInput.isInitialized) mkdirNameInput.text.clear()
        if (::renameRemoteNameInput.isInitialized) renameRemoteNameInput.text.clear()
        if (::uploadSelectionText.isInitialized) uploadSelectionText.text = "No local upload file selected."
        if (::transferStateText.isInitialized) transferStateText.text = "No transfer started."
        showMessage("Transfers reset", "Transfer fields reset.")
    }

    private fun resetConnectionForm() {
        disconnect()
        if (::protocolSpinner.isInitialized) protocolSpinner.setSelection(ConnectionProtocol.FTP.ordinal)
        if (::hostInput.isInitialized) hostInput.text.clear()
        if (::portInput.isInitialized) portInput.setText(ConnectionProtocol.FTP.defaultPort.toString())
        if (::usernameInput.isInitialized) usernameInput.text.clear()
        if (::hostKeyFingerprintInput.isInitialized) hostKeyFingerprintInput.text.clear()
        if (::remotePathInput.isInitialized) remotePathInput.setText("/")
        showMessage("Connection reset", "Connection form reset.")
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
            password = passwordInput.text.toString(),
            hostKeyFingerprint = hostKeyFingerprintInput.text.toString().trim(),
            remotePath = remotePathInput.text.toString().trim().ifBlank { "/" }
        )
    }

    private fun activeTransferProfile(): ConnectionProfile? {
        val profile = activeProfile
        if (profile == null) {
            showMessage("Connect first", "Open a server session before running file actions.")
            return null
        }
        return profile.copy(remotePath = remotePathInput.text.toString().trim().ifBlank { profile.remotePath })
    }

    private fun showReachable(profile: ConnectionProfile, result: ConnectionProbeResult) {
        if (!uiReady()) return
        statusTitle.text = result.title
        statusDetail.text = result.detail
        remoteRows.removeAllViews()
        result.rows.forEach { remoteRows.addView(remoteRow(it)) }
        activityRows.removeAllViews()
        activityRows.addView(row("Connection", "Ready for ${profile.protocol.label} file actions."))
        activityRows.addView(row("Security", "Password is kept in memory and cleared on disconnect."))
        transferStateText.text = if (lastCompletedTransferPath.isBlank()) {
            "Ready for guarded file actions."
        } else {
            "Last completed remote path: $lastCompletedTransferPath"
        }
    }

    private fun showConnectionError(error: Throwable) {
        if (!uiReady()) return
        statusTitle.text = "Connection unavailable"
        statusDetail.text = error.message ?: "The selected endpoint did not open a session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Files", "No server session is active."))
        activityRows.removeAllViews()
        activityRows.addView(row("Transfers", "No active connection."))
        transferStateText.text = "No transfer can run until the connection opens."
    }

    private fun showMessage(title: String, detail: String) {
        if (!::statusTitle.isInitialized || !::statusDetail.isInitialized) return
        statusTitle.text = title
        statusDetail.text = detail
        if (::transferStateText.isInitialized) transferStateText.text = detail
    }

    private fun showIdleState() {
        if (!uiReady()) return
        statusTitle.text = "Ready"
        statusDetail.text = "No active server session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Files", "Connect to a server to load remote files."))
        activityRows.removeAllViews()
        activityRows.addView(row("Transfers", "Connect first, then choose a file action."))
        transferStateText.text = "No transfer started."
        setBusy(false)
    }

    private fun setBusy(busy: Boolean) {
        if (!::connectButton.isInitialized) return
        val hasActiveSession = activeProfile != null
        connectButton.isEnabled = !busy && !hasActiveSession
        refreshButton.isEnabled = !busy && hasActiveSession
        disconnectButton.isEnabled = hasActiveSession
        remoteActionButtons.forEach { it.isEnabled = !busy && hasActiveSession }
        busySensitiveLocalButtons.forEach { it.isEnabled = !busy }
        sessionDisconnectButtons.forEach { it.isEnabled = hasActiveSession }
    }

    private fun uploadOrPickFile() {
        if (selectedUploadUri == null) selectUploadFile() else uploadSelectedFile()
    }

    private fun downloadRemoteFile() {
        val profile = activeTransferProfile() ?: return
        val remotePath = requiredRemoteFilePath(profile) ?: return
        val outputFile = downloadTarget(remotePath)
        runTransfer(
            title = "Downloading",
            detail = "Saving $remotePath to Android downloads."
        ) { cancellation ->
            controller.downloadRemote(profile, remotePath, outputFile, cancellation)
        }
    }

    private fun uploadSelectedFile() {
        val profile = activeTransferProfile() ?: return
        val uri = selectedUploadUri
        if (uri == null) {
            showMessage("Choose a local file", "Pick an Android document before uploading.")
            return
        }
        val remoteTarget = uploadTargetPath(profile) ?: return
        val uploadName = selectedUploadDisplayName.ifBlank { "selected file" }
        confirmUploadTarget(remoteTarget, uploadName) {
            runTransfer(
                title = "Uploading",
                detail = "Sending $uploadName to $remoteTarget.",
                refreshAfter = true
            ) { cancellation ->
                val input = contentResolver.openInputStream(uri)
                    ?: throw IllegalStateException("Unable to open selected Android document.")
                controller.uploadRemote(profile, input, remoteTarget, cancellation)
            }
        }
    }

    private fun deleteRemoteFile() {
        val profile = activeTransferProfile() ?: return
        val remotePath = requiredRemoteFilePath(profile) ?: return
        if (remotePath.split('/').filter { it.isNotBlank() }.isEmpty()) {
            showMessage("Unsafe delete blocked", "Ghost FTP will not delete the remote root path.")
            return
        }
        confirmDestructiveRemoteAction(
            title = "Delete remote entry?",
            message = "This permanently removes $remotePath from the active server. Empty folders are supported; non-empty folders are never removed recursively.",
            confirmLabel = "Delete"
        ) {
            runTransfer(
                title = "Deleting",
                detail = "Removing $remotePath from the active server.",
                refreshAfter = true
            ) { cancellation ->
                controller.deleteRemoteFile(profile, remotePath, cancellation)
            }
        }
    }

    private fun renameRemoteEntry() {
        val profile = activeTransferProfile() ?: return
        val sourcePath = requiredRemoteFilePath(profile) ?: return
        if (sourcePath.split('/').filter { it.isNotBlank() }.isEmpty()) {
            showMessage("Unsafe rename blocked", "Ghost FTP will not rename the remote root path.")
            return
        }
        val rawTarget = renameRemoteNameInput.text.toString().trim()
        if (rawTarget.isBlank()) {
            showMessage("Rename target is required", "Enter a new remote name or absolute remote path.")
            return
        }
        val destinationPath = normalizeRemoteInput(rawTarget, profile.remotePath) ?: return
        if (destinationPath == sourcePath) {
            showMessage("Rename target is unchanged", "Choose a different remote name or path.")
            return
        }
        confirmDestructiveRemoteAction(
            title = "Rename remote entry?",
            message = "Rename $sourcePath to $destinationPath?",
            confirmLabel = "Rename"
        ) {
            runTransfer(
                title = "Renaming",
                detail = "Renaming $sourcePath to $destinationPath.",
                refreshAfter = true
            ) { cancellation ->
                controller.renameRemote(profile, sourcePath, destinationPath, cancellation)
            }
        }
    }

    private fun createRemoteFolder() {
        val profile = activeTransferProfile() ?: return
        val folderName = mkdirNameInput.text.toString().trim()
        if (folderName.isBlank()) {
            showMessage("Folder name is required", "Enter a folder name or absolute remote folder path.")
            return
        }
        val remoteTarget = normalizeRemoteInput(folderName, profile.remotePath) ?: return
        runTransfer(
            title = "Creating folder",
            detail = "Creating $remoteTarget on the active server.",
            refreshAfter = true
        ) { cancellation ->
            controller.createRemoteDirectory(profile, remoteTarget, cancellation)
        }
    }

    private fun selectUploadFile() {
        if (closingOrDestroyed()) return
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = "*/*"
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION)
        }
        runCatching { startActivityForResult(intent, PICK_UPLOAD_REQUEST) }
            .onFailure { showMessage("File picker unavailable", it.message ?: "Android could not open a document picker.") }
    }

    private fun runTransfer(
        title: String,
        detail: String,
        refreshAfter: Boolean = false,
        action: (OperationCancellation) -> TransferResult
    ) {
        if (operationInFlight) return
        val generation = ++operationGeneration
        val cancellation = OperationCancellation()
        activeCancellation = cancellation
        operationInFlight = true
        setBusy(true)
        statusTitle.text = title
        statusDetail.text = detail
        transferStateText.text = detail
        appendActivity(title, detail)
        thread(name = "ghostftp-android-transfer") {
            val result = runCatching { action(cancellation) }
            safeUi {
                if (generation != operationGeneration) return@safeUi
                activeCancellation = null
                operationInFlight = false
                setBusy(false)
                result.fold(
                    onSuccess = { showTransferResult(it, refreshAfter) },
                    onFailure = { showTransferFailure(it) }
                )
            }
        }
    }

    private fun showTransferResult(result: TransferResult, refreshAfter: Boolean) {
        if (!uiReady()) return
        lastCompletedTransferPath = result.remotePath
        statusTitle.text = result.title
        statusDetail.text = result.detail
        transferStateText.text = result.detail
        appendActivity(result.title, result.detail)
        if (refreshAfter && activeProfile != null) refreshActive()
    }

    private fun showTransferFailure(error: Throwable) {
        if (!uiReady()) return
        val canceled = error is OperationCanceledException
        val detail = error.message ?: if (canceled) "Operation canceled." else "The transfer action did not complete."
        statusTitle.text = if (canceled) "Transfer canceled" else "Transfer failed"
        statusDetail.text = detail
        transferStateText.text = detail
        appendActivity(if (canceled) "Transfer canceled" else "Transfer failed", detail)
    }

    private fun restoreUiState(state: Bundle) {
        protocolSpinner.setSelection(state.getInt(STATE_PROTOCOL, 0))
        hostInput.setText(state.getString(STATE_HOST).orEmpty())
        portInput.setText(state.getString(STATE_PORT).orEmpty())
        usernameInput.setText(state.getString(STATE_USERNAME).orEmpty())
        hostKeyFingerprintInput.setText(state.getString(STATE_FINGERPRINT).orEmpty())
        remotePathInput.setText(state.getString(STATE_REMOTE_PATH).orEmpty().ifBlank { "/" })
        transferRemotePathInput.setText(state.getString(STATE_TRANSFER_REMOTE_PATH).orEmpty())
        uploadRemoteNameInput.setText(state.getString(STATE_UPLOAD_REMOTE_NAME).orEmpty())
        mkdirNameInput.setText(state.getString(STATE_MKDIR_NAME).orEmpty())
        renameRemoteNameInput.setText(state.getString(STATE_RENAME_REMOTE_NAME).orEmpty())
        val restoredUploadDisplayName = state.getString(STATE_UPLOAD_DISPLAY_NAME).orEmpty()
        val restoredUploadUri = state.getString(STATE_UPLOAD_URI)
            ?.takeIf { it.isNotBlank() }
            ?.let(Uri::parse)
        val restoredUploadReadable = restoredUploadUri?.let(::canReadUploadUri) == true
        selectedUploadUri = restoredUploadUri?.takeIf { restoredUploadReadable }
        selectedUploadDisplayName = restoredUploadDisplayName.takeIf { restoredUploadReadable }.orEmpty()
        lastCompletedTransferPath = state.getString(STATE_LAST_COMPLETED_PATH).orEmpty()
        activeWorkspace = state.getString(STATE_WORKSPACE)
            ?.let { saved -> runCatching { Workspace.valueOf(saved) }.getOrNull() }
            ?: Workspace.FILES
        setWorkspace(activeWorkspace, announce = false)
        setNavigationOpen(
            state.getBoolean(STATE_NAVIGATION_OPEN, resources.configuration.screenWidthDp >= 600),
            announce = false
        )

        passwordInput.text.clear()
        activeProfile = null
        activeCancellation = null
        operationInFlight = false
        if (selectedUploadUri != null && selectedUploadDisplayName.isNotBlank()) {
            uploadSelectionText.text = "Selected local file: $selectedUploadDisplayName"
        }
        statusTitle.text = "Ready"
        statusDetail.text = if (restoredUploadUri != null && !restoredUploadReadable) {
            "Android restored non-secret workspace state, but the previous local file is no longer readable. Choose it again before uploading."
        } else {
            "Android restored non-secret workspace state. Reconnect to authenticate before remote actions."
        }
        transferStateText.text = if (lastCompletedTransferPath.isBlank()) {
            "Previous session ended. Reconnect to continue."
        } else {
            "Last completed remote path: $lastCompletedTransferPath"
        }
        setBusy(false)
    }

    private fun requiredRemoteFilePath(profile: ConnectionProfile): String? {
        val raw = transferRemotePathInput.text.toString().trim()
        if (raw.isBlank()) {
            showMessage("Remote path is required", "Select a remote file or folder, or enter an absolute remote path.")
            return null
        }
        return normalizeRemoteInput(raw, profile.remotePath)
    }

    private fun uploadTargetPath(profile: ConnectionProfile): String? {
        val targetName = uploadRemoteNameInput.text.toString().trim().ifBlank { selectedUploadDisplayName }
        if (targetName.isBlank()) {
            showMessage("Upload target is required", "Choose a local file and enter the target file name or path.")
            return null
        }
        return normalizeRemoteInput(targetName, profile.remotePath)
    }

    private fun normalizeRemoteInput(raw: String, directory: String): String? {
        val resolved = if (raw.startsWith('/')) raw else joinRemotePath(directory, raw)
        val blocked = resolved.split('/').filter { it.isNotBlank() }.any { it == "." || it == ".." }
        if (blocked) {
            showMessage("Unsupported remote path", "Use a direct remote path without dot path segments.")
            return null
        }
        return resolved
    }

    private fun confirmUploadTarget(remoteTarget: String, localName: String, onConfirm: () -> Unit) {
        requestInlineConfirmation(
            title = "Upload to remote path?",
            detail = "Upload $localName to $remoteTarget on the active server.",
            confirmLabel = "Upload",
            onConfirm = onConfirm
        )
    }

    private fun confirmDestructiveRemoteAction(
        title: String,
        message: String,
        confirmLabel: String,
        onConfirm: () -> Unit
    ) {
        requestInlineConfirmation(
            title = title,
            detail = message,
            confirmLabel = confirmLabel,
            onConfirm = onConfirm
        )
    }

    private fun remoteRow(item: RemoteRow): View = row(item.name, item.detail).apply {
        val target = item.remotePath ?: return@apply
        when {
            item.isDirectory -> {
                setOnClickListener {
                    remotePathInput.setText(target)
                    openConnection()
                }
                setOnLongClickListener {
                    selectRemoteEntry(item)
                    setWorkspace(Workspace.TRANSFERS)
                    true
                }
            }
            item.isFile -> setOnClickListener {
                selectRemoteEntry(item)
            }
        }
    }

    private fun selectRemoteEntry(item: RemoteRow) {
        val target = item.remotePath ?: return
        transferRemotePathInput.setText(target)
        renameRemoteNameInput.setText(target.substringAfterLast('/'))
        transferStateText.text = if (item.isDirectory) {
            "Selected remote folder: $target"
        } else {
            "Selected remote file: $target"
        }
        appendActivity("Selected", target)
    }

    private fun appendActivity(title: String, detail: String) {
        if (!::activityRows.isInitialized || closingOrDestroyed()) return
        activityRows.addView(row(title, detail), 0)
        while (activityRows.childCount > MAX_ACTIVITY_ROWS) {
            activityRows.removeViewAt(activityRows.childCount - 1)
        }
    }

    private fun downloadTarget(remotePath: String): File {
        val primaryDir = getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS)
        val dir = when {
            primaryDir != null && (primaryDir.exists() || primaryDir.mkdirs()) -> primaryDir
            else -> File(filesDir, "downloads").apply { mkdirs() }
        }
        val baseName = safeFileName(remotePath.substringAfterLast('/').ifBlank { "ghostftp-download.bin" })
        var candidate = File(dir, baseName)
        var index = 1
        val stem = baseName.substringBeforeLast('.', baseName)
        val extension = baseName.substringAfterLast('.', missingDelimiterValue = "")
        while (candidate.exists()) {
            val name = if (extension.isBlank()) "$stem-$index" else "$stem-$index.$extension"
            candidate = File(dir, name)
            index += 1
        }
        return candidate
    }

    private fun canReadUploadUri(uri: Uri): Boolean = runCatching {
        contentResolver.openInputStream(uri)?.use { true } ?: false
    }.getOrDefault(false)

    private fun displayNameFor(uri: Uri): String {
        return runCatching {
            contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
                val index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                if (index >= 0 && cursor.moveToFirst()) {
                    val value = cursor.getString(index)
                    if (!value.isNullOrBlank()) return@runCatching safeFileName(value)
                }
            }
            safeFileName(uri.lastPathSegment?.substringAfterLast('/') ?: "ghostftp-upload.bin")
        }.getOrElse { "ghostftp-upload.bin" }
    }

    private fun safeFileName(value: String): String {
        return value.trim().replace(Regex("[^A-Za-z0-9._-]"), "_").ifBlank { "ghostftp-file.bin" }
    }

    private fun joinRemotePath(directory: String, child: String): String {
        val safeChild = child.trim().trimStart('/')
        val base = directory.trim().ifBlank { "/" }.trimEnd('/')
        return if (base.isBlank()) "/$safeChild" else "$base/$safeChild"
    }

    private fun safeUi(block: () -> Unit) {
        if (closingOrDestroyed()) return
        runOnUiThread {
            if (!closingOrDestroyed() && uiReady()) block()
        }
    }

    private fun closingOrDestroyed(): Boolean {
        return activityClosing || isFinishing || isDestroyed
    }

    private fun uiReady(): Boolean {
        return !closingOrDestroyed() &&
            ::statusTitle.isInitialized &&
            ::statusDetail.isInitialized &&
            ::remoteRows.isInitialized &&
            ::activityRows.isInitialized &&
            ::transferStateText.isInitialized
    }

    private fun panel(strong: Boolean = false): LinearLayout = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(14), dp(14), dp(14), dp(14))
        background = rounded(if (strong) Brand.panelStrong else Brand.panel, dp(18), Brand.border)
    }

    private fun row(title: String, detail: String): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(12), dp(10), dp(12), dp(10))
        background = rounded(Brand.row, dp(14), Brand.borderSubtle)
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
            setLineSpacing(0f, 1.12f)
        })
        val params = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
        params.setMargins(0, dp(8), 0, 0)
        layoutParams = params
    }

    private fun officialLinkRow(title: String, detail: String, url: String): View =
        row(title, detail).apply {
            isClickable = true
            isFocusable = true
            contentDescription = "Open Ghost FTP $title"
            setOnClickListener { openOfficialLink(url) }
        }

    private fun openOfficialLink(url: String) {
        if (url !in OFFICIAL_LINKS) {
            showMessage("Link blocked", "Ghost FTP refused to open an unapproved external address.")
            return
        }
        val uri = Uri.parse(url)
        if (uri.scheme != "https") {
            showMessage("Link blocked", "Ghost FTP only opens secure HTTPS product links.")
            return
        }
        runCatching {
            startActivity(Intent(Intent.ACTION_VIEW, uri).apply {
                addCategory(Intent.CATEGORY_BROWSABLE)
            })
        }.onFailure {
            showMessage("Unable to open link", "No browser is available for this Ghost FTP link.")
        }
    }

    private fun input(hintText: String, type: Int): EditText = EditText(this).apply {
        hint = hintText
        inputType = type
        setSingleLine(true)
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
        setLineSpacing(0f, 1.15f)
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

    private fun badge(value: String): TextView = TextView(this).apply {
        text = value
        setTextColor(Brand.text)
        textSize = 12f
        typeface = Typeface.DEFAULT_BOLD
        gravity = Gravity.CENTER
        setPadding(dp(10), dp(5), dp(10), dp(5))
        background = rounded(Brand.badge, dp(999), Brand.border)
    }

    private fun workspaceNavItem(workspace: Workspace): TextView {
        val view = TextView(this).apply {
            text = getString(workspace.labelRes)
            contentDescription = getString(R.string.workspace_open, getString(workspace.labelRes))
            textSize = if (resources.configuration.screenWidthDp >= 600) 13f else 11f
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            setPadding(dp(6), dp(8), dp(6), dp(8))
            isClickable = true
            isFocusable = true
            setOnClickListener { setWorkspace(workspace) }
        }
        workspaceNavButtons[workspace] = view
        styleWorkspaceNavItem(view, selected = workspace == activeWorkspace)
        return view
    }

    private fun styleWorkspaceNavItem(view: TextView, selected: Boolean) {
        view.setTextColor(if (selected) Brand.accent else Brand.textSoft)
        view.background = rounded(
            if (selected) Brand.accentSurface else Brand.panel,
            dp(14),
            if (selected) Brand.accent else Brand.borderSubtle
        )
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

    private fun toolbarButton(value: String, destructive: Boolean = false, onClick: () -> Unit): Button = Button(this).apply {
        text = value
        contentDescription = "$value action"
        setTextColor(if (destructive) Brand.danger else Brand.text)
        textSize = 12f
        typeface = Typeface.DEFAULT_BOLD
        background = rounded(if (destructive) Brand.dangerSurface else Brand.input, dp(14), if (destructive) Brand.danger else Brand.border)
        setOnClickListener { onClick() }
    }

    private fun trackRemoteAction(button: Button): Button {
        remoteActionButtons.add(button)
        return button
    }

    private fun trackBusySensitiveLocalAction(button: Button): Button {
        busySensitiveLocalButtons.add(button)
        return button
    }

    private fun trackSessionDisconnect(button: Button): Button {
        sessionDisconnectButtons.add(button)
        return button
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
        dp(46),
        weight
    )

    private fun dp(value: Int): Int = (value * resources.displayMetrics.density).roundToInt()

    private object Brand {
        val background: Int = Color.rgb(13, 17, 23)
        val panel: Int = Color.rgb(22, 27, 34)
        val panelStrong: Int = Color.rgb(28, 33, 40)
        val row: Int = Color.rgb(22, 27, 34)
        val input: Int = Color.rgb(13, 17, 23)
        val badge: Int = Color.rgb(33, 38, 45)
        val border: Int = Color.rgb(48, 54, 61)
        val borderSubtle: Int = Color.rgb(33, 38, 45)
        val accent: Int = Color.rgb(47, 129, 247)
        val accentSurface: Int = Color.rgb(15, 39, 68)
        val danger: Int = Color.rgb(248, 81, 73)
        val dangerSurface: Int = Color.rgb(48, 27, 32)
        val text: Int = Color.rgb(230, 237, 243)
        val textSoft: Int = Color.rgb(190, 202, 214)
        val muted: Int = Color.rgb(139, 148, 158)
    }

    private companion object {
        const val PICK_UPLOAD_REQUEST = 22091
        const val MAX_ACTIVITY_ROWS = 8

        const val STATE_PROTOCOL = "ghostftp.protocol"
        const val STATE_HOST = "ghostftp.host"
        const val STATE_PORT = "ghostftp.port"
        const val STATE_USERNAME = "ghostftp.username"
        const val STATE_FINGERPRINT = "ghostftp.fingerprint"
        const val STATE_REMOTE_PATH = "ghostftp.remotePath"
        const val STATE_TRANSFER_REMOTE_PATH = "ghostftp.transferRemotePath"
        const val STATE_UPLOAD_REMOTE_NAME = "ghostftp.uploadRemoteName"
        const val STATE_MKDIR_NAME = "ghostftp.mkdirName"
        const val STATE_RENAME_REMOTE_NAME = "ghostftp.renameRemoteName"
        const val STATE_UPLOAD_URI = "ghostftp.uploadUri"
        const val STATE_UPLOAD_DISPLAY_NAME = "ghostftp.uploadDisplayName"
        const val STATE_LAST_COMPLETED_PATH = "ghostftp.lastCompletedPath"
        const val STATE_WORKSPACE = "ghostftp.workspace"
        const val STATE_NAVIGATION_OPEN = "ghostftp.navigationOpen"

        const val WEBSITE_URL = "https://ghostftp.com/"
        const val SUPPORT_URL = "https://ghostftp.com/support/"
        const val DOCUMENTATION_URL = "https://ghostftp.com/docs/"
        const val PRIVACY_URL = "https://ghostftp.com/privacy/"
        const val EULA_URL = "https://github.com/bren-wp/Ghost-FTP/blob/main/EULA.txt"
        val OFFICIAL_LINKS = setOf(
            WEBSITE_URL,
            SUPPORT_URL,
            DOCUMENTATION_URL,
            PRIVACY_URL,
            EULA_URL
        )
    }
}

private class GhostMarkView(context: Context) : View(context) {
    private val bodyPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val eyePaint = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.rgb(10, 43, 81) }
    private val bodyPath = Path()

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        val w = width.toFloat()
        val h = height.toFloat()
        bodyPaint.shader = LinearGradient(
            0f,
            0f,
            w,
            h,
            intArrayOf(Color.rgb(244, 253, 255), Color.rgb(190, 235, 255), Color.rgb(66, 174, 255)),
            floatArrayOf(0f, 0.45f, 1f),
            Shader.TileMode.CLAMP
        )
        bodyPath.reset()
        bodyPath.moveTo(w * 0.16f, h * 0.80f)
        bodyPath.cubicTo(w * 0.25f, h * 0.76f, w * 0.30f, h * 0.66f, w * 0.31f, h * 0.54f)
        bodyPath.cubicTo(w * 0.34f, h * 0.25f, w * 0.44f, h * 0.13f, w * 0.50f, h * 0.13f)
        bodyPath.cubicTo(w * 0.64f, h * 0.13f, w * 0.71f, h * 0.32f, w * 0.73f, h * 0.54f)
        bodyPath.cubicTo(w * 0.74f, h * 0.66f, w * 0.80f, h * 0.76f, w * 0.84f, h * 0.80f)
        bodyPath.cubicTo(w * 0.77f, h * 0.89f, w * 0.66f, h * 0.88f, w * 0.61f, h * 0.77f)
        bodyPath.cubicTo(w * 0.57f, h * 0.88f, w * 0.43f, h * 0.88f, w * 0.39f, h * 0.77f)
        bodyPath.cubicTo(w * 0.34f, h * 0.88f, w * 0.22f, h * 0.89f, w * 0.16f, h * 0.80f)
        bodyPath.close()
        canvas.drawPath(bodyPath, bodyPaint)
        canvas.drawOval(RectF(w * 0.39f, h * 0.42f, w * 0.47f, h * 0.56f), eyePaint)
        canvas.drawOval(RectF(w * 0.57f, h * 0.42f, w * 0.65f, h * 0.56f), eyePaint)
    }
}