#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def write(path: str, text: str) -> None:
    (ROOT / path).write_text(text, encoding="utf-8", newline="\n")


def replace_once(path: str, old: str, new: str) -> None:
    text = read(path)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"PATCH_FAILED: {path}: expected one match, got {count}")
    write(path, text.replace(old, new, 1))


ops = "GhostFTP WEB/app/Operations/RemoteOperations.php"
replace_once(
    ops,
    """        $backup = null;
        $staged = true;
        $operationError = null;
""",
    """        $backup = null;
        $staged = true;
        $operationError = null;
        $recoveryHint = null;
""",
)

replace_once(
    ops,
    """            if ($existing !== null) {
                $backup = $this->temporarySibling($parent, 'GhostFTP-backup');
                $this->client->rename($remotePath, $backup);
            }
""",
    """            if ($existing !== null) {
                $backupCandidate = $this->temporarySibling($parent, 'GhostFTP-backup');
                try {
                    $this->client->rename($remotePath, $backupCandidate);
                    $backup = $backupCandidate;
                } catch (\\Throwable $backupError) {
                    // Some servers can complete a rename and still report a transport error.
                    // Re-read both paths before deciding whether the original stayed in place,
                    // moved to the recovery name or entered an unverifiable state.
                    $targetPresence = $this->remotePathPresence($remotePath);
                    $backupPresence = $this->remotePathPresence($backupCandidate);
                    if ($backupPresence === true) {
                        $backup = $backupCandidate;
                        throw new RuntimeException(
                            'Remote server je prijavio pogrešku tijekom izrade sigurnosne kopije, ali sigurnosna kopija je dostupna kao ' . PathGuard::basename($backup) . '. Nova datoteka nije aktivirana. Provjeri odredišnu putanju i sigurnosnu kopiju prije ponavljanja operacije.',
                            0,
                            $backupError
                        );
                    }
                    if ($backupPresence === false && $targetPresence === true) {
                        throw $backupError;
                    }
                    $recoveryHint = $backupCandidate;
                    throw new RuntimeException(
                        'Remote server je prijavio pogrešku tijekom izrade sigurnosne kopije i nije moguće potvrditi potpuno stanje izvorne datoteke. Nova datoteka nije aktivirana. Prije ponavljanja operacije provjeri odredišnu putanju i moguću recovery putanju ' . PathGuard::basename($backupCandidate) . '.',
                        0,
                        $backupError
                    );
                }
            }
""",
)

replace_once(
    ops,
    """                if ($backup !== null) {
                    $message .= ' Izvorna verzija je sačuvana kao ' . PathGuard::basename($backup) . '.';
                }
                throw new RuntimeException($message, 0, $operationError);
""",
    """                if ($backup !== null) {
                    $message .= ' Izvorna verzija je sačuvana kao ' . PathGuard::basename($backup) . '.';
                } elseif ($recoveryHint !== null) {
                    $message .= ' Provjeri i moguću recovery putanju ' . PathGuard::basename($recoveryHint) . '.';
                }
                throw new RuntimeException($message, 0, $operationError);
""",
)

replace_once(
    ops,
    """    private function cleanupRemoteTemporary(string $path): bool
""",
    """    private function remotePathPresence(string $path): ?bool
    {
        try {
            return $this->exists($path);
        } catch (\\Throwable) {
            return null;
        }
    }

    private function cleanupRemoteTemporary(string $path): bool
""",
)

# Permanently pin the ambiguous-backup recovery invariant in the Web source audit.
replace_once(
    "scripts/audit_web.py",
    '''            "private function cleanupRemoteTemporary(string $path): bool",\n            "cleanupRemoteTemporary($staging)",\n''',
    '''            "private function cleanupRemoteTemporary(string $path): bool",\n            "private function remotePathPresence(string $path): ?bool",\n            "$backupCandidate = $this->temporarySibling($parent, 'GhostFTP-backup');",\n            "pogrešku tijekom izrade sigurnosne kopije",\n            "$recoveryHint = $backupCandidate;",\n            "cleanupRemoteTemporary($staging)",\n''',
)
replace_once(
    "scripts/audit_web.py",
    '''    if "$this->client->upload($localFile, $staging);\\n            $staged = true;" in operations:\n        fail("atomic upload takes staging cleanup ownership only after transport success")\n\n''',
    '''    if "$this->client->upload($localFile, $staging);\\n            $staged = true;" in operations:\n        fail("atomic upload takes staging cleanup ownership only after transport success")\n    if "$backup = $this->temporarySibling($parent, 'GhostFTP-backup');\\n                $this->client->rename($remotePath, $backup);" in operations:\n        fail("atomic upload does not reconcile ambiguous backup-creation rename outcomes")\n\n''',
)

# Runtime regression: server moves the original to backup but reports rename failure.
test = ROOT / "GhostFTP WEB/tests/atomic-upload-backup-ambiguity.php"
if test.exists():
    raise SystemExit("PATCH_FAILED: atomic-upload-backup-ambiguity.php already exists")
test.write_text(r'''<?php
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
''', encoding="utf-8", newline="\n")

# Document the additional recovery boundary without changing the 1.0.7 version.
replace_once(
    "CHANGELOG.md",
    "- Added verified cleanup for residual staging files; if deletion/absence cannot be confirmed, Ghost FTP reports the generated staging recovery name without exposing nested transport error text.\n",
    "- Added verified cleanup for residual staging files; if deletion/absence cannot be confirmed, Ghost FTP reports the generated staging recovery name without exposing nested transport error text.\n- Reconciles ambiguous backup-creation renames where a server moves the original file but still reports an error, surfacing the confirmed recovery backup name and cleaning staging without activating the replacement.\n",
)
replace_once(
    "docs/SECURITY.md",
    "- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.\n",
    "- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.\n- Backup creation is also reconciled after rename errors: Ghost FTP re-reads the target and candidate backup paths so a move-then-error response cannot hide the confirmed recovery filename or accidentally continue promotion.\n",
)

print("GHOST_FTP_1_0_7_BACKUP_AMBIGUITY_PATCH=APPLIED")
