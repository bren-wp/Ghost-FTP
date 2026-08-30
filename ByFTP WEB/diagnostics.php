<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\Auth;
use ByFTP\Storage\UserWorkspace;

Auth::requireAuth();

$isHttps = ((!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') || strtolower((string)($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '')) === 'https');
$requestPath = (string)(parse_url((string)($_SERVER['REQUEST_URI'] ?? ''), PHP_URL_PATH) ?? '');
$expectedDiagnosticsPath = rtrim(byftp_url('diagnostics'), '/');
$rewriteReady = rtrim($requestPath, '/') === $expectedDiagnosticsPath;
$noIndexReady = is_file(BYFTP_ROOT . '/robots.txt');
$userWorkspace = UserWorkspace::ensure(Auth::id());
$pwaReady = is_file(BYFTP_ROOT . '/manifest.webmanifest') && is_file(BYFTP_ROOT . '/service-worker.js');
$checks = [
    ['ByFTP Web', 'v' . BYFTP_VERSION, true, 'core'],
    ['PHP verzija', PHP_VERSION, version_compare(PHP_VERSION, '8.1.0', '>='), 'core'],
    ['FTP / FTPS', extension_loaded('ftp') ? (function_exists('ftp_ssl_connect') ? 'FTP + FTPS dostupni' : 'FTP dostupan, FTPS nije dostupan') : 'FTP ekstenzija nije dostupna', extension_loaded('ftp'), 'transfer'],
    ['FTPS identitet servera', function_exists('ftp_ssl_connect') ? 'PHP ext-ftp ne provjerava peer certifikat; za provjerljiv identitet koristi SFTP fingerprint' : 'FTPS nije dostupan', !function_exists('ftp_ssl_connect'), 'optional'],
    ['SFTP (ssh2)', extension_loaded('ssh2') ? 'Dostupan: password + SSH key autentikacija' : 'Nije dostupan; FTP/FTPS i dalje rade', extension_loaded('ssh2'), 'optional'],
    ['ZIP', class_exists(ZipArchive::class) ? 'Dostupan: ZIP/create/extract/download mapa' : 'Nije dostupan: ZIP opcije su onemogućene', class_exists(ZipArchive::class), 'optional'],
    ['Sodium', extension_loaded('sodium') ? 'Dostupan' : 'Nije dostupan; koristit će se OpenSSL ako postoji', extension_loaded('sodium'), 'security'],
    ['OpenSSL', extension_loaded('openssl') ? OPENSSL_VERSION_TEXT : 'Nije dostupan', extension_loaded('openssl'), 'security'],
    ['Šifriranje vjerodajnica', (function_exists('sodium_crypto_secretbox') || function_exists('openssl_encrypt')) ? 'Dostupno' : 'Nije dostupno', function_exists('sodium_crypto_secretbox') || function_exists('openssl_encrypt'), 'security'],
    ['storage/ zapisiv', is_writable(BYFTP_STORAGE) ? 'Da' : 'Ne', is_writable(BYFTP_STORAGE), 'storage'],
    ['storage/tmp zapisiv', is_writable(BYFTP_STORAGE . '/tmp') ? 'Da' : 'Ne', is_writable(BYFTP_STORAGE . '/tmp'), 'storage'],
    ['Korisnički workspace', is_writable($userWorkspace) ? 'Izoliran i zapisiv' : 'Nije zapisiv', is_writable($userWorkspace), 'storage'],
    ['PWA instalacija', $pwaReady ? ($isHttps ? 'Manifest + service worker spremni' : 'Datoteke spremne; za instalaciju koristi HTTPS') : 'Manifest ili service worker nedostaje', $pwaReady && $isHttps, 'optional'],
    ['Friendly URL konfiguracija', $rewriteReady ? 'Clean ruta /diagnostics radi' : 'Otvori /diagnostics i provjeri mod_rewrite/LiteSpeed pravila', $rewriteReady, 'routing'],
    ['Zaštita od indeksiranja', $noIndexReady ? 'robots + meta + X-Robots-Tag' : 'robots.txt nedostaje', $noIndexReady, 'security'],
    ['Privatni/local server targeti', byftp_private_hosts_allowed() ? 'Dopušteni administratorskom postavkom' : 'Blokirani (preporučeno za shared hosting)', !byftp_private_hosts_allowed(), 'security'],
    ['file_uploads', filter_var(ini_get('file_uploads'), FILTER_VALIDATE_BOOLEAN) ? 'Uključeno' : 'Isključeno', filter_var(ini_get('file_uploads'), FILTER_VALIDATE_BOOLEAN), 'transfer'],
    ['upload_max_filesize', (string)ini_get('upload_max_filesize'), true, 'limits'],
    ['post_max_size', (string)ini_get('post_max_size'), true, 'limits'],
    ['memory_limit', (string)ini_get('memory_limit'), true, 'limits'],
    ['max_execution_time', (string)ini_get('max_execution_time') . ' s', true, 'limits'],
    ['max_file_uploads', (string)ini_get('max_file_uploads'), true, 'limits'],
    ['session.gc_maxlifetime', (string)ini_get('session.gc_maxlifetime') . ' s', (int)ini_get('session.gc_maxlifetime') >= byftp_session_limits()['idle'], 'limits'],
    ['HTTPS', $isHttps ? 'Aktivan' : 'Nije detektiran', $isHttps, 'security'],
];
$requiredOk = version_compare(PHP_VERSION, '8.1.0', '>=') && extension_loaded('ftp') && (function_exists('sodium_crypto_secretbox') || function_exists('openssl_encrypt')) && is_writable(BYFTP_STORAGE) && is_writable(BYFTP_STORAGE . '/tmp');
?><!doctype html>
<html lang="hr">
<head>
    <?php $pageTitle = 'Dijagnostika · ' . byftp_app_name(); require __DIR__ . '/app/Views/head.php'; ?>
    </head>
<body class="auth-page brendigo-auth diagnostics-page">
<a class="skip-link" href="#main">Preskoči na dijagnostiku</a>
<main id="main" class="auth-card diagnostics-card auth-card-premium">
    <?php require __DIR__ . '/app/Views/auth-brand.php'; ?>
    <p class="eyebrow">Shared hosting health check</p>
    <h1>Dijagnostika sustava.</h1>
    <p class="muted">Provjera runtime mogućnosti, sigurnosti, clean URL konfiguracije i limita koji utječu na prijenose.</p>
    <div class="diagnostic-status <?= $requiredOk ? 'is-good' : 'is-warning' ?>">
        <strong><?= $requiredOk ? 'Osnovni zahtjevi su zadovoljeni' : 'Potrebna je dorada hostinga' ?></strong>
        <span><?= $requiredOk ? 'ByFTP može pokrenuti osnovne FTP funkcije.' : 'Provjeri označene obavezne stavke prije produkcijskog rada.' ?></span>
    </div>
    <div class="diagnostic-list">
        <?php foreach ($checks as [$name, $value, $ok, $group]): ?>
            <div class="diagnostic-row" data-group="<?= byftp_e((string)$group) ?>">
                <span><strong><?= byftp_e((string)$name) ?></strong><small><?= byftp_e((string)$value) ?></small></span>
                <b class="<?= $ok ? 'ok' : 'bad' ?>" aria-label="<?= $ok ? 'Dostupno' : 'Nedostupno ili nije aktivno' ?>"><?= $ok ? '✓' : '!' ?></b>
            </div>
        <?php endforeach; ?>
    </div>
    <p class="small muted diagnostic-note">SFTP i ZIP su opcionalni. HTTPS je snažno preporučen. Stvarna podrška za clean URL zahtijeva Apache/LiteSpeed <code>mod_rewrite</code> ili ekvivalentnu konfiguraciju hostinga.</p>
    <div class="diagnostic-actions">
        <a href="<?= byftp_e(byftp_url('app')) ?>" class="button primary">Natrag u ByFTP</a>
        <button class="button ghost" type="button" data-install-app>Instaliraj aplikaciju</button><a href="<?= byftp_e(byftp_url('logout')) ?>" class="button ghost">Odjava</a>
    </div>
</main>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
