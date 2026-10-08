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
import android.view.WindowManager
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Spinner
import android.widget.TextView
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.io.File
import java.util.Locale
import kotlin.concurrent.thread
import kotlin.math.roundToInt

class MainActivity : Activity() {
    private enum class Workspace(val labelRes: Int? = null, val fixedLabel: String? = null) {
        FILES(R.string.workspace_files),
        SITES(R.string.workspace_sites),
        TRANSFERS(R.string.workspace_transfers),
        SYNC(null, "Sync & Backup"),
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
    private lateinit var sftpKeepAliveInput: EditText
    private lateinit var sftpKeepAliveGroup: View
    private lateinit var remotePathInput: EditText
    private lateinit var transferRemotePathInput: EditText
    private lateinit var uploadRemoteNameInput: EditText
    private lateinit var mkdirNameInput: EditText
    private lateinit var renameRemoteNameInput: EditText
    private lateinit var uploadSelectionText: TextView
    private lateinit var transferStateText: TextView
    private lateinit var syncBackupStatus: TextView
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
    private lateinit var syncSection: View
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
        window.addFlags(WindowManager.LayoutParams.FLAG_SECURE)
        clearLegacyPersistedDocumentGrants()
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
        if (::sftpKeepAliveInput.isInitialized) outState.putString(STATE_KEEP_ALIVE, sftpKeepAliveInput.text.toString())
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
        if (resultCode != RESULT_OK || !uiReady()) return

        when (requestCode) {
            PICK_UPLOAD_REQUEST -> handleUploadSelection(data)
            CREATE_SETTINGS_BACKUP_REQUEST -> data?.data?.let(::writeConnectionSettingsBackup)
            RESTORE_SETTINGS_BACKUP_REQUEST -> data?.data?.let(::restoreConnectionSettingsBackup)
        }
    }

    private fun clearLegacyPersistedDocumentGrants() {
        for (permission in contentResolver.persistedUriPermissions) {
            var flags = 0
            if (permission.isReadPermission) flags = flags or Intent.FLAG_GRANT_READ_URI_PERMISSION
            if (permission.isWritePermission) flags = flags or Intent.FLAG_GRANT_WRITE_URI_PERMISSION
            if (flags == 0) continue
            runCatching {
                contentResolver.releasePersistableUriPermission(permission.uri, flags)
            }
        }
    }

    private fun handleUploadSelection(data: Intent?) {
        val uri = data?.data ?: return
        if (!canReadUploadUri(uri)) {
            selectedUploadUri = null
            selectedUploadDisplayName = ""
            showMessage(
                getString(R.string.msg_selected_file_unavailable_title),
                getString(R.string.msg_selected_file_unavailable_detail)
            )
            appendActivity(
                getString(R.string.activity_upload_selection_rejected),
                getString(R.string.activity_upload_selection_rejected_detail)
            )
            return
        }
        selectedUploadUri = uri
        selectedUploadDisplayName = displayNameFor(uri)
        setWorkspace(Workspace.TRANSFERS)
        uploadSelectionText.text = getString(R.string.state_selected_local_file, selectedUploadDisplayName)
        if (uploadRemoteNameInput.text.toString().isBlank()) {
            uploadRemoteNameInput.setText(selectedUploadDisplayName)
        }
        transferStateText.text = getString(R.string.state_upload_file_selected, selectedUploadDisplayName)
        appendActivity(
            getString(R.string.action_upload),
            getString(R.string.activity_upload_selected_detail, selectedUploadDisplayName)
        )
    }

    private fun buildContent(): View {
        sitesSection = buildConnectionCard()
        filesSection = buildFilesCard()
        transfersSection = buildTransfersCard()
        syncSection = buildSyncBackupCard()
        settingsSection = buildSettingsCard()
        aboutSection = buildAboutCard()

        workspaceContainer = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(filesSection)
            addView(sitesSection)
            addView(transfersSection)
            addView(syncSection)
            addView(settingsSection)
            addView(aboutSection)
        }

        // The 480px visual reference uses an always-visible compact rail.
        // Very narrow devices keep the overlay so file actions remain usable.
        val compactNavigation = usesOverlayNavigation()
        navigationOpen = !compactNavigation
        val railWidth = when {
            compactNavigation -> dp(152)
            resources.configuration.screenWidthDp < 600 -> dp(88)
            else -> dp(184)
        }
        navigationRail = buildNavigationRail().apply {
            visibility = if (navigationOpen) View.VISIBLE else View.GONE
        }
        navigationScrim = View(this).apply {
            setBackgroundColor(Color.argb(156, 0, 0, 0))
            visibility = View.GONE
            isClickable = true
            isFocusable = true
            contentDescription = getString(R.string.nav_close_overlay)
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
                text = getString(R.string.label_private_session)
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
            contentDescription = if (navigationOpen) getString(R.string.nav_close) else getString(R.string.nav_open)
            textSize = 20f
            minWidth = 0
            minimumWidth = 0
        }
        if (usesOverlayNavigation()) {
            titleRow.addView(navigationToggleButton, LinearLayout.LayoutParams(dp(48), dp(44)))
            titleRow.addView(gap(8))
        }
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
            text = workspaceLabel(activeWorkspace)
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
        addView(label(getString(R.string.label_session)))
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
        contentDescription = getString(R.string.action_confirm)

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
            contentDescription = getString(R.string.action_cancel)
        }, buttonParams(weight = 1f))
        actions.addView(gap(8))
        confirmationAction = primaryButton(getString(R.string.action_confirm)) {
            val action = pendingConfirmation
            clearInlineConfirmation()
            if (!closingOrDestroyed()) action?.invoke()
        }.apply {
            contentDescription = getString(R.string.action_confirm)
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
        confirmationAction.contentDescription = confirmLabel
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
        addView(sectionDescription(getString(R.string.desc_sites)))

        protocolSpinner = Spinner(this@MainActivity).apply {
            contentDescription = getString(R.string.field_protocol)
            adapter = ArrayAdapter(
                this@MainActivity,
                android.R.layout.simple_spinner_item,
                ConnectionProtocol.entries.map { it.label }
            ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        }
        addView(formLabel(getString(R.string.field_protocol)))
        addView(protocolSpinner)
        protocolSecurityText = TextView(this@MainActivity).apply {
            textSize = 12f
            setPadding(0, dp(6), 0, dp(4))
        }
        addView(protocolSecurityText)

        hostInput = input(getString(R.string.field_host), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel(getString(R.string.field_host)))
        addView(hostInput)

        portInput = input("21", InputType.TYPE_CLASS_NUMBER)
        addView(formLabel(getString(R.string.field_port)))
        addView(portInput)

        usernameInput = input(getString(R.string.field_username), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_NORMAL)
        addView(formLabel(getString(R.string.field_username)))
        addView(usernameInput)

        passwordInput = input(getString(R.string.field_password), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD).apply {
            // Session credentials are memory-only: do not persist view state and
            // do not expose this field to Android Autofill services.
            isSaveEnabled = false
            importantForAutofill = View.IMPORTANT_FOR_AUTOFILL_NO_EXCLUDE_DESCENDANTS
            filterTouchesWhenObscured = true
        }
        addView(formLabel(getString(R.string.field_password)))
        addView(passwordInput)

        hostKeyFingerprintInput = input(getString(R.string.hint_sftp_fingerprint), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_NORMAL)
        sftpFingerprintGroup = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            addView(formLabel(getString(R.string.field_sftp_fingerprint)))
            addView(hostKeyFingerprintInput)
        }
        addView(sftpFingerprintGroup)

        sftpKeepAliveInput = input("15", InputType.TYPE_CLASS_NUMBER).apply {
            setText("15")
        }
        sftpKeepAliveGroup = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            addView(formLabel("SFTP keep-alive (seconds)"))
            addView(sftpKeepAliveInput)
        }
        addView(sftpKeepAliveGroup)

        remotePathInput = input("/", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        remotePathInput.setText("/")
        addView(formLabel(getString(R.string.field_remote_path)))
        addView(remotePathInput)

        val actions = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(14), 0, 0)
        }
        connectButton = primaryButton(getString(R.string.action_connect)) { openConnection() }.apply {
            contentDescription = getString(R.string.action_connect)
        }
        disconnectButton = secondaryButton(getString(R.string.action_disconnect)) { disconnect() }.apply {
            contentDescription = getString(R.string.action_disconnect)
        }
        refreshButton = secondaryButton(getString(R.string.action_refresh)) { refreshActive() }.apply {
            contentDescription = getString(R.string.action_refresh)
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
        if (::sftpKeepAliveGroup.isInitialized) {
            sftpKeepAliveGroup.visibility =
                if (protocol == ConnectionProtocol.SFTP) View.VISIBLE else View.GONE
        }
        if (::protocolSecurityText.isInitialized) {
            val (message, color) = when (protocol) {
                ConnectionProtocol.FTP ->
                    getString(R.string.security_ftp_warning) to Brand.danger
                ConnectionProtocol.EXPLICIT_FTPS ->
                    getString(R.string.security_ftps_info) to Brand.textSoft
                ConnectionProtocol.SFTP ->
                    getString(R.string.security_sftp_info) to Brand.textSoft
            }
            protocolSecurityText.text = message
            protocolSecurityText.setTextColor(color)
            protocolSecurityText.contentDescription = message
        }
    }

    private fun buildFilesCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_files)))
        addView(sectionDescription(getString(R.string.desc_files)))
        remoteRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(remoteRows)
    }

    private fun buildTransfersCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_transfers)))
        addView(sectionDescription(getString(R.string.desc_transfers)))

        transferRemotePathInput = input(getString(R.string.hint_transfer_remote_path), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel(getString(R.string.field_transfer_remote_path)))
        addView(transferRemotePathInput)

        uploadRemoteNameInput = input(getString(R.string.hint_upload_target), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel(getString(R.string.field_upload_target)))
        addView(uploadRemoteNameInput)

        mkdirNameInput = input(getString(R.string.hint_folder_target), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel(getString(R.string.field_folder_target)))
        addView(mkdirNameInput)

        renameRemoteNameInput = input(getString(R.string.hint_rename_target), InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel(getString(R.string.field_rename_target)))
        addView(renameRemoteNameInput)

        uploadSelectionText = TextView(this@MainActivity).apply {
            text = getString(R.string.state_no_local_file)
            setTextColor(Brand.textSoft)
            textSize = 13f
            setPadding(0, dp(10), 0, dp(4))
        }
        addView(uploadSelectionText)

        transferStateText = TextView(this@MainActivity).apply {
            text = getString(R.string.state_no_transfer)
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
            contentDescription = getString(R.string.action_pick_file)
        }), buttonParams(weight = 1f))
        uploadRow.addView(gap(8))
        uploadRow.addView(trackRemoteAction(secondaryButton(getString(R.string.action_upload)) { uploadSelectedFile() }.apply {
            contentDescription = getString(R.string.action_upload)
        }), buttonParams(weight = 1f))
        addView(uploadRow)

        activityRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        addView(activityRows)
    }

    private fun buildSyncBackupCard(): View = panel().apply {
        addView(sectionTitle(workspaceLabel(Workspace.SYNC)))
        addView(
            sectionDescription(
                "Back up and restore non-secret connection settings through Android documents. Passwords are never included."
            )
        )

        syncBackupStatus = TextView(this@MainActivity).apply {
            text = "No settings backup has been created or restored in this session."
            setTextColor(Brand.textSoft)
            textSize = 13f
            setPadding(0, dp(8), 0, dp(8))
        }
        addView(syncBackupStatus)

        val actions = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        actions.addView(
            trackBusySensitiveLocalAction(
                secondaryButton("Back up settings") { createConnectionSettingsBackup() }.apply {
                    contentDescription = "Back up settings"
                }
            ),
            buttonParams(weight = 1f)
        )
        actions.addView(gap(8))
        actions.addView(
            trackBusySensitiveLocalAction(
                secondaryButton("Restore settings") { chooseConnectionSettingsBackup() }.apply {
                    contentDescription = "Restore settings"
                }
            ),
            buttonParams(weight = 1f)
        )
        addView(actions)
    }

    private fun buildSettingsCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_settings)))
        addView(sectionDescription(getString(R.string.desc_settings)))
        addView(row(getString(R.string.label_privacy), getString(R.string.settings_no_tracking)))
        addView(row(getString(R.string.label_credentials), getString(R.string.settings_credentials)))
        addView(row(getString(R.string.label_security), getString(R.string.settings_security)))

        val firstRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        firstRow.addView(secondaryButton(getString(R.string.action_clear_activity)) { clearActivityLog() }.apply {
            contentDescription = getString(R.string.action_clear_activity)
        }, buttonParams(weight = 1f))
        firstRow.addView(gap(8))
        firstRow.addView(trackBusySensitiveLocalAction(secondaryButton(getString(R.string.action_reset_transfers)) { resetTransferFields() }.apply {
            contentDescription = getString(R.string.action_reset_transfers)
        }), buttonParams(weight = 1f))
        addView(firstRow)

        val secondRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(8), 0, 0)
        }
        secondRow.addView(trackBusySensitiveLocalAction(secondaryButton(getString(R.string.action_reset_connection)) { resetConnectionForm() }.apply {
            contentDescription = getString(R.string.action_reset_connection)
        }), buttonParams(weight = 1f))
        secondRow.addView(gap(8))
        secondRow.addView(trackSessionDisconnect(secondaryButton(getString(R.string.action_disconnect)) { disconnect() }.apply {
            contentDescription = "${getString(R.string.workspace_settings)} ${getString(R.string.action_disconnect)}"
        }), buttonParams(weight = 1f))
        addView(secondRow)
    }

    private fun buildAboutCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_about)))
        addView(sectionDescription("Ghost FTP ${ReleaseInfo.VERSION_DISPLAY} · Build ${ReleaseInfo.BUILD}"))
        addView(row(getString(R.string.label_product), "Ghost FTP · Brendigo"))
        addView(row(getString(R.string.label_protocols), "FTP · Explicit FTPS · SFTP"))
        addView(officialLinkRow(getString(R.string.label_support), getString(R.string.about_support_desc), SUPPORT_URL))
        addView(officialLinkRow(getString(R.string.label_documentation), getString(R.string.about_docs_desc), DOCUMENTATION_URL))
        addView(officialLinkRow(getString(R.string.label_privacy_policy), getString(R.string.about_privacy_desc), PRIVACY_URL))
        addView(officialLinkRow(getString(R.string.label_eula), getString(R.string.about_eula_desc), EULA_URL))
        addView(officialLinkRow(getString(R.string.label_official_website), getString(R.string.about_website_desc), WEBSITE_URL))
    }

    private fun setWorkspace(workspace: Workspace, announce: Boolean = true) {
        activeWorkspace = workspace
        val targets = mapOf(
            Workspace.FILES to filesSection,
            Workspace.SITES to sitesSection,
            Workspace.TRANSFERS to transfersSection,
            Workspace.SYNC to syncSection,
            Workspace.SETTINGS to settingsSection,
            Workspace.ABOUT to aboutSection
        )
        targets.forEach { (key, view) ->
            view.visibility = if (key == workspace) View.VISIBLE else View.GONE
        }
        workspaceNavButtons.forEach { (key, view) ->
            styleWorkspaceNavItem(view, selected = key == workspace)
        }
        if (::workspaceTitle.isInitialized) workspaceTitle.text = workspaceLabel(workspace)
        if (::workspaceContainer.isInitialized) {
            workspaceContainer.contentDescription = getString(R.string.workspace_content, workspaceLabel(workspace))
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
                    workspaceContainer.announceForAccessibility(getString(R.string.workspace_announce, workspaceLabel(workspace)))
                }
            }
        }
        if (announce && usesOverlayNavigation()) {
            setNavigationOpen(false, announce = false)
        }
    }

    private fun usesOverlayNavigation(): Boolean =
        resources.configuration.screenWidthDp < 360

    private fun setNavigationOpen(open: Boolean, announce: Boolean = true) {
        val compactNavigation = usesOverlayNavigation()
        // Rotation / Activity restore must never hide a permanently docked rail.
        navigationOpen = if (compactNavigation) open else true
        if (::navigationRail.isInitialized) {
            navigationRail.visibility = if (navigationOpen) View.VISIBLE else View.GONE
        }
        if (::navigationScrim.isInitialized) {
            navigationScrim.visibility =
                if (compactNavigation && navigationOpen) View.VISIBLE else View.GONE
            navigationScrim.isFocusable = compactNavigation && navigationOpen
        }
        if (::navigationToggleButton.isInitialized) {
            navigationToggleButton.text = if (open) "←" else "☰"
            navigationToggleButton.contentDescription =
                if (open) getString(R.string.nav_close) else getString(R.string.nav_open)
            if (announce) {
                navigationToggleButton.announceForAccessibility(
                    if (open) getString(R.string.nav_opened) else getString(R.string.nav_closed)
                )
            }
        }
    }

    private fun buildFooter(): View = panel().apply {
        addView(TextView(this@MainActivity).apply {
            text = "Ghost FTP · Brendigo · ${getString(R.string.label_private_session)}"
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
        statusTitle.text = getString(R.string.status_opening_protocol, profile.protocol.label)
        statusDetail.text = getString(R.string.status_loading_remote, profile.remotePath, profile.host, profile.port)

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
                        showConnectionError(it, profile)
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
        if (::uploadSelectionText.isInitialized) uploadSelectionText.text = getString(R.string.state_no_local_file)
        if (::transferStateText.isInitialized) transferStateText.text = getString(R.string.state_no_transfer)
        showIdleState()
    }

    private fun clearActivityLog() {
        if (::activityRows.isInitialized) activityRows.removeAllViews()
        showMessage(getString(R.string.action_clear_activity), getString(R.string.msg_activity_cleared_detail))
    }

    private fun resetTransferFields() {
        selectedUploadUri = null
        selectedUploadDisplayName = ""
        lastCompletedTransferPath = ""
        if (::transferRemotePathInput.isInitialized) transferRemotePathInput.text.clear()
        if (::uploadRemoteNameInput.isInitialized) uploadRemoteNameInput.text.clear()
        if (::mkdirNameInput.isInitialized) mkdirNameInput.text.clear()
        if (::renameRemoteNameInput.isInitialized) renameRemoteNameInput.text.clear()
        if (::uploadSelectionText.isInitialized) uploadSelectionText.text = getString(R.string.state_no_local_file)
        if (::transferStateText.isInitialized) transferStateText.text = getString(R.string.state_no_transfer)
        showMessage(getString(R.string.action_reset_transfers), getString(R.string.msg_transfers_reset_detail))
    }

    private fun resetConnectionForm() {
        disconnect()
        if (::protocolSpinner.isInitialized) protocolSpinner.setSelection(ConnectionProtocol.FTP.ordinal)
        if (::hostInput.isInitialized) hostInput.text.clear()
        if (::portInput.isInitialized) portInput.setText(ConnectionProtocol.FTP.defaultPort.toString())
        if (::usernameInput.isInitialized) usernameInput.text.clear()
        if (::hostKeyFingerprintInput.isInitialized) hostKeyFingerprintInput.text.clear()
        if (::sftpKeepAliveInput.isInitialized) sftpKeepAliveInput.setText("15")
        if (::remotePathInput.isInitialized) remotePathInput.setText("/")
        showMessage(getString(R.string.action_reset_connection), getString(R.string.msg_connection_reset_detail))
    }

    private fun readProfile(): ConnectionProfile? {
        val protocol = ConnectionProtocol.fromIndex(protocolSpinner.selectedItemPosition)
        val port = portInput.text.toString().trim().ifBlank { protocol.defaultPort.toString() }.toIntOrNull()
        if (port == null || port !in 1..65535) {
            showMessage(getString(R.string.field_port), getString(R.string.msg_invalid_port_detail))
            return null
        }
        val keepAliveSeconds = sftpKeepAliveInput.text.toString().trim().ifBlank { "15" }.toIntOrNull()
        if (protocol == ConnectionProtocol.SFTP && (keepAliveSeconds == null || keepAliveSeconds !in 5..300)) {
            showMessage(getString(R.string.label_connection), "Use an SFTP keep-alive interval between 5 and 300 seconds.")
            return null
        }
        val host = hostInput.text.toString().trim()
        if (host.isBlank()) {
            showMessage(getString(R.string.field_host), getString(R.string.msg_host_required_detail))
            return null
        }
        return ConnectionProfile(
            protocol = protocol,
            host = host,
            port = port,
            username = usernameInput.text.toString().trim(),
            password = passwordInput.text.toString(),
            hostKeyFingerprint = hostKeyFingerprintInput.text.toString().trim(),
            remotePath = remotePathInput.text.toString().trim().ifBlank { "/" },
            keepAliveSeconds = keepAliveSeconds ?: 15
        )
    }

    private fun activeTransferProfile(): ConnectionProfile? {
        val profile = activeProfile
        if (profile == null) {
            showMessage(getString(R.string.action_connect), getString(R.string.msg_connect_first_detail))
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
        activityRows.addView(row(getString(R.string.label_connection), getString(R.string.activity_connection_ready, profile.protocol.label)))
        activityRows.addView(row(getString(R.string.label_security), getString(R.string.activity_password_memory)))
        transferStateText.text = if (lastCompletedTransferPath.isBlank()) {
            getString(R.string.state_ready_guarded_actions)
        } else {
            getString(R.string.state_last_completed_remote_path, lastCompletedTransferPath)
        }
    }

    private fun showConnectionError(error: Throwable, profile: ConnectionProfile) {
        if (!uiReady()) return
        statusTitle.text = getString(R.string.status_connection_unavailable)
        val rawDetail = error.message ?: getString(R.string.status_connection_unavailable_detail)
        statusDetail.text = redactSensitiveErrorText(rawDetail, listOf(profile.password))
        remoteRows.removeAllViews()
        remoteRows.addView(row(getString(R.string.workspace_files), getString(R.string.state_no_server_session)))
        activityRows.removeAllViews()
        activityRows.addView(row(getString(R.string.workspace_transfers), getString(R.string.state_no_active_connection)))
        transferStateText.text = getString(R.string.state_transfer_wait_connection)
    }

    private fun showMessage(title: String, detail: String) {
        if (!::statusTitle.isInitialized || !::statusDetail.isInitialized) return
        statusTitle.text = title
        statusDetail.text = detail
        if (::transferStateText.isInitialized) transferStateText.text = detail
    }

    private fun showIdleState() {
        if (!uiReady()) return
        statusTitle.text = getString(R.string.status_ready)
        statusDetail.text = getString(R.string.status_no_active_session)
        remoteRows.removeAllViews()
        remoteRows.addView(row(getString(R.string.workspace_files), getString(R.string.state_connect_to_load_files)))
        activityRows.removeAllViews()
        activityRows.addView(row(getString(R.string.workspace_transfers), getString(R.string.state_connect_then_choose_action)))
        transferStateText.text = getString(R.string.state_no_transfer)
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
            title = getString(R.string.action_download),
            detail = getString(R.string.detail_download_remote, remotePath)
        ) { cancellation ->
            controller.downloadRemote(profile, remotePath, outputFile, cancellation)
        }
    }

    private fun uploadSelectedFile() {
        val profile = activeTransferProfile() ?: return
        val uri = selectedUploadUri
        if (uri == null) {
            showMessage(getString(R.string.action_pick_file), getString(R.string.msg_choose_local_file_detail))
            return
        }
        val remoteTarget = uploadTargetPath(profile) ?: return
        val uploadName = selectedUploadDisplayName.ifBlank { getString(R.string.label_selected_file) }
        confirmUploadTarget(remoteTarget, uploadName) {
            runTransfer(
                title = getString(R.string.action_upload),
                detail = getString(R.string.detail_upload_remote, uploadName, remoteTarget),
                refreshAfter = true
            ) { cancellation ->
                val input = contentResolver.openInputStream(uri)
                    ?: throw IllegalStateException(getString(R.string.error_open_selected_document))
                controller.uploadRemote(profile, input, remoteTarget, cancellation)
            }
        }
    }

    private fun deleteRemoteFile() {
        val profile = activeTransferProfile() ?: return
        val remotePath = requiredRemoteFilePath(profile) ?: return
        if (remotePath.split('/').filter { it.isNotBlank() }.isEmpty()) {
            showMessage(getString(R.string.action_delete), getString(R.string.msg_unsafe_delete_detail))
            return
        }
        confirmDestructiveRemoteAction(
            title = getString(R.string.confirm_delete_title),
            message = getString(R.string.confirm_delete_detail, remotePath),
            confirmLabel = getString(R.string.action_delete)
        ) {
            runTransfer(
                title = getString(R.string.action_delete),
                detail = getString(R.string.detail_delete_remote, remotePath),
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
            showMessage(getString(R.string.action_rename), getString(R.string.msg_unsafe_rename_detail))
            return
        }
        val rawTarget = renameRemoteNameInput.text.toString().trim()
        if (rawTarget.isBlank()) {
            showMessage(getString(R.string.field_rename_target), getString(R.string.msg_rename_target_required_detail))
            return
        }
        val destinationPath = normalizeRemoteInput(rawTarget, profile.remotePath) ?: return
        if (destinationPath == sourcePath) {
            showMessage(getString(R.string.field_rename_target), getString(R.string.msg_rename_target_unchanged_detail))
            return
        }
        confirmDestructiveRemoteAction(
            title = getString(R.string.confirm_rename_title),
            message = getString(R.string.confirm_rename_detail, sourcePath, destinationPath),
            confirmLabel = getString(R.string.action_rename)
        ) {
            runTransfer(
                title = getString(R.string.action_rename),
                detail = getString(R.string.detail_rename_remote, sourcePath, destinationPath),
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
            showMessage(getString(R.string.field_folder_target), getString(R.string.msg_folder_name_required_detail))
            return
        }
        val remoteTarget = normalizeRemoteInput(folderName, profile.remotePath) ?: return
        runTransfer(
            title = getString(R.string.action_new_folder),
            detail = getString(R.string.detail_create_folder, remoteTarget),
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
            .onFailure {
                showMessage(
                    getString(R.string.action_pick_file),
                    it.message ?: getString(R.string.msg_file_picker_unavailable_detail)
                )
            }
    }

    private fun createConnectionSettingsBackup() {
        if (closingOrDestroyed() || operationInFlight) return
        val intent = Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = "application/json"
            putExtra(Intent.EXTRA_TITLE, "GhostFTP-Connection.json")
        }
        runCatching { startActivityForResult(intent, CREATE_SETTINGS_BACKUP_REQUEST) }
            .onFailure {
                showMessage(workspaceLabel(Workspace.SYNC), "The Android document picker could not be opened.")
            }
    }

    private fun chooseConnectionSettingsBackup() {
        if (closingOrDestroyed() || operationInFlight) return
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = "application/json"
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION)
        }
        runCatching { startActivityForResult(intent, RESTORE_SETTINGS_BACKUP_REQUEST) }
            .onFailure {
                showMessage(workspaceLabel(Workspace.SYNC), "The Android document picker could not be opened.")
            }
    }

    private fun writeConnectionSettingsBackup(uri: Uri) {
        val protocol = ConnectionProtocol.fromIndex(protocolSpinner.selectedItemPosition)
        val json = JSONObject()
            .put("schemaVersion", 1)
            .put("protocol", protocol.name)
            .put("host", hostInput.text.toString().trim())
            .put("port", portInput.text.toString().trim().ifBlank { protocol.defaultPort.toString() }.toIntOrNull() ?: protocol.defaultPort)
            .put("username", usernameInput.text.toString().trim())
            .put("hostKeyFingerprint", hostKeyFingerprintInput.text.toString().trim())
            .put("keepAliveSeconds", sftpKeepAliveInput.text.toString().trim().ifBlank { "15" }.toIntOrNull() ?: 15)
            .put("remotePath", remotePathInput.text.toString().trim().ifBlank { "/" })

        runCatching {
            contentResolver.openOutputStream(uri, "wt")?.use { output ->
                output.write(json.toString(2).toByteArray(Charsets.UTF_8))
                output.flush()
            } ?: error("The selected backup document could not be opened for writing.")
        }.fold(
            onSuccess = {
                syncBackupStatus.text = "Connection settings backed up. The session password was not exported."
                showMessage(workspaceLabel(Workspace.SYNC), syncBackupStatus.text.toString())
            },
            onFailure = {
                syncBackupStatus.text = "The connection-settings backup could not be written."
                showMessage(workspaceLabel(Workspace.SYNC), syncBackupStatus.text.toString())
            }
        )
    }

    private fun restoreConnectionSettingsBackup(uri: Uri) {
        runCatching {
            val json = JSONObject(readBoundedDocumentText(uri))
            require(json.optInt("schemaVersion", 0) == 1) { "Unsupported Ghost FTP backup schema." }

            val protocol = ConnectionProtocol.valueOf(json.getString("protocol"))
            val port = json.optInt("port", protocol.defaultPort)
            require(port in 1..65535) { "Invalid port in Ghost FTP backup." }

            val keepAlive = json.optInt("keepAliveSeconds", 15)
            require(protocol != ConnectionProtocol.SFTP || keepAlive in 5..300) {
                "Invalid SFTP keep-alive interval in Ghost FTP backup."
            }

            disconnect()
            protocolSpinner.setSelection(protocol.ordinal)
            hostInput.setText(json.optString("host").trim())
            portInput.setText(port.toString())
            usernameInput.setText(json.optString("username").trim())
            passwordInput.text.clear()
            hostKeyFingerprintInput.setText(json.optString("hostKeyFingerprint").trim())
            sftpKeepAliveInput.setText(keepAlive.toString())
            remotePathInput.setText(json.optString("remotePath", "/").trim().ifBlank { "/" })
            selectedProtocol = protocol
            updateProtocolSpecificFields(protocol)
        }.fold(
            onSuccess = {
                syncBackupStatus.text = "Connection settings restored. Reconnect and enter the password again."
                showMessage(workspaceLabel(Workspace.SYNC), syncBackupStatus.text.toString())
            },
            onFailure = {
                syncBackupStatus.text = "The selected Ghost FTP settings backup is invalid or unreadable."
                showMessage(workspaceLabel(Workspace.SYNC), syncBackupStatus.text.toString())
            }
        )
    }

    private fun readBoundedDocumentText(uri: Uri, maximumBytes: Int = 256 * 1024): String {
        val input = contentResolver.openInputStream(uri)
            ?: error("The selected backup document could not be opened.")
        input.use { stream ->
            val output = ByteArrayOutputStream()
            val buffer = ByteArray(8 * 1024)
            while (true) {
                val read = stream.read(buffer)
                if (read < 0) break
                require(output.size() + read <= maximumBytes) { "Ghost FTP settings backup is too large." }
                output.write(buffer, 0, read)
            }
            return output.toString(Charsets.UTF_8.name())
        }
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
        val rawDetail = error.message ?: if (canceled) {
            getString(R.string.msg_operation_canceled)
        } else {
            getString(R.string.msg_transfer_failed_detail)
        }
        val detail = redactSensitiveErrorText(
            rawDetail,
            listOfNotNull(activeProfile?.password)
        )
        val title = if (canceled) {
            getString(R.string.status_transfer_canceled)
        } else {
            getString(R.string.status_transfer_failed)
        }
        statusTitle.text = title
        statusDetail.text = detail
        transferStateText.text = detail
        appendActivity(title, detail)
    }

    private fun restoreUiState(state: Bundle) {
        protocolSpinner.setSelection(state.getInt(STATE_PROTOCOL, 0))
        hostInput.setText(state.getString(STATE_HOST).orEmpty())
        portInput.setText(state.getString(STATE_PORT).orEmpty())
        usernameInput.setText(state.getString(STATE_USERNAME).orEmpty())
        hostKeyFingerprintInput.setText(state.getString(STATE_FINGERPRINT).orEmpty())
        sftpKeepAliveInput.setText(state.getString(STATE_KEEP_ALIVE).orEmpty().ifBlank { "15" })
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
            state.getBoolean(STATE_NAVIGATION_OPEN, !usesOverlayNavigation()),
            announce = false
        )

        passwordInput.text.clear()
        activeProfile = null
        activeCancellation = null
        operationInFlight = false
        if (selectedUploadUri != null && selectedUploadDisplayName.isNotBlank()) {
            uploadSelectionText.text = getString(R.string.state_selected_local_file, selectedUploadDisplayName)
        }
        statusTitle.text = getString(R.string.status_ready)
        statusDetail.text = if (restoredUploadUri != null && !restoredUploadReadable) {
            getString(R.string.restore_upload_unreadable)
        } else {
            getString(R.string.restore_reconnect)
        }
        transferStateText.text = if (lastCompletedTransferPath.isBlank()) {
            getString(R.string.restore_previous_session_ended)
        } else {
            getString(R.string.state_last_completed_remote_path, lastCompletedTransferPath)
        }
        setBusy(false)
    }

    private fun requiredRemoteFilePath(profile: ConnectionProfile): String? {
        val raw = transferRemotePathInput.text.toString().trim()
        if (raw.isBlank()) {
            showMessage(getString(R.string.field_transfer_remote_path), getString(R.string.msg_remote_path_required_detail))
            return null
        }
        return normalizeRemoteInput(raw, profile.remotePath)
    }

    private fun uploadTargetPath(profile: ConnectionProfile): String? {
        val targetName = uploadRemoteNameInput.text.toString().trim().ifBlank { selectedUploadDisplayName }
        if (targetName.isBlank()) {
            showMessage(getString(R.string.field_upload_target), getString(R.string.msg_upload_target_required_detail))
            return null
        }
        return normalizeRemoteInput(targetName, profile.remotePath)
    }

    private fun normalizeRemoteInput(raw: String, directory: String): String? {
        val resolved = if (raw.startsWith('/')) raw else joinRemotePath(directory, raw)
        val blocked = resolved.split('/').filter { it.isNotBlank() }.any { it == "." || it == ".." }
        if (blocked) {
            showMessage(getString(R.string.field_transfer_remote_path), getString(R.string.msg_unsupported_remote_path_detail))
            return null
        }
        return resolved
    }

    private fun confirmUploadTarget(remoteTarget: String, localName: String, onConfirm: () -> Unit) {
        requestInlineConfirmation(
            title = getString(R.string.confirm_upload_title),
            detail = getString(R.string.confirm_upload_detail, localName, remoteTarget),
            confirmLabel = getString(R.string.action_upload),
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
            getString(R.string.state_selected_remote_folder, target)
        } else {
            getString(R.string.state_selected_remote_file, target)
        }
        appendActivity(getString(R.string.label_selected), target)
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
            contentDescription = getString(R.string.link_open, title)
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

    private fun workspaceLabel(workspace: Workspace): String =
        workspace.labelRes?.let { getString(it) } ?: workspace.fixedLabel ?: workspace.name

    private fun workspaceNavItem(workspace: Workspace): TextView {
        val label = workspaceLabel(workspace)
        val view = TextView(this).apply {
            text = label
            contentDescription = getString(R.string.workspace_open, label)
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
        filterTouchesWhenObscured = true
        text = value
        setTextColor(Brand.background)
        textSize = 14f
        typeface = Typeface.DEFAULT_BOLD
        background = rounded(Brand.accent, dp(14), Brand.accent)
        setOnClickListener { onClick() }
    }

    private fun secondaryButton(value: String, onClick: () -> Unit): Button = Button(this).apply {
        filterTouchesWhenObscured = true
        text = value
        setTextColor(Brand.text)
        textSize = 14f
        background = rounded(Brand.input, dp(14), Brand.border)
        setOnClickListener { onClick() }
    }

    private fun toolbarButton(value: String, destructive: Boolean = false, onClick: () -> Unit): Button = Button(this).apply {
        filterTouchesWhenObscured = true
        text = value
        contentDescription = value
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
        val background: Int = Color.rgb(7, 14, 26)
        val panel: Int = Color.rgb(13, 25, 43)
        val panelStrong: Int = Color.rgb(17, 31, 52)
        val row: Int = Color.rgb(13, 25, 43)
        val input: Int = Color.rgb(7, 14, 26)
        val badge: Int = Color.rgb(25, 47, 72)
        val border: Int = Color.rgb(39, 70, 99)
        val borderSubtle: Int = Color.rgb(25, 47, 72)
        val accent: Int = Color.rgb(56, 171, 255)
        val accentSurface: Int = Color.rgb(19, 48, 72)
        val danger: Int = Color.rgb(255, 114, 132)
        val dangerSurface: Int = Color.rgb(57, 32, 45)
        val text: Int = Color.rgb(234, 246, 255)
        val textSoft: Int = Color.rgb(184, 205, 225)
        val muted: Int = Color.rgb(119, 144, 172)
    }

    private companion object {
        const val PICK_UPLOAD_REQUEST = 22091
        const val CREATE_SETTINGS_BACKUP_REQUEST = 22092
        const val RESTORE_SETTINGS_BACKUP_REQUEST = 22093
        const val MAX_ACTIVITY_ROWS = 8

        const val STATE_PROTOCOL = "ghostftp.protocol"
        const val STATE_HOST = "ghostftp.host"
        const val STATE_PORT = "ghostftp.port"
        const val STATE_USERNAME = "ghostftp.username"
        const val STATE_FINGERPRINT = "ghostftp.fingerprint"
        const val STATE_KEEP_ALIVE = "ghostftp.keepAlive"
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