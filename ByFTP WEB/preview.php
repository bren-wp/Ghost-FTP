<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Operations\RemoteOperations;
use ByFTP\Remote\ClientFactory;
use ByFTP\Remote\PathGuard;
use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Storage\ProfileStore;

if (!Auth::check()) { http_response_code(401); exit('Sesija je istekla.'); }
if ($_SERVER['REQUEST_METHOD'] !== 'POST') { http_response_code(405); exit('Method not allowed.'); }
if (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) { http_response_code(419); exit('Sigurnosni token nije valjan.'); }

$userId = Auth::id();
byftp_release_session_lock();
$profileId = (string)($_POST['profile_id'] ?? '');
$profile = (new ProfileStore($userId))->find($profileId, true);
if (!$profile) { http_response_code(404); exit('Profil nije pronađen.'); }
$path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
$extension = strtolower(pathinfo(PathGuard::basename($path), PATHINFO_EXTENSION));
if (!in_array($extension, ['jpg', 'jpeg', 'png', 'gif', 'webp'], true)) {
    http_response_code(415); exit('Ova vrsta datoteke nema sigurni ugrađeni pregled.');
}

$tmp = tempnam(BYFTP_STORAGE . '/tmp', 'preview-');
if ($tmp === false) { http_response_code(500); exit('Privremena datoteka nije dostupna.'); }
$client = null;
try {
    $client = ClientFactory::make($profile);
    $ops = new RemoteOperations($client);
    $item = $ops->stat($path);
    if ($item === null || ($item['type'] ?? 'file') !== 'file') {
        throw new RuntimeException('Datoteka više ne postoji.');
    }
    $maxPreviewBytes = 10485760;
    $reportedSize = max(0, (int)($item['size'] ?? 0));
    if ($reportedSize > $maxPreviewBytes) {
        throw new RuntimeException('Pregled slike podržava datoteke do 10 MiB.');
    }
    \byftp_assert_temp_capacity($maxPreviewBytes);
    $size = $client->download($path, $tmp, $maxPreviewBytes);
    if ($size < 1) {
        throw new RuntimeException('Pregled slike podržava datoteke do 10 MiB.');
    }
    $info = @getimagesize($tmp);
    $mime = is_array($info) ? (string)($info['mime'] ?? '') : '';
    if (!in_array($mime, ['image/jpeg', 'image/png', 'image/gif', 'image/webp'], true)) {
        throw new RuntimeException('Datoteka nije valjana podržana slika.');
    }
    header('Content-Type: ' . $mime);
    header('Content-Length: ' . $size);
    header('Content-Disposition: inline; filename="preview"');
    header('Cache-Control: no-store, private');
    header("Content-Security-Policy: default-src 'none'; sandbox");
    AppLogger::event('file.preview', ['profile_id' => $profileId, 'path' => $path, 'bytes' => $size]);
    readfile($tmp);
} catch (Throwable $e) {
    AppLogger::event('preview.error', ['profile_id' => $profileId, 'path' => $path, 'error' => byftp_truncate($e->getMessage(), 300)]);
    if (!headers_sent()) { http_response_code(400); header('Content-Type: text/plain; charset=utf-8'); }
    echo $e->getMessage();
} finally {
    if ($client) $client->disconnect();
    @unlink($tmp);
}
