package com.ghostftp.android

/** Pure navigation rule: user edits must remain intact until discard is confirmed. */
internal object EditorNavigationGuard {
    fun needsDiscardConfirmation(
        editorOpen: Boolean,
        modified: Boolean,
        destinationIsFiles: Boolean
    ): Boolean = editorOpen && modified && !destinationIsFiles
}
