package com.byftp.client.remote;

import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertEquals;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.List;
import org.junit.Test;

public final class TransferStreamsTest {
    @Test public void inputReportsCumulativeBytesWithoutChangingPayload() throws Exception {
        byte[] payload = new byte[]{1, 2, 3, 4, 5, 6};
        List<Long> progress = new ArrayList<>();
        byte[] buffer = new byte[4];

        try (InputStream input = TransferStreams.monitor(new ByteArrayInputStream(payload), progress::add)) {
            assertEquals(4, input.read(buffer, 0, buffer.length));
            assertEquals(5, input.read());
            assertEquals(1, input.read(buffer, 0, buffer.length));
            assertEquals(-1, input.read());
        }

        assertEquals(List.of(4L, 5L, 6L), progress);
    }

    @Test public void outputReportsCumulativeBytesWithoutChangingPayload() throws Exception {
        ByteArrayOutputStream raw = new ByteArrayOutputStream();
        List<Long> progress = new ArrayList<>();

        try (OutputStream output = TransferStreams.monitor(raw, progress::add)) {
            output.write(new byte[]{9, 8, 7}, 0, 3);
            output.write(6);
        }

        assertArrayEquals(new byte[]{9, 8, 7, 6}, raw.toByteArray());
        assertEquals(List.of(3L, 4L), progress);
    }
}
