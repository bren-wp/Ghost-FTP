<?php
declare(strict_types=1);

require __DIR__ . '/../app/helpers.php';

$passed = 0;
$failed = 0;

function check(bool $condition, string $label): void
{
    global $passed, $failed;
    if ($condition) {
        $passed++;
        return;
    }
    $failed++;
    fwrite(STDERR, "FAIL: {$label}\n");
}

$long = GhostFTP_archive_download_name(str_repeat('a', 200) . '.zip');
check(strlen($long) === 120, 'long archive name is capped at 120 ASCII bytes');
check(str_ends_with($long, '.zip'), 'long archive name preserves zip extension after truncation');
check(substr($long, 0, -4) === str_repeat('a', 116), 'long archive name truncates the base before appending extension');

check(
    GhostFTP_archive_download_name('backup.ZIP') === 'backup.zip',
    'existing ZIP extension is normalized without duplication'
);
check(
    GhostFTP_archive_download_name('backup') === 'backup.zip',
    'missing ZIP extension is appended'
);
check(
    GhostFTP_archive_download_name('.zip') === 'ghost-ftp-download.zip',
    'empty archive basename falls back to a stable filename'
);
check(
    GhostFTP_archive_download_name(" report\r\n2026?.zip ") === 'report-2026-.zip',
    'unsafe archive filename characters are sanitized before header use'
);
check(
    str_ends_with(GhostFTP_archive_download_name(str_repeat('č', 200)), '.zip'),
    'multibyte long archive name still preserves zip extension'
);

if ($failed > 0) {
    fwrite(STDERR, "WEB_ARCHIVE_DOWNLOAD_NAME_TEST=FAIL passed={$passed} failed={$failed}\n");
    exit(1);
}

echo "WEB_ARCHIVE_DOWNLOAD_NAME_TEST=PASS ({$passed})\n";
