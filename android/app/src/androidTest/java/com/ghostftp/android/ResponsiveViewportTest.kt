package com.ghostftp.android

import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Rect
import android.view.View
import android.view.ViewGroup
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
 * Runs once per emulator viewport supplied by run-instrumentation-smoke.sh.
 * Captures ACTUAL idle screen hierarchies; no mock FTP servers, credentials,
 * fabricated transfers or static reference screenshots are injected.
 */
@RunWith(AndroidJUnit4::class)
class ResponsiveViewportTest {
    private val instrumentation = InstrumentationRegistry.getInstrumentation()

    @Test
    fun realEightWorkspacesRemainNavigableAcrossPhoneSizes() {
        val output = File(instrumentation.targetContext.filesDir, "premium-responsive")
        assertTrue(output.isDirectory || output.mkdirs())
        val names = listOf(
            "files" to R.string.workspace_files,
            "sites" to R.string.workspace_sites,
            "transfers" to R.string.workspace_transfers,
            "sync" to R.string.workspace_sync,
            "cloud" to R.string.workspace_cloud,
            "bookmarks" to R.string.workspace_bookmarks,
            "settings" to R.string.workspace_settings,
            "about" to R.string.workspace_about
        )
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            for ((name, labelRes) in names) {
                scenario.onActivity { activity ->
                    val root = activity.window.decorView
                    val open = find(root) {
                        it.contentDescription?.toString() ==
                            activity.getString(R.string.nav_open) && it.isShown
                    }
                    open?.performClick()
                    val item = find(root) {
                        it is TextView && it.contentDescription?.toString() ==
                            activity.getString(R.string.workspace_open,
                                activity.getString(labelRes))
                    }
                    assertNotNull("Missing accessible $name destination", item)
                    assertTrue("Destination $name is not visible or tappable",
                        item!!.isShown && item.performClick())
                }
                instrumentation.waitForIdleSync()
                scenario.onActivity { activity ->
                    val root = activity.window.decorView
                    assertTrue("The app content must be laid out on $name",
                        root.width >= 240 && root.height >= 240)
                    val toolbar = find(root) {
                        it.contentDescription?.toString() == "File actions"
                    }
                    if (name == "files") {
                        assertNotNull("Files actions are required", toolbar)
                        assertTrue("Files actions are clipped/off-screen",
                            toolbar!!.isShown && toolbar.width > 0 && toolbar.height >= 48)
                        val actionBar = toolbar as ViewGroup
                        assertEquals("Five live actions expected",
                            5, actionBar.childCount)
                        for (i in 0 until actionBar.childCount) {
                            val action = actionBar.getChildAt(i)
                            val visible = Rect()
                            assertTrue("File action $i clipped from narrow viewport",
                                action.getGlobalVisibleRect(visible) &&
                                    visible.width() >= 40 && visible.height() >= 40)
                        }
                    }
                    val bitmap = Bitmap.createBitmap(
                        root.width, root.height, Bitmap.Config.ARGB_8888)
                    try {
                        bitmap.eraseColor(android.graphics.Color.rgb(7, 14, 26))
                        root.draw(Canvas(bitmap))
                        val file = File(output, "$name.png")
                        file.outputStream().use {
                            assertTrue("Cannot encode real responsive view $name",
                                bitmap.compress(Bitmap.CompressFormat.PNG, 100, it))
                        }
                        assertTrue("Empty real screenshot $name", file.length() > 3000L)
                    } finally {
                        bitmap.recycle()
                    }
                }
            }
        }
    }

    private fun find(root: View, predicate: (View) -> Boolean): View? {
        if (predicate(root)) return root
        if (root is ViewGroup) {
            for (i in 0 until root.childCount) {
                val match = find(root.getChildAt(i), predicate)
                if (match != null) return match
            }
        }
        return null
    }
}
