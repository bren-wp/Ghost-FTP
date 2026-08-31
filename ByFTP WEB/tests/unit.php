<?php
declare(strict_types=1);

function byftp_truncate(string $value, int $length): string
{
    return substr($value, 0, $length);
}

$testStorage = sys_get_temp_dir() . '/byftp-web-unit-' . bin2hex(random_bytes(6));
define('BYFTP_STORAGE', $testStorage);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/Security/HostGuard.php';
require __DIR__ . '/../app/Storage/JsonStore.php';
require __DIR__ . '/../app/Security/RateLimiter.php';
require __DIR__ . '/../app/Storage/ProfileStore.php';

use ByFTP\Remote\PathGuard;
use ByFTP\Security\HostGuard;
use ByFTP\Security\RateLimiter;
use ByFTP\Storage\ProfileStore;

$passed = 0;
$failed = 0;

function check(bool $condition, string $label): void
{
    global $passed, $failed;
    if ($condition) {
        $passed++;
        return;
    }
    $failed++;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function throws(callable $fn, string $label): void
{
    try {
        $fn();
        check(false, $label);
    } catch (Throwable) {
        check(true, $label);
    }
}

check(PathGuard::normalizeRelative('/public_html/assets') === '/public_html/assets', 'canonical path preserved');
throws(fn() => PathGuard::normalizeRelative('/public_html/../secret'), 'traversal rejected');
throws(fn() => PathGuard::normalizeRelative('/public_html//assets'), 'duplicate separator rejected');
throws(fn() => PathGuard::normalizeRelative('public_html\\assets'), 'backslash rejected');
throws(fn() => PathGuard::segment("line\nbreak.txt"), 'remote name controls rejected');
throws(fn() => HostGuard::connectionTargets('127.0.0.1', false), 'private host blocked by default');
check(HostGuard::connectionTargets('127.0.0.1', true) === ['127.0.0.1'], 'private host allowed only when explicit');
throws(fn() => HostGuard::connectionTargets(' example.com', true), 'host edge whitespace rejected');

$reflection = new ReflectionClass(ProfileStore::class);
$store = $reflection->newInstanceWithoutConstructor();
$method = $reflection->getMethod('normalizeInput');
$method->setAccessible(true);
$base = [
    'label' => 'Test', 'protocol' => 'sftp', 'host' => 'example.com', 'port' => '22',
    'base_path' => '/', 'username' => 'user', 'password' => 'password', 'timeout' => '30',
    'host_fingerprint' => '', 'auth_method' => 'password',
];
$normalized = $method->invoke($store, $base);
check($normalized['host'] === 'example.com' && $normalized['port'] === 22, 'canonical profile accepted');

$bad = $base; $bad['host'] = 'example.com ';
throws(fn() => $method->invoke($store, $bad), 'profile host is not trimmed into validity');
$bad = $base; $bad['port'] = '22junk';
throws(fn() => $method->invoke($store, $bad), 'port suffix rejected');
$bad = $base; $bad['base_path'] = '/a/../b';
throws(fn() => $method->invoke($store, $bad), 'profile traversal rejected');
$bad = $base; $bad['host_fingerprint'] = ' SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';
throws(fn() => $method->invoke($store, $bad), 'fingerprint edge whitespace rejected');
$bad = $base; $bad['username'] = "user\r\nnext";
throws(fn() => $method->invoke($store, $bad), 'credential protocol controls rejected');

try {
    $limiter = new RateLimiter(1, 3600);
    $rateKey = 'login:203.0.113.10';
    $limiter->hit($rateKey);
    $limiter->hit($rateKey); // Creates a last-known-good backup containing stale hits.
    check($limiter->blocked($rateKey), 'rate limiter blocks at threshold');
    $limiter->clear($rateKey);
    check(!$limiter->blocked($rateKey), 'rate limiter clear removes stale backup hits');
} finally {
    $logDir = BYFTP_STORAGE . '/logs';
    foreach (glob($logDir . '/*') ?: [] as $path) {
        if (is_file($path)) {
            @unlink($path);
        }
    }
    @rmdir($logDir);
    @rmdir(BYFTP_STORAGE);
}

if ($failed > 0) {
    fwrite(STDERR, "WEB_UNIT_TESTS=FAIL passed={$passed} failed={$failed}\n");
    exit(1);
}
echo "WEB_UNIT_TESTS=PASS ({$passed})\n";
