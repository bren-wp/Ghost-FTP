package com.ghostftp.android

import androidx.test.core.app.ActivityScenario
import androidx.test.espresso.Espresso.onView
import androidx.test.espresso.action.ViewActions.click
import androidx.test.espresso.action.ViewActions.scrollTo
import androidx.test.espresso.assertion.ViewAssertions.matches
import androidx.test.espresso.matcher.ViewMatchers.isDisplayed
import androidx.test.espresso.matcher.ViewMatchers.withContentDescription
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
    fun primaryWorkspaceAndToolbarRender() {
        onView(withText("Ghost FTP")).check(matches(isDisplayed()))
        onView(withContentDescription("Open Files workspace")).check(matches(isDisplayed()))
        for (label in listOf("Refresh", "Upload", "Download", "New Folder", "Delete")) {
            onView(withContentDescription("$label action")).check(matches(isDisplayed()))
        }
    }

    @Test
    fun workspaceNavigationWorksClickByClick() {
        onView(withContentDescription("Open Sites workspace")).perform(scrollTo(), click())
        onView(withText("Protocol")).perform(scrollTo()).check(matches(isDisplayed()))

        onView(withContentDescription("Open Transfers workspace")).perform(scrollTo(), click())
        onView(withText("No transfer started.")).perform(scrollTo()).check(matches(isDisplayed()))

        onView(withContentDescription("Open Settings workspace")).perform(scrollTo(), click())
        onView(withText("No required tracking, analytics or telemetry.")).perform(scrollTo()).check(matches(isDisplayed()))

        onView(withContentDescription("Open Help & About workspace")).perform(scrollTo(), click())
        onView(withText("Ghost FTP by Brendigo")).perform(scrollTo()).check(matches(isDisplayed()))

        onView(withContentDescription("Open Files workspace")).perform(scrollTo(), click())
        onView(withText("Connect to a server to load remote files.")).perform(scrollTo()).check(matches(isDisplayed()))
    }

    @Test
    fun connectionValidationAndIdleRecoveryWorkClickByClick() {
        onView(withText("Connect")).perform(scrollTo(), click())
        onView(withText("Host is required")).check(matches(isDisplayed()))

        onView(withText("Disconnect")).perform(scrollTo(), click())
        onView(withText("Ready")).check(matches(isDisplayed()))

        onView(withContentDescription("Refresh action")).perform(scrollTo(), click())
        onView(withText("Ready")).check(matches(isDisplayed()))
    }

    @Test
    fun guardedFileActionsRequireActiveSession() {
        for (label in listOf("Download", "New Folder", "Delete")) {
            onView(withContentDescription("$label action")).perform(scrollTo(), click())
            onView(withText("Connect first")).check(matches(isDisplayed()))
            onView(withText("Disconnect")).perform(scrollTo(), click())
            onView(withText("Ready")).check(matches(isDisplayed()))
        }
    }
}
