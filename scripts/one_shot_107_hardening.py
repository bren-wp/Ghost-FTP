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
replace_once("VERSION", "1.0.6\n", "1.0.7\n")
replace_once("GhostFTP WEB/VERSION", "1.0.6\n", "1.0.7\n")
replace_once('GhostFTP WEB/composer.json', '"version": "1.0.6"', '"version": "1.0.7"')
replace_once("GhostFTP WEB/service-worker.js", "ghostftp-static-v1.0.6", "ghostftp-static-v1.0.7")

old_upload = r'''    public function uploadAtomic(string $localFile, string $remotePath): void
    {
        $remotePath = PathGuard::ensureNotRoot($remotePath);
        $this->ensureDirectory(PathGuard::parent($remotePath));
        $existing = $this->stat($remotePath);
        if (($existing['type'] ?? null) === 'dir') throw new RuntimeException('Datoteka ne može prepisati postojeći direktorij istog naziva.');
        $parent = PathGuard::parent($remotePath);
        $staging = $this->temporarySibling($parent, 'GhostFTP-upload');
        $backup = null;
        $staged = false;
        try {
            $this->client->upload($localFile, $staging);
            $staged = true;
            if ($existing !== null) {
                $backup = $this->temporarySibling($parent, 'GhostFTP-backup');
                $this->client->rename($remotePath, $backup);
            }
            try {
                $this->client->rename($staging, $remotePath);
                $staged = false;
            } catch (\Throwable $promoteError) {
                if ($backup !== null) {
                    if (!$this->exists($remotePath)) {
                        try {
                            $this->client->rename($backup, $remotePath);
                            $backup = null;
                        } catch (\Throwable $restoreError) {
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
            if ($backup !== null) {
                try { $this->client->delete($backup, false); } catch (\Throwable) {}
            }
        } finally {
            if ($staged) {
                try { $this->client->delete($staging, false); } catch (\Throwable) {}
            }
        }
    }
'''
new_upload = r'''    public function uploadAtomic(string $localFile, string $remotePath): void
    {
        $remotePath = PathGuard::ensureNotRoot($remotePath);
        $this->ensureDirectory(PathGuard::parent($remotePath));
        $existing = $this->stat($remotePath);
        if (($existing['type'] ?? null) === 'dir') throw new RuntimeException('Datoteka ne može prepisati postojeći direktorij istog naziva.');
        $parent = PathGuard::parent($remotePath);
        $staging = $this->temporarySibling($parent, 'GhostFTP-upload');
        $backup = null;
        $staged = true;
        $operationError = null;

        try {
            // Take cleanup ownership before transport upload starts. FTP/SFTP can create
            // a partial remote file and then report failure, so ownership cannot begin only
            // after upload() returns successfully.
            $this->client->upload($localFile, $staging);
            if ($existing !== null) {
                $backup = $this->temporarySibling($parent, 'GhostFTP-backup');
                $this->client->rename($remotePath, $backup);
            }
            try {
                $this->client->rename($staging, $remotePath);
                $staged = false;
            } catch (\Throwable $promoteError) {
                if ($backup !== null) {
                    if (!$this->exists($remotePath)) {
                        try {
                            $this->client->rename($backup, $remotePath);
                            $backup = null;
                        } catch (\Throwable $restoreError) {
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

            if ($backup !== null) {
                if (!$this->cleanupRemoteTemporary($backup)) {
                    throw new RuntimeException(
                        'Nova datoteka je aktivna, ali sigurnosnu kopiju prethodne verzije nije moguće potvrđeno ukloniti. Nemoj ponavljati upload. Nakon provjere nove datoteke ručno obriši ' . PathGuard::basename($backup) . '.',
                    );
                }
                $backup = null;
            }
        } catch (\Throwable $e) {
            $operationError = $e;
        }

        if ($staged) {
            if (!$this->cleanupRemoteTemporary($staging)) {
                $message = 'Operacija nije dovršena, a privremenu remote datoteku nije moguće potvrđeno ukloniti. Prije ponavljanja operacije provjeri i po potrebi obriši ' . PathGuard::basename($staging) . '.';
                if ($backup !== null) {
                    $message .= ' Izvorna verzija je sačuvana kao ' . PathGuard::basename($backup) . '.';
                }
                throw new RuntimeException($message, 0, $operationError);
            }
            $staged = false;
        }

        if ($operationError !== null) {
            throw $operationError;
        }
    }
'''
replace_once("GhostFTP WEB/app/Operations/RemoteOperations.php", old_upload, new_upload)

helper_marker = r'''    private function temporarySibling(string $parent, string $prefix): string
'''
helper = r'''    private function cleanupRemoteTemporary(string $path): bool
    {
        try {
            $this->client->delete($path, false);
            return true;
        } catch (\Throwable) {
            // A transport can report delete failure even when the object is already gone.
            // Verify absence before escalating; if verification itself is unavailable, fail
            // closed so the caller can expose the recovery name instead of silently leaking data.
            try {
                return !$this->exists($path);
            } catch (\Throwable) {
                return false;
            }
        }
    }

    private function temporarySibling(string $parent, string $prefix): string
'''
replace_once("GhostFTP WEB/app/Operations/RemoteOperations.php", helper_marker, helper)

# Durable source-audit pins for staging ownership and old-backup cleanup.
audit_marker = r'''    if "ponovi brisanje: ' . $e->getMessage()" in user_store:
        fail("user-delete wrapper re-exposes raw nested Throwable text")

    remote_interface = read("GhostFTP WEB/app/Remote/RemoteClientInterface.php")
'''
audit_new = r'''    if "ponovi brisanje: ' . $e->getMessage()" in user_store:
        fail("user-delete wrapper re-exposes raw nested Throwable text")

    require(
        operations,
        (
            "$staged = true;",
            "private function cleanupRemoteTemporary(string $path): bool",
            "cleanupRemoteTemporary($staging)",
            "cleanupRemoteTemporary($backup)",
            "Nova datoteka je aktivna, ali sigurnosnu kopiju prethodne verzije nije moguće potvrđeno ukloniti.",
            "privremenu remote datoteku nije moguće potvrđeno ukloniti",
        ),
        "app/Operations/RemoteOperations.php",
    )
    if "try { $this->client->delete($backup, false); } catch (\\Throwable) {}" in operations:
        fail("atomic upload silently swallows old-backup deletion failure")
    if "$this->client->upload($localFile, $staging);\n            $staged = true;" in operations:
        fail("atomic upload takes staging cleanup ownership only after transport success")

    remote_interface = read("GhostFTP WEB/app/Remote/RemoteClientInterface.php")
'''
replace_once("scripts/audit_web.py", audit_marker, audit_new)

# Runtime regression for residual remote data across partial upload and backup cleanup.
test_path = ROOT / "GhostFTP WEB/tests/atomic-upload-residual-cleanup.php"
if test_path.exists():
    raise SystemExit("PATCH_FAILED: atomic-upload-residual-cleanup.php already exists")
test_path.write_text(r'''<?php
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
''', encoding="utf-8", newline="\n")

# Documentation and release metadata.
replace_once("README.md", "Current Ghost FTP version: **1.0.6**", "Current Ghost FTP version: **1.0.7**")
replace_once(
    "README.md",
    "Ghost FTP 1.0.6 hardens atomic Web overwrite recovery: a failed promotion can no longer hide a failed restore or an ambiguous server-side rename outcome. The original version remains discoverable under its recovery backup name and the public error stays actionable without exposing nested transport internals. Repository hygiene now also rejects temporary one-shot workflows in production source.",
    "Ghost FTP 1.0.7 closes residual-data gaps around atomic Web uploads. Cleanup ownership now begins before FTP/SFTP staging starts, so partial transport failures cannot silently bypass staging cleanup; if cleanup cannot be verified, the recovery filename is surfaced safely. A successfully promoted file also reports an explicit partial-success state when the old backup cannot be removed, preventing silent retention of the previous remote contents.",
)
replace_once(
    "README.md",
    "`ghostftp-v1.0.5` and `ghostftp-v1.0.6`.",
    "`ghostftp-v1.0.5`, `ghostftp-v1.0.6` and `ghostftp-v1.0.7`.",
)
replace_once(
    "README.md",
    "`1.0.5` → `1.0.6`), backward-compatible",
    "`1.0.5` → `1.0.6` → `1.0.7`), backward-compatible",
)

replace_once("docs/SECURITY.md", "**Current Ghost FTP release: 1.0.6**", "**Current Ghost FTP release: 1.0.7**")
replace_once(
    "docs/SECURITY.md",
    "- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.\n",
    "- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.\n- Staging cleanup ownership begins before FTP/SFTP upload starts, so a partial transport failure triggers verified remote-temp cleanup; an unverifiable cleanup exposes only the generated staging recovery name.\n- After successful promotion, failure to remove the previous-version backup is reported as an explicit partial-success state with the backup name and a do-not-retry warning instead of silently retaining old remote data.\n",
)

changelog = read("CHANGELOG.md")
entry = """## 1.0.7 - 2026-09-05

- Took atomic-upload staging cleanup ownership before FTP/SFTP transfer begins, closing the gap where a partial remote staging file could survive when the transport failed before returning success.
- Added verified cleanup for residual staging files; if deletion/absence cannot be confirmed, Ghost FTP reports the generated staging recovery name without exposing nested transport error text.
- Stopped silently swallowing failure to remove an old-version backup after successful promotion. The new file remains active and the user receives an explicit partial-success warning with the retained backup name and a do-not-retry instruction.
- Added runtime regression coverage for partial-upload cleanup success, cleanup failure, target preservation, old-backup retention and nested-error non-disclosure.
- Extended the Web source audit so future changes cannot reintroduce post-success staging ownership or silent backup-deletion swallowing.

"""
needle = "# Changelog\n\n## 1.0.6 - 2026-09-05\n"
if changelog.count(needle) != 1:
    raise SystemExit("PATCH_FAILED: CHANGELOG insertion marker mismatch")
write("CHANGELOG.md", changelog.replace(needle, "# Changelog\n\n" + entry + "## 1.0.6 - 2026-09-05\n", 1))

print("GHOST_FTP_1_0_7_HARDENING_PATCH=APPLIED")
