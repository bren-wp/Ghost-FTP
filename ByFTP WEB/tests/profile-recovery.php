<?php
declare(strict_types=1);

use ByFTP\Storage\JsonStore;
use ByFTP\Storage\ProfileStore;
use ByFTP\Storage\UserWorkspace;

function byftp_truncate(string $value, int $length): string
{
    return substr($value, 0, $length);
}

function byftp_config(bool $fresh = false): array
{
    return ['secret_key' => base64_encode(str_repeat('P', 32))];
}

$storage = sys_get_temp_dir() . '/byftp-profile-recovery-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $storage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Security/Crypto.php';
require __DIR__ . '/../app/Storage/UserWorkspace.php';
require __DIR__ . '/../app/Storage/ProfileStore.php';

$failed = false;

function profile_recovery_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function profile_recovery_throws(callable $callback, string $label): void
{
    try {
        $callback();
        profile_recovery_check(false, $label);
    } catch (Throwable) {
        profile_recovery_check(true, $label);
    }
}

$userId = 'profile-recovery-user';
$userDir = '';
$profilePath = '';

try {
    $profiles = new ProfileStore($userId);
    $userDir = UserWorkspace::directory($userId);
    $profilePath = UserWorkspace::file($userId, 'profiles.json');

    $deleted = $profiles->save([
        'label' => 'Deleted FTP',
        'protocol' => 'ftp',
        'host' => 'deleted.example.com',
        'port' => '21',
        'base_path' => '/',
        'username' => 'deleted-user',
        'password' => 'deleted-secret-123',
        'timeout' => '30',
        'auth_method' => 'password',
    ]);
    $deletedId = (string)($deleted['id'] ?? '');
    profile_recovery_check($deletedId !== '', 'deleted-profile regression fixture is created');

    $kept = $profiles->save([
        'label' => 'Kept FTP',
        'protocol' => 'ftp',
        'host' => 'kept.example.com',
        'port' => '21',
        'base_path' => '/',
        'username' => 'kept-user',
        'password' => 'kept-secret-456',
        'timeout' => '30',
        'auth_method' => 'password',
    ]);
    $keptId = (string)($kept['id'] ?? '');
    profile_recovery_check($keptId !== '', 'kept-profile regression fixture is created');

    $profiles->delete($deletedId);
    profile_recovery_check($profiles->find($deletedId, true) === null, 'deleted profile is absent from valid primary generation');
    profile_recovery_check($profiles->find($keptId, true) !== null, 'unrelated profile remains in valid primary generation');

    $backupRaw = @file_get_contents($profilePath . '.bak');
    $backupRows = is_string($backupRaw) ? json_decode($backupRaw, true) : null;
    $backupContainsDeleted = false;
    if (is_array($backupRows)) {
        foreach ($backupRows as $row) {
            if (is_array($row) && ($row['id'] ?? '') === $deletedId) {
                $backupContainsDeleted = true;
                break;
            }
        }
    }
    profile_recovery_check(
        $backupContainsDeleted,
        'backup intentionally contains the deleted encrypted profile used by the regression scenario'
    );

    file_put_contents($profilePath, '{corrupt-profile-primary');
    profile_recovery_throws(
        fn() => (new ProfileStore($userId))->find($deletedId, true),
        'ProfileStore fails closed instead of resurrecting deleted credentials from stale backup'
    );

    $manualRows = (new JsonStore($profilePath))->read();
    profile_recovery_check(
        is_array($manualRows) && count($manualRows) >= 2,
        'generic JsonStore still exposes profile backup for explicit operator recovery'
    );

    @unlink($profilePath);
    profile_recovery_throws(
        fn() => (new ProfileStore($userId))->all(true),
        'ProfileStore fails closed when only stale profiles.json.bak remains'
    );
} finally {
    if ($userDir !== '' && is_dir($userDir)) {
        foreach (glob($userDir . '/*') ?: [] as $file) {
            @unlink($file);
        }
        @rmdir($userDir);
    }
    @rmdir(BYFTP_STORAGE . '/users');
    @rmdir(BYFTP_STORAGE);
}

if ($failed) {
    fwrite(STDERR, "WEB_PROFILE_RECOVERY_TEST=FAIL\n");
    exit(1);
}

echo "WEB_PROFILE_RECOVERY_TEST=PASS\n";
