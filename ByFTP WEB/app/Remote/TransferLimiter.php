<?php
declare(strict_types=1);

namespace ByFTP\Remote;

use RuntimeException;

final class TransferLimiter
{
    public static function normalizeLimit(?int $maxBytes): ?int
    {
        if ($maxBytes !== null && $maxBytes < 0) {
            throw new RuntimeException('Limit prijenosa ne smije biti negativan.');
        }
        return $maxBytes;
    }

    public static function copy(mixed $input, mixed $output, ?int $maxBytes = null): int
    {
        if (!is_resource($input) || !is_resource($output)) {
            throw new RuntimeException('Prijenos nije moguće pokrenuti.');
        }

        $maxBytes = self::normalizeLimit($maxBytes);
        $copied = $maxBytes === null
            ? stream_copy_to_stream($input, $output)
            : stream_copy_to_stream($input, $output, self::probeLength($maxBytes));

        if ($copied === false) {
            throw new RuntimeException('Prijenos nije uspio.');
        }
        if ($maxBytes !== null && $copied > $maxBytes) {
            throw new RuntimeException('Download prelazi dopuštenu veličinu.');
        }
        return $copied;
    }

    public static function streamSize(mixed $stream): int
    {
        if (!is_resource($stream)) {
            return -1;
        }
        @fflush($stream);
        $stat = @fstat($stream);
        return is_array($stat) ? (int)($stat['size'] ?? -1) : -1;
    }

    public static function assertWithinLimit(mixed $stream, ?int $maxBytes): int
    {
        $maxBytes = self::normalizeLimit($maxBytes);
        $size = self::streamSize($stream);
        if ($maxBytes !== null && $size > $maxBytes) {
            throw new RuntimeException('Download prelazi dopuštenu veličinu.');
        }
        return $size;
    }

    private static function probeLength(int $maxBytes): int
    {
        return $maxBytes === PHP_INT_MAX ? PHP_INT_MAX : $maxBytes + 1;
    }
}
