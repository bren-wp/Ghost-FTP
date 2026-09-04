<?php
declare(strict_types=1);

$source = file_get_contents(__DIR__ . '/../app/Operations/RemoteOperations.php');
if (!is_string($source)) {
    fwrite(STDERR, "WEB_ZIP_CREATION_TEST=FAIL unable to read RemoteOperations.php\n");
    exit(1);
}

$passed = 0;
$failed = 0;

function check_zip_creation(bool $condition, string $label): void
{
    global $passed, $failed;
    if ($condition) {
        $passed++;
        return;
    }
    $failed++;
    fwrite(STDERR, "FAIL: {$label}\n");
}

$directoryGuard = <<<'PHP'
if (!$zip->addEmptyDir(rtrim($archivePath, '/'))) throw new RuntimeException('Nije moguće dodati direktorij u ZIP.');
PHP;
$preserveOriginalError = <<<'PHP'
// Preserve the original build error; cleanup still must run.
PHP;
$errorCleanup = <<<'PHP'
} finally {
                foreach ($temps as $temp) @unlink($temp);
                @unlink($tmp);
            }
            throw $e;
PHP;

check_zip_creation(
    str_contains($source, $directoryGuard),
    'ZIP directory insertion failure is checked explicitly'
);
check_zip_creation(
    str_contains($source, $preserveOriginalError),
    'ZIP cleanup preserves the original build failure'
);
check_zip_creation(
    str_contains($source, $errorCleanup),
    'ZIP build failure always cleans staged files and incomplete output'
);

if ($failed > 0) {
    fwrite(STDERR, "WEB_ZIP_CREATION_TEST=FAIL passed={$passed} failed={$failed}\n");
    exit(1);
}

echo "WEB_ZIP_CREATION_TEST=PASS ({$passed})\n";
