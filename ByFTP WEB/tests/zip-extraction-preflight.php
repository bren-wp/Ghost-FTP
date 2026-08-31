<?php
declare(strict_types=1);

if (!class_exists(ZipArchive::class)) {
    // Some cross-platform build runners intentionally do not install the optional PHP
    // ZIP extension. Source-order enforcement still runs everywhere; the real traversal
    // fixture runs on environments where the extraction feature itself is available.
    echo "WEB_ZIP_EXTRACTION_PREFLIGHT_TEST=SKIP_NO_ZIP\n";
    exit(0);
}

$testStorage = sys_get_temp_dir() . '/byftp-zip-preflight-' . bin2hex(random_bytes(6));
mkdir($testStorage . '/tmp', 0700, true);
define('BYFTP_STORAGE', $testStorage);

function byftp_assert_temp_capacity(int $expectedBytes, int $reserveBytes = 16777216): void
{
}

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Operations/RemoteOperations.php';

use ByFTP\Operations\RemoteOperations;
use ByFTP\Remote\RemoteClientInterface;

final class ZipPreflightFakeClient implements RemoteClientInterface
{
    /** @var list<string> */
    private array $mutations = [];

    public function __construct(private readonly string $archivePath)
    {
    }

    public function connect(): void
    {
    }

    public function list(string $path): array
    {
        if ($path === '/') {
            return [[
                'name' => 'archive.zip',
                'type' => 'file',
                'size' => filesize($this->archivePath) ?: 0,
                'modified' => null,
                'permissions' => '',
            ]];
        }
        return [];
    }

    public function makeDirectory(string $path): void
    {
        $this->mutations[] = 'mkdir:' . $path;
    }

    public function rename(string $from, string $to): void
    {
        $this->mutations[] = 'rename:' . $from . '->' . $to;
    }

    public function delete(string $path, bool $directory = false): void
    {
        $this->mutations[] = 'delete:' . $path;
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $this->mutations[] = 'upload:' . $remotePath;
    }

    public function download(string $remotePath, string $localFile): void
    {
        if ($remotePath !== '/archive.zip' || !copy($this->archivePath, $localFile)) {
            throw new RuntimeException('Fake ZIP download failed.');
        }
    }

    public function read(string $remotePath, int $maxBytes = 2097152): string
    {
        throw new RuntimeException('Unexpected read during ZIP extraction preflight test.');
    }

    public function write(string $remotePath, string $content): void
    {
        $this->mutations[] = 'write:' . $remotePath;
    }

    public function chmod(string $path, int $mode): void
    {
        $this->mutations[] = 'chmod:' . $path;
    }

    public function disconnect(): void
    {
    }

    /** @return list<string> */
    public function mutations(): array
    {
        return $this->mutations;
    }
}

$archivePath = $testStorage . '/malicious.zip';
$zip = new ZipArchive();
if ($zip->open($archivePath, ZipArchive::CREATE | ZipArchive::OVERWRITE) !== true) {
    throw new RuntimeException('Unable to create ZIP regression fixture.');
}
if (!$zip->addFromString('safe.txt', 'safe')) {
    throw new RuntimeException('Unable to add safe ZIP regression entry.');
}
if (!$zip->addFromString('../escape.txt', 'escape')) {
    throw new RuntimeException('Unable to add traversal ZIP regression entry.');
}
if (!$zip->close()) {
    throw new RuntimeException('Unable to finalize ZIP regression fixture.');
}

$fixture = new ZipArchive();
if ($fixture->open($archivePath) !== true) {
    throw new RuntimeException('Unable to reopen ZIP regression fixture.');
}
$second = $fixture->statIndex(1);
$fixture->close();
if (!is_array($second) || ($second['name'] ?? '') !== '../escape.txt') {
    throw new RuntimeException('ZIP regression fixture did not preserve the traversal entry.');
}

$client = new ZipPreflightFakeClient($archivePath);
$ops = new RemoteOperations($client);
$threw = false;
try {
    $ops->extractZip('/archive.zip', '/dest');
} catch (Throwable) {
    $threw = true;
}

$failures = [];
if (!$threw) {
    $failures[] = 'unsafe ZIP traversal entry was not rejected';
}
if ($client->mutations() !== []) {
    $failures[] = 'unsafe ZIP is rejected before any remote mutation';
}

foreach (glob($testStorage . '/tmp/*') ?: [] as $path) {
    @unlink($path);
}
@unlink($archivePath);
@rmdir($testStorage . '/tmp');
@rmdir($testStorage);

if ($failures !== []) {
    foreach ($failures as $failure) {
        fwrite(STDERR, 'FAIL: ' . $failure . "\n");
    }
    exit(1);
}

echo "WEB_ZIP_EXTRACTION_PREFLIGHT_TEST=PASS\n";
