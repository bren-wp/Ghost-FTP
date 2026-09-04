<?php
declare(strict_types=1);

$testStorage = sys_get_temp_dir() . '/GhostFTP-atomic-upload-' . bin2hex(random_bytes(6));
mkdir($testStorage . '/tmp', 0700, true);
define('GhostFTP_STORAGE', $testStorage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Operations/RemoteOperations.php';

use GhostFTP\Operations\RemoteOperations;
use GhostFTP\Remote\PathGuard;
use GhostFTP\Remote\RemoteClientInterface;

final class AtomicUploadRecoveryFake implements RemoteClientInterface
{
    /** @var array<string,string> */
    private array $files = ['/target.txt' => 'original'];
    private ?string $backupPath = null;
    private ?string $stagingPath = null;

    public function __construct(private readonly string $mode)
    {
    }

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
        throw new RuntimeException('Unexpected directory creation.');
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
            if ($this->mode === 'ambiguous-promotion') {
                $this->files[$to] = $this->files[$from];
                unset($this->files[$from]);
                throw new RuntimeException('INTERNAL_PROMOTION_RESPONSE_SECRET');
            }
            throw new RuntimeException('INTERNAL_PROMOTION_SECRET');
        }
        if ($this->backupPath !== null && $from === $this->backupPath && $to === '/target.txt') {
            if ($this->mode === 'restore-failure') {
                throw new RuntimeException('INTERNAL_RESTORE_SECRET');
            }
            $this->files[$to] = $this->files[$from];
            unset($this->files[$from]);
            return;
        }
        throw new RuntimeException('Unexpected rename path.');
    }

    public function delete(string $path, bool $directory = false): void
    {
        unset($this->files[$path]);
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $content = file_get_contents($localFile);
        if (!is_string($content)) throw new RuntimeException('Unable to read local fixture.');
        $this->files[$remotePath] = $content;
        $this->stagingPath = $remotePath;
    }

    public function download(string $remotePath, string $localFile): void
    {
        throw new RuntimeException('Unexpected download.');
    }

    public function read(string $remotePath, int $maxBytes = 4194304): string
    {
        throw new RuntimeException('Unexpected read.');
    }

    public function chmod(string $path, int $mode): void
    {
        throw new RuntimeException('Unexpected chmod.');
    }

    public function disconnect(): void {}

    public function backupPath(): ?string { return $this->backupPath; }
    public function stagingPath(): ?string { return $this->stagingPath; }
    /** @return array<string,string> */
    public function files(): array { return $this->files; }
}

function fail_test(string $message): never
{
    fwrite(STDERR, "FAIL: {$message}\n");
    exit(1);
}

$local = $testStorage . '/new.txt';
file_put_contents($local, 'replacement');

$restoreClient = new AtomicUploadRecoveryFake('restore-failure');
$restoreOps = new RemoteOperations($restoreClient);
try {
    $restoreOps->uploadAtomic($local, '/target.txt');
    fail_test('restore-failure scenario unexpectedly succeeded');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'Sigurnosna kopija je sačuvana kao GhostFTP-backup-')) {
        fail_test('restore failure does not expose the safe recovery backup name');
    }
    if (str_contains($message, 'INTERNAL_')) {
        fail_test('restore failure leaked an internal nested exception message');
    }
}
$restoreBackup = $restoreClient->backupPath();
$restoreFiles = $restoreClient->files();
if ($restoreBackup === null || ($restoreFiles[$restoreBackup] ?? null) !== 'original') {
    fail_test('original content was not retained in the recovery backup');
}
if (isset($restoreFiles['/target.txt'])) {
    fail_test('failed restore unexpectedly recreated the destination');
}
$restoreStaging = $restoreClient->stagingPath();
if ($restoreStaging !== null && isset($restoreFiles[$restoreStaging])) {
    fail_test('failed promotion staging object was not cleaned');
}

$ambiguousClient = new AtomicUploadRecoveryFake('ambiguous-promotion');
$ambiguousOps = new RemoteOperations($ambiguousClient);
try {
    $ambiguousOps->uploadAtomic($local, '/target.txt');
    fail_test('ambiguous promotion scenario unexpectedly succeeded');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'odredišna putanja je dostupna') || !str_contains($message, 'Izvorna verzija je sačuvana kao GhostFTP-backup-')) {
        fail_test('ambiguous promotion did not return an actionable recovery message');
    }
    if (str_contains($message, 'INTERNAL_')) {
        fail_test('ambiguous promotion leaked the transport exception message');
    }
}
$ambiguousBackup = $ambiguousClient->backupPath();
$ambiguousFiles = $ambiguousClient->files();
if (($ambiguousFiles['/target.txt'] ?? null) !== 'replacement') {
    fail_test('ambiguous promotion destination state was not preserved');
}
if ($ambiguousBackup === null || ($ambiguousFiles[$ambiguousBackup] ?? null) !== 'original') {
    fail_test('ambiguous promotion did not preserve the original backup');
}

echo "WEB_ATOMIC_UPLOAD_RECOVERY_TEST=PASS\n";
