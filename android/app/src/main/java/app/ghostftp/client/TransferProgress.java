package app.ghostftp.client;

import java.util.Locale;

final class TransferProgress {
    private static final long PUBLISH_INTERVAL_NANOS = 250_000_000L;

    interface Sink {
        void publish(String text);
    }

    private final long totalBytes;
    private final String action;
    private final Sink sink;
    private final long startedNanos;
    private long lastPublishNanos;
    private long lastTransferredBytes;

    TransferProgress(long totalBytes, String action, Sink sink) {
        this.totalBytes = totalBytes > 0L ? totalBytes : -1L;
        this.action = action == null || action.trim().isEmpty() ? "Transferring" : action.trim();
        this.sink = sink;
        this.startedNanos = System.nanoTime();
        this.lastPublishNanos = startedNanos;
    }

    synchronized void onTransferred(long transferredBytes) {
        if (sink == null || transferredBytes < 0L || transferredBytes < lastTransferredBytes) {
            return;
        }
        lastTransferredBytes = transferredBytes;
        long now = System.nanoTime();
        boolean exactKnownCompletion = totalBytes > 0L && transferredBytes == totalBytes;
        if (!exactKnownCompletion && now - lastPublishNanos < PUBLISH_INTERVAL_NANOS) {
            return;
        }
        lastPublishNanos = now;
        sink.publish(formatStatus(transferredBytes, now - startedNanos));
    }

    private String formatStatus(long transferredBytes, long elapsedNanos) {
        StringBuilder text = new StringBuilder(action).append(" — ").append(formatBytes(transferredBytes));
        boolean trustworthyTotal = totalBytes > 0L && transferredBytes <= totalBytes;
        if (trustworthyTotal) {
            int percent = (int) Math.floor((double) transferredBytes * 100.0d / (double) totalBytes);
            text.append(" / ").append(formatBytes(totalBytes)).append(" (").append(percent).append("%)");
        }

        if (elapsedNanos > 0L && transferredBytes > 0L) {
            double seconds = elapsedNanos / 1_000_000_000.0d;
            double bytesPerSecond = transferredBytes / seconds;
            if (Double.isFinite(bytesPerSecond) && bytesPerSecond > 0.0d) {
                text.append(" — ").append(formatRate(bytesPerSecond));
                if (trustworthyTotal && transferredBytes < totalBytes) {
                    long remaining = totalBytes - transferredBytes;
                    long etaSeconds = (long) Math.ceil(remaining / bytesPerSecond);
                    if (etaSeconds >= 0L) {
                        text.append(" — ETA ").append(formatDuration(etaSeconds));
                    }
                }
            }
        }
        return text.toString();
    }

    static String formatBytes(long bytes) {
        if (bytes < 1024L) {
            return bytes + " B";
        }
        double kib = bytes / 1024.0d;
        if (kib < 1024.0d) {
            return String.format(Locale.ROOT, "%.1f KiB", kib);
        }
        double mib = kib / 1024.0d;
        if (mib < 1024.0d) {
            return String.format(Locale.ROOT, "%.1f MiB", mib);
        }
        return String.format(Locale.ROOT, "%.2f GiB", mib / 1024.0d);
    }

    private static String formatRate(double bytesPerSecond) {
        if (bytesPerSecond < 1024.0d) {
            return String.format(Locale.ROOT, "%.0f B/s", bytesPerSecond);
        }
        double kib = bytesPerSecond / 1024.0d;
        if (kib < 1024.0d) {
            return String.format(Locale.ROOT, "%.1f KiB/s", kib);
        }
        return String.format(Locale.ROOT, "%.1f MiB/s", kib / 1024.0d);
    }

    private static String formatDuration(long totalSeconds) {
        long hours = totalSeconds / 3600L;
        long minutes = (totalSeconds % 3600L) / 60L;
        long seconds = totalSeconds % 60L;
        if (hours > 0L) {
            return String.format(Locale.ROOT, "%d:%02d:%02d", hours, minutes, seconds);
        }
        return String.format(Locale.ROOT, "%02d:%02d", minutes, seconds);
    }
}
