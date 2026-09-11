package app.ghostftp.client;

/**
 * Serializes transfer cancellation against the irreversible final-name commit.
 *
 * <p>A cancel request may win while bytes are still moving or while a staged
 * object/document is ready but not yet committed. Once {@link #beginCommit()}
 * succeeds, the final-name operation owns the transfer and a later cancel
 * request is deliberately rejected instead of pretending that an irreversible
 * commit can be rolled back safely.</p>
 */
final class TransferCommitGate {
    enum CancelDisposition {
        INTERRUPT_IO,
        CLEANUP_STAGING,
        TOO_LATE,
        ALREADY_CANCELLED
    }

    private enum Phase {
        TRANSFERRING,
        READY_TO_COMMIT,
        COMMITTING,
        CANCELLED,
        FINISHED
    }

    private Phase phase = Phase.TRANSFERRING;

    synchronized CancelDisposition requestCancel() {
        switch (phase) {
            case TRANSFERRING:
                phase = Phase.CANCELLED;
                return CancelDisposition.INTERRUPT_IO;
            case READY_TO_COMMIT:
                phase = Phase.CANCELLED;
                return CancelDisposition.CLEANUP_STAGING;
            case COMMITTING:
            case FINISHED:
                return CancelDisposition.TOO_LATE;
            case CANCELLED:
            default:
                return CancelDisposition.ALREADY_CANCELLED;
        }
    }

    synchronized boolean markReadyToCommit() {
        if (phase == Phase.CANCELLED) {
            return false;
        }
        if (phase != Phase.TRANSFERRING) {
            throw new IllegalStateException("Transfer is not in the data phase.");
        }
        phase = Phase.READY_TO_COMMIT;
        return true;
    }

    synchronized boolean beginCommit() {
        if (phase == Phase.CANCELLED) {
            return false;
        }
        if (phase != Phase.READY_TO_COMMIT) {
            throw new IllegalStateException("Transfer is not ready to commit.");
        }
        phase = Phase.COMMITTING;
        return true;
    }

    synchronized boolean isCancelled() {
        return phase == Phase.CANCELLED;
    }

    synchronized boolean isCommitting() {
        return phase == Phase.COMMITTING;
    }

    synchronized void finish() {
        if (phase != Phase.COMMITTING) {
            throw new IllegalStateException("Only a committing transfer can finish.");
        }
        phase = Phase.FINISHED;
    }
}
