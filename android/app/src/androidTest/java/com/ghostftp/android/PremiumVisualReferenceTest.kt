package com.ghostftp.android

import android.graphics.Bitmap
import android.graphics.Canvas
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.ScrollView
import android.widget.TextView
import androidx.test.core.app.ActivityScenario
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Renders only the live, empty, debug-only workspace hierarchy at the approved
 * Android reference viewport. These images are QA evidence, not mockup copies.
 *
 * The production FLAG_SECURE window policy remains untouched: rendering the
 * empty view tree inside a test process is deliberately separate from taking
 * unrestricted screenshots of user credentials via Android's capture APIs.
 */
@RunWith(AndroidJUnit4::class)
class PremiumVisualReferenceTest {
    private val instrumentation = InstrumentationRegistry.getInstrumentation()

    @Test
    fun captureNineReferenceScreensAt480x960() {
        val screens = listOf(
            "files" to R.string.workspace_files,
            "sites" to R.string.workspace_sites,
            "bookmarks" to R.string.workspace_bookmarks,
            "transfers" to R.string.workspace_transfers,
            "sync" to null,
            "cloud" to null,
            "settings" to R.string.workspace_settings,
            "about" to R.string.workspace_about
        )
        val output = File(instrumentation.targetContext.filesDir, "premium-qa")
        assertTrue("Cannot create private debug QA directory.", output.isDirectory || output.mkdirs())
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            for ((name, labelRes) in screens) {
                scenario.onActivity { activity ->
                    val label = when (name) {
                        "sync" -> "Sync & Backup"
                        "cloud" -> "Cloud storage"
                        else -> activity.getString(labelRes!!)
                    }
                    val root = activity.window.decorView
                    val toggle = findView(root) {
                        it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                    }
                    if (toggle != null) assertTrue("Navigation drawer must open", toggle.performClick())
                    val description = activity.getString(R.string.workspace_open, label)
                    val destination = findView(root) {
                        it is TextView && it.contentDescription?.toString() == description
                    } as? TextView
                    assertNotNull("Live workspace is missing: $name", destination)
                    assertTrue("Navigation item must be visible before tapping", destination!!.isShown)
                    assertTrue("Workspace cannot be selected: $name", destination.performClick())
                }
                instrumentation.waitForIdleSync()
                scenario.onActivity { activity ->
                    // Do not publish any screenshot if test state contains a
                    // host, account, credential, custom path or object URL.
                    assertNoUserInputInHierarchy(activity.window.decorView)
                    val decor = activity.window.decorView
                    assertEquals("QA capture must use approved 480 px viewport.", 480, decor.width)
                    assertTrue("Application window not laid out.", decor.height > 0)
                    val pixels = Bitmap.createBitmap(decor.width, decor.height, Bitmap.Config.ARGB_8888)
                    try {
                        pixels.eraseColor(android.graphics.Color.rgb(7, 14, 26))
                        decor.draw(Canvas(pixels))
                        val file = File(output, "$name.png")
                        file.outputStream().use { stream ->
                            assertTrue("Cannot encode live $name workspace.", pixels.compress(Bitmap.CompressFormat.PNG, 100, stream))
                        }
                        assertTrue("Empty screenshot: $name", file.length() > 4096L)
                    } finally {
                        pixels.recycle()
                    }
                }
            }
            // Ninth real Android screenshot: Add/Edit Site, General tab
            // (reference #4), using only empty safe fields. No dummy account,
            // password, hostname, protocol response or fake login.
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val toggle = findView(root) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                }
                assertNotNull("Site editor capture needs real drawer navigation", toggle)
                assertTrue(toggle!!.performClick())
                val nav = findView(root) {
                    it.contentDescription?.toString() ==
                        activity.getString(R.string.workspace_open,
                            activity.getString(R.string.workspace_sites))
                }
                assertNotNull("Sites workspace must be available", nav)
                assertTrue(nav!!.performClick())
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val add = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Add Site" && it.isShown
                }
                assertNotNull("Real contextual + Add Site button required", add)
                assertTrue(add!!.performClick())
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                assertNoUserInputInHierarchy(root)
                val general = findView(root) {
                    it.contentDescription?.toString() == "Site editor General tab" && it.isShown
                }
                assertNotNull("Native reference General tab missing", general)
                val decor = activity.window.decorView
                assertEquals(480, decor.width)
                val screenshot = Bitmap.createBitmap(decor.width, decor.height,
                    Bitmap.Config.ARGB_8888)
                try {
                    screenshot.eraseColor(android.graphics.Color.rgb(7, 14, 26))
                    decor.draw(Canvas(screenshot))
                    val target = File(output, "site-editor.png")
                    target.outputStream().use { stream ->
                        assertTrue("Cannot export real Add/Edit Site capture",
                            screenshot.compress(Bitmap.CompressFormat.PNG, 100, stream))
                    }
                    assertTrue("Empty Add/Edit Site screenshot", target.length() > 4096L)
                } finally {
                    screenshot.recycle()
                }
            }
        }

        // Actual reference #11 Search screenshot, from a disconnected fresh
        // Activity. No fake remote rows, hostname, credentials or server.
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val button = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Search files" && it.isShown
                }
                assertNotNull("Real Files search action missing", button)
                assertTrue(button!!.performClick())
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val query = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Search file name" && it.isShown
                }
                assertNotNull("Real editable Search form missing", query)
                captureIdleReference(activity, output, "search")
                val close = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Close Search" && it.isShown
                }
                assertTrue("Search must close back to Files", close?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val open = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                }
                assertTrue("Open the native drawer for reference #2", open?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val siteItem = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == activity.getString(
                        R.string.workspace_open, activity.getString(R.string.workspace_sites))
                }
                assertTrue("Drawer must visibly expose saved-site entry",
                    siteItem?.isShown == true)
                captureIdleReference(activity, output, "drawer")
            }
        }
        val context = instrumentation.targetContext
        val splashIntent = android.content.Intent(context, SplashActivity::class.java)
            .putExtra("ghostftp.qa.hold_splash", true)
        ActivityScenario.launch<SplashActivity>(splashIntent).use { scenario ->
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val brand = findView(activity.window.decorView) {
                    it is TextView && it.text.toString() == "Ghost FTP" && it.isShown
                }
                assertNotNull("Actual branded splash frame missing", brand)
                captureIdleReference(activity, output, "splash")
            }
        }
        // Real transfer-details surface, currently idle; never forge a
        // percentage, transfer speed, ETA or server result for screenshot QA.
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            scenario.onActivity { activity ->
                val open = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                }
                assertTrue("Open drawer before Transfers", open?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val menu = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == activity.getString(
                        R.string.workspace_open, activity.getString(R.string.workspace_transfers))
                }
                assertTrue("Open actual Transfers", menu?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val details = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "View transfer details" && it.isShown
                }
                assertTrue("Real details route must be tappable", details?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val cancel = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == "Cancel active transfer"
                }
                assertTrue("No idle cancel action may be enabled", cancel?.isEnabled == false)
                captureIdleReference(activity, output, "transfer-details")
            }
        }
    }

    @Test
    fun syncReferenceUsesActualFoldersAndBlocksUnimplementedDestructiveMirror() {
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val open = findView(root) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                }
                assertTrue("Sync test needs real navigation", open?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val target = activity.getString(R.string.workspace_open,
                    activity.getString(R.string.workspace_sync))
                val sync = findView(activity.window.decorView) {
                    it.contentDescription?.toString() == target
                }
                assertTrue("Native Sync workspace must be reachable", sync?.performClick() == true)
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val compare = findView(root) {
                    it.contentDescription?.toString() ==
                        "Compare current local and remote directories"
                }
                val sync = findView(root) {
                    it is android.widget.Switch && it.text.toString() == "Synchronize"
                } as? android.widget.Switch
                val mirror = findView(root) {
                    it is android.widget.Switch &&
                        it.text.toString() == "Mirror (delete extra files)"
                } as? android.widget.Switch
                assertNotNull("Reference #12 requires a real Compare action", compare)
                assertTrue("Compare must be usable", compare!!.isEnabled)
                assertNotNull("Sync control should be visible", sync)
                assertNotNull("Destructive mirror control should be visible", mirror)
                assertTrue("Sync may not promise an unimplemented transfer engine",
                    sync?.isEnabled == false && sync.isChecked == false)
                assertTrue("Mirror may not delete files without a verified engine",
                    mirror?.isEnabled == false && mirror.isChecked == false)
                assertTrue("Disconnected comparison should fail closed",
                    compare.performClick())
            }
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val message = findView(activity.window.decorView) {
                    it is TextView &&
                        it.text.toString().contains("Connect and refresh the remote Files pane")
                }
                assertNotNull("No remote listing must not become a fake empty comparison", message)
            }
        }
    }

    @Test
    fun protocolTilesUseActualProtocolSpecificReferenceColours() {
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            scenario.onActivity { activity ->
                val icons = ConnectionProtocol.entries.map {
                    PremiumReferenceStyle.serverIcon(activity, it)
                }
                assertEquals(3, icons.size)
                for ((index, protocol) in ConnectionProtocol.entries.withIndex()) {
                    assertEquals("Protocol accessibility label must not be invented",
                        protocol.label + " server", icons[index].contentDescription)
                    assertNotNull("Protocol tile must have its own background",
                        icons[index].background)
                }
                assertTrue("FTP, FTPS, SFTP must be distinguishable by protocol tint",
                    icons.map { it.currentTextColor }.distinct().size == 3)
            }
        }
    }

    private fun captureIdleReference(
        activity: android.app.Activity, directory: File, filename: String
    ) {
        val decor = activity.window.decorView
        assertNoUserInputInHierarchy(decor)
        assertEquals("Reference width must remain exactly 480 px.", 480, decor.width)
        assertTrue("Actual app screenshot has no height.", decor.height > 0)
        val capture = Bitmap.createBitmap(decor.width, decor.height, Bitmap.Config.ARGB_8888)
        try {
            capture.eraseColor(android.graphics.Color.rgb(7, 14, 26))
            decor.draw(Canvas(capture))
            val target = File(directory, filename + ".png")
            target.outputStream().use {
                assertTrue("Could not encode actual " + filename + " screen",
                    capture.compress(Bitmap.CompressFormat.PNG, 100, it))
            }
            assertTrue("Empty actual visual QA frame: " + filename,
                target.length() > 4096L)
        } finally {
            capture.recycle()
        }
    }

    @Test
    fun premiumChromeAndRailUseReferenceGeometry() {
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val widthDp = activity.resources.configuration.screenWidthDp
                assertEquals("Reference QA emulator must have width 480dp.", 480, widthDp)
                val drawerButton = findView(root) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_open)
                }
                assertNotNull("Reference phone drawer opener missing", drawerButton)
                assertTrue("Reference drawer must open", drawerButton!!.performClick())
            }
            // A View changing GONE -> VISIBLE has width 0 until the
            // actual next layout traversal. Measure after Android layout,
            // not synchronously inside the click callback.
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val density = activity.resources.displayMetrics.density
                val nav = findView(root) {
                    it is TextView &&
                        it.contentDescription?.toString() ==
                        activity.getString(R.string.workspace_open, activity.getString(R.string.workspace_files))
                }
                assertNotNull("Real branded Files navigation missing.", nav)
                var rail: View? = nav
                while (rail != null && rail !is ScrollView) {
                    rail = rail.parent as? View
                }
                assertNotNull("Real scrollable navigation rail missing.", rail)
                assertEquals("280dp drawer must match the supplied phone reference.", (280 * density + .5f).toInt(), rail!!.width)

                val quick = findView(root) {
                    it.contentDescription?.toString() == activity.getString(R.string.nav_close)
                }
                assertNotNull("Full-width top brand bar must remain actionable.", quick)
                val chrome = quick!!.parent as? View
                assertNotNull("Full-width top brand bar missing.", chrome)
                assertEquals("52dp top bar must match approved phone geometry.", (52 * density + .5f).toInt(), chrome!!.height)
                assertEquals("Brand bar must cover viewport.", decorWidth(activity), chrome.width)
            }
        }
    }

    @Test
    fun premiumWorkspacesDoNotShowUnrelatedFilesToolbarOrSessionCard() {
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            instrumentation.waitForIdleSync()
            scenario.onActivity { activity ->
                val root = activity.window.decorView
                val fileActions = findView(root) { it.contentDescription == "File actions" }
                val session = findView(root) { it.contentDescription == "Session status panel" }
                assertNotNull("Files action toolbar must exist.", fileActions)
                assertNotNull("Session status panel must exist.", session)
                assertEquals(View.VISIBLE, fileActions!!.visibility)
                assertEquals(View.GONE, session!!.visibility)

                fun visit(label: String) {
                    val target = findView(root) {
                        it is TextView &&
                            it.contentDescription?.toString() ==
                            activity.getString(R.string.workspace_open, label)
                    }
                    assertNotNull("Workspace navigation item missing: $label", target)
                    assertTrue(target!!.performClick())
                }

                visit(activity.getString(R.string.workspace_about))
                assertEquals("About must not display destructive FTP actions.", View.GONE, fileActions.visibility)
                assertEquals("About must not show Files-only status.", View.GONE, session.visibility)
                visit(activity.getString(R.string.workspace_sites))
                assertEquals("Sites must use its own connection actions.", View.GONE, fileActions.visibility)
                assertEquals(View.GONE, session.visibility)
                visit(activity.getString(R.string.workspace_files))
                assertEquals(View.VISIBLE, fileActions.visibility)
                assertEquals(View.GONE, session.visibility)
            }
        }
    }

    private fun decorWidth(activity: MainActivity): Int = activity.window.decorView.width

    private fun assertNoUserInputInHierarchy(root: View) {
        if (root is EditText) {
            val raw = root.text?.toString()?.trim().orEmpty()
            val allowedDefaults = setOf("", "21", "22", "15", "/", "990", "2222")
            assertTrue("Refusing to export a QA screenshot containing nondefault user input.", raw in allowedDefaults)
        }
        if (root is ViewGroup) {
            for (index in 0 until root.childCount) {
                assertNoUserInputInHierarchy(root.getChildAt(index))
            }
        }
    }

    private fun findView(root: View, predicate: (View) -> Boolean): View? {
        if (predicate(root)) return root
        if (root is ViewGroup) {
            for (index in 0 until root.childCount) {
                val found = findView(root.getChildAt(index), predicate)
                if (found != null) return found
            }
        }
        return null
    }
}
