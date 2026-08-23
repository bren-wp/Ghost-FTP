package com.byftp.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class RemotePathsTest {
    @Test public void joinsAndParentsWithoutTraversalComponents() {
        assertEquals("/public_html/index.html", RemotePaths.child("/public_html", "index.html"));
        assertEquals("/public_html", RemotePaths.parent("/public_html/assets"));
        assertEquals("/", RemotePaths.parent("/public_html"));
    }

    @Test public void rejectsNestedOrDotNames() {
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("../secret"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("a/b"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName(".."));
    }
}
