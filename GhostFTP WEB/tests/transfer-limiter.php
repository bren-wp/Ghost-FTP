<?php
declare(strict_types=1);

require __DIR__ . '/../app/Remote/TransferLimiter.php';

use ByFTP\Remote\TransferLimiter;

$failures = [];

if (TransferLimiter::effectiveLimit(null, null, -1) !== TransferLimiter::UNKNOWN_SIZE_MAX_BYTES) {
    $failures[] = 'unknown-size transfer did not receive the hard fallback limit';
}
if (TransferLimiter::effectiveLimit(null, 0, -1) !== 0) {
    $failures[] = 'known zero-byte listing snapshot was not preserved';
}
if (TransferLimiter::effectiveLimit(null, null, 0) !== 0) {
    $failures[] = 'known zero-byte source size was not preserved';
}
if (TransferLimiter::effectiveLimit(100, 80, 120) !== 80) {
    $failures[] = 'effective transfer limit did not choose the strictest known bound';
}
if (TransferLimiter::effectiveLimit(100, 120, 80) !== 80) {
    $failures[] = 'fresh source size did not tighten the requested transfer limit';
}

$destinationProbe = tempnam(sys_get_temp_dir(), 'byftp-limit-');
if ($destinationProbe === false) {
    fwrite(STDERR, "Unable to create destination capacity fixture.\n");
    exit(1);
}
try {
    $destinationLimit = TransferLimiter::limitForDestination($destinationProbe, 1024, 0);
    if ($destinationLimit < 0 || $destinationLimit > 1024) {
        $failures[] = 'destination capacity guard returned an invalid limit';
    }
} finally {
    @unlink($destinationProbe);
}

$input = fopen('php://temp', 'w+b');
$output = fopen('php://temp', 'w+b');
if (!is_resource($input) || !is_resource($output)) {
    fwrite(STDERR, "Unable to create transfer limiter streams.\n");
    exit(1);
}

try {
    fwrite($input, '0123456789');
    rewind($input);
    $copied = TransferLimiter::copy($input, $output, 10);
    if ($copied !== 10 || TransferLimiter::streamSize($output) !== 10) {
        $failures[] = 'exact-limit transfer did not complete exactly';
    }
} finally {
    fclose($input);
    fclose($output);
}

$input = fopen('php://temp', 'w+b');
$output = fopen('php://temp', 'w+b');
if (!is_resource($input) || !is_resource($output)) {
    fwrite(STDERR, "Unable to create oversized transfer streams.\n");
    exit(1);
}

try {
    fwrite($input, '0123456789X-more-data-must-not-be-copied');
    rewind($input);
    try {
        TransferLimiter::copy($input, $output, 10);
        $failures[] = 'oversized transfer was accepted';
    } catch (\RuntimeException $e) {
        if (!str_contains($e->getMessage(), 'dopuštenu veličinu')) {
            $failures[] = 'oversized transfer failed for an unexpected reason';
        }
    }
    if (TransferLimiter::streamSize($output) !== 11) {
        $failures[] = 'oversized probe copied more than maxBytes + 1';
    }
} finally {
    fclose($input);
    fclose($output);
}

try {
    TransferLimiter::normalizeLimit(-1);
    $failures[] = 'negative transfer limit was accepted';
} catch (\RuntimeException) {
}

$stream = fopen('php://temp', 'w+b');
if (!is_resource($stream)) {
    fwrite(STDERR, "Unable to create stream-size fixture.\n");
    exit(1);
}
try {
    fwrite($stream, '123456');
    try {
        TransferLimiter::assertWithinLimit($stream, 5);
        $failures[] = 'stream-size guard accepted an oversized destination';
    } catch (\RuntimeException) {
    }
} finally {
    fclose($stream);
}

if ($failures !== []) {
    foreach ($failures as $failure) {
        fwrite(STDERR, "TRANSFER_LIMITER_TEST_FAILED: {$failure}\n");
    }
    exit(1);
}

fwrite(STDOUT, "WEB_TRANSFER_LIMITER_TEST=PASS\n");
