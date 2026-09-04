package com.byftp.client.model;

import static org.junit.Assert.*;
import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.Test;

public class VersionBindingTest {
    @Test public void androidBuildReadsRepositoryVersion() throws Exception {
        Path build = Path.of("build.gradle.kts");
        if (!Files.isRegularFile(build)) {
            build = Path.of("android", "app", "build.gradle.kts");
        }
        String text = Files.readString(build);
        assertTrue(text.contains("rootProject.file(\"../VERSION\")"));
        assertTrue(text.contains("versionName = canonicalVersion"));
        assertTrue(text.contains("versionCode = canonicalVersionCode"));
    }
}
