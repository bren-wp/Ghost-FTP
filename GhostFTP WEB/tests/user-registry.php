<?php
declare(strict_types=1);

use ByFTP\Security\RateLimiter;
use ByFTP\Storage\JsonStore;
use ByFTP\Storage\UserStore;
use ByFTP\Storage\UserWorkspace;

$storage = sys_get_temp_dir() . '/byftp-user-registry-' . bin2hex(random_bytes(6));
$externalTarget = sys_get_temp_dir() . '/byftp-user-delete-target-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $storage);

require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Storage/UserWorkspace.php';
require __DIR__ . '/../app/Storage/UserStore.php';
require __DIR__ . '/../app/Security/RateLimiter.php';

$failed = false;

function registry_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function registry_throws(callable $callback, string $label): void
{
    try {
        $callback();
        registry_check(false, $label);
    } catch (Throwable) {
        registry_check(true, $label);
    }
}

try {
    $users = new UserStore();

    // Password writes use compare-and-swap against the hash generation that was actually
    // verified. This deterministic regression models two requests that both verified the
    // same old password before either write: only the first may commit.
    $casUser = $users->create(
        'Password CAS User',
        'password-cas@example.com',
        'password-cas-old-123',
        'user'
    );
    $casId = (string)($casUser['id'] ?? '');
    $casBefore = $users->findById($casId);
    $casOldHash = is_array($casBefore) ? (string)($casBefore['password_hash'] ?? '') : '';
    $casVersion = is_array($casBefore) ? (int)($casBefore['session_version'] ?? 1) : 0;
    $casHashA = password_hash('password-cas-new-a-456', PASSWORD_DEFAULT);
    $casHashB = password_hash('password-cas-new-b-789', PASSWORD_DEFAULT);
    registry_check(
        $casOldHash !== '' && is_string($casHashA) && is_string($casHashB),
        'password CAS regression fixture has valid hash generations'
    );

    $replacePasswordHash = (new ReflectionClass(UserStore::class))->getMethod('replacePasswordHash');
    $replacePasswordHash->setAccessible(true);
    $replacePasswordHash->invoke($users, $casId, $casHashA, $casOldHash);
    registry_throws(
        fn() => $replacePasswordHash->invoke($users, $casId, $casHashB, $casOldHash),
        'stale verified password hash cannot overwrite a newer committed password generation'
    );
    $casAfter = $users->findById($casId);
    registry_check(
        is_array($casAfter)
            && password_verify('password-cas-new-a-456', (string)($casAfter['password_hash'] ?? ''))
            && !password_verify('password-cas-new-b-789', (string)($casAfter['password_hash'] ?? '')),
        'failed stale CAS leaves the first committed password intact'
    );
    registry_check(
        is_array($casAfter) && (int)($casAfter['session_version'] ?? 0) === $casVersion + 1,
        'failed stale CAS does not increment session version a second time'
    );

    // Authentication must be bound to the same generation too. A request that verified the
    // old hash before the password change may not publish a session after the new hash wins.
    $completeAuthentication = (new ReflectionClass(UserStore::class))->getMethod('completeAuthentication');
    $completeAuthentication->setAccessible(true);
    registry_throws(
        fn() => $completeAuthentication->invoke($users, $casId, $casOldHash, null),
        'authentication completion rejects a password hash generation changed after verification'
    );
    $afterStaleAuth = $users->findById($casId);
    registry_check(
        is_array($afterStaleAuth) && ($afterStaleAuth['last_login_at'] ?? null) === null,
        'stale authentication generation is not published as a successful login'
    );

    $currentHash = is_array($afterStaleAuth) ? (string)($afterStaleAuth['password_hash'] ?? '') : '';
    $authenticatedCurrent = $completeAuthentication->invoke($users, $casId, $currentHash, null);
    registry_check(
        is_array($authenticatedCurrent) && !empty($authenticatedCurrent['last_login_at']),
        'current password generation can complete authentication'
    );
    $afterCurrentAuth = $users->findById($casId);
    registry_check(
        is_array($afterCurrentAuth)
            && (int)($afterCurrentAuth['session_version'] ?? 0) === $casVersion + 1,
        'authentication without rehash does not change the session generation'
    );

    // These workspace deletion regressions intentionally exercise POSIX symlink and
    // directory-permission semantics used by shared-hosting deployments. Windows runners
    // still lint and execute the cross-platform registry tests below, but do not provide
    // deterministic symlink privileges or chmod-based unlink denial.
    if (PHP_OS_FAMILY !== 'Windows') {
        // A workspace root is normally a real private directory. If another local process
        // replaces it with a symlink, deletion must unlink only the symlink and never recurse
        // into the external target.
        $symlinkUser = $users->create(
            'Symlink User',
            'symlink@example.com',
            'symlink-password-123',
            'user'
        );
        $symlinkId = (string)($symlinkUser['id'] ?? '');
        $symlinkWorkspace = UserWorkspace::directory($symlinkId);
        @rmdir($symlinkWorkspace);
        @mkdir($externalTarget, 0700, true);
        $externalSentinel = $externalTarget . '/sentinel.txt';
        file_put_contents($externalSentinel, 'must-survive-user-delete');
        $linkCreated = @symlink($externalTarget, $symlinkWorkspace);
        registry_check($linkCreated && is_link($symlinkWorkspace), 'test workspace root symlink is created');
        if ($linkCreated) {
            $users->delete($symlinkId);
            registry_check(is_file($externalSentinel), 'user deletion never traverses workspace root symlink target');
            registry_check(!is_link($symlinkWorkspace), 'workspace root symlink itself is removed');
            registry_check($users->findById($symlinkId) === null, 'symlink workspace user is removed after safe cleanup');
        }

        // Force a cleanup failure with a non-writable nested directory. The account must stay
        // in the registry as inactive/deleting so an administrator has a deterministic retry
        // path instead of an invisible orphaned private workspace.
        $retryUser = $users->create(
            'Retry Delete User',
            'retry-delete@example.com',
            'retry-password-123',
            'user'
        );
        $retryId = (string)($retryUser['id'] ?? '');
        $retryWorkspace = UserWorkspace::directory($retryId);
        $lockedDir = $retryWorkspace . '/locked';
        @mkdir($lockedDir, 0700, true);
        file_put_contents($lockedDir . '/private.json', '{"private":true}');
        @chmod($lockedDir, 0500);

        $deleteFailed = false;
        try {
            $users->delete($retryId);
        } catch (Throwable) {
            $deleteFailed = true;
        }
        @chmod($lockedDir, 0700);

        registry_check($deleteFailed, 'workspace cleanup failure is surfaced to the caller');
        $pendingDelete = $users->findById($retryId);
        registry_check(
            is_array($pendingDelete) && !empty($pendingDelete['deleting']) && empty($pendingDelete['active']),
            'failed workspace cleanup keeps an inactive retryable deleting registry row'
        );

        $users->delete($retryId);
        registry_check($users->findById($retryId) === null, 'retry completes registry deletion after workspace cleanup succeeds');
        registry_check(!is_dir($retryWorkspace) && !is_link($retryWorkspace), 'retry removes the complete private workspace');
    }

    // Rate-limit counters are security state too. Build a valid backup containing a lower
    // attempt count, then corrupt/remove the primary. The limiter must never roll back to
    // that older count because doing so would weaken brute-force protection.
    $rateKey = 'login:203.0.113.50';
    $limiter = new RateLimiter(3, 3600);
    $limiter->hit($rateKey);
    $limiter->hit($rateKey);
    $ratePath = BYFTP_STORAGE . '/logs/rl-' . hash('sha256', $rateKey) . '.json';
    $rateBackup = $ratePath . '.bak';
    $rateBackupRaw = @file_get_contents($rateBackup);
    $rateBackupData = is_string($rateBackupRaw) ? json_decode($rateBackupRaw, true) : null;
    registry_check(
        is_array($rateBackupData) && (int)($rateBackupData['count'] ?? 0) === 1,
        'rate-limit backup contains the prior lower attempt count used by the regression scenario'
    );

    file_put_contents($ratePath, '{corrupt-rate-limit-state');
    registry_throws(
        fn() => $limiter->blocked($rateKey),
        'rate limiter fails closed instead of recovering a lower stale attempt count after primary corruption'
    );
    @unlink($ratePath);
    registry_throws(
        fn() => $limiter->blocked($rateKey),
        'rate limiter fails closed when only an older backup attempt count remains'
    );

    // Existing authentication-registry recovery regression: the generic JSON store may
    // recover a backup, but UserStore must fail closed so old credentials cannot return.
    $created = $users->create(
        'Recovery User',
        'recovery@example.com',
        'old-password-123',
        'user'
    );
    $userId = (string)($created['id'] ?? '');
    registry_check($userId !== '', 'test user is created');

    $users->changePassword(
        $userId,
        'old-password-123',
        'new-password-456',
        true
    );

    $backupPath = BYFTP_STORAGE . '/users.json.bak';
    $backupRaw = @file_get_contents($backupPath);
    $backupRows = is_string($backupRaw) ? json_decode($backupRaw, true) : null;
    $backupHash = '';
    if (is_array($backupRows)) {
        foreach ($backupRows as $row) {
            if (is_array($row) && ($row['id'] ?? '') === $userId) {
                $backupHash = (string)($row['password_hash'] ?? '');
                break;
            }
        }
    }
    registry_check(
        $backupHash !== '' && password_verify('old-password-123', $backupHash),
        'backup contains the prior password generation used by the regression scenario'
    );

    $primaryPath = BYFTP_STORAGE . '/users.json';
    file_put_contents($primaryPath, '{corrupt-user-registry');

    registry_throws(
        fn() => (new UserStore())->authenticate('recovery@example.com', 'old-password-123'),
        'UserStore fails closed instead of authenticating from stale backup after primary corruption'
    );

    $recoverableRows = (new JsonStore($primaryPath))->read();
    registry_check(
        is_array($recoverableRows) && count($recoverableRows) >= 1,
        'generic JsonStore recovery remains available for explicit/manual recovery paths'
    );

    @unlink($primaryPath);
    registry_throws(
        fn() => (new UserStore())->findByEmail('recovery@example.com'),
        'UserStore fails closed when only a stale backup generation remains'
    );
} finally {
    if (is_dir($externalTarget)) {
        @unlink($externalTarget . '/sentinel.txt');
        @rmdir($externalTarget);
    }

    $logsDir = BYFTP_STORAGE . '/logs';
    if (is_dir($logsDir)) {
        foreach (glob($logsDir . '/*') ?: [] as $path) {
            @unlink($path);
        }
        @rmdir($logsDir);
    }

    $usersDir = BYFTP_STORAGE . '/users';
    if (is_dir($usersDir)) {
        foreach (glob($usersDir . '/*') ?: [] as $path) {
            if (is_link($path)) {
                @unlink($path);
                continue;
            }
            if (is_dir($path)) {
                foreach (glob($path . '/*') ?: [] as $child) {
                    if (is_dir($child) && !is_link($child)) {
                        @chmod($child, 0700);
                        foreach (glob($child . '/*') ?: [] as $nested) {
                            @unlink($nested);
                        }
                        @rmdir($child);
                    } else {
                        @unlink($child);
                    }
                }
                @rmdir($path);
            } else {
                @unlink($path);
            }
        }
        @rmdir($usersDir);
    }
    foreach ([
        BYFTP_STORAGE . '/users.json',
        BYFTP_STORAGE . '/users.json.bak',
        BYFTP_STORAGE . '/users.json.lock',
    ] as $path) {
        @unlink($path);
    }
    @rmdir(BYFTP_STORAGE);
}

if ($failed) {
    fwrite(STDERR, "WEB_USER_REGISTRY_TEST=FAIL\n");
    exit(1);
}

echo "WEB_USER_REGISTRY_TEST=PASS\n";
