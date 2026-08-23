package com.byftp.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class RemotePathsTraversalTest {
    @Test public void childRejectsTraversalNames() {
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", "../secret"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.child("/public_html", "."));
    }

    @Test public void parentNeverEscapesRoot() {
        assertEquals("/", RemotePaths.parent("/"));
        assertEquals("/", RemotePaths.parent("/one"));
    }
}
