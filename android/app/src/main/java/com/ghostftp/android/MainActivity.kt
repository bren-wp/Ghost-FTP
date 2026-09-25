package com.ghostftp.android

import android.app.Activity
import android.app.AlertDialog
import android.content.Intent
import android.graphics.Color
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
    private lateinit var queueRows: LinearLayout
    private var activeProfile: ConnectionProfile? = null
    private var selectedUploadUri: Uri? = null
    private var selectedUploadDisplayName: String = ""
    private var lastCompletedTransferPath: String = ""

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.statusBarColor = Brand.background
        window.navigationBarColor = Brand.background
        setContentView(buildContent())
        showIdleState()
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != PICK_UPLOAD_REQUEST || resultCode != RESULT_OK) return
        val uri = data?.data ?: return
        selectedUploadUri = uri
        selectedUploadDisplayName = displayNameFor(uri)
        uploadSelectionText.text = "Selected local file: $selectedUploadDisplayName"
        if (uploadRemoteNameInput.text.toString().isBlank()) {
            uploadRemoteNameInput.setText(selectedUploadDisplayName)
        }
        transferStateText.text = "Upload file selected: $selectedUploadDisplayName"
        appendQueue("Upload selection", "Ready to upload $selectedUploadDisplayName from Android document storage.")
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
            text = "Native Android workspace for secure FTP, explicit FTPS and SFTP access with connection control, remote browsing, uploads, downloads, folder creation and guarded remote cleanup."
            setTextColor(Brand.textSoft)
            textSize = 15f
            setLineSpacing(0f, 1.15f)
        })
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
        }
        addView(statusTitle)
        addView(statusDetail)
    }

    private fun buildConnectionCard(): View = panel().apply {
        addView(sectionTitle("New connection"))
        addView(sectionDescription("Choose protocol, connect to a server and load the selected remote folder. Passwords stay in memory only for the active connection."))

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

        passwordInput = input("Password", InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD)
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
        actions.addView(gap(10))
        actions.addView(disconnectButton, buttonParams(weight = 1f))
        actions.addView(gap(10))
        actions.addView(refreshButton, buttonParams(weight = 1f))
        addView(actions)
    }

    private fun buildWorkspaceCard(): View = panel().apply {
        addView(sectionTitle("Remote workspace"))
        addView(sectionDescription("Tap a folder row to open it. Tap a file row to select it for download or guarded remote cleanup."))
        remoteRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(remoteRows)
    }

    private fun buildTransferCard(): View = panel().apply {
        addView(sectionTitle("Transfer actions"))
        addView(sectionDescription("Download remote files into app-private Android downloads, upload a selected Android document, create folders, or delete a remote file with confirmation."))

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

        val transferRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, 0)
        }
        transferRow.addView(track(primaryButton("Download") { downloadRemoteFile() }), buttonParams(weight = 1f))
        transferRow.addView(gap(10))
        transferRow.addView(track(secondaryButton("Delete file") { deleteRemoteFile() }), buttonParams(weight = 1f))
        addView(transferRow)

        val uploadRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, 0)
        }
        uploadRow.addView(track(secondaryButton("Pick file") { selectUploadFile() }), buttonParams(weight = 1f))
        uploadRow.addView(gap(10))
        uploadRow.addView(track(secondaryButton("Upload") { uploadSelectedFile() }), buttonParams(weight = 1f))
        addView(uploadRow)

        val folderRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, 0)
        }
        folderRow.addView(track(secondaryButton("Create folder") { createRemoteFolder() }), buttonParams(weight = 1f))
        addView(folderRow)

        queueRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(12), 0, 0)
        }
        addView(queueRows)
    }

    private fun buildFooter(): View = panel().apply {
        addView(TextView(this@MainActivity).apply {
            text = "Ghost FTP · Brendigo · Private Android session"
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
        statusDetail.text = "Loading ${profile.remotePath} from ${profile.host}:${profile.port} without storing credentials."

        thread(name = "ghostftp-android-connect") {
            val result = runCatching { controller.listRemote(profile) }
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
        selectedUploadUri = null
        selectedUploadDisplayName = ""
        lastCompletedTransferPath = ""
        passwordInput.text.clear()
        uploadSelectionText.text = "No local upload file selected."
        transferStateText.text = "No transfer started."
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

    private fun showReachable(profile: ConnectionProfile, result: ConnectionProbeResult) {
        statusTitle.text = result.title
        statusDetail.text = result.detail
        remoteRows.removeAllViews()
        result.rows.forEach { remoteRows.addView(remoteRow(it)) }
        queueRows.removeAllViews()
        queueRows.addView(row("Connection", "Ready for ${profile.protocol.label} transfer actions."))
        queueRows.addView(row("Security", "Password remains in memory only and is cleared on disconnect."))
        transferStateText.text = if (lastCompletedTransferPath.isBlank()) {
            "Ready for guarded transfer actions."
        } else {
            "Last completed remote path: $lastCompletedTransferPath"
        }
    }

    private fun showConnectionError(error: Throwable) {
        statusTitle.text = "Connection unavailable"
        statusDetail.text = error.message ?: "The selected endpoint did not open a session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Remote view", "No server session is active."))
        queueRows.removeAllViews()
        queueRows.addView(row("Transfer actions", "No active connection."))
        transferStateText.text = "No transfer can run until the connection opens."
    }

    private fun showMessage(title: String, detail: String) {
        statusTitle.text = title
        statusDetail.text = detail
        if (::transferStateText.isInitialized) transferStateText.text = detail
    }

    private fun showIdleState() {
        statusTitle.text = "Ready"
        statusDetail.text = "No active server session."
        remoteRows.removeAllViews()
        remoteRows.addView(row("Remote view", "Connect to a server endpoint to load remote files."))
        queueRows.removeAllViews()
        queueRows.addView(row("Transfer actions", "Connect first, then choose a remote file or upload target."))
        if (::transferStateText.isInitialized) transferStateText.text = "No transfer started."
        setBusy(false)
    }

    private fun setBusy(busy: Boolean) {
        connectButton.isEnabled = !busy
        refreshButton.isEnabled = !busy
        disconnectButton.isEnabled = !busy
        actionButtons.forEach { it.isEnabled = !busy }
    }

    private fun downloadRemoteFile() {
        val profile = readProfile() ?: return
        val remotePath = requiredRemoteFilePath() ?: return
        val outputFile = downloadTarget(remotePath)
        runTransfer(
            title = "Downloading",
            detail = "Saving $remotePath into Android app-private downloads."
        ) {
            controller.downloadRemote(profile, remotePath, outputFile)
        }
    }

    private fun uploadSelectedFile() {
        val profile = readProfile() ?: return
        val uri = selectedUploadUri
        if (uri == null) {
            showMessage("Choose a local file", "Pick an Android document before uploading.")
            return
        }
        val remoteTarget = uploadTargetPath() ?: return
        confirmUploadTarget(remoteTarget, selectedUploadDisplayName) {
            runTransfer(
                title = "Uploading",
                detail = "Sending $selectedUploadDisplayName to $remoteTarget.",
                refreshAfter = true
            ) {
                val input = contentResolver.openInputStream(uri)
                    ?: throw IllegalStateException("Unable to open selected Android document.")
                controller.uploadRemote(profile, input, remoteTarget)
            }
        }
    }

    private fun deleteRemoteFile() {
        val profile = readProfile() ?: return
        val remotePath = requiredRemoteFilePath() ?: return
        confirmDestructiveRemoteAction(
            title = "Delete remote file?",
            message = "This permanently removes $remotePath from the active server. This action cannot be undone by Ghost FTP Android.",
            confirmLabel = "Delete"
        ) {
            runTransfer(
                title = "Deleting remote file",
                detail = "Removing $remotePath from the active server.",
                refreshAfter = true
            ) {
                controller.deleteRemoteFile(profile, remotePath)
            }
        }
    }

    private fun createRemoteFolder() {
        val profile = readProfile() ?: return
        val folderName = mkdirNameInput.text.toString().trim()
        if (folderName.isBlank()) {
            showMessage("Folder name is required", "Enter a folder name or absolute remote folder path.")
            return
        }
        val remoteTarget = if (folderName.startsWith('/')) folderName else joinRemotePath(profile.remotePath, folderName)
        if (!validateRemoteTarget(remoteTarget)) return
        runTransfer(
            title = "Creating remote folder",
            detail = "Creating $remoteTarget on the active server.",
            refreshAfter = true
        ) {
            controller.createRemoteDirectory(profile, remoteTarget)
        }
    }

    private fun selectUploadFile() {
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = "*/*"
        }
        runCatching { startActivityForResult(intent, PICK_UPLOAD_REQUEST) }
            .onFailure { showMessage("File picker unavailable", it.message ?: "Android could not open a document picker.") }
    }

    private fun runTransfer(
        title: String,
        detail: String,
        refreshAfter: Boolean = false,
        action: () -> TransferResult
    ) {
        setBusy(true)
        statusTitle.text = title
        statusDetail.text = detail
        transferStateText.text = "In progress: $detail"
        appendQueue(title, detail)
        thread(name = "ghostftp-android-transfer") {
            val result = runCatching { action() }
            runOnUiThread {
                setBusy(false)
                result.fold(
                    onSuccess = {
                        showTransferResult(it)
                        if (refreshAfter) {
                            appendQueue("Refreshing remote workspace", "Reloading current remote folder after ${it.title.lowercase(Locale.ROOT)}.")
                            refreshActive()
                        }
                    },
                    onFailure = { showTransferFailure(it) }
                )
            }
        }
    }

    private fun showTransferResult(result: TransferResult) {
        lastCompletedTransferPath = result.remotePath
        statusTitle.text = result.title
        statusDetail.text = result.detail
        transferStateText.text = "${result.title}: ${result.remotePath}"
        appendQueue(result.title, result.detail)
    }

    private fun showTransferFailure(error: Throwable) {
        val detail = error.message ?: "The transfer action did not complete."
        statusTitle.text = "Transfer failed"
        statusDetail.text = detail
        transferStateText.text = "Transfer failed: $detail"
        appendQueue("Transfer failed", detail)
    }

    private fun requiredRemoteFilePath(): String? {
        val raw = transferRemotePathInput.text.toString().trim()
        if (raw.isBlank()) {
            showMessage("Remote file path is required", "Tap a remote file row or enter an absolute remote file path.")
            return null
        }
        val remotePath = if (raw.startsWith('/')) raw else joinRemotePath(remotePathInput.text.toString(), raw)
        return if (validateRemoteTarget(remotePath)) remotePath else null
    }

    private fun uploadTargetPath(): String? {
        val targetName = uploadRemoteNameInput.text.toString().trim().ifBlank { selectedUploadDisplayName }
        if (targetName.isBlank()) {
            showMessage("Upload target is required", "Choose a local file and enter the target file name or path.")
            return null
        }
        val remotePath = if (targetName.startsWith('/')) targetName else joinRemotePath(remotePathInput.text.toString(), targetName)
        return if (validateRemoteTarget(remotePath)) remotePath else null
    }

    private fun validateRemoteTarget(remotePath: String): Boolean {
        val parts = remotePath.split('/').filter { it.isNotBlank() }
        if (parts.any { it == "." || it == ".." }) {
            showMessage("Unsafe remote path", "Remote paths cannot contain . or .. segments.")
            return false
        }
        return true
    }

    private fun confirmUploadTarget(remoteTarget: String, localName: String, onConfirm: () -> Unit) {
        AlertDialog.Builder(this)
            .setTitle("Upload to remote path?")
            .setMessage("Upload $localName to $remoteTarget. If a file with that name already exists, the server may replace it.")
            .setPositiveButton("Upload") { _, _ -> onConfirm() }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun confirmDestructiveRemoteAction(
        title: String,
        message: String,
        confirmLabel: String,
        onConfirm: () -> Unit
    ) {
        AlertDialog.Builder(this)
            .setTitle(title)
            .setMessage(message)
            .setPositiveButton(confirmLabel) { _, _ -> onConfirm() }
            .setNegativeButton("Cancel", null)
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
                appendQueue("Selected remote file", target)
            }
        }
    }

    private fun appendQueue(title: String, detail: String) {
        while (queueRows.childCount >= MAX_QUEUE_ROWS) {
            queueRows.removeViewAt(0)
        }
        queueRows.addView(row(title, detail))
    }

    private fun downloadTarget(remotePath: String): File {
        val dir = getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS)
            ?: File(filesDir, "downloads")
        dir.mkdirs()
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
        contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
            val index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            if (index >= 0 && cursor.moveToFirst()) {
                val value = cursor.getString(index)
                if (!value.isNullOrBlank()) return safeFileName(value)
            }
        }
        return safeFileName(uri.lastPathSegment?.substringAfterLast('/') ?: "ghostftp-upload.bin")
    }

    private fun safeFileName(value: String): String {
        return value.trim().replace(Regex("[^A-Za-z0-9._-]"), "_").ifBlank { "ghostftp-file.bin" }
    }

    private fun joinRemotePath(directory: String, child: String): String {
        val safeChild = child.trim().trimStart('/')
        val base = directory.trim().ifBlank { "/" }.trimEnd('/')
        return if (base.isBlank()) "/$safeChild" else "$base/$safeChild"
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

    private companion object {
        const val PICK_UPLOAD_REQUEST = 22091
        const val MAX_QUEUE_ROWS = 8
    }
}
