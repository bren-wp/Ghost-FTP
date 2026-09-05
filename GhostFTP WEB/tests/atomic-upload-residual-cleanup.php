<?php
declare(strict_types=1);

$testStorage = sys_get_temp_dir() . '/GhostFTP-atomic-residual-' . bin2hex(random_bytes(6));
mkdir($testStorage . '/tmp', 0700, true);
define('GhostFTP_STORAGE', $testStorage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Operations/RemoteOperations.php';

use GhostFTP\Operations\RemoteOperations;
use GhostFTP\Remote\PathGuard;
use GhostFTP\Remote\RemoteClientInterface;

final class ResidualCleanupFake implements RemoteClientInterface
{
    /** @var array<string,string> */
    private array $files = ['/target.txt' => 'original'];
    private ?string $stagingPath = null;
    private ?string $backupPath = null;

    public function __construct(private readonly string $mode) {}
    public function connect(): void {}

    public function list(string $path): array
    {
        if ($path !== '/') return [];
        $items = [];
        foreach ($this->files as $remotePath => $content) {
            if (PathGuard::parent($remotePath) !== '/') continue;
            $items[] = [
                'name' => PathGuard::basename($remotePath),
                'type' => 'file',
                'size' => strlen($content),
                'modified' => null,
                'permissions' => '',
            ];
        }
        return $items;
    }

    public function makeDirectory(string $path): void
    {
        throw new RuntimeException('Unexpected mkdir.');
    }

    public function rename(string $from, string $to): void
    {
        if ($from === '/target.txt' && str_starts_with($to, '/GhostFTP-backup-')) {
            $this->files[$to] = $this->files[$from];
            unset($this->files[$from]);
            $this->backupPath = $to;
            return;
        }
        if ($this->stagingPath !== null && $from === $this->stagingPath && $to === '/target.txt') {
            $this->files[$to] = $this->files[$from];
            unset($this->files[$from]);
            return;
        }
        throw new RuntimeException('Unexpected rename.');
    }

    public function delete(string $path, bool $directory = false): void
    {
        if ($path === $this->stagingPath && $this->mode === 'partial-clean-failure') {
            throw new RuntimeException('INTERNAL_STAGE_DELETE_SECRET');
        }
        if ($path === $this->backupPath && $this->mode === 'backup-delete-failure') {
            throw new RuntimeException('INTERNAL_BACKUP_DELETE_SECRET');
        }
        if (!array_key_exists($path, $this->files)) {
            throw new RuntimeException('Already absent.');
        }
        unset($this->files[$path]);
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $content = file_get_contents($localFile);
        if (!is_string($content)) throw new RuntimeException('Unable to read fixture.');
        $this->stagingPath = $remotePath;
        if (str_starts_with($this->mode, 'partial-clean-')) {
            $this->files[$remotePath] = substr($content, 0, max(1, intdiv(strlen($content), 2)));
            throw new RuntimeException('Upload nije dovršen u cijelosti.');
        }
        $this->files[$remotePath] = $content;
    }

    public function download(string $remotePath, string $localFile): void { throw new RuntimeException('Unexpected download.'); }
    public function read(string $remotePath, int $maxBytes = 4194304): string { throw new RuntimeException('Unexpected read.'); }
    public function chmod(string $path, int $mode): void { throw new RuntimeException('Unexpected chmod.'); }
    public function disconnect(): void {}

    public function stagingPath(): ?string { return $this->stagingPath; }
    public function backupPath(): ?string { return $this->backupPath; }
    /** @return array<string,string> */
    public function files(): array { return $this->files; }
}

function fail_residual(string $message): never
{
    fwrite(STDERR, "FAIL: {$message}\n");
    exit(1);
}

$local = $testStorage . '/replacement.txt';
file_put_contents($local, 'replacement-data');

$cleanClient = new ResidualCleanupFake('partial-clean-success');
try {
    (new RemoteOperations($cleanClient))->uploadAtomic($local, '/target.txt');
    fail_residual('partial upload unexpectedly succeeded');
} catch (RuntimeException $e) {
    if (!str_contains($e->getMessage(), 'Upload nije dovršen')) {
        fail_residual('clean partial-upload failure changed the transport error unexpectedly');
    }
}
$cleanFiles = $cleanClient->files();
$cleanStaging = $cleanClient->stagingPath();
if (($cleanFiles['/target.txt'] ?? null) !== 'original') fail_residual('partial upload changed target before promotion');
if ($cleanStaging !== null && isset($cleanFiles[$cleanStaging])) fail_residual('partial staging file was not removed after transport failure');

$failedCleanupClient = new ResidualCleanupFake('partial-clean-failure');
try {
    (new RemoteOperations($failedCleanupClient))->uploadAtomic($local, '/target.txt');
    fail_residual('partial upload with cleanup failure unexpectedly succeeded');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'privremenu remote datoteku nije moguće potvrđeno ukloniti') || !str_contains($message, 'GhostFTP-upload-')) {
        fail_residual('cleanup failure did not expose the safe staging recovery name');
    }
    if (str_contains($message, 'INTERNAL_')) fail_residual('cleanup failure leaked nested transport text');
}
$failedFiles = $failedCleanupClient->files();
$failedStaging = $failedCleanupClient->stagingPath();
if (($failedFiles['/target.txt'] ?? null) !== 'original') fail_residual('cleanup-failure scenario changed target before promotion');
if ($failedStaging === null || !isset($failedFiles[$failedStaging])) fail_residual('cleanup-failure fixture did not retain the residual staging file');

$backupClient = new ResidualCleanupFake('backup-delete-failure');
try {
    (new RemoteOperations($backupClient))->uploadAtomic($local, '/target.txt');
    fail_residual('backup deletion failure was silently reported as success');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'Nova datoteka je aktivna') || !str_contains($message, 'Nemoj ponavljati upload') || !str_contains($message, 'GhostFTP-backup-')) {
        fail_residual('backup deletion failure is not an actionable partial-success message');
    }
    if (str_contains($message, 'INTERNAL_')) fail_residual('backup deletion failure leaked nested transport text');
}
$backupFiles = $backupClient->files();
$backupPath = $backupClient->backupPath();
if (($backupFiles['/target.txt'] ?? null) !== 'replacement-data') fail_residual('successful promotion was not preserved after backup cleanup failure');
if ($backupPath === null || ($backupFiles[$backupPath] ?? null) !== 'original') fail_residual('old-version backup was not retained for manual cleanup');

echo "WEB_ATOMIC_UPLOAD_RESIDUAL_CLEANUP_TEST=PASS\n";
