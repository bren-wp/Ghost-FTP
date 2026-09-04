<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\Operations\RemoteOperations;
use GhostFTP\Remote\BoundedDownloadInterface;
use GhostFTP\Remote\ClientFactory;
use GhostFTP\Remote\PathGuard;
use GhostFTP\Security\AppLogger;
use GhostFTP\Security\Auth;
use GhostFTP\Storage\ProfileStore;

if (!Auth::check()) {
    http_response_code(401);
    exit('Sesija je istekla.');
}
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    exit('Method not allowed.');
}
if (!GhostFTP_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
    http_response_code(419);
    exit('Sigurnosni token nije valjan.');
}

$userId = Auth::id();
GhostFTP_release_session_lock();
$profileId = (string)($_POST['profile_id'] ?? '');
$profile = (new ProfileStore($userId))->find($profileId, true);
if (!$profile) {
    http_response_code(404);
    exit('Profil nije pronađen.');
}

$path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
$name = PathGuard::basename($path);
$tmp = tempnam(GhostFTP_STORAGE . '/tmp', 'download-');
if ($tmp === false) {
    http_response_code(500);
    exit('Privremena datoteka nije dostupna.');
}

$client = null;
try {
    $client = ClientFactory::make($profile);
    if (!$client instanceof BoundedDownloadInterface) {
        throw new RuntimeException('Remote klijent ne podržava sigurni ograničeni download.');
    }
    $ops = new RemoteOperations($client);
    $item = $ops->stat($path);
    if ($item === null || ($item['type'] ?? 'file') !== 'file') {
        throw new RuntimeException('Datoteka više ne postoji.');
    }
    $reportedSize = max(0, (int)($item['size'] ?? 0));
    $requestedLimit = $reportedSize > 0 ? $reportedSize : null;
    \GhostFTP_assert_temp_capacity($reportedSize);
    $size = $client->downloadBounded($path, $tmp, $requestedLimit);

    header('Content-Type: application/octet-stream');
    header('Content-Disposition: attachment; filename="' . str_replace(['"', "\r", "\n"], '', $name) . '"; filename*=UTF-8\'\'' . rawurlencode($name));
    header('Content-Length: ' . $size);
    header('Cache-Control: no-store, private');
    header('X-Content-Type-Options: nosniff');
    AppLogger::event('file.download', ['profile_id' => $profileId, 'path' => $path, 'bytes' => $size]);
    readfile($tmp);
} catch (Throwable $e) {
    AppLogger::event('download.error', ['profile_id' => $profileId, 'path' => $path, 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) {
        http_response_code(400);
        header('Content-Type: text/plain; charset=utf-8');
    }
    echo 'Download nije uspio: ' . $e->getMessage();
} finally {
    if ($client) {
        $client->disconnect();
    }
    @unlink($tmp);
}
