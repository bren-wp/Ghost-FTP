package com.ghostftp.android

import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import androidx.test.core.app.ActivityScenario
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Uses an in-memory, disconnected editor fixture. Never opens a host, stores
 * secrets or captures a screenshot. Exercises the real Activity navigation
 * path, including the inline discard panel and its cancel behavior.
 */
@RunWith(AndroidJUnit4::class)
class EditorDraftNavigationTest {
    @Test
    fun navigatingAwayMustNotDropDraftOrSelectAnotherWorkspaceBeforeConsent() {
        ActivityScenario.launch(MainActivity::class.java).use { scenario ->
            scenario.onActivity { activity ->
                fun getField(name: String) =
                    MainActivity::class.java.getDeclaredField(name).apply {
                        isAccessible = true
                    }

                val editor = getField("editorContentInput").get(activity) as EditText
                editor.setText("unsaved-qa-text")
                getField("editorOriginalText").set(activity, "original-qa-text")
                getField("editorRemotePath").set(activity, "/qa/unsaved.txt")
                getField("editorMode").setBoolean(activity, true)

                val workspaceField = getField("activeWorkspace")
                val initialWorkspace = workspaceField.get(activity) as Enum<*>
                assertEquals("FILES", initialWorkspace.name)

                val workspaceEnum = initialWorkspace.javaClass
                val settings = workspaceEnum.enumConstants
                    .first { (it as Enum<*>).name == "SETTINGS" }
                val navigate = MainActivity::class.java.getDeclaredMethod(
                    "setWorkspace", workspaceEnum, java.lang.Boolean.TYPE
                ).apply { isAccessible = true }

                navigate.invoke(activity, settings, true)
                assertEquals("Do not select Settings without consent",
                    initialWorkspace, workspaceField.get(activity))
                assertTrue("Editor mode must remain active",
                    getField("editorMode").getBoolean(activity))
                assertEquals("Keep all unsaved bytes intact",
                    "unsaved-qa-text", editor.text.toString())

                val confirmation = getField("confirmationPanel").get(activity) as View
                assertEquals("User must see discard confirmation",
                    View.VISIBLE, confirmation.visibility)

                // Tap the REAL Cancel control, not a mocked callback.
                val cancel = findByDescription(
                    confirmation, activity.getString(R.string.action_cancel))
                assertTrue("Discard panel must have a functional Cancel control",
                    cancel != null && cancel.performClick())
                assertEquals(View.GONE, confirmation.visibility)
                assertEquals(initialWorkspace, workspaceField.get(activity))
                assertEquals("unsaved-qa-text", editor.text.toString())
                assertTrue(getField("editorMode").getBoolean(activity))
            }
        }
    }
    private fun findByDescription(view: View, description: String): View? {
        if (view.contentDescription?.toString() == description) return view
        if (view is ViewGroup) {
            for (index in 0 until view.childCount) {
                val found = findByDescription(view.getChildAt(index), description)
                if (found != null) return found
            }
        }
        return null
    }
}
