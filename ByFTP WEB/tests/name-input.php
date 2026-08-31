<?php
declare(strict_types=1);

require __DIR__ . '/../app/Remote/PathGuard.php';

use ByFTP\Remote\PathGuard;

$passed = 0;
$failed = 0;

function check_name_input(bool $condition, string $label): void
{
    global $passed, $failed;
    if ($condition) {
        $passed++;
        return;
    }
    $failed++;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function throws_name_input(callable $fn, string $label): void
{
    try {
        $fn();
        check_name_input(false, $label);
    } catch (Throwable) {
        check_name_input(true, $label);
    }
}

throws_name_input(
    fn() => PathGuard::segment('nested/file.txt'),
    'single-name validator rejects slash-separated input'
);
throws_name_input(
    fn() => PathGuard::segment('../file.txt'),
    'single-name validator rejects traversal-shaped input'
);

$api = file_get_contents(__DIR__ . '/../api.php');
if (!is_string($api)) {
    fwrite(STDERR, "WEB_NAME_INPUT_TEST=FAIL unable to read api.php\n");
    exit(1);
}

$strictName = "PathGuard::segment((string)(\$_POST['name'] ?? ''))";
$legacyName = "PathGuard::basename((string)(\$_POST['name'] ?? ''))";
check_name_input(
    substr_count($api, $strictName) === 3,
    'mkdir, new_file and rename all use strict single-name validation'
);
check_name_input(
    !str_contains($api, $legacyName),
    'API no longer silently canonicalizes explicit name fields with basename'
);

if ($failed > 0) {
    fwrite(STDERR, "WEB_NAME_INPUT_TEST=FAIL passed={$passed} failed={$failed}\n");
    exit(1);
}

echo "WEB_NAME_INPUT_TEST=PASS ({$passed})\n";
