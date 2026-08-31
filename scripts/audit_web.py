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
            ["git", "ls-files", "--", "ByFTP WEB"],
            cwd=ROOT,
            text=True,
            encoding="utf-8",
            errors="replace",
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

    run_checked(
        [php, "-r", "if (PHP_VERSION_ID < 80100) { fwrite(STDERR, 'PHP 8.1+ required'); exit(2); }"],
        label="PHP version check",
    )

    php_files = sorted(WEB.rglob("*.php"))
    js_files = sorted((WEB / "assets" / "js").glob("*.js"))
    if not php_files:
        fail("no PHP files found under ByFTP WEB")
    if not js_files:
        fail("no JavaScript files found under ByFTP WEB/assets/js")

    for path in php_files:
        run_checked([php, "-l", str(path)], label=f"PHP syntax: {path.relative_to(ROOT)}")
    run_checked([php, str(WEB / "tests" / "unit.php")], label="ByFTP WEB unit tests")
    run_checked(
        [php, str(WEB / "tests" / "user-registry.php")],
        label="ByFTP WEB user registry fail-closed tests",
    )
    run_checked(
        [php, str(WEB / "tests" / "config-security.php")],
        label="ByFTP WEB config fail-closed tests",
    )
    run_checked(
        [php, str(WEB / "tests" / "rate-limiter.php")],
        label="ByFTP WEB atomic rate-limiter tests",
    )
    run_checked(
        [php, str(WEB / "tests" / "profile-recovery.php")],
        label="ByFTP WEB deleted-profile recovery tests",
    )
    for path in js_files:
        run_checked([node, "--check", str(path)], label=f"JavaScript syntax: {path.relative_to(ROOT)}")
    run_checked([node, "--check", str(WEB / "service-worker.js")], label="JavaScript syntax: ByFTP WEB/service-worker.js")

    return len(php_files), len(js_files) + 1


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
        "ByFTP WEB/app/Security/RateLimiter.php", "ByFTP WEB/app/Security/LoginRateLimitGate.php",
        "ByFTP WEB/app/Storage/JsonStore.php", "ByFTP WEB/app/Storage/ProfileStore.php",
        "ByFTP WEB/app/Storage/UserStore.php", "ByFTP WEB/app/Storage/UserWorkspace.php",
        "ByFTP WEB/assets/css/app.css", "ByFTP WEB/assets/css/brendigo.css",
        "ByFTP WEB/assets/js/api.js", "ByFTP WEB/assets/js/app.js",
        "ByFTP WEB/assets/js/pwa.js", "ByFTP WEB/assets/js/settings.js",
        "ByFTP WEB/assets/js/utils.js", "ByFTP WEB/manifest.webmanifest",
        "ByFTP WEB/service-worker.js", "ByFTP WEB/robots.txt", "ByFTP WEB/tests/unit.php",
        "ByFTP WEB/tests/user-registry.php", "ByFTP WEB/tests/config-security.php",
        "ByFTP WEB/tests/rate-limiter.php", "ByFTP WEB/tests/profile-recovery.php",
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
    for marker in (
        "request.mode === 'navigate'",
        "api|login|logout|register|account|users|settings|setup|diagnostics|download|preview",
        "fetch(request)",
    ):
        require(service_worker, marker, "ByFTP WEB/service-worker.js")
    if "cache.put(request" not in service_worker:
        fail("service worker has no explicit static-asset cache path")

    robots = read("ByFTP WEB/robots.txt")
    require(robots, "Disallow: /api", "ByFTP WEB/robots.txt")
    require(robots, "Disallow: /download/", "ByFTP WEB/robots.txt")
    require(robots, "Disallow: /preview/", "ByFTP WEB/robots.txt")
    require(
        read("ByFTP WEB/app/Views/head.php"),
        "noindex,nofollow,noarchive,nosnippet,noimageindex",
        "ByFTP WEB/app/Views/head.php",
    )
    require(
        read("ByFTP WEB/app/bootstrap.php"),
        "X-Robots-Tag: noindex, nofollow, noarchive, nosnippet, noimageindex",
        "ByFTP WEB/app/bootstrap.php",
    )

    path_guard = read("ByFTP WEB/app/Remote/PathGuard.php")
    for marker in ("str_contains($path, '\\\\')", "str_contains($path, '//')", "$part === '.' || $part === '..'", "ensureNotRoot"):
        require(path_guard, marker, "ByFTP WEB/app/Remote/PathGuard.php")
    if "str_replace('\\\\', '/', $path)" in path_guard:
        fail("web path guard rewrites unsafe backslashes instead of rejecting them")

    remote_operations = read("ByFTP WEB/app/Operations/RemoteOperations.php")
    zip_finalization = re.search(
        r"\$closed = \$zip->close\(\);.*?finally\s*\{\s*foreach \(\$temps as \$temp\) @unlink\(\$temp\);\s*\}\s*"
        r"if \(!\$closed\)\s*\{\s*@unlink\(\$tmp\);\s*throw new RuntimeException\('Nije moguće dovršiti ZIP arhivu\.'\);",
        remote_operations,
        re.DOTALL,
    )
    if zip_finalization is None:
        fail("WEB ZIP build does not fail closed when ZipArchive::close() fails")

    profile_store = read("ByFTP WEB/app/Storage/ProfileStore.php")
    for marker in (
        "$rawHost !== trim($rawHost)",
        "preg_match('/^[0-9]{1,5}$/', $rawPort)",
        "preg_match('/[\\r\\n\\x00]/', $username)",
        "PathGuard::normalizeRelative($basePath)",
        "new JsonStore(UserWorkspace::file($userId, 'profiles.json'), false)",
    ):
        require(profile_store, marker, "ByFTP WEB/app/Storage/ProfileStore.php")
    if "trim((string)($input['host']" in profile_store:
        fail("profile host is normalized before fail-closed validation")

    json_store = read("ByFTP WEB/app/Storage/JsonStore.php")
    require(json_store, "private readonly bool $recoverFromBackup = true", "ByFTP WEB/app/Storage/JsonStore.php")
    require(json_store, "if (!$this->recoverFromBackup)", "ByFTP WEB/app/Storage/JsonStore.php")

    user_store = read("ByFTP WEB/app/Storage/UserStore.php")
    require(
        user_store,
        "new JsonStore(BYFTP_STORAGE . '/users.json', false)",
        "ByFTP WEB/app/Storage/UserStore.php",
    )

    helpers = read("ByFTP WEB/app/helpers.php")
    for marker in (
        "new ByFTP\\Storage\\JsonStore($path, false)",
        "if (isset($GLOBALS['byftp_config_error']))",
        "Konfiguracija aplikacije nema valjan encryption ključ",
        "unset($GLOBALS['byftp_config_error']);",
    ):
        require(helpers, marker, "ByFTP WEB/app/helpers.php")

    rate_limiter = read("ByFTP WEB/app/Security/RateLimiter.php")
    for marker in (
        "public function consume(string $key): bool",
        "$this->store($key)->update(function (array $data)",
        "if ($count >= $this->maxAttempts)",
        "$data['count'] = $count + 1;",
    ):
        require(rate_limiter, marker, "ByFTP WEB/app/Security/RateLimiter.php")

    login_gate = read("ByFTP WEB/app/Security/LoginRateLimitGate.php")
    for marker in (
        "if (!$ipLimiter->consume($ipKey))",
        "return $accountLimiter->consume($accountKey);",
    ):
        require(login_gate, marker, "ByFTP WEB/app/Security/LoginRateLimitGate.php")

    host_guard = read("ByFTP WEB/app/Security/HostGuard.php")
    for marker in ("connectionTargets", "FILTER_FLAG_NO_PRIV_RANGE", "FILTER_FLAG_NO_RES_RANGE", "localhost", "dns_get_record"):
        require(host_guard, marker, "ByFTP WEB/app/Security/HostGuard.php")

    ftp = read("ByFTP WEB/app/Remote/FtpClient.php")
    sftp = read("ByFTP WEB/app/Remote/SftpClient.php")
    require(ftp, "$this->profile['password'] = '';", "ByFTP WEB/app/Remote/FtpClient.php")
    for marker in (
        "$this->profile['password'] = '';",
        "$this->profile['private_key'] = '';",
        "$this->profile['key_passphrase'] = '';",
        "verifyHostFingerprint",
    ):
        require(sftp, marker, "ByFTP WEB/app/Remote/SftpClient.php")

    auth = read("ByFTP WEB/app/Security/Auth.php")
    for marker in ("session_regenerate_id(true)", "user_session_version"):
        require(auth, marker, "ByFTP WEB/app/Security/Auth.php")
    require(read("ByFTP WEB/app/bootstrap.php"), "'samesite' => 'Strict'", "ByFTP WEB/app/bootstrap.php")

    login = read("ByFTP WEB/login.php")
    for marker in (
        "function byftp_clear_login_rate_limiters(",
        "auth.rate_limit_clear_failed",
        "LoginRateLimitGate::consume($ipLimiter, $ipKey, $accountLimiter, $accountKey)",
        "auth.rate_limit_consume_failed",
        "if (!Auth::attempt($email, $password))",
        "$migrationFailed = false;",
        "if (!$migrationFailed)",
    ):
        require(login, marker, "ByFTP WEB/login.php")
    if "$accountLimiter->clear(" in login or "$ipLimiter->clear(" in login:
        fail("login performs direct post-auth limiter cleanup outside the fail-soft helper")
    if "$ipLimiter->consume(" in login or "$accountLimiter->consume(" in login:
        fail("login bypasses the ordered IP-first rate-limit gate")
    if "->blocked(" in login or "->hit(" in login:
        fail("login uses split rate-limit check/hit operations instead of atomic pre-auth consume")
    if login.count("Auth::logout();") != 1:
        fail("login must invalidate an authenticated session only for the actual legacy migration failure path")

    register = read("ByFTP WEB/register.php")
    for marker in (
        "$limiter->consume($key)",
        "auth.registration_rate_limit_consume_failed",
        "auth.registration_blocked",
    ):
        require(register, marker, "ByFTP WEB/register.php")
    if "->blocked(" in register or "->hit(" in register:
        fail("registration uses split rate-limit check/hit operations instead of atomic consume")

    setup = read("ByFTP WEB/setup.php")
    for marker in (
        "$configRecoveryRequired = isset($GLOBALS['byftp_config_error']);",
        "$existingDataDetected = $configRecoveryRequired || $hasStoredData();",
        "$setupTransactionStarted = false;",
        "if (isset($GLOBALS['byftp_config_error']))",
        "if ($setupTransactionStarted)",
    ):
        require(setup, marker, "ByFTP WEB/setup.php")
    rollback_match = re.search(r"\$rollbackArtifacts\s*=\s*\[(.*?)\];", setup, re.DOTALL)
    if rollback_match is None:
        fail("ByFTP WEB/setup.php has no explicit failed-setup artifact rollback list")
    rollback_artifacts = rollback_match.group(1)
    for marker in (
        "byftp_config_path()",
        "byftp_config_path() . '.bak'",
        "byftp_config_path() . '.lock'",
        "BYFTP_STORAGE . '/users.json'",
        "BYFTP_STORAGE . '/users.json.bak'",
        "BYFTP_STORAGE . '/users.json.lock'",
    ):
        require(rollback_artifacts, marker, "ByFTP WEB/setup.php rollback artifacts")
    require(setup, "foreach ($rollbackArtifacts as $artifact)", "ByFTP WEB/setup.php")
    require(setup, "unset($GLOBALS['byftp_config_error']);", "ByFTP WEB/setup.php")

    index = read("ByFTP WEB/index.php")
    if "icon-192.png" in index or "icon-512.png" in index:
        fail("web UI references removed duplicate PNG PWA assets")
    require(index, "byftp_asset('images/mark.svg')", "ByFTP WEB/index.php")
    require(index, "webkitdirectory", "ByFTP WEB/index.php")

    tests = read("ByFTP WEB/tests/unit.php")
    for marker in ("22junk", "profile traversal rejected", "host edge whitespace rejected", "credential protocol controls rejected"):
        require(tests, marker, "ByFTP WEB/tests/unit.php")

    registry_tests = read("ByFTP WEB/tests/user-registry.php")
    for marker in (
        "old-password-123",
        "{corrupt-user-registry",
        "fails closed instead of authenticating from stale backup",
        "generic JsonStore recovery remains available",
    ):
        require(registry_tests, marker, "ByFTP WEB/tests/user-registry.php")

    config_tests = read("ByFTP WEB/tests/config-security.php")
    for marker in (
        "{corrupt-app-config",
        "runtime config does not recover stale app.json backup automatically",
        "config update is blocked while the primary config is corrupt",
        "generic JsonStore still exposes backup data for explicit operator recovery",
        "backup-only stale config does not silently configure the application",
    ):
        require(config_tests, marker, "ByFTP WEB/tests/config-security.php")

    limiter_tests = read("ByFTP WEB/tests/rate-limiter.php")
    for marker in (
        "first attempt is atomically admitted",
        "attempt after configured budget is atomically rejected",
        "rejected consume does not inflate persisted attempt count",
        "blocked IP does not create or consume an account-specific rate-limit state",
        "ordered login gate consumes account budget exactly once for admitted IP",
        "atomic consume fails closed when primary rate-limit state is corrupt",
    ):
        require(limiter_tests, marker, "ByFTP WEB/tests/rate-limiter.php")

    profile_recovery_tests = read("ByFTP WEB/tests/profile-recovery.php")
    for marker in (
        "backup intentionally contains the deleted encrypted profile",
        "fails closed instead of resurrecting deleted credentials from stale backup",
        "generic JsonStore still exposes profile backup for explicit operator recovery",
        "fails closed when only stale profiles.json.bak remains",
    ):
        require(profile_recovery_tests, marker, "ByFTP WEB/tests/profile-recovery.php")

    allowed_storage = {
        "ByFTP WEB/storage/.htaccess",
        "ByFTP WEB/storage/logs/.gitkeep",
        "ByFTP WEB/storage/tmp/.gitkeep",
        "ByFTP WEB/storage/users/.gitkeep",
    }
    unexpected_storage = sorted(
        path for path in tracked if path.startswith("ByFTP WEB/storage/") and path not in allowed_storage
    )
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
    print("WEB_USER_REGISTRY_RECOVERY=FAIL_CLOSED")
    print("WEB_CONFIG_SECURITY_POLICY_RECOVERY=FAIL_CLOSED")
    print("WEB_PROFILE_CREDENTIAL_RECOVERY=FAIL_CLOSED")
    print("WEB_LOGIN_RATE_LIMIT_ATTEMPTS=ATOMIC_PRE_AUTH")
    print("WEB_LOGIN_RATE_LIMIT_ORDER=IP_THEN_ACCOUNT_SHORT_CIRCUIT")
    print("WEB_POST_AUTH_LIMITER_RESET=FAIL_SOFT")
    print("WEB_SUBPROCESS_TEXT_ENCODING=UTF8_REPLACE")
    print("WEB_VERSION_SOURCE=ROOT_AND_WEB_VERSION_MATCH")
    print("WEB_PWA_AUTHENTICATED_CACHE=BLOCKED")
    print("WEB_REMOTE_PATHS=FAIL_CLOSED")
    print("WEB_ZIP_FINALIZATION=FAIL_CLOSED")
    print("WEB_PRIVATE_HOSTS=BLOCKED_BY_DEFAULT")
    print("WEB_CREDENTIAL_LIFETIME=POST_AUTH_CLEARED")
    print("WEB_SETUP_FAILED_TRANSACTION_ARTIFACTS=CLEANED")
    print("WEB_RUNTIME_STORAGE=NOT_TRACKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
