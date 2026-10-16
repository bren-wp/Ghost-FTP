package com.ghostftp.android

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.OpenableColumns
import android.provider.DocumentsContract
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
import android.widget.ImageView
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
    private enum class Workspace(val labelRes: Int) {
        FILES(R.string.workspace_files),
        SITES(R.string.workspace_sites),
        TRANSFERS(R.string.workspace_transfers),
        SYNC(R.string.workspace_sync),
        CLOUD(R.string.workspace_cloud),
        BOOKMARKS(R.string.workspace_bookmarks),
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
    private lateinit var siteNameInput: EditText
    private lateinit var savedSitesSpinner: Spinner
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
    private lateinit var transferOverviewText: TextView
    private lateinit var transferQueueSurface: LinearLayout
    private lateinit var transferDetailsSurface: LinearLayout
    private lateinit var transferDetailsStatus: TextView
    private var sessionCompletedOperations = 0
    private var sessionFailedOperations = 0
    private var sessionCancelledOperations = 0
    private lateinit var syncBackupStatus: TextView
    private lateinit var cloudSection: View
    private lateinit var cloudTransferOptionsPanel: View
    private lateinit var bookmarksSection: View
    private lateinit var searchSection: View
    private lateinit var searchResults: LinearLayout
    private lateinit var searchQueryInput: EditText
    private lateinit var searchFeedback: TextView
    private var searchMode = false
    private var searchRemote = true
    private var searchMatchCase = false
    private var searchRecursive = false
    private var lastRemoteInventory: List<RemoteRow> = emptyList()
    private lateinit var bookmarksRows: LinearLayout
    private lateinit var cloudUploadInput: EditText
    private lateinit var cloudDownloadInput: EditText
    private lateinit var cloudStatus: TextView
    private lateinit var cloudCancelButton: Button
    private lateinit var cloudRecoveryNotice: TextView
    private lateinit var cloudRecoveryCleanupButton: Button
    private lateinit var cloudRecoveryKeepButton: Button
    private lateinit var cloudUploadButton: Button
    private lateinit var cloudDownloadButton: Button
    private var cloudCancellation: CloudTransferCancellation? = null
    private var cloudPendingLink: SignedCloudLink? = null
    private var cloudTransferThread: Thread? = null
    private var cloudTransferBusy = false
    private lateinit var protocolSpinner: Spinner
    private lateinit var protocolSecurityText: TextView
    private lateinit var connectButton: Button
    private lateinit var disconnectButton: Button
    private lateinit var refreshButton: Button
    private lateinit var remoteRows: LinearLayout
    private lateinit var localRows: LinearLayout
    private lateinit var remoteDirectoryLabel: TextView
    private lateinit var localDirectoryLabel: TextView
    private var localBrowseDirectory: File? = null
    private lateinit var activityRows: LinearLayout
    private lateinit var contentScroll: ScrollView
    private lateinit var workspaceContainer: LinearLayout
    private lateinit var workspaceTitle: TextView
    private lateinit var phoneChromeTitle: TextView
    private lateinit var phoneMenuButton: Button
    private lateinit var phoneQuickButton: Button
    // Reference workspaces must not inherit Files-only actions or status cards.
    private lateinit var fileActionToolbar: View
    private lateinit var sessionStatusCard: View
    private lateinit var workspaceFeedbackPanel: LinearLayout
    private lateinit var workspaceFeedbackTitle: TextView
    private lateinit var workspaceFeedbackDetail: TextView
    private lateinit var settingsFeedback: TextView
    private lateinit var navigationRail: View
    private lateinit var navigationScrim: View
    private lateinit var navigationToggleButton: Button
    private var navigationOpen = false
    private lateinit var confirmationPanel: LinearLayout
    private lateinit var confirmationTitle: TextView
    private lateinit var confirmationDetail: TextView
    private lateinit var confirmationAction: Button
    private lateinit var inlineNameInput: EditText
    private var pendingConfirmation: (() -> Unit)? = null
    private val workspaceNavButtons = mutableMapOf<Workspace, TextView>()
    private var activeWorkspace = Workspace.FILES
    private lateinit var sitesSection: View
    private lateinit var sitesOverviewPanel: View
    private lateinit var siteEditorPanel: View
    private lateinit var siteFeedback: TextView
    private lateinit var sitesListRows: LinearLayout
    private lateinit var sitesSearch: EditText
    private var siteEditorVisible = false
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
        // Process death skips a transfer worker's finally block. Clear only
        // our app-private cloud staging files before new work can begin.
        CloudStageCleanup.purgeOnProcessStart(cacheDir)
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
        displayPendingCloudRecovery()
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
        outState.putBoolean(STATE_SITE_EDITOR_OPEN, siteEditorVisible)
        outState.putBoolean(STATE_NAVIGATION_OPEN, navigationOpen)
        // Intentionally never persist passwordInput or activeProfile: a recreated
        // Activity must require a fresh authenticated connection.
        super.onSaveInstanceState(outState)
    }

    override fun onDestroy() {
        activityClosing = true
        cloudCancellation?.cancel()
        cloudCancellation = null
        cloudTransferThread?.interrupt()
        cloudTransferThread = null
        cloudPendingLink = null
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
        if (resultCode != RESULT_OK || !uiReady()) {
            if (requestCode == PICK_CLOUD_UPLOAD_REQUEST || requestCode == CREATE_CLOUD_DOWNLOAD_REQUEST) {
                cloudPendingLink = null
                updateCloudPickerButtons()
            }
            return
        }

        when (requestCode) {
            PICK_UPLOAD_REQUEST -> handleUploadSelection(data)
            CREATE_SETTINGS_BACKUP_REQUEST -> data?.data?.let(::writeConnectionSettingsBackup)
            RESTORE_SETTINGS_BACKUP_REQUEST -> data?.data?.let(::restoreConnectionSettingsBackup)
            PICK_CLOUD_UPLOAD_REQUEST, CREATE_CLOUD_DOWNLOAD_REQUEST -> {
                val link = cloudPendingLink
                cloudPendingLink = null
                updateCloudPickerButtons()
                val uri = data?.data
                if (link != null && uri != null) {
                    val upload = requestCode == PICK_CLOUD_UPLOAD_REQUEST
                    if (upload) {
                        runCloudTransfer(upload = true, link = link, uri = uri)
                    } else {
                        // ACTION_CREATE_DOCUMENT already created a potentially
                        // empty user document: journal it BEFORE any bytes move.
                        val recorded = runCatching {
                            CloudDestinationJournal.record(noBackupFilesDir, uri.toString())
                        }.isSuccess
                        if (recorded) {
                            runCloudTransfer(upload = false, link = link, uri = uri)
                        } else {
                            // Recovery storage may be unavailable or already
                            // contain another unresolved document. Never call
                            // DocumentsProvider.deleteDocument on the UI thread.
                            cleanupUnstartedCloudDestination(uri)
                        }
                    }
                } else if (requestCode == CREATE_CLOUD_DOWNLOAD_REQUEST && uri != null) {
                    // Android may recreate this Activity while the system picker
                    // is visible. The URL is deliberately not saved in Bundle;
                    // delete the newly created orphan SAF output when the
                    // missing in-memory authorization prevents any transfer.
                    cleanupUnstartedCloudDestination(uri)
                } else {
                    cloudStatus.text = "No cloud document was selected."
                }
            }
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
        siteEditorPanel = buildConnectionCard()
        sitesOverviewPanel = buildSitesOverview()
        sitesSection = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(sitesOverviewPanel)
            addView(siteEditorPanel)
        }
        siteEditorPanel.visibility = View.GONE
        filesSection = buildFilesCard()
        transfersSection = buildTransfersCard()
        syncSection = buildSyncBackupCard()
        cloudSection = buildCloudCard()
        bookmarksSection = buildBookmarksCard()
        searchSection = buildSearchCard().apply { visibility = View.GONE }
        settingsSection = buildSettingsCard()
        aboutSection = buildAboutCard()

        workspaceContainer = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(filesSection)
            addView(sitesSection)
            addView(transfersSection)
            addView(syncSection)
            addView(cloudSection)
            addView(bookmarksSection)
            addView(searchSection)
            addView(settingsSection)
            addView(aboutSection)
        }

        // User-approved 16-screen reference: full-width phone content with a
        // slide-over drawer; a persistent rail is reserved for tablets.
        val compactNavigation = usesOverlayNavigation()
        navigationOpen = !compactNavigation
        val railWidth = when {
            compactNavigation -> dp(if (resources.configuration.screenWidthDp < 400) 260 else 280)
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
        val outerPadding = if (resources.configuration.screenWidthDp < 600) dp(10) else dp(14)
        val content = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(outerPadding, dp(14), outerPadding, dp(22))
        }
        // Construct the existing live actions, but do not render an extra
        // competing title/toolbar above the premium phone workspace.
        val legacyHeader = buildHeader()
        legacyHeader.visibility =
            if (resources.configuration.screenWidthDp < 600) View.GONE else View.VISIBLE
        content.addView(legacyHeader)
        sessionStatusCard = buildStatusCard().apply {
            contentDescription = "Session status panel"
        }
        content.addView(sessionStatusCard)
        // Status can be hidden in the approved phone reference when no
        // server is connected. Errors and lifecycle recovery MUST still
        // have a visible, accessibility-readable home in the active page.
        workspaceFeedbackPanel = buildWorkspaceFeedback()
        content.addView(workspaceFeedbackPanel)
        content.addView(space(10))
        content.addView(buildInlineConfirmationCard())
        content.addView(space(10))
        content.addView(workspaceContainer)
        // The 16 approved phone references have no persistent "Private
        // session" marketing footer; it consumed viewport space on EVERY
        // Android page. Keep provenance footer on roomy tablets only.
        if (resources.configuration.screenWidthDp >= 600) {
            content.addView(space(10))
            content.addView(buildFooter())
        }

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
        // The approved phone reference has a top title, full-width content
        // and contextual file actions above the Android system navigation.
        return LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(Brand.background)
            if (resources.configuration.screenWidthDp < 600) {
                addView(buildPremiumPhoneChrome(), LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, dp(52)
                ))
            }
            addView(root, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f
            ))
            addView(fileActionToolbar, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ))
        }
    }

    private fun buildPremiumPhoneChrome(): View = LinearLayout(this).apply {
        gravity = Gravity.CENTER_VERTICAL
        orientation = LinearLayout.HORIZONTAL
        setPadding(dp(8), dp(4), dp(10), dp(4))
        setBackgroundColor(Brand.background)

        phoneMenuButton = secondaryButton("☰") {
            if (activeWorkspace == Workspace.SITES && siteEditorVisible) {
                showSitesOverview()
            } else {
                setNavigationOpen(!navigationOpen)
            }
        }.apply {
            contentDescription = getString(R.string.nav_open)
            textSize = 20f
            minWidth = 0
            minimumWidth = 0
        }
        addView(phoneMenuButton, LinearLayout.LayoutParams(dp(44), dp(44)))
        addView(gap(8))
        phoneChromeTitle = TextView(this@MainActivity).apply {
            text = workspaceLabel(activeWorkspace)
            setTextColor(Brand.text)
            typeface = Typeface.DEFAULT_BOLD
            textSize = 18f
            gravity = Gravity.CENTER_VERTICAL
        }
        addView(phoneChromeTitle, LinearLayout.LayoutParams(
            0, ViewGroup.LayoutParams.MATCH_PARENT, 1f
        ))
        phoneQuickButton = secondaryButton("⋯") { }.apply {
            contentDescription = "Quick navigation"
            textSize = 22f
            minWidth = 0
            minimumWidth = 0
        }
        phoneQuickButton.setOnClickListener {
            when {
                activeWorkspace == Workspace.SITES && !siteEditorVisible -> showSiteEditor(null)
                activeWorkspace == Workspace.BOOKMARKS -> promptSaveCurrentBookmark()
                activeWorkspace == Workspace.FILES && !searchMode -> openSearch()
                searchMode -> closeSearch()
                else -> showPhoneQuickMenu()
            }
        }
        addView(phoneQuickButton, LinearLayout.LayoutParams(dp(46), dp(42)))
        updatePhoneChromeActions()
    }

    private fun showPhoneQuickMenu() {
        android.widget.PopupMenu(this, phoneQuickButton).apply {
                menu.add(0, 1, 0, workspaceLabel(Workspace.SETTINGS))
                menu.add(0, 2, 1, workspaceLabel(Workspace.CLOUD))
                menu.add(0, 3, 2, workspaceLabel(Workspace.ABOUT))
                setOnMenuItemClickListener { item ->
                    when (item.itemId) {
                        1 -> setWorkspace(Workspace.SETTINGS)
                        2 -> setWorkspace(Workspace.CLOUD)
                        3 -> setWorkspace(Workspace.ABOUT)
                    }
                    true
                }
            }.show()
    }

    private fun updatePhoneChromeActions() {
        if (!::phoneQuickButton.isInitialized) return
        if (::phoneMenuButton.isInitialized) {
            if (activeWorkspace == Workspace.SITES && siteEditorVisible) {
                phoneMenuButton.text = "‹"
                phoneMenuButton.contentDescription = "Back to Sites"
            } else if (phoneMenuButton.contentDescription == "Back to Sites") {
                phoneMenuButton.text = if (navigationOpen) "←" else "☰"
                phoneMenuButton.contentDescription =
                    if (navigationOpen) getString(R.string.nav_close)
                    else getString(R.string.nav_open)
            }
        }
        val sitesOverview = activeWorkspace == Workspace.SITES && !siteEditorVisible
        val bookmarks = activeWorkspace == Workspace.BOOKMARKS
        when {
            searchMode -> {
                phoneQuickButton.text = "×"
                phoneQuickButton.contentDescription = "Close Search"
            }
            activeWorkspace == Workspace.FILES -> {
                phoneQuickButton.text = "⌕"
                phoneQuickButton.contentDescription = "Search files"
            }
            sitesOverview -> {
                phoneQuickButton.text = "+"
                phoneQuickButton.contentDescription = "Add Site"
            }
            bookmarks -> {
                phoneQuickButton.text = "+"
                phoneQuickButton.contentDescription = getString(R.string.bookmarks_add)
            }
            else -> {
                phoneQuickButton.text = "⋯"
                phoneQuickButton.contentDescription = "Quick navigation"
            }
        }
    }

    private fun buildNavigationRail(): View = ScrollView(this).apply {
        isFillViewport = true
        isVerticalScrollBarEnabled = false
        importantForAccessibility = View.IMPORTANT_FOR_ACCESSIBILITY_NO
        background = rounded(Brand.panelStrong, 0, Brand.border)

        val rail = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.START
            val iconRail = false
            setPadding(dp(13), dp(16), dp(13), dp(16))

            addView(LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                addView(ImageView(this@MainActivity).apply {
                    setImageResource(R.drawable.ic_ghost_premium_splash)
                    contentDescription = "Ghost FTP"
                }, LinearLayout.LayoutParams(dp(44), dp(44)))
                addView(PremiumReferenceStyle.caption(this@MainActivity,
                    "Ghost FTP", 19f).apply { typeface = Typeface.DEFAULT_BOLD },
                    LinearLayout.LayoutParams(0, dp(44), 1f))
            })
            addView(space(19))

            Workspace.entries.forEach { workspace ->
                addView(
                    workspaceNavItem(workspace),
                    LinearLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        dp(if (resources.configuration.screenWidthDp >= 600) 52 else 48)
                    )
                )
                addView(space(3))
            }

            addView(space(12))
            if (!iconRail) addView(TextView(this@MainActivity).apply {
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
        if (resources.configuration.screenWidthDp !in 360..599) addView(titleRow)

        workspaceTitle = TextView(this@MainActivity).apply {
            text = workspaceLabel(activeWorkspace)
            setTextColor(Brand.textSoft)
            textSize = 13f
            typeface = Typeface.DEFAULT_BOLD
            setPadding(0, dp(8), 0, 0)
        }
        addView(workspaceTitle)

        addView(space(12))
        // Reference #6: one five-action footer, not permanent toolbars above files.
        fileActionToolbar = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(5), dp(4), dp(5), dp(4))
            setBackgroundColor(PremiumReferenceStyle.raised)
            contentDescription = "File actions"
            val actions = listOf(
                trackRemoteAction(toolbarButton("⇧\n" + getString(R.string.action_upload)) {
                    uploadOrPickFile()
                }.apply { contentDescription = getString(R.string.action_upload) }),
                trackRemoteAction(toolbarButton("⇩\n" + getString(R.string.action_download)) {
                    downloadRemoteFile()
                }.apply { contentDescription = getString(R.string.action_download) }),
                trackRemoteAction(toolbarButton("▱\n" + getString(R.string.action_new_folder)) {
                    promptNewRemoteFolder()
                }.apply { contentDescription = getString(R.string.action_new_folder) }),
                trackRemoteAction(toolbarButton("⌫\n" + getString(R.string.action_delete)) {
                    deleteRemoteFile()
                }.apply { contentDescription = getString(R.string.action_delete) })
            )
            actions.forEach { item ->
                item.setAllCaps(false)
                item.textSize = 11f
                item.setPadding(dp(2), dp(3), dp(2), dp(3))
                addView(item, LinearLayout.LayoutParams(0, dp(51), 1f))
            }
            val more = toolbarButton("⋯\nMore") { }.apply {
                setAllCaps(false)
                textSize = 11f
                contentDescription = "More file actions"
            }
            more.setOnClickListener {
                android.widget.PopupMenu(this@MainActivity, more).apply {
                    menu.add(0, 1, 0, getString(R.string.action_refresh))
                    menu.add(0, 2, 1, getString(R.string.action_rename))
                    menu.add(0, 3, 2, getString(R.string.workspace_transfers))
                    setOnMenuItemClickListener { selected ->
                        when (selected.itemId) {
                            1 -> refreshActive()
                            2 -> promptRenameRemoteEntry()
                            3 -> setWorkspace(Workspace.TRANSFERS)
                        }
                        true
                    }
                }.show()
            }
            addView(more, LinearLayout.LayoutParams(0, dp(51), 1f))
        }
    }

    private fun promptNewRemoteFolder() {
        if (activeProfile == null) {
            showMessage(getString(R.string.action_new_folder), "Connect to a server first.")
            return
        }
        requestInlineConfirmation(getString(R.string.action_new_folder),
            "Enter the new remote folder name.", getString(R.string.action_confirm)) {
            mkdirNameInput.setText(inlineNameInput.text.toString())
            createRemoteFolder()
        }
        inlineNameInput.setText("")
        inlineNameInput.hint = getString(R.string.hint_folder_target)
        inlineNameInput.visibility = View.VISIBLE
        contentScroll.post { contentScroll.scrollTo(0, 0) }
    }

    private fun promptRenameRemoteEntry() {
        if (activeProfile == null) {
            showMessage(getString(R.string.action_rename), "Connect and select a remote file first.")
            return
        }
        requestInlineConfirmation(getString(R.string.action_rename),
            "Enter a new name for the selected remote item.",
            getString(R.string.action_confirm)) {
            renameRemoteNameInput.setText(inlineNameInput.text.toString())
            renameRemoteEntry()
        }
        inlineNameInput.setText(renameRemoteNameInput.text.toString())
        inlineNameInput.hint = getString(R.string.hint_rename_target)
        inlineNameInput.visibility = View.VISIBLE
        contentScroll.post { contentScroll.scrollTo(0, 0) }
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
        // A panel built without saving this reference makes every destructive
        // confirmation a silent no-op (including saved-site removal).
        confirmationPanel = this
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
        inlineNameInput = input("Name", InputType.TYPE_CLASS_TEXT).apply {
            visibility = View.GONE
            contentDescription = "Remote action name"
        }
        addView(inlineNameInput, LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, dp(46)
        ))

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
        if (::inlineNameInput.isInitialized) inlineNameInput.visibility = View.GONE
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
        addView(sectionDescription("Quick Connect uses these credentials immediately. Save new connection profile is a separate action; passwords are never saved."))
        siteFeedback = TextView(this@MainActivity).apply {
            visibility = View.GONE
            textSize = 13f
            setTextColor(PremiumReferenceStyle.secondary)
            setPadding(dp(10), dp(9), dp(10), dp(9))
            background = PremiumReferenceStyle.surface(this@MainActivity,
                PremiumReferenceStyle.raised, 8)
            contentDescription = "Site connection feedback"
        }
        addView(siteFeedback)

        val savedProfiles = SavedSiteProfiles.read(this@MainActivity)
        addView(formLabel("Saved connection profiles"))
        savedSitesSpinner = Spinner(this@MainActivity).apply {
            contentDescription = "Saved connection profiles"
            adapter = ArrayAdapter(this@MainActivity, android.R.layout.simple_spinner_item,
                listOf("Select saved profile") + savedProfiles.map { it.name }
            ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        }
        addView(savedSitesSpinner)
        addView(secondaryButton("Use saved profile") {
            val selected = savedSitesSpinner.selectedItemPosition - 1
            val entry = SavedSiteProfiles.read(this@MainActivity).getOrNull(selected)
            if (entry == null) {
                showMessage("Saved profiles", "Select a saved profile first.")
            } else {
                selectConnectionProtocolAndPort(entry.protocol, entry.port)
                hostInput.setText(entry.host)
                usernameInput.setText(entry.username)
                passwordInput.text.clear()
                hostKeyFingerprintInput.setText(entry.fingerprint)
                remotePathInput.setText(entry.remotePath)
                sftpKeepAliveInput.setText(entry.keepAliveSeconds.toString())
                siteNameInput.setText(entry.name)
                showMessage("Saved profile loaded", "Enter your password and tap Connect. No saved password was read.")
            }
        })
        addView(space(12))
        siteNameInput = input("Name for saved profile", InputType.TYPE_CLASS_TEXT)
        addView(formLabel("Saved profile name (new or edit; optional for Quick Connect)"))
        addView(siteNameInput)

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
        addView(space(8))
        addView(secondaryButton("Save new connection profile") {
            val profile = readProfile() ?: return@secondaryButton
            val name = siteNameInput.text.toString().trim()
            if (name.isEmpty()) {
                showMessage("Site name required", "Enter a name before saving a reusable connection profile.")
            } else if (SavedSiteProfiles.save(this@MainActivity, name, profile)) {
                // Saving is distinct from Connect, and never persists secrets.
                passwordInput.text.clear()
                refreshSavedProfiles(name)
                showMessage("Saved site", "Profile saved without a password. Select it later, enter credentials and tap Connect.")
            } else {
                showMessage("Could not save site", "Check the name, duplicate profiles or 100-profile limit. Existing sites were kept.")
            }
        })
        addView(space(8))
        addView(secondaryButton("Update selected profile") {
            val selected = savedSitesSpinner.selectedItemPosition - 1
            val original = SavedSiteProfiles.read(this@MainActivity).getOrNull(selected)
            if (original == null) {
                showMessage("Saved profiles", "Select the profile to update first.")
            } else {
                val profile = readProfile() ?: return@secondaryButton
                val name = siteNameInput.text.toString().trim()
                if (SavedSiteProfiles.update(this@MainActivity, original.name, name, profile)) {
                    passwordInput.text.clear()
                    refreshSavedProfiles(name)
                    showMessage("Saved profile updated", "Only the selected profile was changed. No password was saved.")
                } else {
                    showMessage("Could not update profile", "Check the name, duplicates or stored data integrity. Other profiles were preserved.")
                }
            }
        })
        addView(space(8))
        addView(secondaryButton("Remove selected profile") {
            val selected = savedSitesSpinner.selectedItemPosition - 1
            val entry = SavedSiteProfiles.read(this@MainActivity).getOrNull(selected)
            if (entry == null) {
                showMessage("Saved profiles", "Select the profile to remove first.")
            } else {
                requestInlineConfirmation(
                    "Remove saved profile?",
                    "Remove ${entry.name} from this device? Other saved sites will be kept.",
                    "Remove profile"
                ) {
                    if (SavedSiteProfiles.remove(this@MainActivity, entry.name)) {
                        passwordInput.text.clear()
                        refreshSavedProfiles()
                        showMessage("Profile removed", "The selected saved profile was removed. Other sites were preserved.")
                    } else {
                        showMessage("Could not remove profile", "Stored data changed or cannot be written. Other sites were kept.")
                    }
                }
            }
        })
        // Reference frame #4: General / Advanced / Sync segmented editor.
        // Move existing native controls into contextual tabs WITHOUT replacing
        // password handling, save/update/connection handlers or validation.
        val original = (0 until childCount).map { getChildAt(it) }
        val generalViews = mutableSetOf<View>(siteFeedback)
        for (field in listOf<View>(
            siteNameInput, protocolSpinner, hostInput, portInput,
            usernameInput, passwordInput
        )) {
            generalViews.add(field)
            val index = original.indexOf(field)
            if (index > 0 && original[index - 1] is TextView) {
                generalViews.add(original[index - 1])
            }
        }
        original.filterTo(generalViews) { view ->
            view is ViewGroup && (0 until view.childCount)
                .any { index -> view.getChildAt(index) === connectButton }
        }
        original.filterTo(generalViews) { view ->
            view is Button && view.text.toString() == "Save new connection profile"
        }
        val general = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        val advanced = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
        }
        removeAllViews()
        val tabs = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            background = PremiumReferenceStyle.surface(this@MainActivity,
                PremiumReferenceStyle.raised, 8)
        }
        fun tab(label: String, selected: Boolean, action: () -> Unit): TextView =
            PremiumReferenceStyle.action(this@MainActivity, label, selected, action).apply {
                textSize = 12f
                contentDescription = "Site editor " + label + " tab"
            }
        lateinit var generalTab: TextView
        lateinit var advancedTab: TextView
        fun activateGeneral() {
            general.visibility = View.VISIBLE
            advanced.visibility = View.GONE
            generalTab.background = PremiumReferenceStyle.actionBackground(this@MainActivity)
            advancedTab.background = PremiumReferenceStyle.surface(this@MainActivity)
        }
        fun activateAdvanced() {
            general.visibility = View.GONE
            advanced.visibility = View.VISIBLE
            advancedTab.background = PremiumReferenceStyle.actionBackground(this@MainActivity)
            generalTab.background = PremiumReferenceStyle.surface(this@MainActivity)
        }
        generalTab = tab("General", true) { activateGeneral() }
        advancedTab = tab("Advanced", false) { activateAdvanced() }
        tabs.addView(generalTab, LinearLayout.LayoutParams(0, dp(39), 1f))
        tabs.addView(advancedTab, LinearLayout.LayoutParams(0, dp(39), 1f))
        tabs.addView(tab("Sync", false) { setWorkspace(Workspace.SYNC) },
            LinearLayout.LayoutParams(0, dp(39), 1f))
        addView(tabs)
        addView(space(12))
        original.forEach { view ->
            (if (view in generalViews) general else advanced).addView(view)
        }
        general.addView(space(8))
        general.addView(secondaryButton(getString(R.string.action_cancel)) {
            showSitesOverview()
        }.apply { contentDescription = "Cancel site editing" })
        addView(general)
        addView(advanced)
    }

    // Real metadata-only Site Manager; no example hosts or imported passwords.
    // The editor reuses the existing secure ConnectionProfile/controller logic.
    private fun buildSitesOverview(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(4), dp(6), dp(4), dp(8))
        val heading = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        heading.addView(sectionTitle(getString(R.string.workspace_sites)),
            LinearLayout.LayoutParams(0, dp(44), 1f))
        heading.addView(PremiumReferenceStyle.action(this@MainActivity, "+", true) {
            showSiteEditor(null)
        }.apply {
            contentDescription = "Add Site"
        }, LinearLayout.LayoutParams(dp(44), dp(42)))
        if (resources.configuration.screenWidthDp >= 600) addView(heading)
        sitesSearch = input("Search saved sites…", InputType.TYPE_CLASS_TEXT).apply {
            contentDescription = "Search sites"
            addTextChangedListener(object : android.text.TextWatcher {
                override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) = Unit
                override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {
                    refreshSiteRows()
                }
                override fun afterTextChanged(s: android.text.Editable?) = Unit
            })
        }
        addView(sitesSearch, LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, dp(44)
        ))
        addView(space(10))
        sitesListRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(sitesListRows)
        // Render only metadata actually present in the private site store.
        refreshSiteRows()
    }

    private fun refreshSiteRows() {
        if (!::sitesListRows.isInitialized || closingOrDestroyed()) return
        sitesListRows.removeAllViews()
        val q = if (::sitesSearch.isInitialized) sitesSearch.text.toString().trim() else ""
        val matches = SavedSiteProfiles.read(this).filter {
            q.isBlank() || it.name.contains(q, ignoreCase = true) ||
                it.host.contains(q, ignoreCase = true)
        }
        if (matches.isEmpty()) {
            sitesListRows.addView(row("No matching saved sites",
                "Tap + to add a real FTP, FTPS or SFTP server. Credentials are not saved."))
            return
        }
        matches.forEach { site ->
            val item = LinearLayout(this).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(dp(10), dp(10), dp(8), dp(10))
                background = PremiumReferenceStyle.surface(this@MainActivity,
                    PremiumReferenceStyle.panel, 10)
                isClickable = true
                isFocusable = true
                contentDescription = "Edit saved site " + site.name
                setOnClickListener { showSiteEditor(site) }
            }
            item.addView(PremiumReferenceStyle.caption(this, "▣", 22f).apply {
                setTextColor(PremiumReferenceStyle.purpleLight)
                gravity = Gravity.CENTER
                background = PremiumReferenceStyle.surface(this@MainActivity,
                    PremiumReferenceStyle.selected, 8)
            }, LinearLayout.LayoutParams(dp(40), dp(40)))
            item.addView(gap(10))
            val details = LinearLayout(this).apply {
                orientation = LinearLayout.VERTICAL
                addView(PremiumReferenceStyle.caption(this@MainActivity, site.name, 15f).apply {
                    typeface = Typeface.DEFAULT_BOLD
                })
                addView(PremiumReferenceStyle.caption(this@MainActivity,
                    site.protocol.label + " · " + site.host, 12f, true))
            }
            item.addView(details, LinearLayout.LayoutParams(0,
                ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
            item.addView(PremiumReferenceStyle.action(this, "⋮") {
                android.widget.PopupMenu(this@MainActivity, item.getChildAt(item.childCount - 1)).apply {
                    menu.add(0, 1, 0, "Edit")
                    menu.add(0, 2, 1, "Connect")
                    menu.add(0, 3, 2, "Remove")
                    setOnMenuItemClickListener { menuItem ->
                        when (menuItem.itemId) {
                            1, 2 -> showSiteEditor(site)
                            3 -> requestInlineConfirmation("Remove saved site?",
                                "Remove " + site.name + " from this device?", "Remove") {
                                if (SavedSiteProfiles.remove(this@MainActivity, site.name)) {
                                    refreshSavedProfiles()
                                    refreshSiteRows()
                                } else showMessage("Site removal failed",
                                    "The saved profile was preserved.")
                            }
                        }
                        true
                    }
                }.show()
            }.apply { contentDescription = "Site options " + site.name },
                LinearLayout.LayoutParams(dp(40), dp(40)))
            sitesListRows.addView(item, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(6) })
        }
    }

    private fun showSiteEditor(site: SavedSiteDefinition?) {
        siteEditorVisible = true
        siteFeedback.visibility = View.GONE
        siteEditorPanel.visibility = View.VISIBLE
        sitesOverviewPanel.visibility = View.GONE
        if (site == null) {
            siteNameInput.text.clear()
            hostInput.text.clear()
            usernameInput.text.clear()
            passwordInput.text.clear()
            savedSitesSpinner.setSelection(0)
        } else {
            val index = SavedSiteProfiles.read(this).indexOfFirst {
                it.name.equals(site.name, ignoreCase = true)
            }
            if (index >= 0) savedSitesSpinner.setSelection(index + 1)
            // Editing must actually load the selected metadata, not just
            // highlight a hidden spinner in the Advanced tab.
            siteNameInput.setText(site.name)
            selectConnectionProtocolAndPort(site.protocol, site.port)
            hostInput.setText(site.host)
            usernameInput.setText(site.username)
            hostKeyFingerprintInput.setText(site.fingerprint)
            sftpKeepAliveInput.setText(site.keepAliveSeconds.toString())
            remotePathInput.setText(site.remotePath)
            passwordInput.text.clear()
        }
        setWorkspace(Workspace.SITES)
        updatePhoneChromeActions()
    }

    private fun showSitesOverview() {
        siteEditorVisible = false
        siteEditorPanel.visibility = View.GONE
        sitesOverviewPanel.visibility = View.VISIBLE
        if (::phoneChromeTitle.isInitialized) phoneChromeTitle.text =
            getString(R.string.workspace_sites)
        refreshSiteRows()
        updatePhoneChromeActions()
        contentScroll.post { contentScroll.scrollTo(0, 0) }
    }

    private fun refreshSavedProfiles(selectedName: String? = null) {
        val profiles = SavedSiteProfiles.read(this)
        savedSitesSpinner.adapter = ArrayAdapter(
            this, android.R.layout.simple_spinner_item,
            listOf("Select saved profile") + profiles.map { it.name }
        ).also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
        val index = profiles.indexOfFirst { it.name.equals(selectedName, ignoreCase = true) }
        savedSitesSpinner.setSelection(index + 1)
        if (::sitesListRows.isInitialized) refreshSiteRows()
    }

    private fun selectConnectionProtocolAndPort(protocol: ConnectionProtocol, port: Int) {
        // Update the logical selection before setSelection: Spinner callbacks may be
        // synchronous, deferred, or omitted when the same item is already selected.
        // The next matching callback must never rewrite this saved custom port.
        selectedProtocol = protocol
        protocolSpinner.setSelection(protocol.ordinal)
        portInput.setText(port.toString())
        updateProtocolSpecificFields(protocol)
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
                // A delayed callback for an earlier selection must not undo a
                // more recent saved-site load or a subsequent manual selection.
                if (position != protocolSpinner.selectedItemPosition) return
                val nextProtocol = ConnectionProtocol.fromIndex(position)
                portOnProtocolChange(
                    previous = selectedProtocol,
                    next = nextProtocol,
                    currentPort = portInput.text.toString()
                )?.let { portInput.setText(it) }
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

    private fun buildFilesCard(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        val twoPane = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.TOP
        }
        fun pane(label: String, isRemote: Boolean): LinearLayout =
            LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(dp(7), dp(8), dp(7), dp(10))
                background = PremiumReferenceStyle.surface(this@MainActivity)
                val header = TextView(this@MainActivity).apply {
                    text = label
                    textSize = 13f
                    typeface = Typeface.DEFAULT_BOLD
                    setTextColor(PremiumReferenceStyle.text)
                    setPadding(dp(4), dp(4), dp(4), dp(9))
                    ellipsize = android.text.TextUtils.TruncateAt.MIDDLE
                    maxLines = 1
                }
                if (isRemote) remoteDirectoryLabel = header else localDirectoryLabel = header
                addView(header)
                val entries = LinearLayout(this@MainActivity).apply {
                    orientation = LinearLayout.VERTICAL
                }
                if (isRemote) remoteRows = entries else localRows = entries
                addView(entries)
            }
        twoPane.addView(pane("Remote /", true),
            LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                .apply { marginEnd = dp(4) })
        twoPane.addView(pane("Local", false),
            LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                .apply { marginStart = dp(4) })
        addView(twoPane)
        refreshLocalPane()
    }

    // Android SAF does not grant arbitrary /storage/emulated/0 traversal.
    // The local pane is a REAL, bounded app-owned downloads directory, never
    // a fake listing of system Download/Pictures/DCIM folders.
    private fun refreshLocalPane() {
        if (!::localRows.isInitialized) return
        val base = (getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS) ?: filesDir)
            .canonicalFile
        val current = (localBrowseDirectory ?: base).canonicalFile
        val inside = current.path == base.path ||
            current.path.startsWith(base.path + File.separator)
        if (!inside || !current.isDirectory) localBrowseDirectory = base
        else localBrowseDirectory = current
        val directory = localBrowseDirectory ?: base
        localDirectoryLabel.text = "Local · " + directory.name
        localRows.removeAllViews()
        if (directory != base) {
            localRows.addView(compactBrowserRow("..", true) {
                localBrowseDirectory = directory.parentFile
                refreshLocalPane()
            })
        }
        val entries = directory.listFiles()
            ?.filter {
                runCatching {
                    val path = it.canonicalFile.path
                    path.startsWith(base.path + File.separator) && !it.isHidden
                }.getOrDefault(false)
            }?.sortedWith(compareBy({ !it.isDirectory }, { it.name.lowercase(Locale.ROOT) }))
            ?.take(150).orEmpty()
        if (entries.isEmpty()) {
            localRows.addView(PremiumReferenceStyle.caption(this, "No local files", 11f, true))
        }
        entries.forEach { file ->
            localRows.addView(compactBrowserRow(file.name, file.isDirectory) {
                if (file.isDirectory) {
                    localBrowseDirectory = file
                    refreshLocalPane()
                } else {
                    android.widget.Toast.makeText(this@MainActivity,
                        file.name + " · use Upload to choose a document", android.widget.Toast.LENGTH_SHORT).show()
                }
            })
        }
    }

    private fun compactBrowserRow(name: String, directory: Boolean, onClick: () -> Unit): View =
        LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            minimumHeight = dp(43)
            addView(ImageView(this@MainActivity).apply {
                setImageResource(if (directory) R.drawable.ic_premium_file_folder
                    else R.drawable.ic_premium_file_document)
                contentDescription = if (directory) "Folder" else "File"
                scaleType = ImageView.ScaleType.CENTER_INSIDE
                setPadding(dp(2), dp(6), dp(3), dp(6))
            }, LinearLayout.LayoutParams(dp(25), dp(36)))
            addView(PremiumReferenceStyle.caption(this@MainActivity, name, 12f).apply {
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
            }, LinearLayout.LayoutParams(0, dp(40), 1f))
            isClickable = true
            isFocusable = true
            contentDescription = (if (directory) "Open folder " else "File ") + name
            setOnClickListener { onClick() }
        }

    /** Reference screen #11: a real, scoped search of actual file entries.
     * The current remote directory is searched from the verified last server
     * response. Local search traverses ONLY the app-owned Downloads sandbox.
     * Neither claims global remote indexing nor reads outside Android SAF.
     */
    private fun buildSearchCard(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(4), dp(4), dp(4), dp(14))
        val scope = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
        }
        val remoteButton = PremiumReferenceStyle.action(this@MainActivity,
            "Remote", true) {
            searchRemote = true
            searchFeedback.text = "Search currently listed remote folder"
        }
        val localButton = PremiumReferenceStyle.action(this@MainActivity,
            "Local") {
            searchRemote = false
            searchFeedback.text = "Search app-owned Downloads"
        }
        scope.addView(remoteButton, LinearLayout.LayoutParams(0, dp(39), 1f))
        scope.addView(localButton, LinearLayout.LayoutParams(0, dp(39), 1f))
        addView(PremiumReferenceStyle.caption(this@MainActivity, "Location", 12f, true))
        addView(scope)
        addView(space(11))
        addView(PremiumReferenceStyle.caption(this@MainActivity, "File name", 12f, true))
        searchQueryInput = input("*.php", InputType.TYPE_CLASS_TEXT).apply {
            contentDescription = "Search file name"
        }
        addView(searchQueryInput, LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, dp(46)))
        addView(space(8))
        addView(PremiumReferenceStyle.caption(this@MainActivity,
            "Supports * and ? filename wildcards", 11f, true))
        val match = android.widget.Switch(this@MainActivity).apply {
            text = "Match case"
            textSize = 13f
            setTextColor(Brand.text)
            isChecked = false
            setOnCheckedChangeListener { _, value -> searchMatchCase = value }
        }
        addView(match)
        val recurse = android.widget.Switch(this@MainActivity).apply {
            text = "Search subdirectories"
            textSize = 13f
            setTextColor(Brand.text)
            setOnCheckedChangeListener { _, value -> searchRecursive = value }
        }
        addView(recurse)
        searchFeedback = PremiumReferenceStyle.caption(this@MainActivity,
            "Search the actual connected folder or app-owned local files.", 12f, true)
        addView(searchFeedback)
        addView(space(11))
        addView(PremiumReferenceStyle.action(this@MainActivity, "Search", true) {
            runScopedSearch()
        }.apply {
            contentDescription = "Run file search"
        }, LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, dp(45)))
        addView(space(11))
        searchResults = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            contentDescription = "Actual search results"
        }
        addView(searchResults)
    }

    private fun openSearch() {
        if (!::searchSection.isInitialized) return
        setWorkspace(Workspace.FILES, announce = false)
        searchMode = true
        filesSection.visibility = View.GONE
        searchSection.visibility = View.VISIBLE
        fileActionToolbar.visibility = View.GONE
        phoneChromeTitle.text = "Search"
        updatePhoneChromeActions()
        contentScroll.post { contentScroll.scrollTo(0, workspaceContainer.top) }
    }

    private fun closeSearch() {
        searchMode = false
        searchSection.visibility = View.GONE
        filesSection.visibility = View.VISIBLE
        setWorkspace(Workspace.FILES, announce = false)
    }

    private fun runScopedSearch() {
        if (!::searchResults.isInitialized) return
        searchResults.removeAllViews()
        val namePattern = searchQueryInput.text.toString().trim()
        if (namePattern.isBlank() || namePattern.length > 80) {
            searchFeedback.text = "Enter a filename (up to 80 characters)."
            return
        }
        val expression = buildString {
            append("^")
            namePattern.forEach { ch ->
                when (ch) {
                    '*' -> append(".*")
                    '?' -> append(".")
                    else -> append(Regex.escape(ch.toString()))
                }
            }
            append("$")
        }
        val pattern = try {
            Regex(expression, if (searchMatchCase) emptySet() else
                setOf(RegexOption.IGNORE_CASE))
        } catch (_: IllegalArgumentException) {
            searchFeedback.text = "Invalid search pattern."
            return
        }
        if (searchRemote) {
            if (activeProfile == null) {
                searchFeedback.text = "Connect to a server before searching."
                return
            }
            if (searchRecursive) {
                searchFeedback.text = "Recursive remote search is not available yet. " +
                    "Disable subdirectories to search the loaded remote folder safely."
                return
            }
            val matches = lastRemoteInventory.filter {
                (it.isFile || it.isDirectory) && pattern.matches(it.name)
            }.take(150)
            matches.forEach { item ->
                searchResults.addView(compactBrowserRow(item.name, item.isDirectory) {
                    selectRemoteEntry(item)
                    closeSearch()
                })
            }
            searchFeedback.text = matches.size.toString() +
                " matches in the current remote listing"
        } else {
            val base = (getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS)
                ?: filesDir).canonicalFile
            val queue = java.util.ArrayDeque<Pair<File, Int>>()
            queue.add(base to 0)
            var scanned = 0
            var matches = 0
            while (queue.isNotEmpty() && scanned < 1000 && matches < 150) {
                val (directory, depth) = queue.removeFirst()
                val entries = directory.listFiles()?.sortedBy { it.name } ?: emptyList()
                for (entry in entries) {
                    scanned += 1
                    if (scanned > 1000) break
                    val canonical = runCatching { entry.canonicalFile }.getOrNull()
                        ?: continue
                    if (!canonical.path.startsWith(base.path + File.separator)) continue
                    if (entry.isDirectory && searchRecursive && depth < 4) {
                        queue.add(entry to depth + 1)
                    }
                    if (pattern.matches(entry.name)) {
                        matches += 1
                        searchResults.addView(compactBrowserRow(entry.name,
                            entry.isDirectory) {
                            if (entry.isDirectory) {
                                localBrowseDirectory = entry
                                refreshLocalPane()
                            }
                            closeSearch()
                        })
                        if (matches >= 150) break
                    }
                }
            }
            searchFeedback.text = matches.toString() +
                " results from app-owned local storage"
        }
    }

    private fun buildTransfersCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_transfers)))
        addView(sectionDescription(getString(R.string.desc_transfers)))

        // Session-only, real remote-operation outcomes. Unlike the desktop
        // engine this activity does not own a persisted transfer queue.
        // Never show imaginary history, speed or progress percentages.
        transferOverviewText = TextView(this@MainActivity).apply {
            setTextColor(Brand.textSoft)
            textSize = 12f
            setPadding(dp(12), dp(12), dp(12), dp(12))
            background = rounded(Brand.panelStrong, dp(12), Brand.border)
            contentDescription = "Session operation status"
        }
        updateSessionOperationOverview()
        addView(transferOverviewText)
        addView(space(12))

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

        // Reference #7 uses Queue / History tabs, NOT a page of raw editable
        // transfer paths. This app currently has a single active operation and
        // session-local real history, not a fabricated multi-file queue.
        val original = (0 until childCount).map { getChildAt(it) }
        removeAllViews()
        val queuePanel = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        val historyPanel = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
        }
        val optionsPanel = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
        }
        for (view in original) {
            when (view) {
                transferOverviewText, transferStateText -> queuePanel.addView(view)
                activityRows -> historyPanel.addView(view)
                else -> optionsPanel.addView(view)
            }
        }
        fun setTab(queue: Boolean) {
            queuePanel.visibility = if (queue) View.VISIBLE else View.GONE
            historyPanel.visibility = if (queue) View.GONE else View.VISIBLE
            if (!queue) optionsPanel.visibility = View.GONE
        }
        addView(LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            background = PremiumReferenceStyle.surface(this@MainActivity,
                PremiumReferenceStyle.raised, 8)
            addView(PremiumReferenceStyle.action(this@MainActivity, "Queue", true) {
                setTab(true)
            }.apply {
                contentDescription = "Transfer Queue tab"
            }, LinearLayout.LayoutParams(0, dp(40), 1f))
            addView(PremiumReferenceStyle.action(this@MainActivity, "History") {
                setTab(false)
            }.apply {
                contentDescription = "Transfer History tab"
            }, LinearLayout.LayoutParams(0, dp(40), 1f))
        })
        addView(space(12))
        addView(queuePanel)
        addView(historyPanel)
        addView(space(8))
        addView(secondaryButton("Transfer options") {
            setTab(true)
            optionsPanel.visibility =
                if (optionsPanel.visibility == View.VISIBLE) View.GONE else View.VISIBLE
        }.apply { contentDescription = "Transfer options" })
        addView(optionsPanel)

        // Reference #8: a real transfer-details route, never invented progress
        // bars, speed, ETA or synthetic queue jobs. This Android engine runs a
        // single cancellable operation and preserves its actual status/path.
        transferOverviewText.isClickable = true
        transferOverviewText.isFocusable = true
        transferOverviewText.contentDescription = "View transfer details"
        transferOverviewText.setOnClickListener { openTransferDetails() }
        val queueContent = (0 until childCount).map { getChildAt(it) }
        removeAllViews()
        transferQueueSurface = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            queueContent.forEach { addView(it) }
        }
        addView(transferQueueSurface)
        transferDetailsSurface = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
            setPadding(dp(3), dp(4), dp(3), dp(8))
            addView(PremiumReferenceStyle.caption(this@MainActivity,
                "Transfer Details", 19f).apply {
                typeface = Typeface.DEFAULT_BOLD
            })
            addView(space(13))
            transferDetailsStatus = PremiumReferenceStyle.caption(this@MainActivity,
                "No active transfer", 13f, true)
            addView(transferDetailsStatus)
            addView(space(10))
            addView(LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                addView(PremiumReferenceStyle.action(this@MainActivity, "Back") {
                    closeTransferDetails()
                }, LinearLayout.LayoutParams(0, dp(44), 1f))
                addView(gap(8))
                addView(PremiumReferenceStyle.action(this@MainActivity, "Cancel") {
                    activeCancellation?.cancel()
                    transferDetailsStatus.text =
                        "Cancellation requested. Waiting for the current I/O checkpoint."
                }, LinearLayout.LayoutParams(0, dp(44), 1f))
                addView(gap(8))
                addView(PremiumReferenceStyle.action(this@MainActivity, "Retry") {
                    closeTransferDetails()
                    optionsPanel.visibility = View.VISIBLE
                }, LinearLayout.LayoutParams(0, dp(44), 1f))
            })
        }
        addView(transferDetailsSurface)
    }

    private fun openTransferDetails() {
        if (!::transferDetailsSurface.isInitialized) return
        val path = transferRemotePathInput.text.toString().trim()
        val actual = when {
            operationInFlight -> "Transfer in progress"
            lastCompletedTransferPath.isNotBlank() -> "Last operation completed"
            else -> "No active transfer"
        }
        transferDetailsStatus.text = actual + "\n" +
            (path.ifBlank { lastCompletedTransferPath }.ifBlank {
                "Select a remote file or start a transfer."
            }) + "\n" + transferStateText.text.toString()
        transferQueueSurface.visibility = View.GONE
        transferDetailsSurface.visibility = View.VISIBLE
        if (::phoneChromeTitle.isInitialized) phoneChromeTitle.text = "Transfer Details"
        contentScroll.post { contentScroll.scrollTo(0, workspaceContainer.top) }
    }

    private fun closeTransferDetails() {
        transferDetailsSurface.visibility = View.GONE
        transferQueueSurface.visibility = View.VISIBLE
        if (::phoneChromeTitle.isInitialized)
            phoneChromeTitle.text = workspaceLabel(Workspace.TRANSFERS)
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

    private fun cloudLinkField(hintText: String): EditText = EditText(this).apply {
        hint = hintText
        contentDescription = hintText
        inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
        isSaveEnabled = false
        isSingleLine = true
        textSize = 14f
        setTextColor(Brand.text)
        setHintTextColor(Brand.muted)
        setPadding(dp(14), dp(12), dp(14), dp(12))
        background = rounded(Brand.input, dp(14), Brand.border)
    }

    private fun buildCloudCard(): View = panel().apply {
        addView(sectionTitle(getString(R.string.workspace_cloud)))
        addView(sectionDescription(
            "Direct, private object transfers to S3 / R2 / B2, Google Cloud Storage " +
                "and Azure SAS using short-lived provider-signed HTTPS links. " +
                "Links never enter settings, backups or activity history."
        ))
        addView(space(12))
        addView(formLabel("Signed upload link (HTTPS PUT)"))
        cloudUploadInput = cloudLinkField("Paste short-lived signed PUT URL")
        addView(cloudUploadInput)
        addView(space(8))
        cloudUploadButton = primaryButton("Choose file and upload") { chooseCloudFile(upload = true) }.apply {
            contentDescription = "Choose file and upload"
        }
        addView(cloudUploadButton)
        addView(space(18))
        addView(formLabel("Signed download link (HTTPS GET)"))
        cloudDownloadInput = cloudLinkField("Paste short-lived signed GET URL")
        addView(cloudDownloadInput)
        addView(space(8))
        cloudDownloadButton = secondaryButton("Choose destination and download") { chooseCloudFile(upload = false) }.apply {
            contentDescription = "Choose destination and download"
        }
        addView(cloudDownloadButton)
        addView(space(14))
        cloudStatus = sectionDescription(getString(R.string.cloud_no_transfer))
        addView(cloudStatus)
        addView(space(8))
        cloudCancelButton = secondaryButton(getString(R.string.action_cancel_cloud)) { cancelCloudTransfer() }.apply {
            contentDescription = getString(R.string.action_cancel_cloud)
            isEnabled = false
        }
        addView(cloudCancelButton)
        addView(space(12))
        cloudRecoveryNotice = sectionDescription(
            "A previous cloud download may have left an incomplete document. Review it before continuing."
        ).apply { visibility = View.GONE }
        addView(cloudRecoveryNotice)
        cloudRecoveryCleanupButton = secondaryButton("Try removing incomplete document") {
            cleanupPendingCloudRecovery()
        }.apply {
            contentDescription = "Try removing incomplete document"
            visibility = View.GONE
        }
        addView(cloudRecoveryCleanupButton)
        cloudRecoveryKeepButton = secondaryButton("Keep document and dismiss warning") {
            if (CloudDestinationJournal.clear(noBackupFilesDir)) {
                cloudStatus.text = "Recovery reminder dismissed. The document was not changed."
                displayPendingCloudRecovery()
            } else {
                cloudStatus.text = "Could not clear recovery reminder. Check app storage."
            }
        }.apply {
            contentDescription = "Keep document and dismiss warning"
            visibility = View.GONE
        }
        addView(cloudRecoveryKeepButton)
        addView(space(8))
        addView(sectionDescription(
            "Generate an object-specific signed URL in your cloud provider. " +
                "Google Drive account browsing requires a separate OAuth flow; " +
                "a Drive sharing-page link cannot be used here."
        ))

        // Reference #9 provider choices are clearly DISCONNECTED: an OAuth
        // provider login is not implemented on Android. Do not invent accounts,
        // auth sessions or API capabilities. A tap opens the working signed
        // HTTPS object-transfer path and explains its actual limitation.
        val savedContent = (0 until childCount).map { getChildAt(it) }
        removeAllViews()
        val providers = listOf(
            "Google Drive" to "G",
            "Dropbox" to "D",
            "OneDrive" to "O",
            "iCloud" to "i"
        )
        val advanced = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
        }
        cloudTransferOptionsPanel = advanced
        val feedback = PremiumReferenceStyle.caption(this@MainActivity, "", 12f, true).apply {
            visibility = View.GONE
        }
        providers.forEach { (provider, monogram) ->
            addView(LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(dp(11), dp(6), dp(10), dp(6))
                minimumHeight = dp(57)
                background = PremiumReferenceStyle.surface(this@MainActivity,
                    PremiumReferenceStyle.raised, 9)
                addView(PremiumReferenceStyle.caption(this@MainActivity, monogram, 22f).apply {
                    gravity = Gravity.CENTER
                    setTextColor(PremiumReferenceStyle.purpleLight)
                    typeface = Typeface.DEFAULT_BOLD
                }, LinearLayout.LayoutParams(dp(36), dp(37)))
                addView(LinearLayout(this@MainActivity).apply {
                    orientation = LinearLayout.VERTICAL
                    addView(PremiumReferenceStyle.caption(this@MainActivity, provider, 13f))
                    addView(PremiumReferenceStyle.caption(this@MainActivity,
                        "Not connected", 11f, true))
                }, LinearLayout.LayoutParams(0,
                    ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
                addView(PremiumReferenceStyle.caption(this@MainActivity, "›", 20f, true))
                contentDescription = provider + " — not connected"
                isClickable = true
                isFocusable = true
                setOnClickListener {
                    feedback.text = provider +
                        " account login is not yet available. For supported cloud storage, " +
                        "use a provider-signed HTTPS object link below."
                    feedback.visibility = View.VISIBLE
                    advanced.visibility = View.VISIBLE
                }
            }, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(6) })
        }
        addView(feedback)
        addView(space(7))
        addView(secondaryButton("Cloud transfer options") {
            advanced.visibility = if (advanced.visibility == View.VISIBLE)
                View.GONE else View.VISIBLE
        }.apply { contentDescription = "Cloud transfer options" })
        savedContent.forEach { advanced.addView(it) }
        addView(advanced)
    }

    private fun updateCloudPickerButtons() {
        if (!::cloudUploadButton.isInitialized || !::cloudDownloadButton.isInitialized) return
        val canChoose = !cloudTransferBusy && cloudPendingLink == null &&
            !CloudDestinationJournal.hasUnresolved(noBackupFilesDir)
        cloudUploadButton.isEnabled = canChoose
        cloudDownloadButton.isEnabled = canChoose
    }

    private fun chooseCloudFile(upload: Boolean) {
        if (CloudDestinationJournal.hasUnresolved(noBackupFilesDir)) {
            cloudStatus.text = "Resolve the previous incomplete download reminder before starting another transfer."
            displayPendingCloudRecovery()
            return
        }
        if (cloudTransferBusy || cloudPendingLink != null) {
            cloudStatus.text = "Finish or cancel the current cloud operation before starting another."
            return
        }
        val input = if (upload) cloudUploadInput else cloudDownloadInput
        val link = try {
            SignedCloudLink(input.text.toString().trim())
        } catch (_: IllegalArgumentException) {
            cloudStatus.text = "Invalid link. Use a signed public HTTPS object URL, not a sharing page."
            input.text.clear()
            return
        }
        // The validated signed URL is retained only for the pending picker
        // operation; remove its plaintext from the view before launching
        // another Activity, including when the user cancels the picker.
        input.text.clear()
        cloudPendingLink = link
        updateCloudPickerButtons()
        val intent = if (upload) {
            Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                addCategory(Intent.CATEGORY_OPENABLE)
                type = "*/*"
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
        } else {
            Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
                addCategory(Intent.CATEGORY_OPENABLE)
                type = "application/octet-stream"
                putExtra(Intent.EXTRA_TITLE, "cloud-download.bin")
                // Post-write integrity check must read back the final SAF
                // document, not merely trust the provider's output stream.
                addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION or Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
        }
        try {
            startActivityForResult(
                intent, if (upload) PICK_CLOUD_UPLOAD_REQUEST else CREATE_CLOUD_DOWNLOAD_REQUEST
            )
        } catch (_: Exception) {
            cloudPendingLink = null
            updateCloudPickerButtons()
            cloudStatus.text = "No document picker is available."
        }
    }

    private fun displayPendingCloudRecovery() {
        if (!::cloudRecoveryNotice.isInitialized) return
        val unresolved = CloudDestinationJournal.hasUnresolved(noBackupFilesDir)
        // Show real incomplete-download safety warnings even when signed-URL
        // transfer controls are normally collapsed under provider cards.
        if (unresolved && ::cloudTransferOptionsPanel.isInitialized) {
            cloudTransferOptionsPanel.visibility = View.VISIBLE
        }
        val visibility = if (unresolved) View.VISIBLE else View.GONE
        cloudRecoveryNotice.text = if (unresolved &&
            CloudDestinationJournal.pending(noBackupFilesDir) == null) {
            "Cloud recovery details were damaged. The previous document may still be incomplete. " +
                "Review your cloud provider's files manually before dismissing this warning."
        } else {
            "A previous cloud download may have left an incomplete document. Review it before continuing."
        }
        cloudRecoveryCleanupButton.isEnabled = unresolved &&
            CloudDestinationJournal.pending(noBackupFilesDir) != null
        cloudRecoveryNotice.visibility = visibility
        cloudRecoveryCleanupButton.visibility = visibility
        cloudRecoveryKeepButton.visibility = visibility
        updateCloudPickerButtons()
    }

    private fun cleanupPendingCloudRecovery() {
        val recorded = CloudDestinationJournal.pending(noBackupFilesDir)
        if (recorded == null) {
            cloudStatus.text = "Cannot identify the previous document from the recovery record. " +
                "Check your provider and dismiss this warning only after reviewing its files."
            return
        }
        // DocumentProvider.deleteDocument may block on Binder: never perform
        // remote provider I/O on the Android UI thread.
        cloudRecoveryCleanupButton.isEnabled = false
        cloudRecoveryKeepButton.isEnabled = false
        thread(start = true, isDaemon = true, name = "GhostFTP-CloudRecovery") {
            val removed = removeIncompleteCloudDestination(Uri.parse(recorded))
            val cleared = removed && CloudDestinationJournal.clear(noBackupFilesDir)
            runOnUiThread {
                if (activityClosing || !::cloudStatus.isInitialized) return@runOnUiThread
                cloudRecoveryCleanupButton.isEnabled = true
                cloudRecoveryKeepButton.isEnabled = true
                cloudStatus.text = if (cleared) {
                    "Recovered: the previous incomplete document was removed."
                } else {
                    "Cleanup unavailable: the provider may have revoked access. " +
                        "Remove the previous document manually, then choose Keep to dismiss this warning."
                }
                displayPendingCloudRecovery()
            }
        }
    }

    private fun cancelCloudTransfer() {
        if (!cloudTransferBusy) return
        cloudCancellation?.cancel()
        cloudTransferThread?.interrupt()
        cloudCancelButton.isEnabled = false
        cloudStatus.text = "Canceling cloud transfer…"
    }

    /**
     * A canceled/recreated picker can return a newly created SAF document even
     * when the signed capability was intentionally not retained. Commit a
     * recovery hint synchronously before scheduling provider I/O, then attempt
     * deletion on a daemon worker so a blocked Binder provider cannot freeze
     * the Activity. Never clear a different download's durable recovery hint.
     */
    private fun cleanupUnstartedCloudDestination(target: Uri) {
        val directory = noBackupFilesDir
        val destination = target.toString()
        val recorded = CloudDestinationJournal.pending(directory) == destination ||
            runCatching { CloudDestinationJournal.record(directory, destination) }.isSuccess
        cloudStatus.text = "Removing unused cloud destination…"
        displayPendingCloudRecovery()
        thread(start = true, isDaemon = true, name = "GhostFTP-CloudOrphanCleanup") {
            val removed = removeIncompleteCloudDestination(target)
            val cleared = removed && recorded &&
                CloudDestinationJournal.clearIfMatches(directory, destination)
            runOnUiThread {
                if (activityClosing || !::cloudStatus.isInitialized) return@runOnUiThread
                displayPendingCloudRecovery()
                cloudStatus.text = when {
                    cleared -> "Cloud download was not started. The unused destination was removed."
                    removed -> "Unused cloud document was removed. Check any earlier recovery reminder."
                    recorded -> "Cloud destination could not be removed. A recovery reminder was saved."
                    else -> "Cloud destination could not be removed or journaled. Remove the empty document manually."
                }
            }
        }
    }

    /** ACTION_CREATE_DOCUMENT always creates a new document; clean up on failure. */
    private fun removeIncompleteCloudDestination(target: Uri): Boolean = runCatching {
        DocumentsContract.isDocumentUri(this, target) &&
            DocumentsContract.deleteDocument(contentResolver, target)
    }.getOrDefault(false)

    private fun runCloudTransfer(upload: Boolean, link: SignedCloudLink, uri: Uri) {
        if (cloudTransferBusy || activityClosing) return
        val cancellation = CloudTransferCancellation()
        cloudCancellation = cancellation
        cloudTransferBusy = true
        updateCloudPickerButtons()
        cloudCancelButton.isEnabled = true
        cloudUploadInput.text.clear()
        cloudDownloadInput.text.clear()
        cloudStatus.text = if (upload) "Uploading cloud object…" else "Downloading cloud object…"
        cloudTransferThread = thread(start = true, name = "GhostFTP-CloudTransfer") {
            val success = try {
                if (upload) {
                    SignedCloudTransfer.upload(contentResolver, uri, link, cancellation)
                } else {
                    SignedCloudTransfer.download(contentResolver, uri, link, cacheDir, cancellation)
                }
                cancellation.ensureActive()
                true
            } catch (_: Exception) {
                // Do not display exception text: a URL is a bearer credential.
                false
            }
            val wasCanceled = cancellation.isCancelled || Thread.currentThread().isInterrupted
            // A newly created SAF document may be partially written. Never
            // leave that possibility unreported to the user after a failure.
            val leftoverDestination = !success && !upload && !removeIncompleteCloudDestination(uri)
            // A successful verified document or a confirmed successful delete
            // no longer needs a recovery hint. An uncertain output keeps it.
            if (!upload && (success || !leftoverDestination)) {
                CloudDestinationJournal.clearIfMatches(noBackupFilesDir, uri.toString())
            }
            runOnUiThread {
                if (activityClosing || cloudCancellation !== cancellation) return@runOnUiThread
                cloudTransferBusy = false
                updateCloudPickerButtons()
                cloudCancellation = null
                cloudTransferThread = null
                cloudCancelButton.isEnabled = false
                displayPendingCloudRecovery()
                cloudStatus.text = when {
                    success && upload -> "Cloud upload completed successfully."
                    success -> "Cloud download completed successfully."
                    leftoverDestination -> "Download failed or was canceled. The incomplete destination may remain; remove it manually."
                    !upload -> "Cloud download failed or was canceled. The created destination was removed."
                    wasCanceled -> "Cloud upload canceled. Check the remote object before retrying."
                    else -> "Cloud upload failed. Check link expiry, provider permissions and remote object state before retrying."
                }
            }
        }
    }

    /** Screen 10 reference: native, compact saved remote-directory tiles. */
    private fun buildBookmarksCard(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(4), dp(4), dp(4), dp(12))
        val titleRow = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        titleRow.addView(sectionTitle(getString(R.string.workspace_bookmarks)),
            LinearLayout.LayoutParams(0, dp(48), 1f))
        titleRow.addView(PremiumReferenceStyle.action(this@MainActivity, "+", primary = true) {
            promptSaveCurrentBookmark()
        }.apply { contentDescription = getString(R.string.bookmarks_add) },
            LinearLayout.LayoutParams(dp(44), dp(42)))
        if (resources.configuration.screenWidthDp >= 600) addView(titleRow)
        bookmarksRows = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
        }
        addView(bookmarksRows)
        refreshBookmarks()
    }

    private fun refreshBookmarks() {
        if (!::bookmarksRows.isInitialized || closingOrDestroyed()) return
        bookmarksRows.removeAllViews()
        val saved = SavedRemoteBookmarks.read(this)
        if (saved.isEmpty()) {
            bookmarksRows.addView(row(
                getString(R.string.bookmarks_empty_title),
                getString(R.string.bookmarks_empty_detail)
            ))
            return
        }
        saved.forEach { bookmark ->
            val tile = LinearLayout(this).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(dp(10), dp(8), dp(8), dp(8))
                background = PremiumReferenceStyle.surface(this@MainActivity)
                minimumHeight = dp(62)
                isClickable = true
                isFocusable = true
                contentDescription = "Open bookmark " + bookmark.title
                setOnClickListener { openRemoteBookmark(bookmark) }
            }
            val folder = PremiumReferenceStyle.caption(this, "▣", 21f).apply {
                gravity = Gravity.CENTER
                setTextColor(PremiumReferenceStyle.yellow)
            }
            tile.addView(folder, LinearLayout.LayoutParams(dp(36), dp(42)))
            val description = LinearLayout(this).apply {
                orientation = LinearLayout.VERTICAL
                addView(PremiumReferenceStyle.caption(this@MainActivity, bookmark.title, 14f).apply {
                    typeface = Typeface.DEFAULT_BOLD
                })
                addView(PremiumReferenceStyle.caption(this@MainActivity,
                    bookmark.siteName + " · " + bookmark.remoteDirectory, 11f, muted = true).apply {
                    maxLines = 1
                    ellipsize = android.text.TextUtils.TruncateAt.MIDDLE
                })
            }
            tile.addView(description, LinearLayout.LayoutParams(0, dp(45), 1f))
            val menuButton = PremiumReferenceStyle.action(this, "⋮") {}.apply {
                contentDescription = "Bookmark options " + bookmark.title
            }
            menuButton.setOnClickListener {
                android.widget.PopupMenu(this@MainActivity, menuButton).apply {
                    menu.add(0, 1, 0, getString(R.string.action_delete))
                    setOnMenuItemClickListener {
                        requestInlineConfirmation(
                            getString(R.string.bookmarks_remove_title),
                            bookmark.title + " · " + bookmark.remoteDirectory,
                            getString(R.string.action_delete)
                        ) {
                            if (SavedRemoteBookmarks.remove(this@MainActivity, bookmark)) {
                                refreshBookmarks()
                            } else {
                                showMessage("Bookmark was not removed", "Saved bookmarks were preserved.")
                            }
                        }
                        true
                    }
                }.show()
            }
            tile.addView(menuButton, LinearLayout.LayoutParams(dp(42), dp(42)))
            bookmarksRows.addView(tile, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(6) })
        }
    }

    private fun promptSaveCurrentBookmark() {
        val active = activeProfile
        if (active == null) {
            showMessage(getString(R.string.workspace_bookmarks),
                getString(R.string.bookmarks_connect_first))
            return
        }
        val site = SavedSiteProfiles.read(this).firstOrNull {
            it.host.equals(active.host, ignoreCase = true) &&
                it.port == active.port && it.protocol == active.protocol &&
                it.username == active.username
        }
        if (site == null) {
            showMessage(getString(R.string.workspace_bookmarks),
                getString(R.string.bookmarks_save_site_first))
            return
        }
        val path = remotePathInput.text.toString().trim().ifEmpty { active.remotePath }
        val title = if (path == "/") site.name else path.trimEnd('/').substringAfterLast('/')
        // Use the existing branded, accessible inline confirmation rather
        // than a platform popup. The Android premium contract requires all
        // real actions and confirmations to stay inside the app viewport.
        requestInlineConfirmation(
            getString(R.string.bookmarks_add),
            site.name + " · " + path,
            getString(R.string.action_confirm)
        ) {
            val candidate = SavedRemoteBookmark(
                inlineNameInput.text.toString().trim(), site.name, path
            )
            inlineNameInput.text.clear()
            if (SavedRemoteBookmarks.add(this, candidate)) {
                refreshBookmarks()
            } else {
                showMessage(getString(R.string.workspace_bookmarks),
                    getString(R.string.bookmarks_invalid))
            }
        }
        inlineNameInput.contentDescription = getString(R.string.bookmarks_add)
        inlineNameInput.setText(title.take(80))
        inlineNameInput.visibility = View.VISIBLE
        inlineNameInput.selectAll()
    }

    private fun openRemoteBookmark(bookmark: SavedRemoteBookmark) {
        // A bookmark cannot restore credentials or bypass SSH host-key trust.
        val site = SavedSiteProfiles.read(this).firstOrNull {
            it.name.equals(bookmark.siteName, ignoreCase = true)
        }
        if (site == null) {
            showMessage(getString(R.string.workspace_bookmarks),
                getString(R.string.bookmarks_site_missing))
            return
        }
        val active = activeProfile
        if (active != null && active.protocol == site.protocol &&
            active.host.equals(site.host, ignoreCase = true) &&
            active.port == site.port && active.username == site.username) {
            remotePathInput.setText(bookmark.remoteDirectory)
            setWorkspace(Workspace.FILES)
            refreshActive()
        } else {
            showSiteEditor(site)
            remotePathInput.setText(bookmark.remoteDirectory)
            showMessage(getString(R.string.workspace_bookmarks),
                getString(R.string.bookmarks_reconnect))
        }
    }

    /** Two-column clickable dashboard from reference screen #15. */
    private fun settingsCategory(
        title: String, iconResource: Int, onClick: () -> Unit
    ): View = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        setPadding(dp(8), 0, dp(8), 0)
        minimumHeight = dp(45)
        background = PremiumReferenceStyle.surface(this@MainActivity)
        addView(ImageView(this@MainActivity).apply {
            setImageResource(iconResource)
            imageTintList = android.content.res.ColorStateList.valueOf(Brand.textSoft)
            importantForAccessibility = View.IMPORTANT_FOR_ACCESSIBILITY_NO
        }, LinearLayout.LayoutParams(dp(19), dp(19)))
        addView(PremiumReferenceStyle.caption(this@MainActivity, title, 12f).apply {
            typeface = Typeface.DEFAULT_BOLD
            setPadding(dp(6), 0, 0, 0)
            maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
        }, LinearLayout.LayoutParams(0, dp(36), 1f))
        addView(PremiumReferenceStyle.caption(this@MainActivity, "›", 21f, true),
            LinearLayout.LayoutParams(dp(11), dp(34)))
        contentDescription = "Settings: " + title
        isClickable = true
        isFocusable = true
        setOnClickListener { onClick() }
    }

    private fun buildSettingsCard(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        val advanced = panel().apply {
            visibility = View.GONE
            // Retain translated context in the real Advanced settings view,
            // without adding an extra paragraph to reference frame #15.
            addView(sectionDescription(getString(R.string.desc_settings)))
            addView(sectionDescription(getString(R.string.settings_no_tracking)))
            addView(row(getString(R.string.label_credentials), getString(R.string.settings_credentials)))
            addView(row(getString(R.string.label_security), getString(R.string.settings_security)))
            fun pair(first: Button, second: Button) {
                addView(LinearLayout(this@MainActivity).apply {
                    orientation = LinearLayout.HORIZONTAL
                    setPadding(0, dp(5), 0, dp(5))
                    addView(first, buttonParams(1f))
                    addView(gap(8))
                    addView(second, buttonParams(1f))
                })
            }
            pair(secondaryButton(getString(R.string.action_clear_activity)) {
                clearActivityLog()
            }.apply { contentDescription = getString(R.string.action_clear_activity) },
                trackBusySensitiveLocalAction(
                    secondaryButton(getString(R.string.action_reset_transfers)) {
                        resetTransferFields()
                    }.apply { contentDescription = getString(R.string.action_reset_transfers) }
                ))
            pair(trackBusySensitiveLocalAction(
                    secondaryButton(getString(R.string.action_reset_connection)) {
                        resetConnectionForm()
                    }.apply { contentDescription = getString(R.string.action_reset_connection) }
                ), trackSessionDisconnect(
                    secondaryButton(getString(R.string.action_disconnect)) {
                        disconnect()
                    }.apply {
                        contentDescription = "${getString(R.string.workspace_settings)} ${getString(R.string.action_disconnect)}"
                    }
                ))
            addView(space(12))
            addView(officialLinkRow(getString(R.string.label_support),
                getString(R.string.about_support_desc), SUPPORT_URL))
            addView(officialLinkRow(getString(R.string.label_contact),
                getString(R.string.about_contact_desc), CONTACT_URL))
            addView(officialLinkRow(getString(R.string.label_privacy_policy),
                getString(R.string.about_privacy_desc), PRIVACY_URL))
            addView(officialLinkRow(getString(R.string.label_eula),
                getString(R.string.about_eula_desc), EULA_URL))
            addView(aboutFactRow(getString(R.string.label_author), "Brendigo"))
        }
        val details = PremiumReferenceStyle.panelColumn(this@MainActivity).apply {
            visibility = View.GONE
        }
        fun showDetails(name: String, message: String) {
            advanced.visibility = View.GONE
            details.removeAllViews()
            details.addView(PremiumReferenceStyle.caption(this@MainActivity, name, 15f).apply {
                typeface = Typeface.DEFAULT_BOLD
            })
            details.addView(space(6))
            details.addView(PremiumReferenceStyle.caption(this@MainActivity, message, 12f, true))
            details.visibility = View.VISIBLE
        }
        val grid = listOf(
            listOf(Triple("Appearance", R.drawable.ic_premium_nav_settings, 1),
                Triple("Security", R.drawable.ic_premium_nav_settings, 2)),
            listOf(Triple("Connections", R.drawable.ic_premium_nav_sites, 3),
                Triple("Notifications", R.drawable.ic_premium_nav_about, 4)),
            listOf(Triple("Transfers", R.drawable.ic_premium_nav_transfers, 5),
                Triple("Advanced", R.drawable.ic_premium_nav_settings, 6)),
            listOf(Triple("Cloud Storage", R.drawable.ic_premium_nav_cloud, 7),
                Triple("About", R.drawable.ic_premium_nav_about, 8))
        )
        grid.forEach { line ->
            addView(LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                line.forEachIndexed { index, item ->
                    if (index != 0) addView(gap(8))
                    addView(settingsCategory(item.first, item.second) {
                        when (item.third) {
                            1 -> showDetails(item.first, "Dark premium appearance is enabled.")
                            2 -> showDetails(item.first, getString(R.string.settings_security) +
                                "\n" + getString(R.string.settings_no_tracking))
                            3 -> { showSitesOverview(); setWorkspace(Workspace.SITES) }
                            4 -> {
                                val intent = Intent(
                                    android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                                    Uri.parse("package:" + packageName)
                                )
                                runCatching { startActivity(intent) }.onFailure {
                                    showDetails(item.first, "Android controls notification permissions.")
                                }
                            }
                            5 -> setWorkspace(Workspace.TRANSFERS)
                            6 -> {
                                details.visibility = View.GONE
                                advanced.visibility = if (advanced.visibility == View.VISIBLE)
                                    View.GONE else View.VISIBLE
                            }
                            7 -> setWorkspace(Workspace.CLOUD)
                            8 -> setWorkspace(Workspace.ABOUT)
                        }
                    }, LinearLayout.LayoutParams(0, dp(47), 1f))
                }
            }, LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(7) })
        }
        addView(details)
        settingsFeedback = TextView(this@MainActivity).apply {
            visibility = View.GONE
            contentDescription = "Settings action feedback"
            setTextColor(Brand.textSoft)
            textSize = 12f
            setPadding(dp(12), dp(10), dp(12), dp(10))
            background = PremiumReferenceStyle.surface(this@MainActivity)
        }
        addView(settingsFeedback)
        addView(advanced)
    }

    // Matches the approved mobile About hierarchy with real product data.
    // The enclosing workspace ScrollView keeps resources reachable on small screens.
    /** The compact center-aligned About page from approved Android frame 16. */
    private fun buildAboutCard(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(2), dp(8), dp(2), dp(6))
        addView(LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            addView(GhostMarkView(this@MainActivity),
                LinearLayout.LayoutParams(dp(52), dp(52)))
            addView(PremiumReferenceStyle.caption(this@MainActivity, "Ghost FTP", 15f).apply {
                typeface = Typeface.DEFAULT_BOLD
                gravity = Gravity.CENTER
            })
            addView(PremiumReferenceStyle.caption(this@MainActivity,
                "Version " + ReleaseInfo.VERSION_DISPLAY, 11f, true).apply {
                gravity = Gravity.CENTER
            })
            // Real product attribution; no placeholder version or author.
            addView(PremiumReferenceStyle.caption(this@MainActivity,
                "Ghost FTP · Brendigo", 10f, true).apply {
                gravity = Gravity.CENTER
            })
        })
        addView(space(19))
        val additionalLinks = LinearLayout(this@MainActivity).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
            addView(officialLinkRow(getString(R.string.label_documentation),
                getString(R.string.about_docs_desc), DOCUMENTATION_URL))
            addView(officialLinkRow(getString(R.string.label_support),
                getString(R.string.about_support_desc), SUPPORT_URL))
            addView(officialLinkRow(getString(R.string.label_contact),
                getString(R.string.about_contact_desc), CONTACT_URL))
            addView(officialLinkRow(getString(R.string.label_privacy_policy),
                getString(R.string.about_privacy_desc), PRIVACY_URL))
            addView(officialLinkRow(getString(R.string.label_eula),
                getString(R.string.about_eula_desc), EULA_URL))
            addView(officialLinkRow(getString(R.string.label_project_repository),
                getString(R.string.about_repository_desc), REPOSITORY_URL))
        }
        addView(settingsCategory("Licenses", R.drawable.ic_premium_nav_files) {
            additionalLinks.visibility =
                if (additionalLinks.visibility == View.VISIBLE) View.GONE else View.VISIBLE
        })
        addView(space(6))
        addView(settingsCategory("Terms of Service", R.drawable.ic_premium_nav_files) {
            openOfficialLink(EULA_URL)
        })
        addView(space(6))
        addView(settingsCategory("Check for Updates", R.drawable.ic_premium_nav_settings) {
            openOfficialLink(RELEASES_URL)
        })
        addView(additionalLinks)
    }

    // Reference About uses concise label/value rows, not editable settings controls.
    private fun aboutFactRow(title: String, detail: String): View =
        LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.TOP
            setPadding(dp(2), dp(12), dp(2), dp(12))
            contentDescription = "$title: $detail"
            addView(TextView(this@MainActivity).apply {
                text = title
                textSize = 12f
                typeface = Typeface.DEFAULT_BOLD
                setTextColor(Brand.text)
            }, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 0.38f))
            addView(TextView(this@MainActivity).apply {
                text = detail
                textSize = 12f
                setTextColor(Brand.textSoft)
                gravity = Gravity.END
                setLineSpacing(0f, 1.12f)
            }, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 0.62f))
        }

    private fun setWorkspace(workspace: Workspace, announce: Boolean = true) {
        activeWorkspace = workspace
        searchMode = false
        if (::searchSection.isInitialized) searchSection.visibility = View.GONE
        if (::workspaceFeedbackPanel.isInitialized) {
            workspaceFeedbackPanel.visibility = View.GONE
        }
        val targets = mapOf(
            Workspace.FILES to filesSection,
            Workspace.SITES to sitesSection,
            Workspace.TRANSFERS to transfersSection,
            Workspace.SYNC to syncSection,
            Workspace.CLOUD to cloudSection,
            Workspace.BOOKMARKS to bookmarksSection,
            Workspace.SETTINGS to settingsSection,
            Workspace.ABOUT to aboutSection
        )
        targets.forEach { (key, view) ->
            view.visibility = if (key == workspace) View.VISIBLE else View.GONE
        }
        workspaceNavButtons.forEach { (key, view) ->
            styleWorkspaceNavItem(view, selected = key == workspace)
        }
        if (workspace != Workspace.SITES && siteEditorVisible) {
            siteEditorVisible = false
            siteEditorPanel.visibility = View.GONE
            sitesOverviewPanel.visibility = View.VISIBLE
        }
        if (::workspaceTitle.isInitialized) workspaceTitle.text = workspaceLabel(workspace)
        if (::phoneChromeTitle.isInitialized) {
            phoneChromeTitle.text = if (workspace == Workspace.SITES && siteEditorVisible)
                "Add / Edit Site" else workspaceLabel(workspace)
        }
        updatePhoneChromeActions()
        // Do not show stale success/status messages when switching workspaces.
        if (::settingsFeedback.isInitialized) settingsFeedback.visibility = View.GONE
        if (::siteFeedback.isInitialized && workspace != Workspace.SITES) {
            siteFeedback.visibility = View.GONE
        }
        if (::fileActionToolbar.isInitialized) {
            fileActionToolbar.visibility = if (workspace == Workspace.FILES) View.VISIBLE else View.GONE
        }
        if (::sessionStatusCard.isInitialized) {
            sessionStatusCard.visibility =
                if (activeProfile != null && (workspace == Workspace.FILES || workspace == Workspace.SITES))
                    View.VISIBLE else View.GONE
        }
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
        if (workspace == Workspace.BOOKMARKS && ::bookmarksRows.isInitialized) {
            refreshBookmarks()
        }
        if (workspace == Workspace.SITES && ::sitesListRows.isInitialized && !siteEditorVisible) {
            refreshSiteRows()
        }
        if (announce && usesOverlayNavigation()) {
            setNavigationOpen(false, announce = false)
        }
    }

    @Suppress("DEPRECATION")
    override fun onBackPressed() {
        when {
            usesOverlayNavigation() && navigationOpen -> setNavigationOpen(false)
            searchMode -> closeSearch()
            activeWorkspace == Workspace.TRANSFERS &&
                ::transferDetailsSurface.isInitialized &&
                transferDetailsSurface.visibility == View.VISIBLE -> closeTransferDetails()
            activeWorkspace == Workspace.SITES && siteEditorVisible -> showSitesOverview()
            activeWorkspace != Workspace.FILES -> setWorkspace(Workspace.FILES)
            else -> super.onBackPressed()
        }
    }

    private fun usesOverlayNavigation(): Boolean =
        resources.configuration.screenWidthDp < 600

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
        if (::phoneMenuButton.isInitialized) {
            phoneMenuButton.text = if (navigationOpen) "←" else "☰"
            phoneMenuButton.contentDescription =
                if (navigationOpen) getString(R.string.nav_close) else getString(R.string.nav_open)
        }
        if (::navigationToggleButton.isInitialized) {
            navigationToggleButton.text = if (navigationOpen) "←" else "☰"
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
        if (::protocolSpinner.isInitialized && ::portInput.isInitialized) {
            selectConnectionProtocolAndPort(ConnectionProtocol.FTP, ConnectionProtocol.FTP.defaultPort)
        }
        if (::hostInput.isInitialized) hostInput.text.clear()
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
        lastRemoteInventory = result.rows.filter { it.remotePath != null }
        if (::remoteDirectoryLabel.isInitialized) {
            remoteDirectoryLabel.text = profile.remotePath
        }
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
        val safeDetail = redactSensitiveErrorText(rawDetail, listOf(profile.password))
        if (activeWorkspace == Workspace.SITES && siteEditorVisible) {
            siteFeedback.text = safeDetail
            siteFeedback.visibility = View.VISIBLE
            statusDetail.text = ""
        } else statusDetail.text = safeDetail
        remoteRows.removeAllViews()
        remoteRows.addView(row(getString(R.string.workspace_files), getString(R.string.state_no_server_session)))
        activityRows.removeAllViews()
        activityRows.addView(row(getString(R.string.workspace_transfers), getString(R.string.state_no_active_connection)))
        transferStateText.text = getString(R.string.state_transfer_wait_connection)
    }

    private fun buildWorkspaceFeedback(): LinearLayout = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        visibility = View.GONE
        setPadding(dp(12), dp(8), dp(12), dp(8))
        background = PremiumReferenceStyle.surface(this@MainActivity)
        contentDescription = "Current workspace feedback"
        workspaceFeedbackTitle = TextView(this@MainActivity).apply {
            setTextColor(Brand.text)
            typeface = Typeface.DEFAULT_BOLD
            textSize = 13f
        }
        workspaceFeedbackDetail = TextView(this@MainActivity).apply {
            setTextColor(Brand.textSoft)
            textSize = 12f
        }
        addView(workspaceFeedbackTitle)
        addView(workspaceFeedbackDetail)
    }

    private fun showMessage(title: String, detail: String) {
        if (!::statusTitle.isInitialized || !::statusDetail.isInitialized) return
        if (activeWorkspace == Workspace.SITES && siteEditorVisible &&
            ::siteFeedback.isInitialized) {
            siteFeedback.text = detail
            siteFeedback.visibility = View.VISIBLE
            return
        }
        if (activeWorkspace == Workspace.SETTINGS && ::settingsFeedback.isInitialized) {
            // Only the currently visible Settings notice owns this message.
            // Updating hidden Files/Transfers labels as well makes screen
            // readers and hierarchy tests pick the invisible duplicate first.
            settingsFeedback.text = detail
            settingsFeedback.visibility = View.VISIBLE
            return
        }
        statusTitle.text = title
        statusDetail.text = detail
        if (::transferStateText.isInitialized) transferStateText.text = detail
        if (::workspaceFeedbackPanel.isInitialized) {
            workspaceFeedbackTitle.text = title
            workspaceFeedbackDetail.text = detail
            workspaceFeedbackPanel.visibility = View.VISIBLE
            workspaceFeedbackPanel.announceForAccessibility("$title. $detail")
        }
    }

    private fun showIdleState() {
        if (!uiReady()) return
        statusTitle.text = getString(R.string.status_ready)
        statusDetail.text = getString(R.string.status_no_active_session)
        lastRemoteInventory = emptyList()
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
            selectConnectionProtocolAndPort(protocol, port)
            hostInput.setText(json.optString("host").trim())
            usernameInput.setText(json.optString("username").trim())
            passwordInput.text.clear()
            hostKeyFingerprintInput.setText(json.optString("hostKeyFingerprint").trim())
            sftpKeepAliveInput.setText(keepAlive.toString())
            remotePathInput.setText(json.optString("remotePath", "/").trim().ifBlank { "/" })
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
        updateSessionOperationOverview()
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

    private fun updateSessionOperationOverview() {
        if (!::transferOverviewText.isInitialized || closingOrDestroyed()) return
        val active = if (operationInFlight) 1 else 0
        val message = "This session · $active active · " +
            "$sessionCompletedOperations completed · " +
            "$sessionFailedOperations failed · $sessionCancelledOperations canceled"
        transferOverviewText.text = message
        transferOverviewText.contentDescription = message
    }

    private fun showTransferResult(result: TransferResult, refreshAfter: Boolean) {
        if (!uiReady()) return
        lastCompletedTransferPath = result.remotePath
        sessionCompletedOperations++
        updateSessionOperationOverview()
        statusTitle.text = result.title
        statusDetail.text = result.detail
        transferStateText.text = result.detail
        appendActivity(result.title, result.detail)
        if (::localRows.isInitialized) refreshLocalPane()
        if (refreshAfter && activeProfile != null) refreshActive()
    }

    private fun showTransferFailure(error: Throwable) {
        if (!uiReady()) return
        val canceled = error is OperationCanceledException
        if (canceled) sessionCancelledOperations++ else sessionFailedOperations++
        updateSessionOperationOverview()
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
        siteEditorVisible = state.getBoolean(STATE_SITE_EDITOR_OPEN, false)
        siteEditorPanel.visibility = if (siteEditorVisible) View.VISIBLE else View.GONE
        sitesOverviewPanel.visibility = if (siteEditorVisible) View.GONE else View.VISIBLE
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
        // Security: credentials and authenticated transport never survive a
        // recreated Activity. Display reconnection state in the actual Sites
        // editor or active page, not a hidden disconnected session card.
        showMessage(getString(R.string.status_ready), statusDetail.text.toString())
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

    private fun remoteRow(item: RemoteRow): View {
        val target = item.remotePath
        if (target == null || (!item.isDirectory && !item.isFile)) {
            return row(item.name, item.detail)
        }
        return compactBrowserRow(item.name, item.isDirectory) {
            if (item.isDirectory) {
                remotePathInput.setText(target)
                openConnection()
            } else {
                selectRemoteEntry(item)
            }
        }.apply {
            setOnLongClickListener {
                selectRemoteEntry(item)
                setWorkspace(Workspace.TRANSFERS)
                true
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

    private fun workspaceLabel(workspace: Workspace): String = getString(workspace.labelRes)

    private fun workspaceNavItem(workspace: Workspace): TextView {
        val label = workspaceLabel(workspace)
        val iconRail = false
        // Keep the localized full label available even on the compact icon rail.
        val displayLabel = label
        val iconResource = when (workspace) {
            Workspace.FILES -> R.drawable.ic_premium_nav_files
            Workspace.SITES -> R.drawable.ic_premium_nav_sites
            Workspace.TRANSFERS -> R.drawable.ic_premium_nav_transfers
            Workspace.SYNC -> R.drawable.ic_premium_nav_sync
            Workspace.CLOUD -> R.drawable.ic_premium_nav_cloud
            Workspace.BOOKMARKS -> R.drawable.ic_premium_nav_bookmarks
            Workspace.SETTINGS -> R.drawable.ic_premium_nav_settings
            Workspace.ABOUT -> R.drawable.ic_premium_nav_about
        }
        val view = TextView(this).apply {
            text = displayLabel
            contentDescription = getString(R.string.workspace_open, label)
            textSize = 14f
            maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
            typeface = Typeface.DEFAULT
            gravity = Gravity.CENTER_VERTICAL or Gravity.START
            compoundDrawablePadding = dp(14)
            val icon = getDrawable(iconResource)?.mutate()
            icon?.setBounds(0, 0, dp(20), dp(20))
            if (resources.configuration.screenWidthDp < 600) {
                setCompoundDrawables(icon, null, null, null)
            } else {
                setCompoundDrawables(null, icon, null, null)
            }
            setPadding(dp(14), dp(4), dp(8), dp(4))
            isClickable = true
            isFocusable = true
            setOnClickListener { setWorkspace(workspace) }
        }
        workspaceNavButtons[workspace] = view
        styleWorkspaceNavItem(view, selected = workspace == activeWorkspace)
        return view
    }

    private fun styleWorkspaceNavItem(view: TextView, selected: Boolean) {
        val color = if (selected) Brand.text else Brand.textSoft
        view.setTextColor(color)
        view.compoundDrawables.forEach { drawable ->
            drawable?.setTint(if (selected) Brand.accent else Brand.textSoft)
        }
        view.background = rounded(
            if (selected) Brand.accentSurface else Brand.background,
            dp(9),
            if (selected) Brand.border else Brand.background
        )
    }

    private fun primaryButton(value: String, onClick: () -> Unit): Button = Button(this).apply {
        filterTouchesWhenObscured = true
        text = value
        setTextColor(Brand.background)
        textSize = 14f
        typeface = Typeface.DEFAULT_BOLD
        background = PremiumReferenceStyle.actionBackground(this@MainActivity)
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
        val row: Int = panel
        val input: Int = background
        val badge: Int = Color.rgb(25, 47, 72)
        val border: Int = Color.rgb(39, 70, 99)
        val borderSubtle: Int = Color.rgb(25, 47, 72)
        val accent: Int = PremiumReferenceStyle.purple
        val accentSurface: Int = PremiumReferenceStyle.selected
        val success: Int = Color.rgb(38, 220, 143)
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
        const val PICK_CLOUD_UPLOAD_REQUEST = 22094
        const val CREATE_CLOUD_DOWNLOAD_REQUEST = 22095
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
        const val STATE_SITE_EDITOR_OPEN = "ghostftp.siteEditorOpen"
        const val STATE_NAVIGATION_OPEN = "ghostftp.navigationOpen"

        const val REPOSITORY_URL = "https://github.com/bren-wp/Ghost-FTP"
        const val RELEASES_URL = "https://github.com/bren-wp/Ghost-FTP/releases"
        const val SUPPORT_URL = "https://ghostftp.com/hr/support"
        const val DOCUMENTATION_URL = "https://ghostftp.com/hr/how-to-download"
        const val PRIVACY_URL = "https://ghostftp.com/hr/privacy"
        const val EULA_URL = "https://ghostftp.com/hr/terms"
        const val CONTACT_URL = "https://ghostftp.com/hr/contact"
        val OFFICIAL_LINKS = setOf(
            REPOSITORY_URL,
            RELEASES_URL,
            SUPPORT_URL,
            DOCUMENTATION_URL,
            PRIVACY_URL,
            EULA_URL,
            CONTACT_URL
        )
    }
}

// Render the exact same approved drawable in the sidebar, app header and
// launcher. A hand-drawn Canvas approximation drifted from the premium SVG.
private class GhostMarkView(context: Context) : ImageView(context) {
    init {
        setImageResource(R.drawable.ic_ghost_ftp)
        scaleType = ScaleType.FIT_CENTER
        contentDescription = context.getString(R.string.app_name)
    }
}
