<?php
declare(strict_types=1);

require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Remote/BoundedDownloadInterface.php';
require __DIR__ . '/../app/Remote/TransferLimiter.php';
require __DIR__ . '/../app/Remote/FtpClient.php';

use ByFTP\Remote\FtpClient;

$client = new FtpClient([]);
$parse = new \ReflectionMethod(FtpClient::class, 'parseRawList');

$items = $parse->invoke($client, [
    '-rw-r--r-- 1 user group 12 Sep 4 12:00 report -> archive.txt',
    'lrwxrwxrwx 1 user group 6 Sep 4 12:00 current -> target',
    'drwxr-xr-x 2 user group 4096 Sep 4 12:00 public_html',
]);

$byName = [];
foreach ($items as $item) {
    if (is_array($item) && isset($item['name'])) {
        $byName[(string)$item['name']] = $item;
    }
}

$failures = [];
if (!isset($byName['report -> archive.txt'])) {
    $failures[] = 'regular filename containing " -> " was truncated';
}
if (isset($byName['report'])) {
    $failures[] = 'truncated alias for a regular filename was emitted';
}
if (!isset($byName['current'])) {
    $failures[] = 'Unix symlink display name was not normalized';
}
if (!isset($byName['public_html']) || ($byName['public_html']['type'] ?? null) !== 'dir') {
    $failures[] = 'directory parsing regressed';
}

if ($failures !== []) {
    foreach ($failures as $failure) {
        fwrite(STDERR, "FTP_LISTING_TEST_FAILED: {$failure}\n");
    }
    exit(1);
}

fwrite(STDOUT, "WEB_FTP_LISTING_TEST=PASS\n");
