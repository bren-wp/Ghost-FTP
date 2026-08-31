<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Operations\RemoteOperations;
use ByFTP\Remote\ClientFactory;
use ByFTP\Remote\PathGuard;
use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Storage\ProfileStore;

if (!Auth::check()) {
    http_response_code(401);
    exit('Sesija je istekla.');
}
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    exit('Method not allowed.');
}
if (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
    http_response_code(419);
    exit('Sigurnosni token nije valjan.');
}

$userId = Auth::id();
byftp_release_session_lock();
$profileId = (string)($_POST['profile_id'] ?? '');
$profile = (new ProfileStore($userId))->find($profileId, true);
if (!$profile) {
    http_response_code(404);
    exit('Profil nije pronađen.');
}

$paths = byftp_string_paths((string)($_POST['paths'] ?? '[]'), 200);
$name = byftp_archive_download_name((string)($_POST['name'] ?? 'byftp-download.zip'));

$tmp = tempnam(BYFTP_STORAGE . '/tmp', 'archive-');
if ($tmp === false) {
    http_response_code(500);
    exit('Privremena datoteka nije dostupna.');
}
@unlink($tmp);
$tmp .= '.zip';
$client = null;

try {
    $client = ClientFactory::make($profile);
    $ops = new RemoteOperations($client);
    $ops->buildZipToLocal($paths, $tmp);
    $size = filesize($tmp);

    header('Content-Type: application/zip');
    header('Content-Disposition: attachment; filename="' . str_replace(['"', "\r", "\n"], '', $name) . '"; filename*=UTF-8\'\'' . rawurlencode($name));
    if ($size !== false) {
        header('Content-Length: ' . $size);
    }
    header('Cache-Control: no-store, private');
    header('X-Content-Type-Options: nosniff');
    AppLogger::event('archive.download', ['profile_id' => $profileId, 'count' => count($paths), 'bytes' => $size ?: 0]);
    readfile($tmp);
} catch (Throwable $e) {
    AppLogger::event('archive_download.error', ['profile_id' => $profileId, 'error' => byftp_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(400);
        header('Content-Type: text/plain; charset=utf-8');
    }
    echo 'Preuzimanje ZIP arhive nije uspjelo: ' . $e->getMessage();
} finally {
    if ($client) {
        $client->disconnect();
    }
    @unlink($tmp);
}
