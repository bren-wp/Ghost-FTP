package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidResponsiveLayoutTest {
    @Test fun phoneRemainsPhoneAfterLandscapeRotation() {
        assertTrue(AndroidResponsiveLayout.usesPhoneNavigation(320))
        assertTrue(AndroidResponsiveLayout.usesPhoneNavigation(393))
        assertTrue(AndroidResponsiveLayout.usesPhoneNavigation(599))
        assertFalse(AndroidResponsiveLayout.usesPhoneNavigation(600))
        assertFalse(AndroidResponsiveLayout.usesPhoneNavigation(840))
    }

    @Test fun narrowDevicesGetSinglePaneWithFullWidthFiles() {
        for (width in listOf(240, 320, 360, 393, 411)) {
            assertFalse(AndroidResponsiveLayout.canShowDualFilePanes(width))
        }
        for (width in listOf(420, 480, 600, 800, 960)) {
            assertTrue(AndroidResponsiveLayout.canShowDualFilePanes(width))
        }
    }

    @Test fun navigationDrawerAlwaysLeavesTapSpaceForClosing() {
        for (width in listOf(240, 280, 320, 360, 393, 480, 599, 800)) {
            val drawer = AndroidResponsiveLayout.drawerWidthDp(width)
            assertTrue(drawer <= width)
            assertTrue(drawer <= 280)
            assertTrue(drawer > 0)
            assertTrue(width - drawer >= 48)
        }
    }

    @Test fun shortScreensAndLargeFontsAvoidFixedClippingAssumptions() {
        assertEquals(220, AndroidResponsiveLayout.filePaneMinHeightDp(320))
        assertEquals(220, AndroidResponsiveLayout.filePaneMinHeightDp(400))
        assertEquals(720, AndroidResponsiveLayout.filePaneMinHeightDp(1000))
        assertTrue(AndroidResponsiveLayout.compactFileActions(320, 1f))
        assertTrue(AndroidResponsiveLayout.compactFileActions(420, 1.5f))
        assertFalse(AndroidResponsiveLayout.compactFileActions(480, 1f))
        assertEquals(60, AndroidResponsiveLayout.navigationItemHeightDp(1.5f, false))
        assertEquals(64, AndroidResponsiveLayout.navigationItemHeightDp(1.5f, true))
    }
}
