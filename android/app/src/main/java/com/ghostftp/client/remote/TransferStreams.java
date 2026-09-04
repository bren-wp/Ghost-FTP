package com.byftp.client.remote;

import java.io.FilterInputStream;
import java.io.FilterOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.Objects;

/**
 * Counts bytes at the stream boundary without changing the underlying transfer protocol.
 * The listener receives the cumulative byte count after successful reads/writes only.
 */
public final class TransferStreams {
    private TransferStreams() {}

    @FunctionalInterface
    public interface Listener {
        void onBytesTransferred(long bytes);
    }

    public static InputStream monitor(InputStream source, Listener listener) {
        return new ProgressInputStream(
            Objects.requireNonNull(source, "source"),
            Objects.requireNonNull(listener, "listener")
        );
    }

    public static OutputStream monitor(OutputStream destination, Listener listener) {
        return new ProgressOutputStream(
            Objects.requireNonNull(destination, "destination"),
            Objects.requireNonNull(listener, "listener")
        );
    }

    private static final class ProgressInputStream extends FilterInputStream {
        private final Listener listener;
        private long transferred;

        private ProgressInputStream(InputStream source, Listener listener) {
            super(source);
            this.listener = listener;
        }

        @Override public int read() throws IOException {
            int value = in.read();
            if (value >= 0) report(1);
            return value;
        }

        @Override public int read(byte[] buffer, int offset, int length) throws IOException {
            int read = in.read(buffer, offset, length);
            if (read > 0) report(read);
            return read;
        }

        private void report(int bytes) {
            transferred += bytes;
            listener.onBytesTransferred(transferred);
        }
    }

    private static final class ProgressOutputStream extends FilterOutputStream {
        private final Listener listener;
        private long transferred;

        private ProgressOutputStream(OutputStream destination, Listener listener) {
            super(destination);
            this.listener = listener;
        }

        @Override public void write(int value) throws IOException {
            out.write(value);
            report(1);
        }

        @Override public void write(byte[] buffer, int offset, int length) throws IOException {
            out.write(buffer, offset, length);
            if (length > 0) report(length);
        }

        private void report(int bytes) {
            transferred += bytes;
            listener.onBytesTransferred(transferred);
        }
    }
}
