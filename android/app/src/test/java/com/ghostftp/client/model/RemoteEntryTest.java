package com.ghostftp.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class RemoteEntryTest {
    @Test public void rejectsUnsafeRemoteNames() {
        assertThrows(IllegalArgumentException.class, () -> new RemoteEntry("../secret", false, 1, 0));
        assertThrows(IllegalArgumentException.class, () -> new RemoteEntry("a/b", false, 1, 0));
    }

    @Test public void formatsDirectoryAndFileLabels() {
        assertTrue(new RemoteEntry("public_html", true, 0, 0).displayLabel().contains("public_html"));
        assertTrue(new RemoteEntry("index.html", false, 2048, 0).displayLabel().contains("2.0 KB"));
    }
}
