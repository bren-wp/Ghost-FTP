<?php
declare(strict_types=1);

$failed = false;

function metadata_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

$root = dirname(__DIR__, 2);
$web = dirname(__DIR__);
$version = trim((string)file_get_contents($root . '/VERSION'));
$webVersion = trim((string)file_get_contents($web . '/VERSION'));
$composerRaw = (string)file_get_contents($web . '/composer.json');
$composer = json_decode($composerRaw, true, 512, JSON_THROW_ON_ERROR);
$serviceWorker = (string)file_get_contents($web . '/service-worker.js');

metadata_check((bool)preg_match('/^\d+\.\d+\.\d+$/', $version), 'canonical VERSION is semantic');
metadata_check($webVersion === $version, 'web VERSION matches canonical VERSION');
metadata_check(($composer['version'] ?? null) === $version, 'Composer version matches canonical VERSION');

$description = (string)($composer['description'] ?? '');
metadata_check(str_contains($description, 'Ghost FTP'), 'Composer description uses public Ghost FTP brand');
metadata_check(!preg_match('/\b\d+\.\d+\.\d+\b/', $description), 'Composer description does not hard-code a release version');
metadata_check(str_contains($serviceWorker, "ghostftp-static-v{$version}"), 'PWA cache namespace matches canonical VERSION');

if ($failed) {
    fwrite(STDERR, "WEB_VERSION_METADATA_TEST=FAIL\n");
    exit(1);
}

echo "WEB_VERSION_METADATA_TEST=PASS\n";
