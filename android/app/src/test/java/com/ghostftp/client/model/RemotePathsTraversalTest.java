package com.GhostFTP.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class RemotePathsTraversalTest {
    @Test public void childRejectsTraversalAndNoncanonicalNames() {
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", "../secret"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", "."));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", " name"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", "name "));
    }

    @Test public void directoryRejectsTraversalAndSeparatorRewrites() {
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.normalizeDirectory("/public_html/../etc"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.normalizeDirectory("/public_html/./assets"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.normalizeDirectory("/public_html//assets"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.normalizeDirectory("/public_html\\assets"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.normalizeDirectory("public_html"));
    }

    @Test public void parentNeverEscapesRoot() {
        assertEquals("/", RemotePaths.parent("/"));
        assertEquals("/", RemotePaths.parent("/one"));
        assertEquals("/one", RemotePaths.parent("/one/two/"));
    }
}
