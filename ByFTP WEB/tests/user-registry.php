<?php
declare(strict_types=1);

use ByFTP\Storage\JsonStore;
use ByFTP\Storage\UserStore;
use ByFTP\Storage\UserWorkspace;

$storage = sys_get_temp_dir() . '/byftp-user-registry-' . bin2hex(random_bytes(6));
$externalTarget = sys_get_temp_dir() . '/byftp-user-delete-target-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $storage);

require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Storage/UserWorkspace.php';
require __DIR__ . '/../app/Storage/UserStore.php';

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
        is_array($recoverableRows) && count($recoverableRows) === 1,
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
