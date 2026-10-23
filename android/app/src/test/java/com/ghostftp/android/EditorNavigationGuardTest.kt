package com.ghostftp.android

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EditorNavigationGuardTest {
    @Test fun changedEditorRequiresUserConsentBeforeLeavingFiles() {
        assertTrue(EditorNavigationGuard.needsDiscardConfirmation(true, true, false))
    }

    @Test fun unchangedEditorMayCloseWithoutFalseConfirmation() {
        assertFalse(EditorNavigationGuard.needsDiscardConfirmation(true, false, false))
    }

    @Test fun stayingInFilesDoesNotDiscardEditor() {
        assertFalse(EditorNavigationGuard.needsDiscardConfirmation(true, true, true))
    }

    @Test fun ordinaryNavigationIsUnaffectedWithoutOpenEditor() {
        assertFalse(EditorNavigationGuard.needsDiscardConfirmation(false, true, false))
    }
}
