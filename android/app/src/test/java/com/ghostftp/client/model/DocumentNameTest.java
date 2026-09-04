package com.GhostFTP.client.model;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

public final class DocumentNameTest {
    @Test public void keepsProviderNameWhenPresent() {
        assertEquals("photo.jpg", DocumentName.resolve("photo.jpg", "ignored.bin"));
    }

    @Test public void fallsBackToUriTailForNullOrBlankProviderName() {
        assertEquals("photo.jpg", DocumentName.resolve(null, "document/42/photo.jpg"));
        assertEquals("photo.jpg", DocumentName.resolve("   ", "document/42/photo.jpg"));
    }

    @Test public void usesDeterministicFallbackWhenMetadataIsMissing() {
        assertEquals("upload.bin", DocumentName.resolve(null, null));
        assertEquals("upload.bin", DocumentName.resolve("", "folder/"));
    }
}
