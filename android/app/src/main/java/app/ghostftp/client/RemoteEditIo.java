package app.ghostftp.client;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.OutputStream;

final class RemoteEditIo {
    private RemoteEditIo() {
    }

    static RemoteTextDocument.Snapshot open(FtpSession session, String remotePath) throws IOException {
        return RemoteTextDocument.decode(downloadBounded(session, remotePath));
    }

    static RemoteTextDocument.Snapshot reload(FtpSession session, String remotePath) throws IOException {
        return open(session, remotePath);
    }

    static RemoteTextDocument.Snapshot save(
            FtpSession session,
            String remotePath,
            String baselineSha256,
            RemoteTextDocument.LineEnding lineEnding,
            String originalMode,
            String editorText) throws IOException {
        if (session == null || !session.isConnected()) {
            throw new IOException("Remote Edit requires the active server connection.");
        }
        byte[] latest = downloadBounded(session, remotePath);
        RemoteTextDocument.requireUnchanged(latest, baselineSha256);
        session.requireRemoteModeUnchanged(remotePath, originalMode);

        byte[] replacement = RemoteTextDocument.encodeForSave(editorText, lineEnding);
        TransferCommitGate uploadGate = new TransferCommitGate();
        String mode = originalMode == null ? "" : originalMode.trim();
        if (mode.isEmpty()) {
            session.upload(remotePath, new ByteArrayInputStream(replacement), uploadGate);
        } else {
            session.uploadPreservingMode(remotePath, new ByteArrayInputStream(replacement), uploadGate, mode);
        }

        byte[] readBack = downloadBounded(session, remotePath);
        String expected = RemoteTextDocument.sha256(replacement);
        String actual = RemoteTextDocument.sha256(readBack);
        if (!expected.equals(actual)) {
            throw new IOException("Remote Edit save could not be verified by read-back; reconnect and inspect the remote file before retrying.");
        }
        session.requireRemoteModeUnchanged(remotePath, mode);
        return RemoteTextDocument.decode(readBack);
    }

    static byte[] downloadBounded(FtpSession session, String remotePath) throws IOException {
        if (session == null || !session.isConnected()) {
            throw new IOException("Remote Edit requires the active server connection.");
        }
        BoundedOutputStream out = new BoundedOutputStream(WorkspaceOps.MAX_REMOTE_EDIT_BYTES);
        TransferCommitGate gate = new TransferCommitGate();
        session.download(remotePath, out, gate);
        if (!gate.beginCommit()) {
            session.closeCancelledTransferSession();
            throw new IOException("Remote Edit read was cancelled before verification.");
        }
        gate.finish();
        return out.toByteArray();
    }

    private static final class BoundedOutputStream extends OutputStream {
        private final int maxBytes;
        private final ByteArrayOutputStream delegate = new ByteArrayOutputStream();

        BoundedOutputStream(int maxBytes) {
            this.maxBytes = maxBytes;
        }

        @Override
        public void write(int value) throws IOException {
            ensureCapacity(1);
            delegate.write(value);
        }

        @Override
        public void write(byte[] buffer, int offset, int length) throws IOException {
            if (buffer == null) {
                throw new NullPointerException("buffer");
            }
            if (offset < 0 || length < 0 || offset > buffer.length - length) {
                throw new IndexOutOfBoundsException();
            }
            ensureCapacity(length);
            delegate.write(buffer, offset, length);
        }

        byte[] toByteArray() {
            return delegate.toByteArray();
        }

        private void ensureCapacity(int additional) throws IOException {
            if (additional > maxBytes - delegate.size()) {
                throw new IOException("Remote Edit file exceeded the 1 MiB safety limit while downloading.");
            }
        }
    }
}
