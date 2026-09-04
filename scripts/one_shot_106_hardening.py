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


# Canonical version surfaces.
replace_once("VERSION", "1.0.5\n", "1.0.6\n")
replace_once("GhostFTP WEB/VERSION", "1.0.5\n", "1.0.6\n")
replace_once('GhostFTP WEB/composer.json', '"version": "1.0.5"', '"version": "1.0.6"')
replace_once("GhostFTP WEB/service-worker.js", "ghostftp-static-v1.0.5", "ghostftp-static-v1.0.6")

# Atomic upload recovery must distinguish a normal promotion error from a
# recovery failure or an ambiguous server outcome. Never discard the only
# known-good backup and never expose raw nested exception messages.
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    """            } catch (\\Throwable $promoteError) {
                if ($backup !== null && !$this->exists($remotePath)) {
                    try { $this->client->rename($backup, $remotePath); $backup = null; } catch (\\Throwable) {}
                }
                throw $promoteError;
            }
""",
    """            } catch (\\Throwable $promoteError) {
                if ($backup !== null) {
                    if (!$this->exists($remotePath)) {
                        try {
                            $this->client->rename($backup, $remotePath);
                            $backup = null;
                        } catch (\\Throwable $restoreError) {
                            throw new RuntimeException(
                                'Nova datoteka nije aktivirana, a izvornu datoteku nije moguće automatski vratiti. Sigurnosna kopija je sačuvana kao ' . PathGuard::basename($backup) . '. Nemoj ponavljati operaciju dok se kopija ne vrati.',
                                0,
                                $restoreError
                            );
                        }
                    } else {
                        throw new RuntimeException(
                            'Remote server je prijavio pogrešku tijekom aktivacije nove datoteke, ali odredišna putanja je dostupna. Izvorna verzija je sačuvana kao ' . PathGuard::basename($backup) . '. Provjeri obje datoteke prije nastavka.',
                            0,
                            $promoteError
                        );
                    }
                }
                throw $promoteError;
            }
""",
)

# Repository hygiene: one-shot helper workflows are temporary tooling and must
# never remain in the production source tree after their run completes.
replace_once(
    "scripts/audit_repository.py",
    """    missing = sorted(required - tracked_paths)
    if missing:
        errors.append("required tracked files missing: " + ", ".join(missing))

    text_count = 0
""",
    """    missing = sorted(required - tracked_paths)
    if missing:
        errors.append("required tracked files missing: " + ", ".join(missing))

    one_shot_workflows = sorted(
        path for path in tracked_paths if path.startswith(".github/workflows/one-shot-")
    )
    if one_shot_workflows:
        errors.append(
            "temporary one-shot workflow is tracked: " + ", ".join(one_shot_workflows)
        )

    text_count = 0
""",
)
replace_once(
    "scripts/audit_repository.py",
    '    print("REPOSITORY_GENERATED_ARTIFACTS=BLOCKED")\n',
    '    print("REPOSITORY_GENERATED_ARTIFACTS=BLOCKED")\n    print("REPOSITORY_ONE_SHOT_WORKFLOWS=BLOCKED")\n',
)

# Focused runtime regression for both critical recovery outcomes.
test_path = ROOT / "GhostFTP WEB/tests/atomic-upload-recovery.php"
if test_path.exists():
    raise SystemExit("PATCH_FAILED: atomic-upload-recovery.php already exists")
test_path.write_text(r'''<?php
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
''', encoding="utf-8", newline="\n")

# Documentation/version surfaces.
replace_once(
    "README.md",
    "Current Ghost FTP version: **1.0.5**",
    "Current Ghost FTP version: **1.0.6**",
)
replace_once(
    "README.md",
    "Ghost FTP 1.0.5 extends the 1.0.4 safe-error boundary across HTML account/setup/admin flows and removes nested exception wrappers that could re-expose internal error text. It also removes the unused transport-level write contract so browser editor writes have one bounded, exact-write, atomic implementation path, while preserving the existing upload, SFTP trust and batch-mutation protections.",
    "Ghost FTP 1.0.6 hardens atomic Web overwrite recovery: a failed promotion can no longer hide a failed restore or an ambiguous server-side rename outcome. The original version remains discoverable under its recovery backup name and the public error stays actionable without exposing nested transport internals. Repository hygiene now also rejects temporary one-shot workflows in production source.",
)
replace_once(
    "README.md",
    "`ghostftp-v1.0.4` and `ghostftp-v1.0.5`.",
    "`ghostftp-v1.0.4`, `ghostftp-v1.0.5` and `ghostftp-v1.0.6`.",
)
replace_once(
    "README.md",
    "`1.0.4` → `1.0.5`), backward-compatible",
    "`1.0.4` → `1.0.5` → `1.0.6`), backward-compatible",
)

replace_once(
    "docs/SECURITY.md",
    "**Current Ghost FTP release: 1.0.5**",
    "**Current Ghost FTP release: 1.0.6**",
)
replace_once(
    "docs/SECURITY.md",
    "- Multi-file upload validates the complete request shape, temporary upload identity and normalized remote destination set before the first remote mutation.\n",
    "- Multi-file upload validates the complete request shape, temporary upload identity and normalized remote destination set before the first remote mutation.\n- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.\n",
)
replace_once(
    "docs/SECURITY.md",
    "The repository-wide audit checks tracked files for case-insensitive path collisions, Windows-reserved components, symlinks, generated/cache artifacts, malformed UTF-8 text, NUL/BOM issues, trailing whitespace, missing final newlines, merge-conflict markers and stale current-release references.",
    "The repository-wide audit checks tracked files for case-insensitive path collisions, Windows-reserved components, symlinks, generated/cache artifacts, temporary one-shot workflows, malformed UTF-8 text, NUL/BOM issues, trailing whitespace, missing final newlines, merge-conflict markers and stale current-release references.",
)

changelog = read("CHANGELOG.md")
entry = """## 1.0.6 - 2026-09-05

- Hardened Web atomic overwrite recovery so a failed promotion cannot silently hide a failed restoration of the original file.
- Detects ambiguous remote rename outcomes where the server reports a promotion error after the destination path becomes available; both the new destination and original recovery backup are preserved for inspection.
- Recovery errors expose the generated backup name needed for manual restoration while never concatenating raw nested transport exception text into the public message.
- Added a focused runtime regression covering failed restore, ambiguous promotion, backup preservation and staging cleanup.
- Removed the leaked one-shot 1.0.5 audit workflow from production source and made the repository audit reject tracked `.github/workflows/one-shot-*` helpers permanently.

"""
needle = "# Changelog\n\n## 1.0.5 - 2026-09-05\n"
if changelog.count(needle) != 1:
    raise SystemExit("PATCH_FAILED: CHANGELOG insertion marker mismatch")
write("CHANGELOG.md", changelog.replace(needle, "# Changelog\n\n" + entry + "## 1.0.5 - 2026-09-05\n", 1))

print("GHOST_FTP_1_0_6_HARDENING_PATCH=APPLIED")
