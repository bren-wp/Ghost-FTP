#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    (ROOT / rel).write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"PATCH_FAILED: {rel}: expected exactly one match, got {count}: {old[:120]!r}")
    write(rel, text.replace(old, new, 1))


def replace_all_exact(rel: str, old: str, new: str, expected: int) -> None:
    text = read(rel)
    count = text.count(old)
    if count != expected:
        raise SystemExit(f"PATCH_FAILED: {rel}: expected {expected} matches, got {count}: {old[:120]!r}")
    write(rel, text.replace(old, new))


# Canonical release version surfaces.
replace_once("VERSION", "1.0.2\n", "1.0.3\n")
replace_once("GhostFTP WEB/VERSION", "1.0.2\n", "1.0.3\n")
replace_once("GhostFTP WEB/service-worker.js", "ghostftp-static-v1.0.2", "ghostftp-static-v1.0.3")

composer_path = ROOT / "GhostFTP WEB/composer.json"
composer = json.loads(composer_path.read_text(encoding="utf-8"))
if composer.get("version") != "1.0.2":
    raise SystemExit(f"PATCH_FAILED: unexpected Composer version: {composer.get('version')!r}")
composer["version"] = "1.0.3"
composer_path.write_text(json.dumps(composer, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

# Profile input must fail before persistence when SFTP trust is incomplete.
replace_once(
    "GhostFTP WEB/app/Storage/ProfileStore.php",
    "        if ($fingerprint !== '') {\n            $validFingerprint = false;",
    "        if ($protocol === 'sftp' && $fingerprint === '') {\n            throw new RuntimeException('SFTP zahtijeva SHA-256 host fingerprint prije spremanja ili povezivanja. Provjeri fingerprint servera iz pouzdanog izvora.');\n        }\n\n        if ($fingerprint !== '') {\n            $validFingerprint = false;",
)

# Centralize inline editor/new-file limits and require complete local staging writes.
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "final class RemoteOperations\n{\n    private const MAX_DEPTH = 40;",
    "final class RemoteOperations\n{\n    public const MAX_INLINE_CONTENT_BYTES = 4194304;\n\n    private const MAX_DEPTH = 40;",
)
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "    public function __construct(private readonly RemoteClientInterface $client)\n    {\n    }\n\n    public function createFile(string $path, string $content = ''): void\n    {\n        $path = PathGuard::ensureNotRoot($path);",
    "    public function __construct(private readonly RemoteClientInterface $client)\n    {\n    }\n\n    public static function assertInlineContentSize(string $content): void\n    {\n        if (strlen($content) > self::MAX_INLINE_CONTENT_BYTES) {\n            throw new RuntimeException('Sadržaj je prevelik za web editor. Maksimum je 4 MiB.');\n        }\n    }\n\n    public function createFile(string $path, string $content = ''): void\n    {\n        self::assertInlineContentSize($content);\n        $path = PathGuard::ensureNotRoot($path);",
)
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "    public function writeAtomic(string $remotePath, string $content): void\n    {\n        $tmp = $this->tempFile('write-');\n        try {\n            if (file_put_contents($tmp, $content, LOCK_EX) === false) throw new RuntimeException('Nije moguće pripremiti sadržaj za spremanje.');\n            $this->uploadAtomic($tmp, $remotePath);\n        } finally {\n            @unlink($tmp);\n        }\n    }",
    "    public function writeAtomic(string $remotePath, string $content): void\n    {\n        self::assertInlineContentSize($content);\n        $contentBytes = strlen($content);\n        \\GhostFTP_assert_temp_capacity($contentBytes);\n        $tmp = $this->tempFile('write-');\n        try {\n            $written = file_put_contents($tmp, $content, LOCK_EX);\n            if (!is_int($written) || $written !== $contentBytes) {\n                throw new RuntimeException('Nije moguće pripremiti cijeli sadržaj za spremanje.');\n            }\n            $this->uploadAtomic($tmp, $remotePath);\n        } finally {\n            @unlink($tmp);\n        }\n    }",
)
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "        foreach ($items as $item) {\n            if (!is_array($item)) continue;\n            $source = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));",
    "        foreach ($items as $item) {\n            if (!is_array($item)) {\n                throw new RuntimeException('Popis za batch preimenovanje sadrži neispravnu stavku.');\n            }\n            $source = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));",
)
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "            if ($source === $destination) continue;\n            $sources[$source] = true;\n            $plan[] = ['source'=>$source,'destination'=>$destination];",
    "            if ($source === $destination) continue;\n            if (isset($sources[$source])) {\n                throw new RuntimeException('Ista izvorna stavka ne smije biti navedena više puta u batch preimenovanju.');\n            }\n            $sources[$source] = true;\n            $plan[] = ['source'=>$source,'destination'=>$destination];",
)
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "                    // Only a fully validated archive is allowed to mutate remote state.\n                    // Only a fully validated and materialized archive is allowed to mutate remote state.",
    "                    // Only a fully validated and materialized archive is allowed to mutate remote state.",
)

# Reject oversized editor writes before any conditional remote read.
replace_once(
    "GhostFTP WEB/api.php",
    "            $content = (string)($_POST['content'] ?? '');\n            if (strlen($content) > 4194304) throw new RuntimeException('Sadržaj je prevelik za web editor. Maksimum je 4 MiB.');",
    "            $content = (string)($_POST['content'] ?? '');\n            RemoteOperations::assertInlineContentSize($content);",
)

# Fail closed for malformed destructive batch inputs instead of partially applying them.
replace_once(
    "GhostFTP WEB/api.php",
    "            $type = (string)($_POST['type'] ?? 'file') === 'dir' ? 'dir' : 'file';\n            $recursive = !isset($_POST['recursive']) || filter_var($_POST['recursive'], FILTER_VALIDATE_BOOLEAN);",
    "            $type = (string)($_POST['type'] ?? 'file');\n            if (!in_array($type, ['file', 'dir'], true)) throw new RuntimeException('Vrsta stavke za brisanje nije valjana.');\n            $recursive = !isset($_POST['recursive']) || filter_var($_POST['recursive'], FILTER_VALIDATE_BOOLEAN);",
)
replace_once(
    "GhostFTP WEB/api.php",
    "            foreach ($items as $item) {\n                if (!is_array($item)) continue;\n                $path = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));\n                $type = (string)($item['type'] ?? 'file') === 'dir' ? 'dir' : 'file';\n                $ops->deleteRecursive($path, $type);",
    "            foreach ($items as $item) {\n                if (!is_array($item)) throw new RuntimeException('Popis za skupno brisanje sadrži neispravnu stavku.');\n                $path = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));\n                $type = (string)($item['type'] ?? 'file');\n                if (!in_array($type, ['file', 'dir'], true)) throw new RuntimeException('Vrsta stavke za skupno brisanje nije valjana.');\n                $ops->deleteRecursive($path, $type);",
)

# SFTP upload integrity and private-key temp-file hygiene.
replace_once(
    "GhostFTP WEB/app/Remote/SftpClient.php",
    "        try {\n            $copied = stream_copy_to_stream($in, $out);\n            if ($copied === false) throw new RuntimeException('Upload nije uspio.');\n            $expected = is_array($sourceStat) ? (int)($sourceStat['size'] ?? -1) : -1;\n            if ($expected >= 0 && $copied !== $expected) throw new RuntimeException('Upload nije dovršen u cijelosti.');\n        } finally {\n            fclose($in);\n            fclose($out);\n        }",
    "        $expected = is_array($sourceStat) ? (int)($sourceStat['size'] ?? -1) : -1;\n        try {\n            $copied = stream_copy_to_stream($in, $out);\n            if ($copied === false) throw new RuntimeException('Upload nije uspio.');\n            if ($expected >= 0 && $copied !== $expected) throw new RuntimeException('Upload nije dovršen u cijelosti.');\n        } finally {\n            fclose($in);\n            fclose($out);\n        }\n        if ($expected >= 0) {\n            $remoteStat = @ssh2_sftp_lstat($this->sftp, $this->full($remotePath));\n            $remoteSize = is_array($remoteStat) && array_key_exists('size', $remoteStat) && is_numeric($remoteStat['size'])\n                ? (int)$remoteStat['size']\n                : -1;\n            if ($remoteSize >= 0 && $remoteSize !== $expected) {\n                throw new RuntimeException('Upload je završio s neočekivanom veličinom datoteke. Prijenos nije pouzdan.');\n            }\n        }",
)
replace_once(
    "GhostFTP WEB/app/Remote/SftpClient.php",
    "        try {\n            if (file_put_contents($pub, $publicKey . (str_ends_with($publicKey, \"\\n\") ? '' : \"\\n\"), LOCK_EX) === false || file_put_contents($priv, $privateKey . (str_ends_with($privateKey, \"\\n\") ? '' : \"\\n\"), LOCK_EX) === false) {\n                throw new RuntimeException('Nije moguće zapisati privremene SFTP ključeve.');\n            }\n            @chmod($pub, 0600);\n            @chmod($priv, 0600);",
    "        try {\n            @chmod($pub, 0600);\n            @chmod($priv, 0600);\n            $publicMaterial = $publicKey . (str_ends_with($publicKey, \"\\n\") ? '' : \"\\n\");\n            $privateMaterial = $privateKey . (str_ends_with($privateKey, \"\\n\") ? '' : \"\\n\");\n            $publicWritten = file_put_contents($pub, $publicMaterial, LOCK_EX);\n            $privateWritten = file_put_contents($priv, $privateMaterial, LOCK_EX);\n            if (!is_int($publicWritten) || $publicWritten !== strlen($publicMaterial)\n                || !is_int($privateWritten) || $privateWritten !== strlen($privateMaterial)) {\n                throw new RuntimeException('Nije moguće zapisati cijele privremene SFTP ključeve.');\n            }\n            @chmod($pub, 0600);\n            @chmod($priv, 0600);",
)

# Regression coverage: canonical SFTP profile policy, inline-content bounds and batch preflight.
replace_once(
    "GhostFTP WEB/tests/unit.php",
    "    'host_fingerprint' => '', 'auth_method' => 'password',",
    "    'host_fingerprint' => str_repeat('a', 64), 'auth_method' => 'password',",
)
replace_once(
    "GhostFTP WEB/tests/unit.php",
    "$bad = $base; $bad['host_fingerprint'] = ' SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';\nthrows(fn() => $method->invoke($store, $bad), 'fingerprint edge whitespace rejected');",
    "$bad = $base; $bad['host_fingerprint'] = ' SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';\nthrows(fn() => $method->invoke($store, $bad), 'fingerprint edge whitespace rejected');\n$bad = $base; $bad['host_fingerprint'] = '';\nthrows(fn() => $method->invoke($store, $bad), 'SFTP profile without pinned fingerprint rejected before persistence');",
)
replace_once(
    "GhostFTP WEB/tests/unit.php",
    "throws(fn() => HostGuard::connectionTargets(' example.com', true), 'host edge whitespace rejected');\n\n$reflection = new ReflectionClass(ProfileStore::class);",
    "throws(fn() => HostGuard::connectionTargets(' example.com', true), 'host edge whitespace rejected');\ncheck(strlen(str_repeat('x', RemoteOperations::MAX_INLINE_CONTENT_BYTES)) === RemoteOperations::MAX_INLINE_CONTENT_BYTES, 'inline editor limit constant is stable');\nRemoteOperations::assertInlineContentSize(str_repeat('x', RemoteOperations::MAX_INLINE_CONTENT_BYTES));\nthrows(fn() => RemoteOperations::assertInlineContentSize(str_repeat('x', RemoteOperations::MAX_INLINE_CONTENT_BYTES + 1)), 'inline editor content above 4 MiB rejected');\n\n$reflection = new ReflectionClass(ProfileStore::class);",
)
replace_once(
    "GhostFTP WEB/tests/unit.php",
    "$renameClient = new BatchRenameFakeClient(['/a' => 'A', '/x-a' => 'B'], 4);\n$renameOps = new RemoteOperations($renameClient);",
    "$preflightClient = new BatchRenameFakeClient(['/a' => 'A', '/b' => 'B'], 999);\n$preflightOps = new RemoteOperations($preflightClient);\nthrows(fn() => $preflightOps->batchRename([['path' => '/a'], 'invalid'], 'a', 'z', '', ''), 'batch rename rejects malformed rows before mutation');\ncheck($preflightClient->files() === ['/a' => 'A', '/b' => 'B'], 'malformed batch rename leaves remote state untouched');\nthrows(fn() => $preflightOps->batchRename([['path' => '/a'], ['path' => '/a']], 'a', 'z', '', ''), 'batch rename rejects duplicate source rows before mutation');\ncheck($preflightClient->files() === ['/a' => 'A', '/b' => 'B'], 'duplicate-source batch rename leaves remote state untouched');\n\n$renameClient = new BatchRenameFakeClient(['/a' => 'A', '/x-a' => 'B'], 4);\n$renameOps = new RemoteOperations($renameClient);",
)

# Static web audit must pin these security contracts, not merely rely on runtime tests.
replace_once(
    "scripts/audit_web.py",
    '        "GhostFTP WEB/app/Remote/PathGuard.php",\n        "GhostFTP WEB/app/Remote/FtpClient.php",',
    '        "GhostFTP WEB/app/Remote/PathGuard.php",\n        "GhostFTP WEB/app/Remote/ClientFactory.php",\n        "GhostFTP WEB/app/Remote/FtpClient.php",',
)
replace_once(
    "scripts/audit_web.py",
    "    profiles = read(\"GhostFTP WEB/app/Storage/ProfileStore.php\")\n    require(\n        profiles,\n        (\n            \"$rawHost !== trim($rawHost)\",\n            \"preg_match('/^[0-9]{1,5}$/', $rawPort)\",\n            \"PathGuard::normalizeRelative($basePath)\",\n        ),\n        \"app/Storage/ProfileStore.php\",\n    )",
    "    profiles = read(\"GhostFTP WEB/app/Storage/ProfileStore.php\")\n    require(\n        profiles,\n        (\n            \"$rawHost !== trim($rawHost)\",\n            \"preg_match('/^[0-9]{1,5}$/', $rawPort)\",\n            \"PathGuard::normalizeRelative($basePath)\",\n            \"$protocol === 'sftp' && $fingerprint === ''\",\n            \"SFTP zahtijeva SHA-256 host fingerprint prije spremanja ili povezivanja\",\n        ),\n        \"app/Storage/ProfileStore.php\",\n    )",
)
replace_once(
    "scripts/audit_web.py",
    "    ftp = read(\"GhostFTP WEB/app/Remote/FtpClient.php\")\n    require(ftp, (\"$this->profile['password'] = '';\", \"HostGuard::connectionTargets\"), \"app/Remote/FtpClient.php\")\n\n    sftp = read(\"GhostFTP WEB/app/Remote/SftpClient.php\")",
    "    client_factory = read(\"GhostFTP WEB/app/Remote/ClientFactory.php\")\n    require(\n        client_factory,\n        (\n            \"$protocol === 'sftp'\",\n            \"$profile['host_fingerprint']\",\n            \"SFTP zahtijeva SHA-256 host fingerprint prije povezivanja\",\n        ),\n        \"app/Remote/ClientFactory.php\",\n    )\n\n    ftp = read(\"GhostFTP WEB/app/Remote/FtpClient.php\")\n    require(ftp, (\"$this->profile['password'] = '';\", \"HostGuard::connectionTargets\"), \"app/Remote/FtpClient.php\")\n\n    sftp = read(\"GhostFTP WEB/app/Remote/SftpClient.php\")",
)
replace_once(
    "scripts/audit_web.py",
    '            "@unlink($pub)",\n            "@unlink($priv)",',
    '            "@unlink($pub)",\n            "@unlink($priv)",\n            "$publicWritten !== strlen($publicMaterial)",\n            "$privateWritten !== strlen($privateMaterial)",\n            "ssh2_sftp_lstat",',
)
replace_once(
    "scripts/audit_web.py",
    "    sftp = read(\"GhostFTP WEB/app/Remote/SftpClient.php\")",
    "    operations = read(\"GhostFTP WEB/app/Operations/RemoteOperations.php\")\n    require(\n        operations,\n        (\n            \"MAX_INLINE_CONTENT_BYTES = 4194304\",\n            \"assertInlineContentSize\",\n            \"$written !== $contentBytes\",\n            \"Ista izvorna stavka ne smije biti navedena više puta\",\n        ),\n        \"app/Operations/RemoteOperations.php\",\n    )\n\n    sftp = read(\"GhostFTP WEB/app/Remote/SftpClient.php\")",
)

# Documentation audit binds current-release markers and release-contract counts to VERSION.
replace_once(
    "scripts/audit_docs.py",
    "HTML_LINK_RE = re.compile(r\"\\b(?:href|src)\\s*=\\s*[\\\"']([^\\\"']+)[\\\"']\", re.IGNORECASE)\nVERSIONED_DOC_TITLE_RE = re.compile(r\"(?m)^#\\s+(?:Ghost FTP|GhostFTP)\\s+\\d+\\.\\d+\\.\\d+\\s+—\")",
    "HTML_LINK_RE = re.compile(r\"\\b(?:href|src)\\s*=\\s*[\\\"']([^\\\"']+)[\\\"']\", re.IGNORECASE)\nVERSIONED_DOC_TITLE_RE = re.compile(r\"(?m)^#\\s+(?:Ghost FTP|GhostFTP)\\s+\\d+\\.\\d+\\.\\d+\\s+—\")\nCURRENT_RELEASE_RE = re.compile(r\"\\*\\*Current Ghost FTP release:\\s*(\\d+\\.\\d+\\.\\d+)\\*\\*\")",
)
replace_once(
    "scripts/audit_docs.py",
    "    index_text = INDEX.read_text(encoding=\"utf-8\")\n    root_readme = (ROOT / \"README.md\").read_text(encoding=\"utf-8\")",
    "    version = (ROOT / \"VERSION\").read_text(encoding=\"utf-8\").strip()\n    if not re.fullmatch(r\"\\d+\\.\\d+\\.\\d+\", version):\n        fail(f\"invalid canonical VERSION: {version!r}\")\n\n    index_text = INDEX.read_text(encoding=\"utf-8\")\n    root_readme = (ROOT / \"README.md\").read_text(encoding=\"utf-8\")",
)
replace_once(
    "scripts/audit_docs.py",
    "    if not root_readme.startswith(\"# Ghost FTP\\n\"):\n        fail(\"root README does not use the Ghost FTP public title\")",
    "    if not root_readme.startswith(\"# Ghost FTP\\n\"):\n        fail(\"root README does not use the Ghost FTP public title\")\n    if f\"Current Ghost FTP version: **{version}**\" not in root_readme:\n        fail(\"root README current-version marker does not match VERSION\")\n    for path in files:\n        text = path.read_text(encoding=\"utf-8\")\n        for match in CURRENT_RELEASE_RE.finditer(text):\n            if match.group(1) != version:\n                fail(f\"stale current-release marker in {path.relative_to(ROOT)}: {match.group(1)} != {version}\")\n    security_doc = (DOCS / \"SECURITY.md\").read_text(encoding=\"utf-8\")\n    for marker in (\"10 platform artifacts\", \"13 public files\"):\n        if marker not in root_readme or marker not in security_doc:\n            fail(f\"release-contract documentation is missing current marker: {marker}\")",
)
replace_once(
    "scripts/audit_docs.py",
    '    print(f"DOCS_AUDIT=PASS ({len(files)} Markdown files, {len(detailed_docs)} detailed documents)")',
    '    print(f"DOCS_AUDIT=PASS ({version}; {len(files)} Markdown files, {len(detailed_docs)} detailed documents)")',
)

# Current documentation and release history.
replace_once("README.md", "Current Ghost FTP version: **1.0.2**", "Current Ghost FTP version: **1.0.3**")
replace_once(
    "README.md",
    "Ghost FTP 1.0.2 additionally bounds the in-memory Unix runtime-secret store so abandoned secret handles cannot grow without limit. The web credential envelope parser now rejects unsupported formats, oversized inputs and truncated authenticated-encryption payloads before decryption. Web runtime tests also enforce that human-readable Composer metadata cannot silently retain an obsolete release number.",
    "Ghost FTP 1.0.3 tightens Web/PWA fail-closed behavior around SFTP trust, inline editor staging, SFTP key material and batch mutations. SFTP profiles require a canonical pinned SHA-256 host fingerprint before persistence, editor/new-file writes share one 4 MiB boundary with complete local staging checks, SFTP uploads verify remote size when available, and malformed/duplicate batch rename input is rejected before remote mutation.",
)
replace_once(
    "README.md",
    "advances sequentially through patch releases such as `ghostftp-v1.0.1` and `ghostftp-v1.0.2`.",
    "advances sequentially through patch releases such as `ghostftp-v1.0.1`, `ghostftp-v1.0.2` and `ghostftp-v1.0.3`.",
)
replace_once(
    "README.md",
    "Patch releases advance sequentially (`1.0.0` → `1.0.1` → `1.0.2`),",
    "Patch releases advance sequentially (`1.0.0` → `1.0.1` → `1.0.2` → `1.0.3`),",
)

replace_once("docs/SECURITY.md", "**Current Ghost FTP release: 1.0.0**", "**Current Ghost FTP release: 1.0.3**")
replace_once("docs/SECURITY.md", "Ghost FTP 1.0.0 Setup uses", "Ghost FTP Setup uses")
replace_once(
    "docs/SECURITY.md",
    "It assembles exactly eight platform packages plus `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **11 public release files**.",
    "It assembles exactly **10 platform artifacts** plus `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **13 public files** total.",
)
replace_once(
    "docs/SECURITY.md",
    "- SFTP requires a pinned SHA-256 host fingerprint and verifies the connected server key against that pin.\n- Password changes/rehashes use generation-aware compare-and-swap behavior.",
    "- SFTP requires a pinned SHA-256 host fingerprint before a profile can be persisted and verifies the connected server key against that pin again at the client boundary.\n- Inline editor/new-file content is centrally bounded and local staging must be complete before any remote promotion.\n- SFTP key temp files are permission-restricted before key material is written, and uploads verify the resulting remote size when the server exposes it.\n- Destructive/batch mutation inputs fail closed before partial application when their shape or source set is invalid.\n- Password changes/rehashes use generation-aware compare-and-swap behavior.",
)

changelog = read("CHANGELOG.md")
if "## 1.0.3 - 2026-09-05" in changelog:
    raise SystemExit("PATCH_FAILED: CHANGELOG already contains 1.0.3")
entry = """## 1.0.3 - 2026-09-05

- Required a canonical pinned SHA-256 SFTP host fingerprint before Web profiles can be persisted, while retaining the connection-boundary pin requirement as defense in depth.
- Centralized the Web editor/new-file 4 MiB content limit and required complete local staging writes before atomic remote promotion.
- Hardened SFTP temporary key handling with restrictive permissions before key material is written and exact write-length checks.
- Added SFTP upload size verification against remote metadata when available so staging fails closed on incomplete transfers.
- Made batch rename and destructive Web API input validation reject malformed item types and duplicate sources before partial mutation.
- Removed duplicate archive-processing code commentary and expanded regression coverage for the new fail-closed boundaries.
- Bound documentation current-release markers and the 10-artifact/13-file release contract to the canonical VERSION audit.

"""
replace_once("CHANGELOG.md", "# Changelog\n\n", "# Changelog\n\n" + entry)

print("GHOST_FTP_1_0_3_PATCH=APPLIED")
