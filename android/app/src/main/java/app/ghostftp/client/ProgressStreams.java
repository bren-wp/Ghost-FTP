package app.ghostftp.client;

import java.io.FilterInputStream;
import java.io.FilterOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

final class ProgressStreams {
    interface Listener {
        void onTransferred(long bytes);
    }

    private ProgressStreams() {
    }

    static InputStream input(InputStream source, Listener listener) {
        return new CountingInputStream(source, listener);
    }

    static OutputStream output(OutputStream destination, Listener listener) {
        return new CountingOutputStream(destination, listener);
    }

    private static final class CountingInputStream extends FilterInputStream {
        private final Listener listener;
        private long transferred;

        CountingInputStream(InputStream source, Listener listener) {
            super(source);
            this.listener = listener;
        }

        @Override
        public int read() throws IOException {
            int value = super.read();
            if (value != -1) publish(1L);
            return value;
        }

        @Override
        public int read(byte[] buffer, int offset, int length) throws IOException {
            int read = super.read(buffer, offset, length);
            if (read > 0) publish(read);
            return read;
        }

        private void publish(long bytes) {
            transferred += bytes;
            if (listener != null) listener.onTransferred(transferred);
        }
    }

    private static final class CountingOutputStream extends FilterOutputStream {
        private final Listener listener;
        private long transferred;

        CountingOutputStream(OutputStream destination, Listener listener) {
            super(destination);
            this.listener = listener;
        }

        @Override
        public void write(int value) throws IOException {
            out.write(value);
            publish(1L);
        }

        @Override
        public void write(byte[] buffer, int offset, int length) throws IOException {
            out.write(buffer, offset, length);
            if (length > 0) publish(length);
        }

        private void publish(long bytes) {
            transferred += bytes;
            if (listener != null) listener.onTransferred(transferred);
        }
    }
}
