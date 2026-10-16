package com.ghostftp.android

import android.app.Activity
import android.content.Intent
import android.graphics.Rect
import android.net.Uri
import android.os.SystemClock
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.Spinner
import android.widget.TextView
import androidx.test.espresso.Espresso.onView
import androidx.test.espresso.assertion.ViewAssertions.matches
import androidx.test.espresso.matcher.ViewMatchers.isDisplayed
import androidx.test.espresso.matcher.ViewMatchers.withText
import androidx.test.core.app.ActivityScenario
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class MainActivitySmokeTest {
    private lateinit var scenario: ActivityScenario<MainActivity>

    @Before
    fun launch() {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        instrumentation.waitForIdleSync()
        scenario = ActivityScenario.launch(MainActivity::class.java)
        instrumentation.waitForIdleSync()

        scenario.onActivity { activity ->
            val decor = activity.window.decorView
            assertTrue("MainActivity must not be finishing after launch.", !activity.isFinishing)
            assertTrue("MainActivity must not be destroyed after launch.", !activity.isDestroyed)
            assertTrue("MainActivity decor must be attached after launch.", decor.isAttachedToWindow)
            assertTrue("MainActivity decor must be shown after launch.", decor.isShown)
        }
    }

    @After
    fun close() {
        scenario.close()
    }

    @Test
    fun primaryWorkspaceAndToolbarRender() {
        // Mobile reference #6 uses the active page title rather than
        // a duplicated Ghost FTP banner above every workspace.
        assertTextPresent(appString(R.string.workspace_files))
        if (usesOverlayNavigation()) assertNavigationTogglePresent()
        ensureNavigationOpen()
        assertDescriptionPresent(appString(R.string.workspace_open, appString(R.string.workspace_files)))
        for (actionRes in listOf(
            R.string.action_upload,
            R.string.action_download,
            R.string.action_new_folder,
            R.string.action_delete
        )) {
            assertDescriptionPresent(appString(actionRes))
        }
        assertDescriptionPresent("More file actions")
    }

    @Test
    fun referenceFilesMoreExposesRealRenameAndRefreshMenuActions() {
        clickByDescription("More file actions")
        onView(withText(appString(R.string.action_rename))).check(matches(isDisplayed()))
        onView(withText(appString(R.string.action_refresh))).check(matches(isDisplayed()))
    }

    @Test
    fun workspaceNavigationWorksClickByClick() {
        openWorkspace(R.string.workspace_sites)
        assertTextVisibleInViewport(appString(R.string.field_protocol))

        openWorkspace(R.string.workspace_transfers)
        assertTextVisibleInViewport(appString(R.string.state_no_transfer))

        openWorkspaceByLabel("Sync & Backup")
        assertTextVisibleInViewport("No settings backup has been created or restored in this session.")
        assertDescriptionPresent("Back up settings")
        assertDescriptionPresent("Restore settings")

        openWorkspace(R.string.workspace_settings)
        clickByDescription("Settings: Advanced")
        assertTextVisibleInViewport(appString(R.string.settings_no_tracking))
        assertDescriptionPresent(appString(R.string.action_clear_activity))
        assertDescriptionPresent(appString(R.string.action_reset_transfers))
        assertDescriptionPresent(appString(R.string.action_reset_connection))
        assertDescriptionPresent("${appString(R.string.workspace_settings)} ${appString(R.string.action_disconnect)}")

        openWorkspace(R.string.workspace_about)
        assertTextVisibleInViewport("Ghost FTP · Brendigo")

        openWorkspace(R.string.workspace_files)
        assertTextVisibleInViewport(appString(R.string.state_connect_to_load_files))
    }

    @Test
    fun navigationRailCanOpenAndCloseWithoutPopupNavigation() {
        if (!usesOverlayNavigation()) {
            assertDescriptionPresent(appString(R.string.workspace_open, appString(R.string.workspace_files)))
            return
        }
        ensureNavigationOpen()
        assertDescriptionPresent(appString(R.string.nav_close))
        clickByDescription(appString(R.string.nav_close))
        assertDescriptionPresent(appString(R.string.nav_open))
        clickByDescription(appString(R.string.nav_open))
        assertDescriptionPresent(appString(R.string.nav_close))
    }

    @Test
    fun compactNavigationUsesOverlayScrimInsteadOfShrinkingContent() {
        var compact = false
        scenario.onActivity { activity ->
            compact = activity.resources.configuration.screenWidthDp < 600
        }
        if (!compact) return

        ensureNavigationOpen()
        assertDescriptionPresent(appString(R.string.nav_close_overlay))
        clickByDescription(appString(R.string.nav_close_overlay))
        assertDescriptionPresent(appString(R.string.nav_open))
    }

    @Test
    fun brandedNavigationUsesRealVectorIconsAndLocalizedLabels() {
        var isReferencePhone = false
        scenario.onActivity { activity ->
            isReferencePhone = activity.resources.configuration.screenWidthDp in 360..599
        }
        if (isReferencePhone) {
            assertDescriptionPresent(appString(R.string.nav_open))
        }
        ensureNavigationOpen()
        val destinations = listOf(
            appString(R.string.workspace_files),
            appString(R.string.workspace_sites),
            appString(R.string.workspace_transfers),
            "Sync & Backup",
            "Cloud storage",
            appString(R.string.workspace_settings),
            appString(R.string.workspace_about)
        )
        scenario.onActivity { activity ->
            for (label in destinations) {
                val description = appString(R.string.workspace_open, label)
                val item = findView(activity.window.decorView) {
                    it is TextView && it.contentDescription?.toString() == description
                } as? TextView
                assertNotNull("Navigation destination is missing: $description", item)
                assertNotNull("Navigation destination lacks its real vector glyph: $description",
                    item!!.compoundDrawables[1])
                assertEquals("Navigation label must stay localized: $description", label,
                    item.text.toString())
                assertTrue("Navigation item must be actionable: $description", item.isClickable)
            }
        }
    }

    @Test
    fun cloudWorkspaceHasCancellableTransferControls() {
        openWorkspaceByLabel("Cloud storage")
        // "Cloud storage" also occurs in the navigation rail. Checking the
        // workspace container verifies that navigation selected the right
        // screen rather than finding an offscreen rail label first.
        assertDescriptionPresent(appString(R.string.workspace_content, "Cloud storage"))
        assertTextPresent("No cloud transfer in progress.")
        assertDescriptionPresent("Cancel cloud transfer")
        assertDescriptionEnabled("Cancel cloud transfer", false)
        assertDescriptionEnabled("Choose file and upload", true)
        assertDescriptionEnabled("Choose destination and download", true)
    }

    @Test
    fun invalidSignedCloudLinkClearsBearerInputWithoutStartingPicker() {
        openWorkspaceByLabel("Cloud storage")
        scenario.onActivity { activity ->
            val input = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Paste short-lived signed GET URL"
            } as? EditText
            assertNotNull("Signed download input is missing.", input)
            input!!.setText("http://localhost/?sig=secret")
            val button = findView(activity.window.decorView) {
                it.contentDescription?.toString() == "Choose destination and download" && it.isShown
            }
            assertNotNull("Download picker action is missing.", button)
            button!!.performClick()
            assertEquals("Invalid bearer link must not stay in the form.", "", input.text.toString())
            assertTrue("Invalid URL must not disable the picker permanently.", button.isEnabled)
        }
        assertTextPresent("Invalid link. Use a signed public HTTPS object URL, not a sharing page.")
    }

    @Test
    fun interruptedSafDownloadRecoveryIsClickableAndNeverDeletesWithoutConsent() {
        val uri = "content://com.example.missing.documentprovider/document/potentially-incomplete"
        scenario.onActivity { activity ->
            CloudDestinationJournal.record(activity.noBackupFilesDir, uri)
        }
        try {
            scenario.recreate()
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            openWorkspaceByLabel("Cloud storage")
            assertTextPresent(
                "A previous cloud download may have left an incomplete document. Review it before continuing."
            )
            assertDescriptionEnabled("Choose file and upload", false)
            assertDescriptionEnabled("Choose destination and download", false)
            assertDescriptionEnabled("Try removing incomplete document", true)
            clickByDescription("Keep document and dismiss warning")
            assertTextPresent("Recovery reminder dismissed. The document was not changed.")
            assertDescriptionEnabled("Choose file and upload", true)
            assertDescriptionEnabled("Choose destination and download", true)
            scenario.onActivity { activity ->
                assertTrue("Recovery must be cleared only by user consent",
                    !CloudDestinationJournal.hasUnresolved(activity.noBackupFilesDir))
            }
        } finally {
            scenario.onActivity { activity ->
                CloudDestinationJournal.clear(activity.noBackupFilesDir)
            }
        }
    }

    @Test
    fun corruptSafRecoveryBlocksAllNewCloudActionsUntilUserReview() {
        scenario.onActivity { activity ->
            java.io.File(activity.noBackupFilesDir, "pending-cloud-download.uri")
                .writeText("unrecoverable-invalid-uri")
        }
        try {
            scenario.recreate()
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            openWorkspaceByLabel("Cloud storage")
            assertTextPresent(
                "Cloud recovery details were damaged. The previous document may still be incomplete. " +
                    "Review your cloud provider's files manually before dismissing this warning."
            )
            assertDescriptionEnabled("Choose file and upload", false)
            assertDescriptionEnabled("Choose destination and download", false)
            assertDescriptionEnabled("Try removing incomplete document", false)
            assertDescriptionEnabled("Keep document and dismiss warning", true)
            clickByDescription("Keep document and dismiss warning")
            assertDescriptionEnabled("Choose destination and download", true)
        } finally {
            scenario.onActivity { activity ->
                CloudDestinationJournal.clear(activity.noBackupFilesDir)
            }
        }
    }

    @Test
    fun referenceNavigationRailStaysDockedAcrossWorkspaces() {
        if (usesOverlayNavigation()) return
        assertDescriptionPresent(appString(R.string.workspace_open, appString(R.string.workspace_files)))
        openWorkspace(R.string.workspace_sites)
        assertDescriptionPresent(appString(R.string.workspace_open, appString(R.string.workspace_transfers)))
        openWorkspace(R.string.workspace_files)
        assertDescriptionPresent(appString(R.string.workspace_open, appString(R.string.workspace_sites)))
    }

    @Test
    fun referenceSettingsDashboardIsTwoColumnAndNavigable() {
        openWorkspace(R.string.workspace_settings)
        // The reference displays exactly four rows of two real categories;
        // the legacy actions remain behind the interactive Advanced card.
        for (name in listOf("Appearance", "Security", "Connections", "Notifications",
            "Transfers", "Advanced", "Cloud Storage", "About")) {
            assertDescriptionPresent("Settings: " + name)
        }
        clickByDescription("Settings: Appearance")
        assertTextPresent("Dark premium appearance is enabled.")
        clickByDescription("Settings: Security")
        assertDescriptionPresent("Settings: Advanced")
    }

    @Test
    fun settingsShowsVisibleLegalLinksAndBrendigoAuthor() {
        openWorkspace(R.string.workspace_settings)
        clickByDescription("Settings: Advanced")
        for (label in listOf(R.string.label_support, R.string.label_contact, R.string.label_privacy_policy, R.string.label_eula)) {
            val description = appString(R.string.link_open, appString(label))
            scenario.onActivity { activity ->
                val link = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == description && it.isShown
                }
                assertNotNull("Legal link must be visible in Settings: $description", link)
                assertTrue("Legal link must be clickable.", link!!.isClickable)
            }
        }
        scenario.onActivity { activity ->
            val label = appString(R.string.label_author)
            val author = findView(activity.window.decorView) {
                it.contentDescription?.toString() == "$label: Brendigo" && it.isShown
            }
            assertNotNull("Settings must credit Brendigo.", author)
        }
    }

    @Test
    fun helpWorkspaceExposesCanonicalProductLinks() {
        openWorkspace(R.string.workspace_about)
        clickByDescription("Settings: Licenses")
        for (description in listOf(
            appString(R.string.link_open, appString(R.string.label_support)),
            appString(R.string.link_open, appString(R.string.label_contact)),
            appString(R.string.link_open, appString(R.string.label_documentation)),
            appString(R.string.link_open, appString(R.string.label_privacy_policy)),
            appString(R.string.link_open, appString(R.string.label_eula)),
            appString(R.string.link_open, appString(R.string.label_project_repository))
        )) {
            assertDescriptionPresent(description)
        }
    }

    @Test
    fun protocolSelectionUpdatesDefaultsWithoutClobberingCustomPort() {
        openWorkspace(R.string.workspace_sites)

        scenario.onActivity { activity ->
            val protocol = findView(activity.window.decorView) {
                it is Spinner && it.contentDescription?.toString() == appString(R.string.field_protocol)
            } as? Spinner
            val port = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "21"
            } as? EditText
            val fingerprint = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.hint_sftp_fingerprint)
            } as? EditText

            assertNotNull("Protocol selector missing", protocol)
            assertNotNull("Port input missing", port)
            assertNotNull("SFTP fingerprint input missing", fingerprint)
            assertEquals("21", port!!.text.toString())
            assertTrue("SFTP fingerprint must stay hidden for FTP.", !fingerprint!!.isShown)

            protocol!!.setSelection(ConnectionProtocol.SFTP.ordinal)
        }
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)

        scenario.onActivity { activity ->
            val port = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "21"
            } as EditText
            val fingerprint = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.hint_sftp_fingerprint)
            } as EditText
            assertEquals("22", port.text.toString())
            assertTrue("SFTP fingerprint must become visible for SFTP.", fingerprint.isShown)

            port.setText("2222")
            val protocol = findView(activity.window.decorView) {
                it is Spinner && it.contentDescription?.toString() == appString(R.string.field_protocol)
            } as Spinner
            protocol.setSelection(ConnectionProtocol.FTP.ordinal)
        }
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)

        scenario.onActivity { activity ->
            val port = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "21"
            } as EditText
            val fingerprint = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.hint_sftp_fingerprint)
            } as EditText
            assertEquals("2222", port.text.toString())
            assertTrue("SFTP fingerprint must hide again for FTP.", !fingerprint.isShown)
        }
    }

    @Test
    fun savedSftpCustomPortSurvivesLoadEditAndProtocolRoundtrip() {
        // Reproduce the real Spinner sequence behind the v0.95.10 review:
        // loading SFTP on port 21 while FTP is selected, then editing the
        // port and switching away/back must never restore the old saved value.
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val name = "Port regression " + System.nanoTime()
        val fixture = ConnectionProfile(
            protocol = ConnectionProtocol.SFTP,
            host = "sftp.invalid",
            port = 21,
            username = "qa-user",
            password = "not-persisted",
            hostKeyFingerprint = "",
            remotePath = "/",
            keepAliveSeconds = 15
        )
        assertTrue(SavedSiteProfiles.save(context, name, fixture))
        try {
            scenario.recreate()
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            openWorkspace(R.string.workspace_sites)
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val sites = findView(root) {
                    it is Spinner && it.contentDescription?.toString() == "Saved connection profiles"
                } as Spinner
                val index = SavedSiteProfiles.read(context).indexOfFirst { it.name == name }
                assertTrue("Test saved profile missing from inventory", index >= 0)
                sites.setSelection(index + 1)
                val open = findText(root, "Use saved profile")
                assertNotNull("Use saved profile action missing", open)
                assertTrue(open!!.performClick())
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val port = findView(root) {
                    it is EditText && it.hint?.toString() == "21"
                } as EditText
                val protocol = findView(root) {
                    it is Spinner && it.contentDescription?.toString() == appString(R.string.field_protocol)
                } as Spinner
                assertEquals(ConnectionProtocol.SFTP.ordinal, protocol.selectedItemPosition)
                assertEquals("21", port.text.toString())
                port.setText("2021")
                protocol.setSelection(ConnectionProtocol.FTP.ordinal)
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val protocol = findView(root) {
                    it is Spinner && it.contentDescription?.toString() == appString(R.string.field_protocol)
                } as Spinner
                protocol.setSelection(ConnectionProtocol.SFTP.ordinal)
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            scenario.onActivity { activity ->
                val port = findView(activity.window.decorView) {
                    it is EditText && it.hint?.toString() == "21"
                } as EditText
                assertEquals("2021", port.text.toString())
            }
        } finally {
            assertTrue("Failed to remove isolated QA profile", SavedSiteProfiles.remove(context, name))
        }
    }

    @Test
    fun savedSiteEditAndConfirmedRemoveKeepUnrelatedProfiles() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val suffix = System.nanoTime().toString()
        val firstName = "First-" + suffix
        val secondName = "Second-" + suffix
        val renamed = "Edited-" + suffix
        val base = ConnectionProfile(
            protocol = ConnectionProtocol.FTP,
            host = "ftp.invalid",
            port = 21,
            username = "qa-user",
            password = "never-store",
            hostKeyFingerprint = "",
            remotePath = "/",
            keepAliveSeconds = 15
        )
        assertTrue(SavedSiteProfiles.save(context, firstName, base))
        assertTrue(SavedSiteProfiles.save(context, secondName, base.copy(port = 2121)))
        try {
            scenario.recreate()
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            openWorkspace(R.string.workspace_sites)
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val spinner = findView(root) {
                    it is Spinner && it.contentDescription?.toString() == "Saved connection profiles"
                } as Spinner
                val index = SavedSiteProfiles.read(context).indexOfFirst { it.name == secondName }
                assertTrue("Second test site missing", index >= 0)
                spinner.setSelection(index + 1)
                assertTrue(findText(root, "Use saved profile")!!.performClick())
                val siteName = findView(root) {
                    it is EditText && it.hint?.toString() == "Name for saved profile"
                } as EditText
                val port = findView(root) {
                    it is EditText && it.hint?.toString() == "21"
                } as EditText
                siteName.setText(renamed)
                port.setText("2244")
                assertTrue(findText(root, "Update selected profile")!!.performClick())
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            val afterEdit = SavedSiteProfiles.read(context)
            assertTrue(afterEdit.any { it.name == firstName && it.port == 21 })
            assertTrue(afterEdit.any { it.name == renamed && it.port == 2244 })
            assertTrue(afterEdit.none { it.name == secondName })

            // First cancel and verify the profile is not deleted.
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                assertTrue(findText(root, "Remove selected profile")!!.performClick())
                val confirm = findView(root) {
                    it.contentDescription?.toString() == "Remove profile"
                }
                assertNotNull("Inline removal confirmation missing", confirm)
                assertTrue("Confirmation must be visible before any deletion", confirm!!.isShown)
                val cancel = findView(root) {
                    it.contentDescription?.toString() == appString(R.string.action_cancel)
                }
                assertNotNull("Inline confirmation cancel action missing", cancel)
                assertTrue(cancel!!.performClick())
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            assertTrue("Cancel must preserve the selected profile",
                SavedSiteProfiles.read(context).any { it.name == renamed && it.port == 2244 })

            // Then explicitly confirm and verify only the selected entry is removed.
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                assertTrue(findText(root, "Remove selected profile")!!.performClick())
                val confirm = findView(root) {
                    it.contentDescription?.toString() == "Remove profile"
                }
                assertNotNull("Inline removal confirmation missing", confirm)
                assertTrue(confirm!!.isShown)
                assertTrue(confirm.performClick())
            }
            InstrumentationRegistry.getInstrumentation().waitForIdleSync()
            val afterDelete = SavedSiteProfiles.read(context)
            assertTrue(afterDelete.any { it.name == firstName && it.port == 21 })
            assertTrue(afterDelete.none { it.name == renamed })
        } finally {
            for (name in listOf(firstName, secondName, renamed)) {
                if (SavedSiteProfiles.read(context).any { it.name == name }) {
                    SavedSiteProfiles.remove(context, name)
                }
            }
        }
    }

    @Test
    fun connectionValidationAndIdleRecoveryWorkClickByClick() {
        openWorkspace(R.string.workspace_sites)
        clickByDescription(appString(R.string.action_connect))
        assertTextPresent(appString(R.string.msg_host_required_detail))

        assertDescriptionEnabled(appString(R.string.action_disconnect), false)
        assertDescriptionEnabled(appString(R.string.action_refresh), false)
        assertDescriptionEnabled(appString(R.string.action_refresh), false)
    }

    @Test
    fun guardedFileActionsStayDisabledWithoutActiveSession() {
        for (actionRes in listOf(
            R.string.action_upload,
            R.string.action_download,
            R.string.action_new_folder,
            R.string.action_delete
        )) {
            assertDescriptionEnabled(appString(actionRes), false)
        }
        assertDescriptionPresent("More file actions")

        openWorkspace(R.string.workspace_transfers)
        assertDescriptionEnabled(appString(R.string.action_pick_file), true)
        assertDescriptionEnabled(appString(R.string.action_upload), false)

        openWorkspace(R.string.workspace_settings)
        clickByDescription("Settings: Advanced")
        assertDescriptionEnabled(appString(R.string.action_reset_transfers), true)
        assertDescriptionEnabled(appString(R.string.action_reset_connection), true)
        assertDescriptionEnabled("${appString(R.string.workspace_settings)} ${appString(R.string.action_disconnect)}", false)
    }

    @Test
    fun transferAndSettingsActionsAreWiredClickByClick() {
        openWorkspace(R.string.workspace_transfers)
        assertDescriptionPresent(appString(R.string.action_pick_file))
        assertDescriptionPresent(appString(R.string.action_upload))
        assertDescriptionEnabled(appString(R.string.action_pick_file), true)
        assertDescriptionEnabled(appString(R.string.action_upload), false)

        openWorkspace(R.string.workspace_settings)
        clickByDescription("Settings: Advanced")
        assertDescriptionEnabled("${appString(R.string.workspace_settings)} ${appString(R.string.action_disconnect)}", false)
        clickByDescription(appString(R.string.action_clear_activity))
        assertTextPresent(appString(R.string.msg_activity_cleared_detail))

        clickByDescription(appString(R.string.action_reset_transfers))
        assertTextPresent(appString(R.string.msg_transfers_reset_detail))

        clickByDescription(appString(R.string.action_reset_connection))
        assertTextPresent(appString(R.string.msg_connection_reset_detail))

        assertDescriptionEnabled("${appString(R.string.workspace_settings)} ${appString(R.string.action_disconnect)}", false)
    }

    @Test
    fun unreadablePickerUriIsRejectedFailClosed() {
        scenario.onActivity { activity ->
            val callback = MainActivity::class.java.getDeclaredMethod(
                "onActivityResult",
                Int::class.javaPrimitiveType,
                Int::class.javaPrimitiveType,
                Intent::class.java
            )
            callback.isAccessible = true
            callback.invoke(
                activity,
                22091,
                Activity.RESULT_OK,
                Intent().apply {
                    data = Uri.parse("content://com.ghostftp.android.missing/not-readable")
                }
            )
        }
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)

        assertTextPresent(appString(R.string.msg_selected_file_unavailable_title))
        openWorkspace(R.string.workspace_transfers)
        assertDescriptionEnabled(appString(R.string.action_upload), false)
    }

    @Test
    fun bookmarksReferenceWorkspaceHasWorkingAddActionWithoutInventingSites() {
        openWorkspace(R.string.workspace_bookmarks)
        assertTextPresent(appString(R.string.workspace_bookmarks))
        assertDescriptionPresent(appString(R.string.bookmarks_add))
        clickByDescription(appString(R.string.bookmarks_add))
        assertTextPresent(appString(R.string.bookmarks_connect_first))
        // A bookmark never stores credentials or bypasses SSH host-key trust:
        // the add action must refuse to create anything before a live session.
        assertTextNotShown("Production Server")
    }

    @Test
    fun workspacesAreExclusiveInsteadOfOneLongScreen() {
        assertTextVisibleInViewport(appString(R.string.state_connect_to_load_files))
        assertTextNotShown(appString(R.string.field_protocol))

        openWorkspace(R.string.workspace_sites)
        assertTextVisibleInViewport(appString(R.string.field_protocol))
        assertTextNotShown(appString(R.string.state_connect_to_load_files))

        openWorkspace(R.string.workspace_transfers)
        assertTextVisibleInViewport(appString(R.string.state_no_transfer))
        assertTextNotShown(appString(R.string.field_protocol))
    }

    @Test
    fun activityRecreationRestoresNonSecretFieldsButNotPassword() {
        openWorkspace(R.string.workspace_sites)
        scenario.onActivity { activity ->
            val host = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_host)
            } as? EditText
            val username = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_username)
            } as? EditText
            val password = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_password)
            } as? EditText
            val renameTarget = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.hint_rename_target)
            } as? EditText

            assertNotNull("Host input missing", host)
            assertNotNull("Username input missing", username)
            assertNotNull("Password input missing", password)
            assertNotNull("Rename target input missing", renameTarget)
            host!!.setText("ftp.lifecycle.test")
            username!!.setText("ghost-user")
            password!!.setText("must-not-survive")
            renameTarget!!.setText("renamed.txt")
        }

        scenario.recreate()
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)

        scenario.onActivity { activity ->
            val host = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_host)
            } as EditText
            val username = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_username)
            } as EditText
            val password = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.field_password)
            } as EditText
            val renameTarget = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == appString(R.string.hint_rename_target)
            } as EditText

            assertEquals("ftp.lifecycle.test", host.text.toString())
            assertEquals("ghost-user", username.text.toString())
            assertEquals("renamed.txt", renameTarget.text.toString())
            assertTrue("Password must never survive Activity recreation.", password.text.isEmpty())
        }
        assertTextPresent(appString(R.string.restore_reconnect))
        assertTextVisibleInViewport(appString(R.string.field_protocol))
    }

    private fun assertTextNotShown(expected: String) {
        scenario.onActivity { activity ->
            val visible = findView(activity.window.decorView) {
                it is TextView && it.text?.toString() == expected && it.isShown
            }
            assertTrue("Text should not be shown in the active workspace: $expected", visible == null)
        }
    }

    private fun appString(resId: Int, vararg formatArgs: Any): String =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(resId, *formatArgs)

    private fun usesOverlayNavigation(): Boolean {
        var result = false
        scenario.onActivity { activity ->
            result = activity.resources.configuration.screenWidthDp < 600
        }
        return result
    }

    private fun assertNavigationTogglePresent() {
        scenario.onActivity { activity ->
            val root = activity.window.decorView
            val open = findView(root) { it.contentDescription?.toString() == appString(R.string.nav_open) }
            val close = findView(root) { it.contentDescription?.toString() == appString(R.string.nav_close) }
            assertTrue("Navigation toggle must be visible.", open?.isShown == true || close?.isShown == true)
        }
    }

    private fun ensureNavigationOpen() {
        var needsOpen = false
        scenario.onActivity { activity ->
            val root = activity.window.decorView
            val open = findView(root) { it.contentDescription?.toString() == appString(R.string.nav_open) }
            needsOpen = open?.isShown == true
        }
        if (needsOpen) clickByDescription(appString(R.string.nav_open))
    }

    private fun openWorkspace(labelRes: Int) {
        ensureNavigationOpen()
        clickByDescription(appString(R.string.workspace_open, appString(labelRes)))
        // The user-approved Sites screen is a real metadata list. Legacy
        // connection form tests navigate through its working Add Site action.
        if (labelRes == R.string.workspace_transfers) {
            var optionsAlreadyVisible = false
            scenario.onActivity { activity ->
                optionsAlreadyVisible = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == appString(R.string.action_pick_file) &&
                        it.isShown
                } != null
            }
            if (!optionsAlreadyVisible) clickByDescription("Transfer options")
        }
        if (labelRes == R.string.workspace_sites) {
            var addVisible = false
            scenario.onActivity { activity ->
                addVisible = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Add Site" && it.isShown
                } != null
            }
            if (addVisible) clickByDescription("Add Site")
        }
    }

    private fun openWorkspaceByLabel(label: String) {
        ensureNavigationOpen()
        clickByDescription(appString(R.string.workspace_open, label))
    }

    private fun clickByDescription(description: String) {
        scenario.onActivity { activity ->
            val view = findView(activity.window.decorView) {
                it.contentDescription?.toString() == description && it.isShown
            }
            assertNotNull("Missing control with content description: $description", view)
            view!!
            assertTrue("Control must be enabled before click: $description", view.isEnabled)

            val width = view.width.coerceAtLeast(1)
            val height = view.height.coerceAtLeast(1)
            view.requestRectangleOnScreen(Rect(0, 0, width, height), true)
            assertTrue("Click listener did not handle control: $description", view.performClick())
        }
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)
    }

    private fun assertDescriptionEnabled(description: String, expected: Boolean) {
        scenario.onActivity { activity ->
            val view = findView(activity.window.decorView) {
                it.contentDescription?.toString() == description && it.isShown
            }
            assertNotNull("Missing control with content description: $description", view)
            assertEquals(
                "Unexpected enabled state for control: $description",
                expected,
                view!!.isEnabled
            )
        }
    }

    private fun assertDescriptionPresent(description: String) {
        scenario.onActivity { activity ->
            val view = findView(activity.window.decorView) {
                it.contentDescription?.toString() == description && it.isShown
            }
            assertNotNull("Missing control with content description: $description", view)
            assertTrue("Matched control must remain visible: $description", view!!.isShown)
        }
    }

    private fun assertTextPresent(expected: String) {
        scenario.onActivity { activity ->
            // The full-width design keeps an off-screen navigation drawer.
            // A hidden drawer label may precede the visible page heading;
            // assert against a genuinely visible matching TextView.
            val view = findView(activity.window.decorView) {
                it is TextView && it.text?.toString() == expected && it.isShown
            }
            assertNotNull("Missing visible text in MainActivity hierarchy: $expected", view)
        }
    }

    private fun assertTextVisibleInViewport(expected: String) {
        repeat(12) {
            var visible = false
            scenario.onActivity { activity ->
                val view = findView(activity.window.decorView) {
                    it is TextView && it.text?.toString() == expected && it.isShown
                }
                val rect = Rect()
                visible = view?.getGlobalVisibleRect(rect) == true && rect.width() > 0 && rect.height() > 0
            }
            if (visible) return
            SystemClock.sleep(50)
        }
        assertTrue("Text did not become visible after workspace navigation: $expected", false)
    }

    private fun findText(root: View, expected: String): TextView? {
        val match = findView(root) { view ->
            view is TextView && view.text?.toString() == expected
        }
        return match as? TextView
    }

    private fun findView(root: View, predicate: (View) -> Boolean): View? {
        if (predicate(root)) return root
        if (root is ViewGroup) {
            for (index in 0 until root.childCount) {
                val match = findView(root.getChildAt(index), predicate)
                if (match != null) return match
            }
        }
        return null
    }
}
