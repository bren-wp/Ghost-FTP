#!/usr/bin/env python3
"""Validate the ByFTP WEB source, privacy, version and shared-hosting contract."""

from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "ByFTP WEB"


def fail(message: str) -> None:
    raise SystemExit("WEB_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def require(text: str, marker: str, where: str) -> None:
    if marker not in text:
        fail(f"{where} is missing required marker: {marker}")


def tracked_web_files() -> list[str]:
    try:
        output = subprocess.check_output(
            ["git", "ls-files", "--", "ByFTP WEB"], cwd=ROOT, text=True
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"git ls-files failed for ByFTP WEB: {exc}")
    return [line for line in output.splitlines() if line]


def run_checked(command: list[str], *, label: str, cwd: Path = ROOT) -> None:
    try:
        result = subprocess.run(
            command,
            cwd=cwd,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=False,
        )
    except OSError as exc:
        fail(f"{label} could not start: {exc}")
    if result.returncode != 0:
        output = (result.stdout or "").strip()
        if len(output) > 5000:
            output = output[-5000:]
        fail(f"{label} failed with exit code {result.returncode}:\n{output}")


def run_runtime_checks() -> tuple[int, int]:
    php = shutil.which("php")
    node = shutil.which("node")
    if not php:
        fail("PHP CLI is required for ByFTP WEB validation")
    if not node:
        fail("Node.js is required for ByFTP WEB JavaScript syntax validation")

    run_checked([php, "-r", "if (PHP_VERSION_ID < 80100) { fwrite(STDERR, 'PHP 8.1+ required'); exit(2); }"], label="PHP version check")

    php_files = sorted(WEB.rglob("*.php"))
    js_files = sorted((WEB / "assets" / "js").glob("*.js"))
    if not php_files:
        fail("no PHP files found under ByFTP WEB")
    if not js_files:
        fail("no JavaScript files found under ByFTP WEB/assets/js")

    for path in php_files:
        run_checked([php, "-l", str(path)], label=f"PHP syntax: {path.relative_to(ROOT)}")
    run_checked([php, str(WEB / "tests" / "unit.php")], label="ByFTP WEB unit tests")
    for path in js_files:
        run_checked([node, "--check", str(path)], label=f"JavaScript syntax: {path.relative_to(ROOT)}")

    return len(php_files), len(js_files)


def main() -> int:
    version = read("VERSION").strip()
    web_version = read("ByFTP WEB/VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version) or web_version != version:
        fail(f"root/web version mismatch: {version!r} != {web_version!r}")

    required = {
        "ByFTP WEB/.htaccess", "ByFTP WEB/README.md", "ByFTP WEB/VERSION",
        "ByFTP WEB/index.php", "ByFTP WEB/api.php", "ByFTP WEB/setup.php",
        "ByFTP WEB/login.php", "ByFTP WEB/logout.php", "ByFTP WEB/register.php",
        "ByFTP WEB/account.php", "ByFTP WEB/users.php", "ByFTP WEB/settings.php",
        "ByFTP WEB/diagnostics.php", "ByFTP WEB/download.php",
        "ByFTP WEB/download-archive.php", "ByFTP WEB/preview.php",
        "ByFTP WEB/app/bootstrap.php", "ByFTP WEB/app/helpers.php",
        "ByFTP WEB/app/Operations/RemoteOperations.php",
        "ByFTP WEB/app/Remote/FtpClient.php", "ByFTP WEB/app/Remote/SftpClient.php",
        "ByFTP WEB/app/Remote/PathGuard.php", "ByFTP WEB/app/Security/HostGuard.php",
        "ByFTP WEB/app/Security/Auth.php", "ByFTP WEB/app/Security/Crypto.php",
        "ByFTP WEB/app/Storage/JsonStore.php", "ByFTP WEB/app/Storage/ProfileStore.php",
        "ByFTP WEB/app/Storage/UserStore.php", "ByFTP WEB/app/Storage/UserWorkspace.php",
        "ByFTP WEB/assets/css/app.css", "ByFTP WEB/assets/css/brendigo.css",
        "ByFTP WEB/assets/js/api.js", "ByFTP WEB/assets/js/app.js",
        "ByFTP WEB/assets/js/pwa.js", "ByFTP WEB/assets/js/settings.js",
        "ByFTP WEB/assets/js/utils.js", "ByFTP WEB/manifest.webmanifest",
        "ByFTP WEB/service-worker.js", "ByFTP WEB/robots.txt", "ByFTP WEB/tests/unit.php",
        "ByFTP WEB/storage/.htaccess",
    }
    tracked = set(tracked_web_files())
    missing = sorted(required - tracked)
    if missing:
        fail("required web files are not tracked: " + ", ".join(missing))

    composer = json.loads(read("ByFTP WEB/composer.json"))
    if composer.get("version") != version:
        fail("composer.json version does not match root VERSION")

    service_worker = read("ByFTP WEB/service-worker.js")
    require(service_worker, f"byftp-static-v{version}", "ByFTP WEB/service-worker.js")
    for marker in ("request.mode === 'navigate'", "api|login|logout|register|account|users|settings|setup|diagnostics|download|preview", "fetch(request)"):
        require(service_worker, marker, "ByFTP WEB/service-worker.js")
    if "cache.put(request" not in service_worker:
        fail("service worker has no explicit static-asset cache path")

    robots = read("ByFTP WEB/robots.txt")
    require(robots, "Disallow: /api", "ByFTP WEB/robots.txt")
    require(robots, "Disallow: /download/", "ByFTP WEB/robots.txt")
    require(robots, "Disallow: /preview/", "ByFTP WEB/robots.txt")
    require(read("ByFTP WEB/app/Views/head.php"), "noindex,nofollow,noarchive,nosnippet,noimageindex", "ByFTP WEB/app/Views/head.php")
    require(read("ByFTP WEB/app/bootstrap.php"), "X-Robots-Tag: noindex, nofollow, noarchive, nosnippet, noimageindex", "ByFTP WEB/app/bootstrap.php")

    path_guard = read("ByFTP WEB/app/Remote/PathGuard.php")
    for marker in ("str_contains($path, '\\\\')", "str_contains($path, '//')", "$part === '.' || $part === '..'", "ensureNotRoot"):
        require(path_guard, marker, "ByFTP WEB/app/Remote/PathGuard.php")
    if "str_replace('\\\\', '/', $path)" in path_guard:
        fail("web path guard rewrites unsafe backslashes instead of rejecting them")

    profile_store = read("ByFTP WEB/app/Storage/ProfileStore.php")
    for marker in ("$rawHost !== trim($rawHost)", "preg_match('/^[0-9]{1,5}$/', $rawPort)", "preg_match('/[\\r\\n\\x00]/', $username)", "PathGuard::normalizeRelative($basePath)"):
        require(profile_store, marker, "ByFTP WEB/app/Storage/ProfileStore.php")
    if "trim((string)($input['host']" in profile_store:
        fail("profile host is normalized before fail-closed validation")

    host_guard = read("ByFTP WEB/app/Security/HostGuard.php")
    for marker in ("connectionTargets", "FILTER_FLAG_NO_PRIV_RANGE", "FILTER_FLAG_NO_RES_RANGE", "localhost", "dns_get_record"):
        require(host_guard, marker, "ByFTP WEB/app/Security/HostGuard.php")

    ftp = read("ByFTP WEB/app/Remote/FtpClient.php")
    sftp = read("ByFTP WEB/app/Remote/SftpClient.php")
    require(ftp, "$this->profile['password'] = '';", "ByFTP WEB/app/Remote/FtpClient.php")
    for marker in ("$this->profile['password'] = '';", "$this->profile['private_key'] = '';", "$this->profile['key_passphrase'] = '';", "verifyHostFingerprint"):
        require(sftp, marker, "ByFTP WEB/app/Remote/SftpClient.php")

    auth = read("ByFTP WEB/app/Security/Auth.php")
    for marker in ("session_regenerate_id(true)", "user_session_version", "SameSite"):
        if marker == "SameSite":
            require(read("ByFTP WEB/app/bootstrap.php"), "'samesite' => 'Strict'", "ByFTP WEB/app/bootstrap.php")
        else:
            require(auth, marker, "ByFTP WEB/app/Security/Auth.php")

    index = read("ByFTP WEB/index.php")
    if "icon-192.png" in index or "icon-512.png" in index:
        fail("web UI references removed duplicate PNG PWA assets")
    require(index, "assets/images/mark.svg", "ByFTP WEB/index.php")
    require(index, "webkitdirectory", "ByFTP WEB/index.php")

    tests = read("ByFTP WEB/tests/unit.php")
    for marker in ("22junk", "profile traversal rejected", "host edge whitespace rejected", "credential protocol controls rejected"):
        require(tests, marker, "ByFTP WEB/tests/unit.php")

    allowed_storage = {
        "ByFTP WEB/storage/.htaccess",
        "ByFTP WEB/storage/logs/.gitkeep",
        "ByFTP WEB/storage/tmp/.gitkeep",
        "ByFTP WEB/storage/users/.gitkeep",
    }
    unexpected_storage = sorted(path for path in tracked if path.startswith("ByFTP WEB/storage/") and path not in allowed_storage)
    if unexpected_storage:
        fail("runtime/private web storage is tracked: " + ", ".join(unexpected_storage))

    legacy_version = re.compile(r"\b(?:1\.[012]\.\d+|[234]\.\d+\.\d+)\b")
    for path in sorted(tracked):
        if not path.endswith((".php", ".js", ".json", ".md", ".txt", ".webmanifest")):
            continue
        text = read(path)
        if legacy_version.search(text):
            fail(f"legacy/foreign active ByFTP version literal remains in {path}")

    php_count, js_count = run_runtime_checks()

    print(f"WEB_AUDIT=PASS ({version})")
    print(f"WEB_TRACKED_FILES={len(tracked)}")
    print(f"WEB_PHP_SYNTAX_FILES={php_count}")
    print(f"WEB_JS_SYNTAX_FILES={js_count}")
    print("WEB_UNIT_TESTS=PASS")
    print("WEB_VERSION_SOURCE=ROOT_AND_WEB_VERSION_MATCH")
    print("WEB_PWA_AUTHENTICATED_CACHE=BLOCKED")
    print("WEB_REMOTE_PATHS=FAIL_CLOSED")
    print("WEB_PRIVATE_HOSTS=BLOCKED_BY_DEFAULT")
    print("WEB_CREDENTIAL_LIFETIME=POST_AUTH_CLEARED")
    print("WEB_RUNTIME_STORAGE=NOT_TRACKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
