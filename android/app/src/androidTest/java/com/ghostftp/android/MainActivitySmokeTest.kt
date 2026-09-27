package com.ghostftp.android

import androidx.test.core.app.ActivityScenario
import androidx.test.espresso.Espresso.onView
import androidx.test.espresso.action.ViewActions.click
import androidx.test.espresso.assertion.ViewAssertions.matches
import androidx.test.espresso.matcher.ViewMatchers.isDisplayed
import androidx.test.espresso.matcher.ViewMatchers.withText
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.After
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class MainActivitySmokeTest {
    private lateinit var scenario: ActivityScenario<MainActivity>

    @Before
    fun launch() {
        scenario = ActivityScenario.launch(MainActivity::class.java)
    }

    @After
    fun close() {
        scenario.close()
    }

    @Test
    fun primaryWorkspacesAndActionsRender() {
        for (label in listOf(
            "Ghost FTP",
            "Files",
            "Sites",
            "Transfers",
            "Connect",
            "Disconnect",
            "Refresh",
            "Upload",
            "Download",
            "New Folder",
            "Delete"
        )) {
            onView(withText(label)).check(matches(isDisplayed()))
        }
    }

    @Test
    fun connectionValidationAndIdleRecoveryWorkClickByClick() {
        onView(withText("Connect")).perform(click())
        onView(withText("Host is required")).check(matches(isDisplayed()))

        onView(withText("Disconnect")).perform(click())
        onView(withText("Ready")).check(matches(isDisplayed()))

        onView(withText("Refresh")).perform(click())
        onView(withText("Ready")).check(matches(isDisplayed()))
    }

    @Test
    fun guardedFileActionsRequireActiveSession() {
        for (label in listOf("Download", "New Folder", "Delete")) {
            onView(withText(label)).perform(click())
            onView(withText("Connect first")).check(matches(isDisplayed()))
            onView(withText("Disconnect")).perform(click())
            onView(withText("Ready")).check(matches(isDisplayed()))
        }
    }
}
