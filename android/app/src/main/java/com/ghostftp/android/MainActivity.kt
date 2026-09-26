package com.ghostftp.android

import android.app.Activity
import android.app.AlertDialog
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
import android.os.Bundle
import android.os.Environment
import android.provider.OpenableColumns
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
import java.io.File
import java.util.Locale
import kotlin.concurrent.thread
import kotlin.math.roundToInt

class MainActivity : Activity() {
    private val controller = ConnectionController()
    private val actionButtons = mutableListOf<Button>()
    private lateinit var statusTitle: TextView
    private lateinit var statusDetail: TextView
    private lateinit var hostInput: EditText
    private lateinit var portInput: EditText
    private lateinit var usernameInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var hostKeyFingerprintInput: EditText
    private lateinit var remotePathInput: EditText
    private lateinit var transferRemotePathInput: EditText
    private lateinit var uploadRemoteNameInput: EditText
    private lateinit var mkdirNameInput: EditText
    private lateinit var uploadSelectionText: TextView
    private lateinit var transferStateText: TextView
    private lateinit var protocolSpinner: Spinner
    private lateinit var connectButton: Button
    private lateinit var disconnectButton: Button
    private lateinit var refreshButton: Button
    private lateinit var remoteRows: LinearLayout
    private lateinit var activityRows: LinearLayout
    private var activeProfile: ConnectionProfile? = null
    private var selectedUploadUri: Uri? = null
    private var selectedUploadDisplayName: String = ""
    private var lastCompletedTransferPath: String = ""
    private var operationGeneration: Long = 0
    private var operationInFlight = false

    @Volatile
    private var activityClosing = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        activityClosing = false
        window.statusBarColor = Brand.background
        window.navigationBarColor = Brand.background
        setContentView(buildContent())
        showIdleState()
    }

    override fun onDestroy() {
        activityClosing = true
        selectedUploadUri = null
        activeProfile = null
        if (::passwordInput.isInitialized) passwordInput.text.clear()
        super.onDestroy()
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != PICK_UPLOAD_REQUEST || resultCode != RESULT_OK || !uiReady()) return
        val uri = data?.data ?: return
        selectedUploadUri = uri
        selectedUploadDisplayName = displayNameFor(uri)
        uploadSelectionText.text = "Selected local file: $selectedUploadDisplayName"
        if (uploadRemoteNameInput.text.toString().isBlank()) {
            uploadRemoteNameInput.setText(selectedUploadDisplayName)
        }
        transferStateText.text = "Upload file selected: $selectedUploadDisplayName"
        appendActivity("Upload", "Selected $selectedUploadDisplayName from Android document storage.")
    }

    private fun buildContent(): View = ScrollView(this).apply {
        setBackgroundColor(Brand.background)
        addView(
            LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(dp(14), dp(14), dp(14), dp(22))
                addView(buildHeader())
                addView(space(12))
                addView(buildStatusCard())
                addView(space(12))
                addView(buildConnectionCard())
                addView(space(12))
                addView(buildFilesCard())
                addView(space(12))
                addView(buildTransfersCard())
                addView(space(12))
                addView(buildFooter())
            },
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

        val workspaceRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, 0)
        }
        workspaceRow.addView(workspaceChip("Files"))
        workspaceRow.addView(gap(8))
        workspaceRow.addView(TextView(this@MainActivity).apply {
            text = ReleaseInfo.VERSION_DISPLAY
            setTextColor(Brand.muted)
            textSize = 13f
        })
        addView(workspaceRow)

        addView(space(12))
        val toolbar = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        toolbar.addView(toolbarButton("Refresh") { refreshActive() }, buttonParams(weight = 1f))
        toolbar.addView(gap(8))
        toolbar.addView(track(toolbarButton("Upload") { uploadOrPickFile() }), buttonParams(weight = 1f))
        toolbar.addView(gap(8))
        toolbar.addView(track(toolbarButton("Download") { downloadRemoteFile() }), buttonParams(weight = 1f))
        addView(toolbar)

        val toolbarMore = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(8), 0, 0)
        }
        toolbarMore.addView(track(toolbarButton("New Folder") { createRemoteFolder() }), buttonParams(weight = 1f))
        toolbarMore.addView(gap(8))
        toolbarMore.addView(track(toolbarButton("Delete", destructive = true) { deleteRemoteFile() }), buttonParams(weight = 1f))
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

    private fun buildConnectionCard(): View = panel().apply {
        addView(sectionTitle("Sites"))
        addView(sectionDescription("Connect to FTP, explicit FTPS or SFTP. Passwords stay in memory for the active session and are cleared on disconnect."))

        protocolSpinner = Spinner(this@MainActivity).apply {
            adapter = ArrayAdapter(
                this@MainActivity,
                android.R.layout.simple_spinner_item,
                ConnectionProtocol.entries.map { it.label }
            ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        }
        addView(formLabel("Protocol"))
        addView(protocolSpinner)

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
        addView(formLabel("SFTP host key fingerprint"))
        addView(hostKeyFingerprintInput)

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
        actions.addView(gap(8))
        actions.addView(disconnectButton, buttonParams(weight = 1f))
        actions.addView(gap(8))
        actions.addView(refreshButton, buttonParams(weight = 1f))
        addView(actions)
    }

    private fun buildFilesCard(): View = panel().apply {
        addView(sectionTitle("Files"))
        addView(sectionDescription("Open folders, select files, then use the toolbar actions aligned with Ghost FTP desktop."))
        remoteRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(remoteRows)
    }

    private fun buildTransfersCard(): View = panel().apply {
        addView(sectionTitle("Transfers"))
        addView(sectionDescription("Manage the selected remote file, upload target and remote folder action for the active session."))

        transferRemotePathInput = input("/remote/file.txt", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Remote file path"))
        addView(transferRemotePathInput)

        uploadRemoteNameInput = input("Remote upload file name", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Upload target name or path"))
        addView(uploadRemoteNameInput)

        mkdirNameInput = input("New remote folder", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI)
        addView(formLabel("Folder name or path"))
        addView(mkdirNameInput)

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
        uploadRow.addView(track(secondaryButton("Pick file") { selectUploadFile() }), buttonParams(weight = 1f))
        uploadRow.addView(gap(8))
        uploadRow.addView(track(secondaryButton("Upload") { uploadSelectedFile() }), buttonParams(weight = 1f))
        addView(uploadRow)

        activityRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        addView(activityRows)
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
        if (operationInFlight) return
        val profile = readProfile() ?: return
        val generation = ++operationGeneration
        operationInFlight = true
        setBusy(true)
        statusTitle.text = "Opening ${profile.protocol.label}"
        statusDetail.text = "Loading ${profile.remotePath} from ${profile.host}:${profile.port}."

        thread(name = "ghostftp-android-connect") {
            val result = runCatching { controller.listRemote(profile) }
            safeUi {
                if (generation != operationGeneration) return@safeUi
                operationInFlight = false
                setBusy(false)
                result.fold(
                    onSuccess = {
                        activeProfile = profile
                        showReachable(profile, it)
                    },
                    onFailure = {
                        activeProfile = null
                        showConnectionError(it)
                    }
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
        val nextPath = remotePathInput.text.toString().trim().ifBlank { profile.remotePath }
        activeProfile = profile.copy(remotePath = nextPath)
        openConnection()
    }

    private fun disconnect() {
        operationGeneration += 1
        operationInFlight = false
        activeProfile = null
        selectedUploadUri = null
        selectedUploadDisplayName = ""
        lastCompletedTransferPath = ""
        if (::passwordInput.isInitialized) passwordInput.text.clear()
        if (::uploadSelectionText.isInitialized) uploadSelectionText.text = "No local upload file selected."
        if (::transferStateText.isInitialized) transferStateText.text = "No transfer started."
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
        connectButton.isEnabled = !busy
        refreshButton.isEnabled = !busy
        disconnectButton.isEnabled = true
        actionButtons.forEach { it.isEnabled = !busy }
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
        ) {
            controller.downloadRemote(profile, remotePath, outputFile)
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
            ) {
                val input = contentResolver.openInputStream(uri)
                    ?: throw IllegalStateException("Unable to open selected Android document.")
                controller.uploadRemote(profile, input, remoteTarget)
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
            title = "Delete remote file?",
            message = "This permanently removes $remotePath from the active server.",
            confirmLabel = "Delete"
        ) {
            runTransfer(
                title = "Deleting",
                detail = "Removing $remotePath from the active server.",
                refreshAfter = true
            ) {
                controller.deleteRemoteFile(profile, remotePath)
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
        ) {
            controller.createRemoteDirectory(profile, remoteTarget)
        }
    }

    private fun selectUploadFile() {
        if (closingOrDestroyed()) return
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = "*/*"
        }
        runCatching { startActivityForResult(intent, PICK_UPLOAD_REQUEST) }
            .onFailure { showMessage("File picker unavailable", it.message ?: "Android could not open a document picker.") }
    }

    private fun runTransfer(title: String, detail: String, refreshAfter: Boolean = false, action: () -> TransferResult) {
        if (operationInFlight) return
        val generation = ++operationGeneration
        operationInFlight = true
        setBusy(true)
        statusTitle.text = title
        statusDetail.text = detail
        transferStateText.text = detail
        appendActivity(title, detail)
        thread(name = "ghostftp-android-transfer") {
            val result = runCatching { action() }
            safeUi {
                if (generation != operationGeneration) return@safeUi
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
        val detail = error.message ?: "The transfer action did not complete."
        statusTitle.text = "Transfer failed"
        statusDetail.text = detail
        transferStateText.text = detail
        appendActivity("Transfer failed", detail)
    }

    private fun requiredRemoteFilePath(profile: ConnectionProfile): String? {
        val raw = transferRemotePathInput.text.toString().trim()
        if (raw.isBlank()) {
            showMessage("Remote file path is required", "Tap a remote file row or enter an absolute remote file path.")
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
        if (closingOrDestroyed()) return
        AlertDialog.Builder(this)
            .setTitle("Upload to remote path?")
            .setMessage("Upload $localName to $remoteTarget on the active server.")
            .setNegativeButton("Cancel", null)
            .setPositiveButton("Upload") { _, _ -> if (!closingOrDestroyed()) onConfirm() }
            .show()
    }

    private fun confirmDestructiveRemoteAction(title: String, message: String, confirmLabel: String, onConfirm: () -> Unit) {
        if (closingOrDestroyed()) return
        AlertDialog.Builder(this)
            .setTitle(title)
            .setMessage(message)
            .setNegativeButton("Cancel", null)
            .setPositiveButton(confirmLabel) { _, _ -> if (!closingOrDestroyed()) onConfirm() }
            .show()
    }

    private fun remoteRow(item: RemoteRow): View = row(item.name, item.detail).apply {
        val target = item.remotePath ?: return@apply
        when {
            item.isDirectory -> setOnClickListener {
                remotePathInput.setText(target)
                openConnection()
            }
            item.isFile -> setOnClickListener {
                transferRemotePathInput.setText(target)
                transferStateText.text = "Selected remote file: $target"
                appendActivity("Selected", target)
            }
        }
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

    private fun workspaceChip(value: String): TextView = TextView(this).apply {
        text = value
        setTextColor(Brand.background)
        textSize = 13f
        typeface = Typeface.DEFAULT_BOLD
        gravity = Gravity.CENTER
        setPadding(dp(12), dp(6), dp(12), dp(6))
        background = rounded(Brand.accent, dp(999), Brand.accent)
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
        setTextColor(if (destructive) Brand.danger else Brand.text)
        textSize = 12f
        typeface = Typeface.DEFAULT_BOLD
        background = rounded(if (destructive) Brand.dangerSurface else Brand.input, dp(14), if (destructive) Brand.danger else Brand.border)
        setOnClickListener { onClick() }
    }

    private fun track(button: Button): Button {
        actionButtons.add(button)
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
        val danger: Int = Color.rgb(248, 81, 73)
        val dangerSurface: Int = Color.rgb(48, 27, 32)
        val text: Int = Color.rgb(230, 237, 243)
        val textSoft: Int = Color.rgb(190, 202, 214)
        val muted: Int = Color.rgb(139, 148, 158)
    }

    private companion object {
        const val PICK_UPLOAD_REQUEST = 22091
        const val MAX_ACTIVITY_ROWS = 8
        const val MAX_QUEUE_ROWS = MAX_ACTIVITY_ROWS
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