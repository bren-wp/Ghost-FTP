<?php
declare(strict_types=1);

use ByFTP\Storage\JsonStore;
use ByFTP\Storage\UserStore;
use ByFTP\Storage\UserWorkspace;

$storage = sys_get_temp_dir() . '/byftp-user-registry-' . bin2hex(random_bytes(6));
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
    $usersDir = BYFTP_STORAGE . '/users';
    if (is_dir($usersDir)) {
        foreach (glob($usersDir . '/*') ?: [] as $path) {
            if (is_dir($path)) {
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
