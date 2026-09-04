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
replace_once("VERSION", "1.0.3\n", "1.0.4\n")
replace_once("GhostFTP WEB/VERSION", "1.0.3\n", "1.0.4\n")
replace_once('GhostFTP WEB/composer.json', '"version": "1.0.3"', '"version": "1.0.4"')
replace_once("GhostFTP WEB/service-worker.js", "ghostftp-static-v1.0.3", "ghostftp-static-v1.0.4")

# Shared public error boundary. Deliberate application validation errors remain
# user-readable; unexpected Error/TypeError/other Throwable details stay server-side.
helpers = "GhostFTP WEB/app/helpers.php"
replace_once(
    helpers,
    """function GhostFTP_json(array $payload, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE);
    exit;
}

function GhostFTP_truncate(string $value, int $length): string
""",
    """function GhostFTP_json(array $payload, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE);
    exit;
}

function GhostFTP_public_error(Throwable $error): string
{
    if ($error instanceof RuntimeException || $error instanceof InvalidArgumentException) {
        $message = trim(str_replace(["\\r", "\\n", "\\0"], ' ', $error->getMessage()));
        if ($message !== '') {
            return GhostFTP_truncate($message, 300);
        }
    }
    return 'Neočekivana interna pogreška. Pokušaj ponovno ili provjeri zapisnik poslužitelja.';
}

function GhostFTP_public_error_status(Throwable $error): int
{
    return ($error instanceof RuntimeException || $error instanceof InvalidArgumentException) ? 400 : 500;
}

function GhostFTP_truncate(string $value, int $length): string
""",
)
replace_once(
    helpers,
    """    foreach ($rows as $row) {
        if (!is_string($row)) {
            continue;
        }
        $paths[] = GhostFTP\\Remote\\PathGuard::ensureNotRoot($row);
    }
""",
    """    foreach ($rows as $row) {
        if (!is_string($row)) {
            throw new RuntimeException('Popis putanja sadrži neispravnu stavku.');
        }
        $paths[] = GhostFTP\\Remote\\PathGuard::ensureNotRoot($row);
    }
""",
)

# API: preflight the entire upload request before the first remote mutation and
# never expose unexpected Throwable details to the HTTP client.
api = "GhostFTP WEB/api.php"
old_upload = """        case 'upload':
            $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            $files = $_FILES['files'] ?? null;
            if (!$files) throw new RuntimeException('Nema datoteka za upload.');
            $names = is_array($files['name']) ? $files['name'] : [$files['name']];
            $tmps = is_array($files['tmp_name']) ? $files['tmp_name'] : [$files['tmp_name']];
            $errors = is_array($files['error']) ? $files['error'] : [$files['error']];
            $relativePaths = $_POST['relative_paths'] ?? [];
            if (!is_array($relativePaths)) $relativePaths = [$relativePaths];
            if (count($names) > 200) throw new RuntimeException('Previše datoteka u jednom upload zahtjevu.');
            $conflictPolicy = strtolower(trim((string)($_POST['conflict'] ?? 'overwrite')));
            if (!in_array($conflictPolicy, ['overwrite','skip','rename'], true)) throw new RuntimeException('Neispravna politika konflikta pri uploadu.');
            $uploaded = [];
            $skipped = [];
            foreach ($names as $i => $original) {
                $error = (int)($errors[$i] ?? UPLOAD_ERR_NO_FILE);
                if ($error !== UPLOAD_ERR_OK) throw new RuntimeException(GhostFTP_upload_error_message($error, (string)$original));
                $tmp = (string)($tmps[$i] ?? '');
                if (!is_uploaded_file($tmp)) throw new RuntimeException('Privremena upload datoteka nije valjana.');
                $relative = trim((string)($relativePaths[$i] ?? ''));
                if ($relative !== '') {
                    $relative = PathGuard::normalizeRelative('/'.ltrim($relative,'/'));
                    $remote = PathGuard::normalizeRelative(($path === '/' ? '' : $path).$relative);
                } else {
                    $remote = PathGuard::child($path, PathGuard::basename((string)$original));
                }
                $finalRemote = $ops->uploadWithConflict($tmp, $remote, $conflictPolicy);
                if ($finalRemote === null) $skipped[] = $remote; else $uploaded[] = $finalRemote;
            }
            AppLogger::event('file.upload', ['profile_id'=>$profileId,'count'=>count($uploaded),'skipped'=>count($skipped),'path'=>$path]);
            GhostFTP_json(['ok'=>true,'uploaded'=>$uploaded,'skipped'=>$skipped]);
"""
new_upload = """        case 'upload':
            $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            $files = $_FILES['files'] ?? null;
            if (!$files || !is_array($files)) throw new RuntimeException('Nema datoteka za upload.');
            $names = is_array($files['name'] ?? null) ? $files['name'] : [$files['name'] ?? ''];
            $tmps = is_array($files['tmp_name'] ?? null) ? $files['tmp_name'] : [$files['tmp_name'] ?? ''];
            $errors = is_array($files['error'] ?? null) ? $files['error'] : [$files['error'] ?? UPLOAD_ERR_NO_FILE];
            $relativePaths = $_POST['relative_paths'] ?? [];
            if (!is_array($relativePaths)) $relativePaths = [$relativePaths];
            $fileCount = count($names);
            if ($fileCount < 1) throw new RuntimeException('Nema datoteka za upload.');
            if ($fileCount > 200) throw new RuntimeException('Previše datoteka u jednom upload zahtjevu.');
            if (count($tmps) !== $fileCount || count($errors) !== $fileCount || count($relativePaths) > $fileCount) {
                throw new RuntimeException('Upload zahtjev sadrži neusklađene metapodatke datoteka.');
            }
            $conflictPolicy = strtolower(trim((string)($_POST['conflict'] ?? 'overwrite')));
            if (!in_array($conflictPolicy, ['overwrite','skip','rename'], true)) throw new RuntimeException('Neispravna politika konflikta pri uploadu.');

            $uploadPlan = [];
            $seenRemote = [];
            $seenTmp = [];
            foreach ($names as $i => $original) {
                if (!is_string($original) || $original === '') throw new RuntimeException('Naziv upload datoteke nije valjan.');
                $error = (int)$errors[$i];
                if ($error !== UPLOAD_ERR_OK) throw new RuntimeException(GhostFTP_upload_error_message($error, $original));
                $tmp = (string)$tmps[$i];
                if ($tmp === '' || !is_uploaded_file($tmp)) throw new RuntimeException('Privremena upload datoteka nije valjana.');
                if (isset($seenTmp[$tmp])) throw new RuntimeException('Ista privremena upload datoteka ne smije biti navedena više puta.');
                $relative = trim((string)($relativePaths[$i] ?? ''));
                if ($relative !== '') {
                    $relative = PathGuard::normalizeRelative('/'.ltrim($relative,'/'));
                    $remote = PathGuard::normalizeRelative(($path === '/' ? '' : $path).$relative);
                } else {
                    $remote = PathGuard::child($path, PathGuard::basename($original));
                }
                if (isset($seenRemote[$remote])) throw new RuntimeException('Više upload datoteka ne smije ciljati istu udaljenu putanju.');
                $seenTmp[$tmp] = true;
                $seenRemote[$remote] = true;
                $uploadPlan[] = ['tmp' => $tmp, 'remote' => $remote];
            }

            $uploaded = [];
            $skipped = [];
            foreach ($uploadPlan as $item) {
                $remote = (string)$item['remote'];
                $finalRemote = $ops->uploadWithConflict((string)$item['tmp'], $remote, $conflictPolicy);
                if ($finalRemote === null) $skipped[] = $remote; else $uploaded[] = $finalRemote;
            }
            AppLogger::event('file.upload', ['profile_id'=>$profileId,'count'=>count($uploaded),'skipped'=>count($skipped),'path'=>$path]);
            GhostFTP_json(['ok'=>true,'uploaded'=>$uploaded,'skipped'=>$skipped]);
"""
replace_once(api, old_upload, new_upload)
replace_once(
    api,
    """} catch (Throwable $e) {
    AppLogger::event('api.error', ['action'=>$action,'profile_id'=>$profileId,'error'=>GhostFTP_truncate($e->getMessage(),300)]);
    GhostFTP_json(['ok'=>false,'error'=>$e->getMessage()],400);
} finally {
""",
    """} catch (Throwable $e) {
    AppLogger::event('api.error', ['action'=>$action,'profile_id'=>$profileId,'exception'=>get_class($e),'error'=>GhostFTP_truncate($e->getMessage(),300)]);
    GhostFTP_json(['ok'=>false,'error'=>GhostFTP_public_error($e)], GhostFTP_public_error_status($e));
} finally {
""",
)

# Download and preview surfaces use the same safe public error mapping. If the
# binary response already started, do not append a text error into the payload.
replace_once(
    "GhostFTP WEB/download.php",
    """} catch (Throwable $e) {
    AppLogger::event('download.error', ['profile_id' => $profileId, 'path' => $path, 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(400);
        header('Content-Type: text/plain; charset=utf-8');
    }
    echo 'Download nije uspio: ' . $e->getMessage();
} finally {
""",
    """} catch (Throwable $e) {
    AppLogger::event('download.error', ['profile_id' => $profileId, 'path' => $path, 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(GhostFTP_public_error_status($e));
        header('Content-Type: text/plain; charset=utf-8');
        echo 'Download nije uspio: ' . GhostFTP_public_error($e);
    }
} finally {
""",
)
replace_once(
    "GhostFTP WEB/download-archive.php",
    """} catch (Throwable $e) {
    AppLogger::event('archive_download.error', ['profile_id' => $profileId, 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(400);
        header('Content-Type: text/plain; charset=utf-8');
    }
    echo 'Preuzimanje ZIP arhive nije uspjelo: ' . $e->getMessage();
} finally {
""",
    """} catch (Throwable $e) {
    AppLogger::event('archive_download.error', ['profile_id' => $profileId, 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(GhostFTP_public_error_status($e));
        header('Content-Type: text/plain; charset=utf-8');
        echo 'Preuzimanje ZIP arhive nije uspjelo: ' . GhostFTP_public_error($e);
    }
} finally {
""",
)
replace_once(
    "GhostFTP WEB/preview.php",
    """} catch (Throwable $e) {
    AppLogger::event('preview.error', ['profile_id' => $profileId, 'path' => $path, 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) { http_response_code(400); header('Content-Type: text/plain; charset=utf-8'); }
    echo $e->getMessage();
} finally {
""",
    """} catch (Throwable $e) {
    AppLogger::event('preview.error', ['profile_id' => $profileId, 'path' => $path, 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(GhostFTP_public_error_status($e));
        header('Content-Type: text/plain; charset=utf-8');
        echo GhostFTP_public_error($e);
    }
} finally {
""",
)

# Remaining visible brand drift in admin diagnostics.
replace_once("GhostFTP WEB/diagnostics.php", "['GhostFTP Web', 'v' . GhostFTP_VERSION", "['Ghost FTP Web', 'v' . GhostFTP_VERSION")
replace_once("GhostFTP WEB/diagnostics.php", "GhostFTP može pokrenuti osnovne FTP funkcije.", "Ghost FTP može pokrenuti osnovne FTP funkcije.")
replace_once("GhostFTP WEB/diagnostics.php", ">Natrag u GhostFTP<", ">Natrag u Ghost FTP<")

# Regression tests: public error disclosure, strict path-list parsing and full
# upload preflight ordering before any uploadWithConflict call.
public_test = ROOT / "GhostFTP WEB/tests/public-error-boundary.php"
if public_test.exists():
    raise SystemExit("PATCH_FAILED: public-error-boundary.php already exists")
public_test.write_text("""<?php
declare(strict_types=1);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/helpers.php';

$known = GhostFTP_public_error(new RuntimeException("Poznata validacija\\r\\nne smije curiti u više redaka"));
if ($known !== 'Poznata validacija  ne smije curiti u više redaka') {
    fwrite(STDERR, "FAIL: known application error was not normalized.\\n");
    exit(1);
}
if (GhostFTP_public_error_status(new RuntimeException('x')) !== 400) {
    fwrite(STDERR, "FAIL: known application error status is not 400.\\n");
    exit(1);
}
$internal = new TypeError('/srv/private/customer/secret.php: internal detail');
if (GhostFTP_public_error($internal) !== 'Neočekivana interna pogreška. Pokušaj ponovno ili provjeri zapisnik poslužitelja.') {
    fwrite(STDERR, "FAIL: unexpected Throwable detail reached public error text.\\n");
    exit(1);
}
if (GhostFTP_public_error_status($internal) !== 500) {
    fwrite(STDERR, "FAIL: unexpected Throwable status is not 500.\\n");
    exit(1);
}
try {
    GhostFTP_string_paths(json_encode(['/safe', ['bad']]), 10);
    fwrite(STDERR, "FAIL: malformed ZIP path-list row was silently ignored.\\n");
    exit(1);
} catch (RuntimeException $e) {
    if (!str_contains($e->getMessage(), 'neispravnu stavku')) {
        throw $e;
    }
}

echo "WEB_PUBLIC_ERROR_BOUNDARY_TEST=PASS\\n";
""", encoding="utf-8", newline="\n")

upload_test = ROOT / "GhostFTP WEB/tests/api-upload-preflight.php"
if upload_test.exists():
    raise SystemExit("PATCH_FAILED: api-upload-preflight.php already exists")
upload_test.write_text("""<?php
declare(strict_types=1);

$source = file_get_contents(__DIR__ . '/../api.php');
if (!is_string($source)) {
    fwrite(STDERR, "FAIL: unable to inspect API upload source.\\n");
    exit(1);
}
$start = strpos($source, "case 'upload':");
$end = $start !== false ? strpos($source, "throw new RuntimeException('Nepoznata akcija.')", $start) : false;
if ($start === false || $end === false || $end <= $start) {
    fwrite(STDERR, "FAIL: unable to isolate upload API branch.\\n");
    exit(1);
}
$block = substr($source, $start, $end - $start);
$planPos = strpos($block, '$uploadPlan = [];');
$validationLoopPos = strpos($block, 'foreach ($names as $i => $original)');
$shapePos = strpos($block, 'Upload zahtjev sadrži neusklađene metapodatke datoteka.');
$duplicateRemotePos = strpos($block, 'Više upload datoteka ne smije ciljati istu udaljenu putanju.');
$appendPos = strpos($block, '$uploadPlan[] =');
$executionLoopPos = strpos($block, 'foreach ($uploadPlan as $item)');
$uploadPos = strpos($block, 'uploadWithConflict(', $executionLoopPos === false ? 0 : $executionLoopPos);
if (
    $planPos === false || $validationLoopPos === false || $shapePos === false
    || $duplicateRemotePos === false || $appendPos === false
    || $executionLoopPos === false || $uploadPos === false
    || !($shapePos < $validationLoopPos
        && $planPos < $validationLoopPos
        && $validationLoopPos < $duplicateRemotePos
        && $duplicateRemotePos < $appendPos
        && $appendPos < $executionLoopPos
        && $executionLoopPos < $uploadPos)
) {
    fwrite(STDERR, "FAIL: upload request is not fully preflighted before remote mutation.\\n");
    exit(1);
}

echo "WEB_API_UPLOAD_PREFLIGHT_TEST=PASS\\n";
""", encoding="utf-8", newline="\n")

# Audit pins the new boundaries so later refactors cannot silently remove them.
audit = "scripts/audit_web.py"
replace_once(
    audit,
    '    helpers = read("GhostFTP WEB/app/helpers.php")\n    require(helpers, ("function GhostFTP_csrf_token", "hash_equals", "random_bytes"), "app/helpers.php")\n',
    '    helpers = read("GhostFTP WEB/app/helpers.php")\n    require(helpers, ("function GhostFTP_csrf_token", "hash_equals", "random_bytes", "function GhostFTP_public_error", "function GhostFTP_public_error_status", "Neočekivana interna pogreška"), "app/helpers.php")\n\n    api = read("GhostFTP WEB/api.php")\n    require(api, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)", "exception\'=>get_class($e)"), "api.php")\n    if "GhostFTP_json([\'ok\'=>false,\'error\'=>$e->getMessage()]" in api:\n        fail("API exposes raw unexpected exception messages")\n\n    for endpoint in ("download.php", "download-archive.php", "preview.php"):\n        endpoint_source = read(f"GhostFTP WEB/{endpoint}")\n        require(endpoint_source, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)"), endpoint)\n',
)
replace_once(
    audit,
    '    sftp = read("GhostFTP WEB/app/Remote/SftpClient.php")\n',
    '    api = read("GhostFTP WEB/api.php")\n    require(\n        api,\n        (\n            "$uploadPlan = [];",\n            "Upload zahtjev sadrži neusklađene metapodatke datoteka.",\n            "Ista privremena upload datoteka ne smije biti navedena više puta.",\n            "Više upload datoteka ne smije ciljati istu udaljenu putanju.",\n            "foreach ($uploadPlan as $item)",\n        ),\n        "api.php",\n    )\n    helpers = read("GhostFTP WEB/app/helpers.php")\n    require(helpers, ("Popis putanja sadrži neispravnu stavku.",), "app/helpers.php")\n\n    sftp = read("GhostFTP WEB/app/Remote/SftpClient.php")\n',
)

# Documentation and release history.
changelog = read("CHANGELOG.md")
if not changelog.startswith("# Changelog\n\n## 1.0.3"):
    raise SystemExit("PATCH_FAILED: unexpected CHANGELOG head")
write(
    "CHANGELOG.md",
    changelog.replace(
        "# Changelog\n\n",
        """# Changelog

## 1.0.4 - 2026-09-05

- Added a shared fail-closed Web public-error boundary: deliberate validation failures remain actionable, while unexpected PHP/extension Throwable details return a generic 500 response instead of leaking internals.
- Applied the same safe error mapping to API, file download, ZIP download and image preview endpoints, and stopped appending text errors after binary response headers have already been sent.
- Made multi-file upload fully preflighted: request shape, upload status, temporary files, remote paths and duplicate targets are validated before the first remote upload mutation.
- Made ZIP path-list parsing reject malformed non-string rows instead of silently skipping them.
- Added regression tests and source audits for public error disclosure, upload preflight ordering and strict path-list parsing.
- Removed remaining visible `GhostFTP` diagnostics text in favor of the canonical **Ghost FTP** brand.

""",
        1,
    ),
)
replace_once("README.md", "Current Ghost FTP version: **1.0.3**", "Current Ghost FTP version: **1.0.4**")
replace_once("README.md", "`ghostftp-v1.0.1`, `ghostftp-v1.0.2` and `ghostftp-v1.0.3`", "`ghostftp-v1.0.1`, `ghostftp-v1.0.2`, `ghostftp-v1.0.3` and `ghostftp-v1.0.4`")
replace_once("README.md", "(`1.0.0` → `1.0.1` → `1.0.2` → `1.0.3`)", "(`1.0.0` → `1.0.1` → `1.0.2` → `1.0.3` → `1.0.4`)")
replace_once(
    "README.md",
    "Ghost FTP 1.0.3 tightens Web/PWA fail-closed behavior around SFTP trust, inline editor staging, SFTP key material and batch mutations. SFTP profiles require a canonical pinned SHA-256 host fingerprint before persistence, editor/new-file writes share one 4 MiB boundary with complete local staging checks, SFTP uploads verify remote size when available, and malformed/duplicate batch rename input is rejected before remote mutation.",
    "Ghost FTP 1.0.4 extends Web/PWA fail-closed behavior to public error handling and multi-file uploads. Unexpected PHP/extension failures no longer expose raw internal messages to clients, and complete upload request metadata/path validation now finishes before the first remote mutation. The 1.0.3 SFTP trust, staging, key-permission and batch-mutation protections remain enforced by regression audits.",
)
replace_once("docs/SECURITY.md", "**Current Ghost FTP release: 1.0.3**", "**Current Ghost FTP release: 1.0.4**")
replace_once(
    "docs/SECURITY.md",
    "- Destructive/batch mutation inputs fail closed before partial application when their shape or source set is invalid.\n",
    "- Destructive/batch mutation inputs fail closed before partial application when their shape or source set is invalid.\n- Multi-file upload validates the complete request shape, temporary upload identity and normalized remote destination set before the first remote mutation.\n- Public Web error responses expose deliberate validation messages but replace unexpected PHP/extension Throwable details with a generic internal-error response.\n",
)

print("GHOST_FTP_1_0_4_HARDENING_PATCH=APPLIED")
