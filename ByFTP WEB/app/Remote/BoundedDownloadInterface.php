<?php
declare(strict_types=1);

namespace ByFTP\Remote;

interface BoundedDownloadInterface
{
    public function downloadBounded(string $remotePath, string $localFile, ?int $maxBytes = null): int;
}
