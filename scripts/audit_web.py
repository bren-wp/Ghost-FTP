#!/usr/bin/env python3
"""Fail-closed Ghost FTP web/PWA source, runtime, privacy and security audit."""

from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "GhostFTP WEB"  # Legacy source-directory name retained for compatibility.

# These empty files keep runtime-created directories present in source/deploy
# archives. They are the only tracked entries allowed below storage/ besides
# storage/.htaccess; any actual runtime/user data remains release-blocking.
ALLOWED_STORAGE_PLACEHOLDERS = {
    "GhostFTP WEB/storage/logs/.gitkeep",
    "GhostFTP WEB/storage/tmp/.gitkeep",
    "GhostFTP WEB/storage/users/.gitkeep",
}


def fail(message: str) -> None:
    raise SystemExit("WEB_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeError as exc:
        fail(f"{rel} is not valid UTF-8: {exc}")


def require(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{where} is missing required security/runtime marker: {marker}")


def tracked_web_files() -> list[str]:
    try:
        output = subprocess.check_output(
            ["git", "ls-files", "--", "GhostFTP WEB"],
            cwd=ROOT,
            text=True,
            encoding="utf-8",
            errors="replace",
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"git ls-files failed for web source: {exc}")
    return [line for line in output.splitlines() if line]


def run_checked(command: list[str], *, label: str) -> None:
    try:
        result = subprocess.run(
            command,
            cwd=ROOT,
            text=True,
            encoding="utf-8",
            errors="replace",
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=False,
        )
    except OSError as exc:
        fail(f"{label} could not start: {exc}")
    if result.returncode != 0:
        output = (result.stdout or "").strip()
        if len(output) > 6000:
            output = output[-6000:]
        fail(f"{label} failed with exit code {result.returncode}:\n{output}")


def run_runtime_checks() -> tuple[int, int, int]:
    php = shutil.which("php")
    node = shutil.which("node")
    if not php:
        fail("PHP CLI is required for Ghost FTP web validation")
    if not node:
        fail("Node.js is required for Ghost FTP web JavaScript syntax validation")

    run_checked(
        [php, "-r", "if (PHP_VERSION_ID < 80100) { fwrite(STDERR, 'PHP 8.1+ required'); exit(2); }"],
        label="PHP version check",
    )

    php_files = sorted(WEB.rglob("*.php"))
    js_files = sorted((WEB / "assets" / "js").glob("*.js"))
    test_files = sorted((WEB / "tests").glob("*.php"))
    if not php_files:
        fail("no PHP files found under web source")
    if not js_files:
        fail("no JavaScript files found under web assets")
    if not test_files:
        fail("no web runtime regression tests found")

    for path in php_files:
        run_checked([php, "-l", str(path)], label=f"PHP syntax: {path.relative_to(ROOT)}")
    for path in js_files:
        run_checked([node, "--check", str(path)], label=f"JavaScript syntax: {path.relative_to(ROOT)}")
    run_checked([node, "--check", str(WEB / "service-worker.js")], label="JavaScript syntax: service-worker.js")

    for path in test_files:
        run_checked([php, str(path)], label=f"Ghost FTP web test: {path.name}")

    return len(php_files), len(js_files) + 1, len(test_files)


def validate_repository_surface(version: str) -> None:
    tracked = set(tracked_web_files())
    required = {
        "GhostFTP WEB/.htaccess",
        "GhostFTP WEB/README.md",
        "GhostFTP WEB/VERSION",
        "GhostFTP WEB/composer.json",
        "GhostFTP WEB/index.php",
        "GhostFTP WEB/api.php",
        "GhostFTP WEB/setup.php",
        "GhostFTP WEB/login.php",
        "GhostFTP WEB/logout.php",
        "GhostFTP WEB/register.php",
        "GhostFTP WEB/account.php",
        "GhostFTP WEB/settings.php",
        "GhostFTP WEB/download.php",
        "GhostFTP WEB/preview.php",
        "GhostFTP WEB/app/bootstrap.php",
        "GhostFTP WEB/app/helpers.php",
        "GhostFTP WEB/app/Remote/PathGuard.php",
        "GhostFTP WEB/app/Remote/ClientFactory.php",
        "GhostFTP WEB/app/Remote/FtpClient.php",
        "GhostFTP WEB/app/Remote/SftpClient.php",
        "GhostFTP WEB/app/Security/Auth.php",
        "GhostFTP WEB/app/Security/HostGuard.php",
        "GhostFTP WEB/app/Security/RateLimiter.php",
        "GhostFTP WEB/app/Storage/JsonStore.php",
        "GhostFTP WEB/app/Storage/ProfileStore.php",
        "GhostFTP WEB/manifest.webmanifest",
        "GhostFTP WEB/service-worker.js",
        "GhostFTP WEB/robots.txt",
        "GhostFTP WEB/storage/.htaccess",
    }
    missing = sorted(required - tracked)
    if missing:
        fail("required web files are not tracked: " + ", ".join(missing))

    for placeholder in sorted(ALLOWED_STORAGE_PLACEHOLDERS):
        if placeholder not in tracked:
            fail(f"required empty storage placeholder is not tracked: {placeholder}")
        path = ROOT / placeholder
        if not path.is_file() or path.stat().st_size != 0:
            fail(f"storage placeholder must remain an empty regular file: {placeholder}")

    runtime_storage = sorted(
        path
        for path in tracked
        if path.startswith("GhostFTP WEB/storage/")
        and path != "GhostFTP WEB/storage/.htaccess"
        and path not in ALLOWED_STORAGE_PLACEHOLDERS
    )
    if runtime_storage:
        fail("runtime/user storage must not be tracked: " + ", ".join(runtime_storage))

    forbidden_suffixes = (".log", ".tmp", ".bak", ".sqlite", ".sqlite3")
    leaked = sorted(path for path in tracked if path.lower().endswith(forbidden_suffixes))
    if leaked:
        fail("generated/runtime file is tracked: " + ", ".join(leaked))

    composer = json.loads(read("GhostFTP WEB/composer.json"))
    if composer.get("name") != "brendigo/ghost-ftp-web":
        fail("composer package name is not Ghost FTP")
    if composer.get("version") != version:
        fail("composer version does not match canonical VERSION")
    if "Ghost FTP" not in str(composer.get("description", "")):
        fail("composer description does not use Ghost FTP branding")


def validate_public_brand_and_pwa(version: str) -> None:
    manifest = read("GhostFTP WEB/manifest.webmanifest")
    require(manifest, ('"name": "Ghost FTP Remote File Client"', '"short_name": "Ghost FTP"'), "manifest.webmanifest")

    service_worker = read("GhostFTP WEB/service-worker.js")
    require(
        service_worker,
        (
            f"ghostftp-static-v{version}",
            "key.startsWith('GhostFTP-static-')",
            "request.mode === 'navigate'",
            "api|login|logout|register|account|users|settings|setup|diagnostics|download|preview",
            "fetch(request)",
            "cache.put(request",
        ),
        "service-worker.js",
    )
    if f"GhostFTP-static-v{version}" in service_worker:
        fail("current PWA cache namespace still uses the retired public brand")

    readme = read("GhostFTP WEB/README.md")
    if "Ghost FTP" not in readme:
        fail("web README does not identify Ghost FTP")


def validate_http_session_and_csrf_boundaries() -> None:
    bootstrap = read("GhostFTP WEB/app/bootstrap.php")
    require(
        bootstrap,
        (
            "GhostFTP_ROOT . '/VERSION'",
            "HTTP_SEC_FETCH_SITE",
            "X-Content-Type-Options: nosniff",
            "X-Frame-Options: DENY",
            "X-Robots-Tag: noindex, nofollow, noarchive, nosnippet, noimageindex",
            "Referrer-Policy: no-referrer",
            "Content-Security-Policy:",
            "Strict-Transport-Security: max-age=31536000",
            "session.use_strict_mode",
            "session.use_only_cookies",
            "'httponly' => true",
            "'samesite' => 'Strict'",
            "session_regenerate_id(true)",
        ),
        "app/bootstrap.php",
    )

    auth = read("GhostFTP WEB/app/Security/Auth.php")
    require(
        auth,
        (
            "session_regenerate_id(true)",
            "user_session_version",
            "unset($_SESSION['csrf'])",
            "GhostFTP_csrf_token()",
            "unset($user['password_hash'])",
        ),
        "app/Security/Auth.php",
    )

    helpers = read("GhostFTP WEB/app/helpers.php")
    require(helpers, ("function GhostFTP_csrf_token", "hash_equals", "random_bytes", "function GhostFTP_public_error", "function GhostFTP_public_error_status", "Neočekivana interna pogreška"), "app/helpers.php")

    api = read("GhostFTP WEB/api.php")
    require(api, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)", "exception'=>get_class($e)"), "api.php")
    if "GhostFTP_json(['ok'=>false,'error'=>$e->getMessage()]" in api:
        fail("API exposes raw unexpected exception messages")

    for endpoint in ("download.php", "download-archive.php", "preview.php"):
        endpoint_source = read(f"GhostFTP WEB/{endpoint}")
        require(endpoint_source, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)"), endpoint)


def validate_remote_input_and_secret_boundaries() -> None:
    path_guard = read("GhostFTP WEB/app/Remote/PathGuard.php")
    require(
        path_guard,
        (
            "str_contains($path, '\\\\')",
            "str_contains($path, '//')",
            "$part === '.' || $part === '..'",
            "ensureNotRoot",
        ),
        "app/Remote/PathGuard.php",
    )
    if "str_replace('\\\\', '/', $path)" in path_guard:
        fail("path guard rewrites unsafe backslashes instead of rejecting them")

    host_guard = read("GhostFTP WEB/app/Security/HostGuard.php")
    require(
        host_guard,
        (
            "$host !== trim($host)",
            "FILTER_FLAG_NO_PRIV_RANGE",
            "FILTER_FLAG_NO_RES_RANGE",
        ),
        "app/Security/HostGuard.php",
    )

    profiles = read("GhostFTP WEB/app/Storage/ProfileStore.php")
    require(
        profiles,
        (
            "$rawHost !== trim($rawHost)",
            "preg_match('/^[0-9]{1,5}$/', $rawPort)",
            "PathGuard::normalizeRelative($basePath)",
            "$protocol === 'sftp' && $fingerprint === ''",
            "SFTP zahtijeva SHA-256 host fingerprint prije spremanja ili povezivanja",
        ),
        "app/Storage/ProfileStore.php",
    )
    if "trim((string)($input['host']" in profiles:
        fail("profile host is normalized before fail-closed validation")

    client_factory = read("GhostFTP WEB/app/Remote/ClientFactory.php")
    require(
        client_factory,
        (
            "$protocol === 'sftp'",
            "$profile['host_fingerprint']",
            "SFTP zahtijeva SHA-256 host fingerprint prije povezivanja",
        ),
        "app/Remote/ClientFactory.php",
    )

    ftp = read("GhostFTP WEB/app/Remote/FtpClient.php")
    require(ftp, ("$this->profile['password'] = '';", "HostGuard::connectionTargets"), "app/Remote/FtpClient.php")

    operations = read("GhostFTP WEB/app/Operations/RemoteOperations.php")
    require(
        operations,
        (
            "MAX_INLINE_CONTENT_BYTES = 4194304",
            "assertInlineContentSize",
            "$written !== $contentBytes",
            "Ista izvorna stavka ne smije biti navedena više puta",
        ),
        "app/Operations/RemoteOperations.php",
    )

    api = read("GhostFTP WEB/api.php")
    require(
        api,
        (
            "$uploadPlan = [];",
            "Upload zahtjev sadrži neusklađene metapodatke datoteka.",
            "Ista privremena upload datoteka ne smije biti navedena više puta.",
            "Više upload datoteka ne smije ciljati istu udaljenu putanju.",
            "foreach ($uploadPlan as $item)",
        ),
        "api.php",
    )
    helpers = read("GhostFTP WEB/app/helpers.php")
    require(helpers, ("Popis putanja sadrži neispravnu stavku.",), "app/helpers.php")

    sftp = read("GhostFTP WEB/app/Remote/SftpClient.php")
    require(
        sftp,
        (
            "verifyHostFingerprint",
            "SSH2_FINGERPRINT_SHA256",
            "hash_equals",
            "$this->profile['password'] = '';",
            "$this->profile['private_key'] = '';",
            "$this->profile['key_passphrase'] = '';",
            "@unlink($pub)",
            "@unlink($priv)",
            "$publicWritten !== strlen($publicMaterial)",
            "$privateWritten !== strlen($privateMaterial)",
            "DIRECTORY_SEPARATOR === '/'",
            "dozvolama 0600",
            "ssh2_sftp_lstat",
        ),
        "app/Remote/SftpClient.php",
    )


def validate_noindex_and_storage_protection() -> None:
    robots = read("GhostFTP WEB/robots.txt")
    require(robots, ("Disallow: /api", "Disallow: /download/", "Disallow: /preview/"), "robots.txt")

    storage_htaccess = read("GhostFTP WEB/storage/.htaccess")
    if not re.search(r"(?i)(deny\s+from\s+all|require\s+all\s+denied)", storage_htaccess):
        fail("storage/.htaccess does not deny direct HTTP access")


def main() -> int:
    version = read("VERSION").strip()
    web_version = read("GhostFTP WEB/VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid canonical VERSION: {version!r}")
    if web_version != version:
        fail(f"root/web version mismatch: {version!r} != {web_version!r}")

    validate_repository_surface(version)
    validate_public_brand_and_pwa(version)
    validate_http_session_and_csrf_boundaries()
    validate_remote_input_and_secret_boundaries()
    validate_noindex_and_storage_protection()
    php_count, js_count, test_count = run_runtime_checks()

    print(f"WEB_AUDIT=PASS ({version})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("WEB_RUNTIME_STORAGE=BLOCKED_EXCEPT_EMPTY_PLACEHOLDERS")
    print("WEB_PWA_CACHE_NAMESPACE=ghostftp-static")
    print("WEB_LEGACY_CACHE_MIGRATION=GhostFTP-static-cleanup")
    print(f"WEB_PHP_FILES={php_count}")
    print(f"WEB_JS_FILES={js_count}")
    print(f"WEB_RUNTIME_TESTS={test_count}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
