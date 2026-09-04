<?php
declare(strict_types=1);

use ByFTP\Remote\ClientFactory;
use ByFTP\Remote\SftpClient;
use ByFTP\Storage\JsonStore;
use ByFTP\Storage\PreferenceStore;
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
require __DIR__ . '/../app/Remote/RemoteClientInterface.php';
require __DIR__ . '/../app/Remote/BoundedDownloadInterface.php';
require __DIR__ . '/../app/Remote/TransferLimiter.php';
require __DIR__ . '/../app/Remote/SftpClient.php';
require __DIR__ . '/../app/Remote/ClientFactory.php';
require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Security/Crypto.php';
require __DIR__ . '/../app/Storage/UserWorkspace.php';
require __DIR__ . '/../app/Storage/ProfileStore.php';
require __DIR__ . '/../app/Storage/PreferenceStore.php';

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
$preferencePath = '';
$legacyUserId = 'legacy-recovery-user';
$legacyUserDir = '';
$legacySource = '';
$legacyTarget = '';

try {
    // PHP ssh2 does not supply a known_hosts trust decision. The central client factory
    // must therefore reject SFTP before authentication whenever no pinned host key exists.
    profile_recovery_throws(
        fn() => ClientFactory::make(['protocol' => 'sftp', 'host_fingerprint' => '']),
        'SFTP client creation fails closed when no host fingerprint is pinned'
    );
    $pinnedSftp = ClientFactory::make([
        'protocol' => 'sftp',
        'host_fingerprint' => str_repeat('A', 64),
    ]);
    profile_recovery_check(
        $pinnedSftp instanceof SftpClient,
        'SFTP client creation remains available when a host fingerprint is pinned'
    );

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

    // Preferences are privacy state: a user may intentionally clear remembered remote paths.
    // The previous generation can remain in .bak, but runtime recovery must not resurrect it.
    $preferences = new PreferenceStore($userId);
    $preferencePath = UserWorkspace::file($userId, 'preferences.json');
    $preferences->saveClientState([
        'lastProfile' => 'profile-a',
        'showHidden' => true,
        'compactRows' => false,
        'uploadConflict' => 'rename',
        'sort' => ['key' => 'name', 'direction' => 1],
        'lastPaths' => ['profile-a' => '/clients/acme/private'],
        'recentPaths' => ['profile-a' => ['/clients/acme/private', '/clients/acme/invoices']],
    ]);
    $clearedState = $preferences->saveClientState([
        'lastProfile' => '',
        'showHidden' => true,
        'compactRows' => false,
        'uploadConflict' => 'rename',
        'sort' => ['key' => 'name', 'direction' => 1],
        'lastPaths' => [],
        'recentPaths' => [],
    ]);
    profile_recovery_check(
        ($clearedState['lastPaths'] ?? []) === [] && ($clearedState['recentPaths'] ?? []) === [],
        'user-cleared remote path history is absent from the valid preference primary generation'
    );

    $preferenceBackupRaw = @file_get_contents($preferencePath . '.bak');
    $preferenceBackup = is_string($preferenceBackupRaw) ? json_decode($preferenceBackupRaw, true) : null;
    profile_recovery_check(
        is_array($preferenceBackup)
            && (($preferenceBackup['client_state']['lastPaths']['profile-a'] ?? '') === '/clients/acme/private')
            && in_array('/clients/acme/invoices', $preferenceBackup['client_state']['recentPaths']['profile-a'] ?? [], true),
        'backup intentionally retains cleared remote path history for the preference regression scenario'
    );

    file_put_contents($preferencePath, '{corrupt-preference-primary');
    profile_recovery_throws(
        fn() => (new PreferenceStore($userId))->clientState(),
        'PreferenceStore fails closed instead of resurrecting cleared remote path history from stale backup'
    );

    $manualPreferences = (new JsonStore($preferencePath))->read();
    profile_recovery_check(
        is_array($manualPreferences)
            && (($manualPreferences['client_state']['lastPaths']['profile-a'] ?? '') === '/clients/acme/private'),
        'generic JsonStore still exposes preference backup for explicit operator recovery'
    );

    @unlink($preferencePath);
    profile_recovery_throws(
        fn() => (new PreferenceStore($userId))->clientState(),
        'PreferenceStore fails closed when only stale preferences.json.bak remains'
    );

    // Legacy upgrade must not make a different recovery decision than the normal runtime.
    // A stale root backup can contain a profile deleted before the upgrade started.
    $legacyUserDir = UserWorkspace::directory($legacyUserId);
    $legacySource = BYFTP_STORAGE . '/profiles.json';
    $legacyTarget = UserWorkspace::file($legacyUserId, 'profiles.json');
    $legacyStore = new JsonStore($legacySource);
    $legacyStore->write([[
        'id' => 'stale-deleted-profile',
        'label' => 'Stale deleted legacy profile',
        'protocol' => 'ftp',
        'host' => 'legacy.example.com',
        'port' => 21,
        'base_path' => '/',
        'username_enc' => 'stale-encrypted-username',
        'password_enc' => 'stale-encrypted-password',
    ]]);
    $legacyStore->write([]);

    $legacyBackupRaw = @file_get_contents($legacySource . '.bak');
    $legacyBackup = is_string($legacyBackupRaw) ? json_decode($legacyBackupRaw, true) : null;
    profile_recovery_check(
        is_array($legacyBackup)
            && (($legacyBackup[0]['id'] ?? '') === 'stale-deleted-profile'),
        'legacy backup intentionally contains a deleted credential generation for the migration regression'
    );

    file_put_contents($legacySource, '{corrupt-legacy-profile-primary');
    profile_recovery_throws(
        fn() => UserWorkspace::migrateLegacy($legacyUserId),
        'legacy migration fails closed instead of restoring stale profile credentials from backup'
    );
    profile_recovery_check(
        !is_file($legacyTarget) && !is_file($legacyTarget . '.bak'),
        'failed legacy recovery does not create a migrated profile registry'
    );

    @unlink($legacySource);
    profile_recovery_throws(
        fn() => UserWorkspace::migrateLegacy($legacyUserId),
        'legacy migration fails closed when only stale root profiles.json.bak remains'
    );
    profile_recovery_check(
        !is_file($legacyTarget) && !is_file($legacyTarget . '.bak'),
        'backup-only legacy recovery still leaves the target profile registry absent'
    );
} finally {
    foreach ([$legacySource, $legacySource . '.bak', $legacySource . '.lock'] as $legacyPath) {
        if ($legacyPath !== '') {
            @unlink($legacyPath);
        }
    }
    foreach ([$userDir, $legacyUserDir] as $directory) {
        if ($directory !== '' && is_dir($directory)) {
            foreach (glob($directory . '/*') ?: [] as $file) {
                @unlink($file);
            }
            @rmdir($directory);
        }
    }
    @rmdir(BYFTP_STORAGE . '/users');
    @rmdir(BYFTP_STORAGE);
}

if ($failed) {
    fwrite(STDERR, "WEB_PROFILE_RECOVERY_TEST=FAIL\n");
    exit(1);
}

echo "WEB_PROFILE_RECOVERY_TEST=PASS\n";
