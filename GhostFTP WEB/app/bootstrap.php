<?php
declare(strict_types=1);

const BYFTP_ROOT = __DIR__ . '/..';
const BYFTP_STORAGE = BYFTP_ROOT . '/storage';

$version = trim((string)@file_get_contents(BYFTP_ROOT . '/VERSION'));
if (!preg_match('/^\d+\.\d+\.\d+$/', $version)) {
    throw new RuntimeException('ByFTP WEB VERSION datoteka nedostaje ili nije valjana.');
}
define('BYFTP_WEB_VERSION', $version);
define('BYFTP_VERSION', BYFTP_WEB_VERSION);

spl_autoload_register(static function (string $class): void {
    $prefix = 'ByFTP\\';
    if (!str_starts_with($class, $prefix)) {
        return;
    }
    $relative = substr($class, strlen($prefix));
    $file = __DIR__ . '/' . str_replace('\\', '/', $relative) . '.php';
    if (is_file($file)) {
        require $file;
    }
});

require __DIR__ . '/helpers.php';

foreach ([BYFTP_STORAGE, BYFTP_STORAGE . '/tmp', BYFTP_STORAGE . '/logs', BYFTP_STORAGE . '/users'] as $directory) {
    if (!is_dir($directory)) {
        @mkdir($directory, 0700, true);
    }
}

// Aborted PHP requests can leave local transfer/key temp files behind. A bounded,
// probabilistic cleanup keeps shared-hosting storage healthy without requiring cron.
try {
    if (random_int(1, 100) === 1) {
        byftp_cleanup_stale_temp_files();
    }
} catch (Throwable) {
    // Temp cleanup is maintenance only and must never prevent application startup.
}

// Defense in depth in addition to per-form CSRF tokens and SameSite cookies.
// Modern browsers identify requests initiated by a different site via Sec-Fetch-Site.
if (strtoupper((string)($_SERVER['REQUEST_METHOD'] ?? 'GET')) === 'POST'
    && strtolower((string)($_SERVER['HTTP_SEC_FETCH_SITE'] ?? '')) === 'cross-site') {
    http_response_code(403);
    header('Content-Type: text/plain; charset=utf-8');
    exit('Cross-site POST zahtjev je blokiran.');
}

$isHttps = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
    || (strtolower((string)($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '')) === 'https');

if (!headers_sent()) {
    header_remove('X-Powered-By');
    header('X-Content-Type-Options: nosniff');
    header('X-Frame-Options: DENY');
    header('X-Robots-Tag: noindex, nofollow, noarchive, nosnippet, noimageindex');
    header('Referrer-Policy: no-referrer');
    header('Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=(), usb=(), browsing-topics=()');
    header('Cross-Origin-Opener-Policy: same-origin');
    header('Cross-Origin-Resource-Policy: same-origin');
    header('X-Permitted-Cross-Domain-Policies: none');
    header('Origin-Agent-Cluster: ?1');
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');
    header("Content-Security-Policy: default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'");
    if ($isHttps) {
        header('Strict-Transport-Security: max-age=31536000');
    }
}

ini_set('session.use_strict_mode', '1');
ini_set('session.use_only_cookies', '1');
ini_set('session.cookie_httponly', '1');
$cookieBase = byftp_base_path();
$cookiePath = $cookieBase === '' ? '/' : rtrim($cookieBase, '/') . '/';
session_name('BYFTPSESSID');
session_set_cookie_params([
    'lifetime' => 0,
    'path' => $cookiePath,
    'secure' => $isHttps,
    'httponly' => true,
    'samesite' => 'Strict',
]);

// Keep PHP's own session garbage collector from undercutting the configured ByFTP
// idle/max lifetime (many shared hosts default gc_maxlifetime to only 24 minutes).
$sessionLimits = byftp_session_limits(byftp_config());
@ini_set('session.gc_maxlifetime', (string)$sessionLimits['gc']);

if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
}

$now = time();
if (!isset($_SESSION['session_started_at'])) {
    session_regenerate_id(true);
    $_SESSION['session_started_at'] = $now;
    $_SESSION['rotated_at'] = $now;
}

if (!empty($_SESSION['authenticated'])) {
    $limits = byftp_session_limits();
    $idleSeconds = $limits['idle'];
    $maxSeconds = $limits['max'];
    $lastActivity = (int)($_SESSION['last_activity'] ?? $now);
    $startedAt = (int)($_SESSION['session_started_at'] ?? $now);
    if (($now - $lastActivity) > $idleSeconds || ($now - $startedAt) > $maxSeconds) {
        $_SESSION = [];
        session_regenerate_id(true);
    } else {
        $_SESSION['last_activity'] = $now;
    }
}

if (($now - (int)($_SESSION['rotated_at'] ?? 0)) > 1800) {
    session_regenerate_id(true);
    $_SESSION['rotated_at'] = $now;
}
