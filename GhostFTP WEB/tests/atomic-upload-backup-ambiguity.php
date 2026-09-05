<?php
declare(strict_types=1);

$testStorage = sys_get_temp_dir() . '/GhostFTP-backup-ambiguity-' . bin2hex(random_bytes(6));
mkdir($testStorage . '/tmp', 0700, true);
define('GhostFTP_STORAGE', $testStorage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Operations/RemoteOperations.php';

use GhostFTP\Operations\RemoteOperations;
use GhostFTP\Remote\PathGuard;
use GhostFTP\Remote\RemoteClientInterface;

final class BackupAmbiguityFake implements RemoteClientInterface
{
    /** @var array<string,string> */
    private array $files = ['/target.txt' => 'original'];
    private ?string $stagingPath = null;
    private ?string $backupPath = null;

    public function __construct(private readonly bool $failStagingCleanup) {}
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

    public function makeDirectory(string $path): void { throw new RuntimeException('Unexpected mkdir.'); }

    public function rename(string $from, string $to): void
    {
        if ($from === '/target.txt' && str_starts_with($to, '/GhostFTP-backup-')) {
            $this->files[$to] = $this->files[$from];
            unset($this->files[$from]);
            $this->backupPath = $to;
            throw new RuntimeException('INTERNAL_BACKUP_RENAME_SECRET');
        }
        throw new RuntimeException('Unexpected rename.');
    }

    public function delete(string $path, bool $directory = false): void
    {
        if ($path === $this->stagingPath && $this->failStagingCleanup) {
            throw new RuntimeException('INTERNAL_STAGING_CLEANUP_SECRET');
        }
        if (!array_key_exists($path, $this->files)) throw new RuntimeException('Already absent.');
        unset($this->files[$path]);
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $content = file_get_contents($localFile);
        if (!is_string($content)) throw new RuntimeException('Unable to read fixture.');
        $this->stagingPath = $remotePath;
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

function fail_backup_ambiguity(string $message): never
{
    fwrite(STDERR, "FAIL: {$message}\n");
    exit(1);
}

$local = $testStorage . '/replacement.txt';
file_put_contents($local, 'replacement-data');

$client = new BackupAmbiguityFake(false);
try {
    (new RemoteOperations($client))->uploadAtomic($local, '/target.txt');
    fail_backup_ambiguity('ambiguous backup rename unexpectedly succeeded');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'pogrešku tijekom izrade sigurnosne kopije') ||
        !str_contains($message, 'sigurnosna kopija je dostupna kao GhostFTP-backup-') ||
        !str_contains($message, 'Nova datoteka nije aktivirana')) {
        fail_backup_ambiguity('ambiguous backup rename did not return actionable recovery state');
    }
    if (str_contains($message, 'INTERNAL_')) fail_backup_ambiguity('ambiguous backup rename leaked transport internals');
}
$files = $client->files();
$backup = $client->backupPath();
$staging = $client->stagingPath();
if (isset($files['/target.txt'])) fail_backup_ambiguity('target unexpectedly remained after simulated move-then-error backup rename');
if ($backup === null || ($files[$backup] ?? null) !== 'original') fail_backup_ambiguity('original was not preserved under confirmed recovery backup');
if ($staging !== null && isset($files[$staging])) fail_backup_ambiguity('staging was not cleaned after ambiguous backup rename');

$combined = new BackupAmbiguityFake(true);
try {
    (new RemoteOperations($combined))->uploadAtomic($local, '/target.txt');
    fail_backup_ambiguity('combined ambiguous backup + staging cleanup failure unexpectedly succeeded');
} catch (RuntimeException $e) {
    $message = $e->getMessage();
    if (!str_contains($message, 'privremenu remote datoteku nije moguće potvrđeno ukloniti') ||
        !str_contains($message, 'GhostFTP-upload-') ||
        !str_contains($message, 'Izvorna verzija je sačuvana kao GhostFTP-backup-')) {
        fail_backup_ambiguity('combined recovery state does not expose both safe recovery names');
    }
    if (str_contains($message, 'INTERNAL_')) fail_backup_ambiguity('combined recovery state leaked transport internals');
}
$combinedFiles = $combined->files();
$combinedBackup = $combined->backupPath();
$combinedStaging = $combined->stagingPath();
if ($combinedBackup === null || ($combinedFiles[$combinedBackup] ?? null) !== 'original') fail_backup_ambiguity('combined state lost original recovery backup');
if ($combinedStaging === null || !isset($combinedFiles[$combinedStaging])) fail_backup_ambiguity('combined fixture did not retain failed-cleanup staging file');

echo "WEB_ATOMIC_UPLOAD_BACKUP_AMBIGUITY_TEST=PASS\n";
