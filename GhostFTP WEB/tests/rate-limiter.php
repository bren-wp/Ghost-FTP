<?php
declare(strict_types=1);

use ByFTP\Security\LoginRateLimitGate;
use ByFTP\Security\RateLimiter;

$storage = sys_get_temp_dir() . '/byftp-rate-limit-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $storage);

require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Security/RateLimiter.php';
require __DIR__ . '/../app/Security/LoginRateLimitGate.php';

$failed = false;

function limiter_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function limiter_throws(callable $callback, string $label): void
{
    try {
        $callback();
        limiter_check(false, $label);
    } catch (Throwable) {
        limiter_check(true, $label);
    }
}

function limiter_path(string $key): string
{
    return BYFTP_STORAGE . '/logs/rl-' . hash('sha256', $key) . '.json';
}

try {
    $key = 'login-account:atomic-test';
    $limiter = new RateLimiter(2, 3600);

    limiter_check($limiter->consume($key), 'first attempt is atomically admitted');
    limiter_check($limiter->consume($key), 'second attempt is atomically admitted at configured budget');
    limiter_check(!$limiter->consume($key), 'attempt after configured budget is atomically rejected');
    limiter_check($limiter->blocked($key), 'blocked state agrees with consumed attempt budget');

    $path = limiter_path($key);
    $raw = @file_get_contents($path);
    $state = is_string($raw) ? json_decode($raw, true) : null;
    limiter_check(
        is_array($state) && (int)($state['count'] ?? -1) === 2,
        'rejected consume does not inflate persisted attempt count beyond configured threshold'
    );

    $limiter->clear($key);
    limiter_check(!$limiter->blocked($key), 'clear resets the consumed attempt budget');
    limiter_check($limiter->consume($key), 'attempt is admitted again after explicit reset');

    // A source that already exhausted its IP budget must not be able to spend arbitrary
    // per-account budgets by continuing to submit different e-mail addresses.
    $ipKey = 'login-ip:198.51.100.20';
    $accountKey = 'login-account:target@example.test';
    $ipLimiter = new RateLimiter(1, 3600);
    $accountLimiter = new RateLimiter(5, 3600);
    limiter_check($ipLimiter->consume($ipKey), 'IP setup attempt consumes its only budget slot');
    limiter_check(
        !LoginRateLimitGate::consume($ipLimiter, $ipKey, $accountLimiter, $accountKey),
        'blocked IP is rejected by ordered login gate'
    );
    limiter_check(
        !is_file(limiter_path($accountKey)),
        'blocked IP does not create or consume an account-specific rate-limit state'
    );

    $ipLimiter->clear($ipKey);
    limiter_check(
        LoginRateLimitGate::consume($ipLimiter, $ipKey, $accountLimiter, $accountKey),
        'admitted IP consumes the account budget after IP reset'
    );
    $accountRaw = @file_get_contents(limiter_path($accountKey));
    $accountState = is_string($accountRaw) ? json_decode($accountRaw, true) : null;
    limiter_check(
        is_array($accountState) && (int)($accountState['count'] ?? -1) === 1,
        'ordered login gate consumes account budget exactly once for admitted IP'
    );

    // Create a valid backup generation, then corrupt the primary. Security-state recovery
    // must fail closed instead of consuming from an older counter generation.
    $limiter->consume($key);
    limiter_check(is_file($path . '.bak'), 'rate limiter keeps a backup for explicit diagnostics/recovery');
    file_put_contents($path, '{corrupt-rate-limit-primary');
    limiter_throws(
        fn() => $limiter->consume($key),
        'atomic consume fails closed when primary rate-limit state is corrupt'
    );
} finally {
    $logs = BYFTP_STORAGE . '/logs';
    if (is_dir($logs)) {
        foreach (glob($logs . '/*') ?: [] as $file) {
            @unlink($file);
        }
        @rmdir($logs);
    }
    @rmdir(BYFTP_STORAGE);
}

if ($failed) {
    fwrite(STDERR, "WEB_RATE_LIMITER_TEST=FAIL\n");
    exit(1);
}

echo "WEB_RATE_LIMITER_TEST=PASS\n";
