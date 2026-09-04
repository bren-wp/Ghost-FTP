<?php
declare(strict_types=1);

use GhostFTP\Storage\JsonStore;

$storage = sys_get_temp_dir() . '/GhostFTP-config-security-' . bin2hex(random_bytes(6));
define('GhostFTP_STORAGE', $storage);

require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/helpers.php';

$failed = false;

function config_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function config_throws(callable $callback, string $label): void
{
    try {
        $callback();
        config_check(false, $label);
    } catch (Throwable) {
        config_check(true, $label);
    }
}

try {
    @mkdir(GhostFTP_STORAGE, 0700, true);
    $path = GhostFTP_config_path();
    $key = base64_encode(str_repeat('C', 32));

    $oldConfig = [
        'app_name' => 'GhostFTP',
        'secret_key' => $key,
        'allow_registration' => true,
        'allow_private_hosts' => true,
        'session_idle_minutes' => 240,
        'session_max_hours' => 48,
        'version' => '1.7.1',
    ];
    $currentConfig = [
        'app_name' => 'GhostFTP',
        'secret_key' => $key,
        'allow_registration' => false,
        'allow_private_hosts' => false,
        'session_idle_minutes' => 60,
        'session_max_hours' => 8,
        'version' => '1.7.1',
    ];

    $store = new JsonStore($path);
    $store->write($oldConfig);
    $store->write($currentConfig);

    $backupRaw = @file_get_contents($path . '.bak');
    $backup = is_string($backupRaw) ? json_decode($backupRaw, true) : null;
    config_check(
        is_array($backup)
            && !empty($backup['allow_registration'])
            && !empty($backup['allow_private_hosts'])
            && (int)($backup['session_idle_minutes'] ?? 0) === 240,
        'backup contains the older, less restrictive security policy used by the regression scenario'
    );

    file_put_contents($path, '{corrupt-app-config');
    $config = GhostFTP_config(true);
    config_check($config === [], 'runtime config does not recover stale app.json backup automatically');
    config_check(isset($GLOBALS['GhostFTP_config_error']), 'runtime config exposes recovery-required state');
    config_check(!GhostFTP_registration_enabled(), 'registration fails closed while config recovery is required');
    config_check(!GhostFTP_private_hosts_allowed(), 'private host access fails closed while config recovery is required');

    $corruptPrimary = (string)@file_get_contents($path);
    config_throws(
        fn() => GhostFTP_update_config(['allow_registration' => true]),
        'config update is blocked while the primary config is corrupt'
    );
    config_check(
        (string)@file_get_contents($path) === $corruptPrimary,
        'blocked config update does not replace the corrupt primary or encryption state'
    );

    $manualRecovery = (new JsonStore($path))->read();
    config_check(
        !empty($manualRecovery['allow_registration']) && !empty($manualRecovery['allow_private_hosts']),
        'generic JsonStore still exposes backup data for explicit operator recovery'
    );

    @unlink($path);
    $backupOnly = GhostFTP_config(true);
    config_check($backupOnly === [], 'runtime config fails closed when only app.json.bak remains');
    config_check(isset($GLOBALS['GhostFTP_config_error']), 'backup-only config remains marked for manual recovery');
    config_check(!GhostFTP_is_configured(), 'backup-only stale config does not silently configure the application');
} finally {
    foreach ([
        GhostFTP_STORAGE . '/app.json',
        GhostFTP_STORAGE . '/app.json.bak',
        GhostFTP_STORAGE . '/app.json.lock',
    ] as $file) {
        @unlink($file);
    }
    @rmdir(GhostFTP_STORAGE);
}

if ($failed) {
    fwrite(STDERR, "WEB_CONFIG_SECURITY_TEST=FAIL\n");
    exit(1);
}

echo "WEB_CONFIG_SECURITY_TEST=PASS\n";
