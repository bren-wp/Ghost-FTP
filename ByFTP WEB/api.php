<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Operations\RemoteOperations;
use ByFTP\Remote\ClientFactory;
use ByFTP\Remote\PathGuard;
use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Storage\PreferenceStore;
use ByFTP\Storage\ProfileStore;

if (!Auth::check()) {
    byftp_json(['ok' => false, 'error' => 'Sesija je istekla.'], 401);
}
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    byftp_json(['ok' => false, 'error' => 'Method not allowed.'], 405);
}
$contentLength = (int)($_SERVER['CONTENT_LENGTH'] ?? 0);
$postMax = byftp_ini_bytes(ini_get('post_max_size'));
if ($contentLength > 0 && $postMax < PHP_INT_MAX && $contentLength > $postMax && $_POST === [] && $_FILES === []) {
    byftp_json(['ok' => false, 'error' => 'Zahtjev prelazi PHP post_max_size limit ovog hostinga.'], 413);
}

$csrf = $_SERVER['HTTP_X_CSRF_TOKEN'] ?? $_POST['csrf'] ?? null;
if (!byftp_verify_csrf(is_string($csrf) ? $csrf : null)) {
    byftp_json(['ok' => false, 'error' => 'Sigurnosni token nije valjan.'], 419);
}

$action = trim((string)($_POST['action'] ?? ''));
$userId = Auth::id();
byftp_release_session_lock();
$store = new ProfileStore($userId);
$preferences = new PreferenceStore($userId);
$client = null;
$profileId = '';

register_shutdown_function(static function () use (&$client): void {
    if ($client !== null) {
        try { $client->disconnect(); } catch (Throwable) {}
    }
});

try {
    if ($action === 'me') {
        byftp_json(['ok' => true, 'user' => Auth::user(), 'preferences' => $preferences->clientState()]);
    }
    if ($action === 'save_preferences') {
        $payload = json_decode((string)($_POST['preferences'] ?? '{}'), true);
        if (!is_array($payload)) throw new RuntimeException('Postavke nisu valjane.');
        byftp_json(['ok' => true, 'preferences' => $preferences->saveClientState($payload)]);
    }
    if ($action === 'profiles') {
        byftp_json(['ok' => true, 'profiles' => $store->all(false)]);
    }
    if ($action === 'save_profile') {
        $profile = $store->save($_POST);
        AppLogger::event('profile.save', ['profile_id' => $profile['id'] ?? '']);
        byftp_json(['ok' => true, 'profile' => $profile]);
    }
    if ($action === 'delete_profile') {
        $id = (string)($_POST['id'] ?? '');
        $store->delete($id);
        AppLogger::event('profile.delete', ['profile_id' => $id]);
        byftp_json(['ok' => true]);
    }
    if ($action === 'duplicate_profile') {
        $id = (string)($_POST['id'] ?? '');
        $profile = $store->duplicate($id);
        AppLogger::event('profile.duplicate', ['profile_id' => $id, 'new_profile_id' => $profile['id'] ?? '']);
        byftp_json(['ok' => true, 'profile' => $profile]);
    }
    if ($action === 'test_profile_draft') {
        $draft = $store->connectionDraft($_POST);
        $profileId = (string)($draft['id'] ?? 'draft');
        $client = ClientFactory::make($draft);
        $started = microtime(true);
        $client->connect();
        $client->list('/');
        $elapsedMs = max(1, (int)round((microtime(true) - $started) * 1000));
        AppLogger::event('connection.test_draft', ['profile_id' => $profileId, 'protocol' => (string)($draft['protocol'] ?? ''), 'host' => (string)($draft['host'] ?? ''), 'elapsed_ms' => $elapsedMs]);
        byftp_json(['ok' => true, 'message' => 'Veza i početna putanja su dostupne.', 'elapsed_ms' => $elapsedMs]);
    }
    if ($action === 'ping') {
        byftp_json(['ok' => true, 'version' => BYFTP_VERSION, 'time' => gmdate('c')]);
    }

    $profileId = (string)($_POST['profile_id'] ?? '');
    $profile = $store->find($profileId, true);
    if (!$profile) throw new RuntimeException('Profil nije pronađen.');

    if ($action === 'favorites') {
        byftp_json(['ok' => true, 'favorites' => $preferences->favorites($profileId)]);
    }
    if ($action === 'toggle_favorite') {
        $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
        byftp_json(['ok' => true, 'favorites' => $preferences->toggleFavorite($profileId, $path)]);
    }

    $client = ClientFactory::make($profile);
    $ops = new RemoteOperations($client);

    switch ($action) {
        case 'test':
            $client->connect();
            AppLogger::event('connection.test', ['profile_id' => $profileId]);
            byftp_json(['ok' => true, 'message' => 'Veza je uspješna.']);

        case 'list':
            $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            byftp_json(['ok'=>true,'path'=>$path,'items'=>$client->list($path),'favorites'=>$preferences->favorites($profileId),'features'=>['zip'=>class_exists(ZipArchive::class),'checksum'=>true,'batch_rename'=>true,'analyze'=>true]]);

        case 'mkdir':
            $parent = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            $target = PathGuard::child($parent, PathGuard::basename((string)($_POST['name'] ?? '')));
            $client->makeDirectory($target);
            AppLogger::event('directory.create', ['profile_id'=>$profileId,'path'=>$target]);
            byftp_json(['ok'=>true,'path'=>$target]);

        case 'new_file':
            $parent = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            $target = PathGuard::child($parent, PathGuard::basename((string)($_POST['name'] ?? '')));
            $ops->createFile($target, (string)($_POST['content'] ?? ''));
            AppLogger::event('file.create', ['profile_id'=>$profileId,'path'=>$target]);
            byftp_json(['ok'=>true,'path'=>$target]);

        case 'rename':
            $from = PathGuard::ensureNotRoot((string)($_POST['from'] ?? ''));
            $to = PathGuard::child(PathGuard::parent($from), PathGuard::basename((string)($_POST['name'] ?? '')));
            $client->rename($from, $to);
            AppLogger::event('item.rename', ['profile_id'=>$profileId,'from'=>$from,'to'=>$to]);
            byftp_json(['ok'=>true,'path'=>$to]);

        case 'delete':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            $type = (string)($_POST['type'] ?? 'file') === 'dir' ? 'dir' : 'file';
            $recursive = !isset($_POST['recursive']) || filter_var($_POST['recursive'], FILTER_VALIDATE_BOOLEAN);
            if ($type === 'dir' && $recursive) $ops->deleteRecursive($path, 'dir'); else $client->delete($path, $type === 'dir');
            AppLogger::event('item.delete', ['profile_id'=>$profileId,'path'=>$path,'type'=>$type]);
            byftp_json(['ok'=>true]);

        case 'bulk_delete':
            $items = byftp_json_array((string)($_POST['items'] ?? '[]'), 200);
            $deleted = 0;
            foreach ($items as $item) {
                if (!is_array($item)) continue;
                $path = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));
                $type = (string)($item['type'] ?? 'file') === 'dir' ? 'dir' : 'file';
                $ops->deleteRecursive($path, $type);
                $deleted++;
            }
            AppLogger::event('item.bulk_delete', ['profile_id'=>$profileId,'count'=>$deleted]);
            byftp_json(['ok'=>true,'deleted'=>$deleted]);

        case 'copy':
        case 'move':
            $source = PathGuard::ensureNotRoot((string)($_POST['source'] ?? ''));
            $destination = PathGuard::ensureNotRoot((string)($_POST['destination'] ?? ''));
            $policy = strtolower(trim((string)($_POST['conflict'] ?? 'rename')));
            if (!in_array($policy, ['rename','overwrite','skip','error'], true)) throw new RuntimeException('Neispravna politika konflikta.');
            if ($action === 'move' && $source === $destination) byftp_json(['ok'=>true,'path'=>$source,'skipped'=>true]);
            if ($source === $destination && $action === 'copy') {
                $destination = $ops->uniquePath($destination);
            } elseif ($ops->exists($destination)) {
                if ($policy === 'skip') byftp_json(['ok'=>true,'path'=>$destination,'skipped'=>true]);
                if ($policy === 'error') throw new RuntimeException('Na odredištu već postoji stavka istog naziva.');
                if ($policy === 'rename') $destination = $ops->uniquePath($destination);
            }
            if ($action === 'copy') $ops->copy($source, $destination); else $ops->move($source, $destination);
            AppLogger::event('item.'.$action, ['profile_id'=>$profileId,'source'=>$source,'destination'=>$destination,'conflict'=>$policy]);
            byftp_json(['ok'=>true,'path'=>$destination,'skipped'=>false]);

        case 'duplicate':
            $source = PathGuard::ensureNotRoot((string)($_POST['source'] ?? ''));
            $path = $ops->duplicate($source);
            AppLogger::event('item.duplicate', ['profile_id'=>$profileId,'source'=>$source,'destination'=>$path]);
            byftp_json(['ok'=>true,'path'=>$path]);

        case 'read':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            $content = $client->read($path, 4194304);
            byftp_json(['ok'=>true,'content'=>$content,'etag'=>hash('sha256',$content),'bytes'=>strlen($content)]);

        case 'write':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            $content = (string)($_POST['content'] ?? '');
            if (strlen($content) > 4194304) throw new RuntimeException('Sadržaj je prevelik za web editor. Maksimum je 4 MiB.');
            $ifMatch = trim((string)($_POST['if_match'] ?? ''));
            if ($ifMatch !== '') {
                $current = $client->read($path, 4194304);
                if (!hash_equals($ifMatch, hash('sha256',$current))) throw new RuntimeException('Datoteka je izmijenjena na serveru nakon otvaranja. Osvježi editor prije spremanja kako ne bi prepisao tuđe promjene.');
            }
            $ops->writeAtomic($path, $content);
            AppLogger::event('file.write', ['profile_id'=>$profileId,'path'=>$path,'bytes'=>strlen($content)]);
            byftp_json(['ok'=>true,'bytes'=>strlen($content),'etag'=>hash('sha256',$content)]);

        case 'checksum':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            byftp_json(['ok'=>true] + $ops->checksum($path, (string)($_POST['algorithm'] ?? 'sha256')));

        case 'analyze':
            $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            byftp_json(['ok'=>true,'stats'=>$ops->analyze($path)]);

        case 'batch_rename':
            $items = byftp_json_array((string)($_POST['items'] ?? '[]'), 200);
            $result = $ops->batchRename($items, byftp_truncate((string)($_POST['find'] ?? ''),120), byftp_truncate((string)($_POST['replace'] ?? ''),120), byftp_truncate((string)($_POST['prefix'] ?? ''),80), byftp_truncate((string)($_POST['suffix'] ?? ''),80));
            AppLogger::event('item.batch_rename', ['profile_id'=>$profileId,'count'=>$result['renamed'] ?? 0]);
            byftp_json(['ok'=>true] + $result);

        case 'chmod':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            $raw = (string)($_POST['mode'] ?? '');
            if (!preg_match('/^[0-7]{3,4}$/', $raw)) throw new RuntimeException('CHMOD vrijednost nije valjana.');
            if (filter_var($_POST['recursive'] ?? false, FILTER_VALIDATE_BOOLEAN)) {
                $dirRaw = (string)($_POST['dir_mode'] ?? $raw);
                if (!preg_match('/^[0-7]{3,4}$/', $dirRaw)) throw new RuntimeException('CHMOD vrijednost direktorija nije valjana.');
                $ops->chmodRecursive($path, octdec($raw), octdec($dirRaw));
            } else $client->chmod($path, octdec($raw));
            AppLogger::event('item.chmod', ['profile_id'=>$profileId,'path'=>$path,'mode'=>$raw]);
            byftp_json(['ok'=>true]);

        case 'search':
            $root = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            byftp_json(['ok'=>true,'results'=>$ops->search($root, byftp_truncate(trim((string)($_POST['query'] ?? '')),100))]);

        case 'zip':
            $paths = byftp_string_paths((string)($_POST['paths'] ?? '[]'), 200);
            $destination = PathGuard::ensureNotRoot((string)($_POST['destination'] ?? ''));
            $ops->zipPaths($paths, $destination);
            AppLogger::event('archive.create', ['profile_id'=>$profileId,'destination'=>$destination,'count'=>count($paths)]);
            byftp_json(['ok'=>true,'path'=>$destination]);

        case 'extract':
            $path = PathGuard::ensureNotRoot((string)($_POST['path'] ?? ''));
            $destination = PathGuard::normalizeRelative((string)($_POST['destination'] ?? PathGuard::parent($path)));
            $result = $ops->extractZip($path, $destination);
            AppLogger::event('archive.extract', ['profile_id'=>$profileId,'path'=>$path,'destination'=>$destination,'files'=>$result['files']]);
            byftp_json(['ok'=>true] + $result);

        case 'upload':
            $path = PathGuard::normalizeRelative((string)($_POST['path'] ?? '/'));
            $files = $_FILES['files'] ?? null;
            if (!$files) throw new RuntimeException('Nema datoteka za upload.');
            $names = is_array($files['name']) ? $files['name'] : [$files['name']];
            $tmps = is_array($files['tmp_name']) ? $files['tmp_name'] : [$files['tmp_name']];
            $errors = is_array($files['error']) ? $files['error'] : [$files['error']];
            $relativePaths = $_POST['relative_paths'] ?? [];
            if (!is_array($relativePaths)) $relativePaths = [$relativePaths];
            if (count($names) > 200) throw new RuntimeException('Previše datoteka u jednom upload zahtjevu.');
            $conflictPolicy = strtolower(trim((string)($_POST['conflict'] ?? 'overwrite')));
            if (!in_array($conflictPolicy, ['overwrite','skip','rename'], true)) throw new RuntimeException('Neispravna politika konflikta pri uploadu.');
            $uploaded = [];
            $skipped = [];
            foreach ($names as $i => $original) {
                $error = (int)($errors[$i] ?? UPLOAD_ERR_NO_FILE);
                if ($error !== UPLOAD_ERR_OK) throw new RuntimeException(byftp_upload_error_message($error, (string)$original));
                $tmp = (string)($tmps[$i] ?? '');
                if (!is_uploaded_file($tmp)) throw new RuntimeException('Privremena upload datoteka nije valjana.');
                $relative = trim((string)($relativePaths[$i] ?? ''));
                if ($relative !== '') {
                    $relative = PathGuard::normalizeRelative('/'.ltrim($relative,'/'));
                    $remote = PathGuard::normalizeRelative(($path === '/' ? '' : $path).$relative);
                } else {
                    $remote = PathGuard::child($path, PathGuard::basename((string)$original));
                }
                $finalRemote = $ops->uploadWithConflict($tmp, $remote, $conflictPolicy);
                if ($finalRemote === null) $skipped[] = $remote; else $uploaded[] = $finalRemote;
            }
            AppLogger::event('file.upload', ['profile_id'=>$profileId,'count'=>count($uploaded),'skipped'=>count($skipped),'path'=>$path]);
            byftp_json(['ok'=>true,'uploaded'=>$uploaded,'skipped'=>$skipped]);
    }
    throw new RuntimeException('Nepoznata akcija.');
} catch (Throwable $e) {
    AppLogger::event('api.error', ['action'=>$action,'profile_id'=>$profileId,'error'=>byftp_truncate($e->getMessage(),300)]);
    byftp_json(['ok'=>false,'error'=>$e->getMessage()],400);
} finally {
    if ($client) $client->disconnect();
}
