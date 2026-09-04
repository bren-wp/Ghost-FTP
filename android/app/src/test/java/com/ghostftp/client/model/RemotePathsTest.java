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

    @Test public void rejectsWhitespaceAndProtocolControlCharacters() {
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName(" leading.txt"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("trailing.txt "));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("line\nbreak.txt"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("line\rbreak.txt"));
        assertThrows(IllegalArgumentException.class, () -> RemotePaths.validateName("nul\0name.txt"));
        RemotePaths.validateName("normal file.txt");
    }
}
