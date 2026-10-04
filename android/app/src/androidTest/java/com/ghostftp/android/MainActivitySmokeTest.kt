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
        assertTextPresent("Ghost FTP")
        assertDescriptionPresent("Open Files workspace")
        for (label in listOf("Refresh", "Upload", "Download", "New Folder", "Rename", "Delete")) {
            assertDescriptionPresent("$label action")
        }
    }

    @Test
    fun workspaceNavigationWorksClickByClick() {
        clickByDescription("Open Sites workspace")
        assertTextVisibleInViewport("Protocol")

        clickByDescription("Open Transfers workspace")
        assertTextVisibleInViewport("No transfer started.")

        clickByDescription("Open Settings workspace")
        assertTextVisibleInViewport("No required tracking, analytics or telemetry.")
        assertDescriptionPresent("Clear activity log")
        assertDescriptionPresent("Reset transfer fields")
        assertDescriptionPresent("Reset connection form")
        assertDescriptionPresent("Settings disconnect session")

        clickByDescription("Open Help & About workspace")
        assertTextVisibleInViewport("Ghost FTP by Brendigo")

        clickByDescription("Open Files workspace")
        assertTextVisibleInViewport("Connect to a server to load remote files.")
    }

    @Test
    fun protocolSelectionUpdatesDefaultsWithoutClobberingCustomPort() {
        clickByDescription("Open Sites workspace")

        scenario.onActivity { activity ->
            val protocol = findView(activity.window.decorView) {
                it is Spinner && it.contentDescription?.toString() == "Connection protocol"
            } as? Spinner
            val port = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "21"
            } as? EditText
            val fingerprint = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "SHA256 fingerprint for SFTP"
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
                it is EditText && it.hint?.toString() == "SHA256 fingerprint for SFTP"
            } as EditText
            assertEquals("22", port.text.toString())
            assertTrue("SFTP fingerprint must become visible for SFTP.", fingerprint.isShown)

            port.setText("2222")
            val protocol = findView(activity.window.decorView) {
                it is Spinner && it.contentDescription?.toString() == "Connection protocol"
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
                it is EditText && it.hint?.toString() == "SHA256 fingerprint for SFTP"
            } as EditText
            assertEquals("2222", port.text.toString())
            assertTrue("SFTP fingerprint must hide again for FTP.", !fingerprint.isShown)
        }
    }

    @Test
    fun connectionValidationAndIdleRecoveryWorkClickByClick() {
        clickByDescription("Open Sites workspace")
        clickByDescription("Connect to server")
        assertTextPresent("Host is required")

        assertDescriptionEnabled("Disconnect from server", false)
        assertDescriptionEnabled("Refresh current session", false)
        assertDescriptionEnabled("Refresh action", false)
    }

    @Test
    fun guardedFileActionsStayDisabledWithoutActiveSession() {
        for (label in listOf("Refresh", "Upload", "Download", "New Folder", "Rename", "Delete")) {
            assertDescriptionEnabled("$label action", false)
        }

        clickByDescription("Open Transfers workspace")
        assertDescriptionEnabled("Pick upload file", true)
        assertDescriptionEnabled("Upload selected file", false)

        clickByDescription("Open Settings workspace")
        assertDescriptionEnabled("Reset transfer fields", true)
        assertDescriptionEnabled("Reset connection form", true)
        assertDescriptionEnabled("Settings disconnect session", false)
    }

    @Test
    fun transferAndSettingsActionsAreWiredClickByClick() {
        clickByDescription("Open Transfers workspace")
        assertDescriptionPresent("Pick upload file")
        assertDescriptionPresent("Upload selected file")
        assertDescriptionEnabled("Pick upload file", true)
        assertDescriptionEnabled("Upload selected file", false)

        clickByDescription("Open Settings workspace")
        assertDescriptionEnabled("Settings disconnect session", false)
        clickByDescription("Clear activity log")
        assertTextPresent("Activity log cleared.")

        clickByDescription("Reset transfer fields")
        assertTextPresent("Transfer fields reset.")

        clickByDescription("Reset connection form")
        assertTextPresent("Connection form reset.")

        assertDescriptionEnabled("Settings disconnect session", false)
    }

    @Test
    fun unreadablePickerUriIsRejectedFailClosed() {
        scenario.onActivity { activity ->
            activity.onActivityResult(
                22091,
                Activity.RESULT_OK,
                Intent().apply {
                    data = Uri.parse("content://com.ghostftp.android.missing/not-readable")
                }
            )
        }
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        SystemClock.sleep(120)

        assertTextPresent("Selected file unavailable")
        clickByDescription("Open Transfers workspace")
        assertDescriptionEnabled("Upload selected file", false)
    }

    @Test
    fun workspacesAreExclusiveInsteadOfOneLongScreen() {
        assertTextVisibleInViewport("Connect to a server to load remote files.")
        assertTextNotShown("Protocol")

        clickByDescription("Open Sites workspace")
        assertTextVisibleInViewport("Protocol")
        assertTextNotShown("Connect to a server to load remote files.")

        clickByDescription("Open Transfers workspace")
        assertTextVisibleInViewport("No transfer started.")
        assertTextNotShown("Protocol")
    }

    @Test
    fun activityRecreationRestoresNonSecretFieldsButNotPassword() {
        clickByDescription("Open Sites workspace")
        scenario.onActivity { activity ->
            val host = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Host"
            } as? EditText
            val username = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Username"
            } as? EditText
            val password = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Password"
            } as? EditText
            val renameTarget = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "New remote name or path"
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
                it is EditText && it.hint?.toString() == "Host"
            } as EditText
            val username = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Username"
            } as EditText
            val password = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "Password"
            } as EditText
            val renameTarget = findView(activity.window.decorView) {
                it is EditText && it.hint?.toString() == "New remote name or path"
            } as EditText

            assertEquals("ftp.lifecycle.test", host.text.toString())
            assertEquals("ghost-user", username.text.toString())
            assertEquals("renamed.txt", renameTarget.text.toString())
            assertTrue("Password must never survive Activity recreation.", password.text.isEmpty())
        }
        assertTextPresent("Android restored non-secret workspace state. Reconnect to authenticate before remote actions.")
        assertTextVisibleInViewport("Protocol")
    }

    private fun assertTextNotShown(expected: String) {
        scenario.onActivity { activity ->
            val view = findText(activity.window.decorView, expected)
            if (view != null) {
                assertTrue("Text should not be shown in the active workspace: $expected", !view.isShown)
            }
        }
    }

    private fun clickByDescription(description: String) {
        scenario.onActivity { activity ->
            val view = findView(activity.window.decorView) {
                it.contentDescription?.toString() == description
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
                it.contentDescription?.toString() == description
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
                it.contentDescription?.toString() == description
            }
            assertNotNull("Missing control with content description: $description", view)
            assertTrue("Control is not shown in the Activity hierarchy: $description", view!!.isShown)
        }
    }

    private fun assertTextPresent(expected: String) {
        scenario.onActivity { activity ->
            val view = findText(activity.window.decorView, expected)
            assertNotNull("Missing text in MainActivity hierarchy: $expected", view)
            assertTrue("Text is not shown in MainActivity hierarchy: $expected", view!!.isShown)
        }
    }

    private fun assertTextVisibleInViewport(expected: String) {
        repeat(12) {
            var visible = false
            scenario.onActivity { activity ->
                val view = findText(activity.window.decorView, expected)
                assertNotNull("Missing text in MainActivity hierarchy: $expected", view)
                val rect = Rect()
                visible = view!!.getGlobalVisibleRect(rect) && rect.width() > 0 && rect.height() > 0
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
