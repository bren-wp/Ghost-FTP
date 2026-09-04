<?php
declare(strict_types=1);

require __DIR__ . '/../app/Storage/JsonStore.php';

use ByFTP\Storage\JsonStore;
use RuntimeException;

$failures = [];
$root = sys_get_temp_dir() . '/byftp-json-bounds-' . bin2hex(random_bytes(6));
if (!mkdir($root, 0700, true) && !is_dir($root)) {
    fwrite(STDERR, "Unable to create JSON bounds test directory.\n");
    exit(1);
}

try {
    $smallPath = $root . '/small.json';
    $small = new JsonStore($smallPath, false);
    $small->write(['ok' => true, 'value' => 'bounded']);
    $decoded = $small->read();
    if (($decoded['ok'] ?? null) !== true || ($decoded['value'] ?? null) !== 'bounded') {
        $failures[] = 'small JSON state did not round-trip';
    }

    $oversizePath = $root . '/oversize.json';
    $handle = fopen($oversizePath, 'xb');
    if (!is_resource($handle)) {
        throw new RuntimeException('Unable to create oversized JSON state fixture.');
    }
    try {
        if (!ftruncate($handle, (8 * 1024 * 1024) + 1)) {
            throw new RuntimeException('Unable to size oversized JSON state fixture.');
        }
    } finally {
        fclose($handle);
    }

    try {
        (new JsonStore($oversizePath, false))->read();
        $failures[] = 'oversized JSON state was accepted';
    } catch (RuntimeException $e) {
        if (!str_contains($e->getMessage(), 'prevelika')) {
            $failures[] = 'oversized JSON state failed for an unexpected reason';
        }
    }

    try {
        $small->write(['payload' => str_repeat('x', 8 * 1024 * 1024)]);
        $failures[] = 'oversized JSON write was accepted';
    } catch (RuntimeException $e) {
        if (!str_contains($e->getMessage(), 'preveliki')) {
            $failures[] = 'oversized JSON write failed for an unexpected reason';
        }
    }
} finally {
    foreach (glob($root . '/*') ?: [] as $path) {
        @unlink($path);
    }
    @rmdir($root);
}

if ($failures !== []) {
    foreach ($failures as $failure) {
        fwrite(STDERR, "JSON_STORE_BOUNDS_TEST_FAILED: {$failure}\n");
    }
    exit(1);
}

fwrite(STDOUT, "WEB_JSON_STORE_BOUNDS_TEST=PASS\n");
