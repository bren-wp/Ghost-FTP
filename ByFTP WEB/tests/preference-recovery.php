<?php
declare(strict_types=1);

use ByFTP\Storage\JsonStore;
use ByFTP\Storage\PreferenceStore;
use ByFTP\Storage\UserWorkspace;

$storage = sys_get_temp_dir() . '/byftp-preference-recovery-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $storage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Storage/UserWorkspace.php';
require __DIR__ . '/../app/Storage/PreferenceStore.php';

$failed = false;

function preference_recovery_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function preference_recovery_throws(callable $callback, string $label): void
{
    try {
        $callback();
        preference_recovery_check(false, $label);
    } catch (Throwable) {
        preference_recovery_check(true, $label);
    }
}

$userId = 'preference-recovery-user';
$userDir = '';
$preferencePath = '';

try {
    $preferences = new PreferenceStore($userId);
    $userDir = UserWorkspace::directory($userId);
    $preferencePath = UserWorkspace::file($userId, 'preferences.json');

    $sensitiveState = $preferences->saveClientState([
        'lastProfile' => 'profile-a',
        'showHidden' => true,
        'compactRows' => false,
        'uploadConflict' => 'rename',
        'sort' => ['key' => 'name', 'direction' => 1],
        'lastPaths' => ['profile-a' => '/clients/acme/private'],
        'recentPaths' => ['profile-a' => ['/clients/acme/private', '/clients/acme/invoices']],
    ]);
    preference_recovery_check(
        ($sensitiveState['lastPaths']['profile-a'] ?? '') === '/clients/acme/private',
        'privacy regression fixture stores a remote path in the primary generation'
    );

    $clearedState = $preferences->saveClientState([
        'lastProfile' => '',
        'showHidden' => true,
        'compactRows' => false,
        'uploadConflict' => 'rename',
        'sort' => ['key' => 'name', 'direction' => 1],
        'lastPaths' => [],
        'recentPaths' => [],
    ]);
    preference_recovery_check(
        ($clearedState['lastPaths'] ?? []) === [] && ($clearedState['recentPaths'] ?? []) === [],
        'user-cleared remote path history is absent from the valid primary generation'
    );

    $backupRaw = @file_get_contents($preferencePath . '.bak');
    $backup = is_string($backupRaw) ? json_decode($backupRaw, true) : null;
    preference_recovery_check(
        is_array($backup)
            && (($backup['client_state']['lastPaths']['profile-a'] ?? '') === '/clients/acme/private')
            && in_array('/clients/acme/invoices', $backup['client_state']['recentPaths']['profile-a'] ?? [], true),
        'backup intentionally retains cleared remote path history for the regression scenario'
    );

    $primaryRaw = @file_get_contents($preferencePath);
    $primary = is_string($primaryRaw) ? json_decode($primaryRaw, true) : null;
    preference_recovery_check(
        is_array($primary)
            && (($primary['client_state']['lastPaths'] ?? []) === [])
            && (($primary['client_state']['recentPaths'] ?? []) === []),
        'primary preference generation remains cleared before corruption'
    );

    file_put_contents($preferencePath, '{corrupt-preference-primary');
    preference_recovery_throws(
        fn() => (new PreferenceStore($userId))->clientState(),
        'PreferenceStore fails closed instead of resurrecting cleared remote path history from stale backup'
    );

    $manual = (new JsonStore($preferencePath))->read();
    preference_recovery_check(
        is_array($manual)
            && (($manual['client_state']['lastPaths']['profile-a'] ?? '') === '/clients/acme/private'),
        'generic JsonStore still exposes preference backup for explicit operator recovery'
    );

    @unlink($preferencePath);
    preference_recovery_throws(
        fn() => (new PreferenceStore($userId))->clientState(),
        'PreferenceStore fails closed when only stale preferences.json.bak remains'
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
    fwrite(STDERR, "WEB_PREFERENCE_RECOVERY_TEST=FAIL\n");
    exit(1);
}

echo "WEB_PREFERENCE_RECOVERY_TEST=PASS\n";
