<?php
declare(strict_types=1);

$remoteOperationsSource = file_get_contents(__DIR__ . '/../app/Operations/RemoteOperations.php');
if (!is_string($remoteOperationsSource)) {
    fwrite(STDERR, "FAIL: unable to inspect RemoteOperations ZIP extraction source.\n");
    exit(1);
}
$extractStart = strpos($remoteOperationsSource, 'public function extractZip(');
$extractEnd = $extractStart !== false ? strpos($remoteOperationsSource, 'private function buildZip(', $extractStart) : false;
if ($extractStart === false || $extractEnd === false || $extractEnd <= $extractStart) {
    fwrite(STDERR, "FAIL: unable to isolate ZIP extraction source.\n");
    exit(1);
}
$extractSource = substr($remoteOperationsSource, $extractStart, $extractEnd - $extractStart);
$requiredTypesPos = strpos($extractSource, '$requiredTypes = $plannedTypes;');
$remoteStatePos = strpos($extractSource, "$remoteState = ['/'=>'dir'];");
$listingCachePos = strpos($extractSource, '$listingCache = [];');
$existingFileConflictPos = strpos($extractSource, 'ZIP odredište blokira postojeća datoteka.');
$existingDirectoryConflictPos = strpos($extractSource, 'ZIP datoteka ne može prepisati postojeći direktorij.');
$executionPos = strpos($extractSource, '// Only a fully validated archive is allowed to mutate remote state.');
if (
    $requiredTypesPos === false
    || $remoteStatePos === false
    || $listingCachePos === false
    || $existingFileConflictPos === false
    || $existingDirectoryConflictPos === false
    || $executionPos === false
    || !($requiredTypesPos < $remoteStatePos
        && $remoteStatePos <= $listingCachePos
        && $listingCachePos < $existingFileConflictPos
        && $existingFileConflictPos < $executionPos
        && $existingDirectoryConflictPos < $executionPos)
) {
    fwrite(STDERR, "FAIL: existing remote ZIP topology is not fully preflighted before execution.\n");
    exit(1);
}

if (!class_exists(ZipArchive::class)) {
    // Some cross-platform build runners intentionally do not install the optional PHP
    // ZIP extension. Source-order enforcement above still runs everywhere; the real
    // archive fixtures run where the extraction feature itself is available.
    echo "WEB_ZIP_EXTRACTION_PREFLIGHT_TEST=SKIP_NO_ZIP\n";
    exit(0);
}

$testStorage = sys_get_temp_dir() . '/GhostFTP-zip-preflight-' . bin2hex(random_bytes(6));
mkdir($testStorage . '/tmp', 0700, true);
define('GhostFTP_STORAGE', $testStorage);

function GhostFTP_assert_temp_capacity(int $expectedBytes, int $reserveBytes = 16777216): void
{
}

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Operations/RemoteOperations.php';

use GhostFTP\Operations\RemoteOperations;
use GhostFTP\Remote\RemoteClientInterface;

final class ZipPreflightFakeClient implements RemoteClientInterface
{
    /** @var list<string> */
    private array $mutations = [];

    /** @param array<string, list<array<string, mixed>>> $listings */
    public function __construct(
        private readonly string $archivePath,
        private readonly array $listings = [],
    ) {
    }

    public function connect(): void
    {
    }

    public function list(string $path): array
    {
        $items = $this->listings[$path] ?? [];
        if ($path === '/') {
            array_unshift($items, [
                'name' => 'archive.zip',
                'type' => 'file',
                'size' => filesize($this->archivePath) ?: 0,
                'modified' => null,
                'permissions' => '',
            ]);
        }
        return $items;
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

$failures = [];

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
if (!$threw) {
    $failures[] = 'unsafe ZIP traversal entry was not rejected';
}
if ($client->mutations() !== []) {
    $failures[] = 'unsafe ZIP is rejected before any remote mutation';
}

$topologyPath = $testStorage . '/topology.zip';
$zip = new ZipArchive();
if ($zip->open($topologyPath, ZipArchive::CREATE | ZipArchive::OVERWRITE) !== true) {
    throw new RuntimeException('Unable to create ZIP topology regression fixture.');
}
if (!$zip->addFromString('safe.txt', 'safe')) {
    throw new RuntimeException('Unable to add ZIP topology parent file.');
}
if (!$zip->addFromString('safe.txt/child.txt', 'child')) {
    throw new RuntimeException('Unable to add ZIP topology child file.');
}
if (!$zip->close()) {
    throw new RuntimeException('Unable to finalize ZIP topology regression fixture.');
}

$topologyClient = new ZipPreflightFakeClient($topologyPath);
$topologyOps = new RemoteOperations($topologyClient);
$topologyThrew = false;
try {
    $topologyOps->extractZip('/archive.zip', '/dest');
} catch (Throwable) {
    $topologyThrew = true;
}
if (!$topologyThrew) {
    $failures[] = 'conflicting ZIP file/child topology was not rejected';
}
if ($topologyClient->mutations() !== []) {
    $failures[] = 'conflicting ZIP topology is rejected before any remote mutation';
}

$remoteTopologyPath = $testStorage . '/remote-topology.zip';
$zip = new ZipArchive();
if ($zip->open($remoteTopologyPath, ZipArchive::CREATE | ZipArchive::OVERWRITE) !== true) {
    throw new RuntimeException('Unable to create existing remote topology fixture.');
}
if (!$zip->addFromString('first.txt', 'first')) {
    throw new RuntimeException('Unable to add early safe ZIP entry.');
}
if (!$zip->addFromString('blocked/child.txt', 'child')) {
    throw new RuntimeException('Unable to add remote-topology child entry.');
}
if (!$zip->close()) {
    throw new RuntimeException('Unable to finalize existing remote topology fixture.');
}

$remoteTopologyClient = new ZipPreflightFakeClient($remoteTopologyPath, [
    '/' => [[
        'name' => 'dest',
        'type' => 'dir',
        'size' => 0,
        'modified' => null,
        'permissions' => '',
    ]],
    '/dest' => [[
        'name' => 'blocked',
        'type' => 'file',
        'size' => 7,
        'modified' => null,
        'permissions' => '',
    ]],
]);
$remoteTopologyOps = new RemoteOperations($remoteTopologyClient);
$remoteTopologyThrew = false;
try {
    $remoteTopologyOps->extractZip('/archive.zip', '/dest');
} catch (Throwable) {
    $remoteTopologyThrew = true;
}
if (!$remoteTopologyThrew) {
    $failures[] = 'existing remote file blocking a required ZIP directory was not rejected';
}
if ($remoteTopologyClient->mutations() !== []) {
    $failures[] = 'existing remote ZIP topology conflict is rejected before any remote mutation';
}

foreach (glob($testStorage . '/tmp/*') ?: [] as $path) {
    @unlink($path);
}
@unlink($archivePath);
@unlink($topologyPath);
@unlink($remoteTopologyPath);
@rmdir($testStorage . '/tmp');
@rmdir($testStorage);

if ($failures !== []) {
    foreach ($failures as $failure) {
        fwrite(STDERR, 'FAIL: ' . $failure . "\n");
    }
    exit(1);
}

echo "WEB_ZIP_EXTRACTION_PREFLIGHT_TEST=PASS\n";
