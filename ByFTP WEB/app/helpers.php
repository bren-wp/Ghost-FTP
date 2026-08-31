<?php
declare(strict_types=1);

function byftp_config_path(): string
{
    return BYFTP_STORAGE . '/app.json';
}

function byftp_is_configured(): bool
{
    $path = byftp_config_path();
    if (!is_file($path) && !is_file($path . '.bak')) {
        return false;
    }
    $config = byftp_config();
    $key = base64_decode((string)($config['secret_key'] ?? ''), true);
    if (!is_string($key) || strlen($key) !== 32) {
        return false;
    }

    // Legacy single-admin installations used one password hash instead of users.json.
    if (!empty($config['password_hash'])) {
        return true;
    }

    // A valid encryption key without any user can be left behind by an interrupted
    // first setup. Treat that state as recoverable setup instead of an unusable login.
    try {
        return (new ByFTP\Storage\UserStore())->count() > 0;
    } catch (Throwable) {
        // If user storage exists but is unreadable/corrupt, fail closed. Never expose
        // setup in a way that could rotate the key over potentially recoverable data.
        return is_file(BYFTP_STORAGE . '/users.json') || is_file(BYFTP_STORAGE . '/users.json.bak');
    }
}

function byftp_config(bool $fresh = false): array
{
    if ($fresh || !array_key_exists('byftp_config_cache', $GLOBALS)) {
        $path = byftp_config_path();
        if (!is_file($path) && !is_file($path . '.bak')) {
            $GLOBALS['byftp_config_cache'] = [];
            unset($GLOBALS['byftp_config_error']);
        } else {
            try {
                // app.json contains the encryption key and runtime security policy.
                // Keep the adjacent .bak for explicit operator recovery, but never
                // silently roll security settings back to an older generation.
                $GLOBALS['byftp_config_cache'] = (new ByFTP\Storage\JsonStore($path, false))->read([]);
                unset($GLOBALS['byftp_config_error']);
            } catch (Throwable $e) {
                // Fail closed into recovery/setup instead of crashing every request.
                $GLOBALS['byftp_config_cache'] = [];
                $GLOBALS['byftp_config_error'] = $e->getMessage();
            }
        }
    }
    return is_array($GLOBALS['byftp_config_cache'] ?? null) ? $GLOBALS['byftp_config_cache'] : [];
}

function byftp_write_config(array $data): void
{
    $store = new ByFTP\Storage\JsonStore(byftp_config_path());
    $store->write($data);
    $GLOBALS['byftp_config_cache'] = $data;
    unset($GLOBALS['byftp_config_error']);
}

function byftp_update_config(array $changes): array
{
    $current = byftp_config(true);
    if (isset($GLOBALS['byftp_config_error'])) {
        throw new RuntimeException('Konfiguracija aplikacije nije čitljiva. Vrati storage/app.json iz provjerene sigurnosne kopije prije spremanja postavki.');
    }
    $key = base64_decode((string)($current['secret_key'] ?? ''), true);
    if (!is_string($key) || strlen($key) !== 32) {
        throw new RuntimeException('Konfiguracija aplikacije nema valjan encryption ključ. Postavke nisu spremljene.');
    }
    $next = array_replace($current, $changes);
    byftp_write_config($next);
    return $next;
}

function byftp_app_name(): string
{
    return (string)(byftp_config()['app_name'] ?? 'ByFTP');
}

function byftp_registration_enabled(): bool
{
    return !empty(byftp_config()['allow_registration']);
}

function byftp_private_hosts_allowed(): bool
{
    return !empty(byftp_config()['allow_private_hosts']);
}

function byftp_csrf_token(): string
{
    if (empty($_SESSION['csrf'])) {
        $_SESSION['csrf'] = bin2hex(random_bytes(32));
    }
    return (string)$_SESSION['csrf'];
}

function byftp_verify_csrf(?string $token): bool
{
    return is_string($token) && hash_equals(byftp_csrf_token(), $token);
}

function byftp_e(string $value): string
{
    return htmlspecialchars($value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

function byftp_json(array $payload, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE);
    exit;
}

function byftp_truncate(string $value, int $length): string
{
    return function_exists('mb_substr') ? mb_substr($value, 0, $length) : substr($value, 0, $length);
}

function byftp_archive_download_name(string $value): string
{
    $name = trim($value);
    $name = preg_replace('/[^\pL\pN._ -]+/u', '-', $name) ?: 'byftp-download';
    $base = preg_replace('/\.zip$/i', '', $name);
    $base = is_string($base) ? $base : '';
    $base = rtrim(byftp_truncate($base, 116), " .");
    if ($base === '') {
        $base = 'byftp-download';
    }
    return $base . '.zip';
}

function byftp_client_ip(): string
{
    // REMOTE_ADDR is intentionally used directly. Forwarded headers are not trusted by default.
    return (string)($_SERVER['REMOTE_ADDR'] ?? '0.0.0.0');
}

function byftp_json_array(string $json, int $maxItems = 200): array
{
    $decoded = json_decode($json, true);
    if (!is_array($decoded)) {
        throw new RuntimeException('Neispravan JSON popis.');
    }
    if (count($decoded) > $maxItems) {
        throw new RuntimeException('Previše stavki u jednom zahtjevu.');
    }
    return array_values($decoded);
}

function byftp_string_paths(string $json, int $maxItems = 200): array
{
    $rows = byftp_json_array($json, $maxItems);
    $paths = [];
    foreach ($rows as $row) {
        if (!is_string($row)) {
            continue;
        }
        $paths[] = ByFTP\Remote\PathGuard::ensureNotRoot($row);
    }
    return array_values(array_unique($paths));
}

function byftp_base_path(): string
{
    static $base = null;
    if ($base !== null) {
        return $base;
    }

    $root = realpath(BYFTP_ROOT);
    $documentRoot = realpath((string)($_SERVER['DOCUMENT_ROOT'] ?? ''));
    if (is_string($root) && is_string($documentRoot)) {
        $rootNormalized = rtrim(str_replace('\\', '/', $root), '/');
        $documentNormalized = rtrim(str_replace('\\', '/', $documentRoot), '/');
        if ($documentNormalized !== '' && ($rootNormalized === $documentNormalized || str_starts_with($rootNormalized, $documentNormalized . '/'))) {
            $relative = substr($rootNormalized, strlen($documentNormalized));
            return $base = $relative === '' ? '' : '/' . trim($relative, '/');
        }
    }

    $scriptName = str_replace('\\', '/', (string)($_SERVER['SCRIPT_NAME'] ?? ''));
    if ($scriptName === '' || $scriptName === '/') {
        return $base = '';
    }
    if (str_ends_with($scriptName, '/')) {
        return $base = rtrim($scriptName, '/');
    }
    $dir = str_replace('\\', '/', dirname($scriptName));
    return $base = ($dir === '/' || $dir === '.' ? '' : rtrim($dir, '/'));
}

function byftp_url(string $route = 'app'): string
{
    $path = match ($route) {
        'app' => '/',
        'login' => '/login',
        'logout' => '/logout',
        'register' => '/register',
        'account' => '/account',
        'users' => '/users',
        'settings' => '/settings',
        'setup' => '/setup',
        'diagnostics' => '/diagnostics',
        'api' => '/api',
        'download' => '/download/file',
        'download_archive' => '/download/archive',
        'preview' => '/preview/file',
        'manifest' => '/manifest.webmanifest',
        'service_worker' => '/service-worker.js',
        default => throw new InvalidArgumentException('Nepoznata ByFTP ruta.'),
    };
    $base = byftp_base_path();
    if ($path === '/') {
        return ($base === '' ? '' : $base) . '/';
    }
    return $base . $path;
}

function byftp_asset(string $path): string
{
    $clean = ltrim(str_replace('\\', '/', $path), '/');
    if ($clean === '' || str_contains($clean, '..')) {
        throw new InvalidArgumentException('Neispravna putanja asseta.');
    }
    return byftp_base_path() . '/assets/' . $clean . '?v=' . rawurlencode(BYFTP_VERSION);
}

function byftp_redirect(string $route, array $query = []): never
{
    $url = byftp_url($route);
    if ($query !== []) {
        $url .= '?' . http_build_query($query, '', '&', PHP_QUERY_RFC3986);
    }
    header('Location: ' . $url, true, 302);
    exit;
}

function byftp_ini_bytes(string|false $value): int
{
    if ($value === false) {
        return PHP_INT_MAX;
    }
    $value = trim($value);
    if ($value === '' || $value === '-1' || $value === '0') {
        return PHP_INT_MAX;
    }
    if (!preg_match('/^([0-9]+(?:\.[0-9]+)?)\s*([kmgtpe]?)b?$/i', $value, $match)) {
        return (int)$value;
    }
    $number = (float)$match[1];
    $unit = strtolower($match[2]);
    $powers = ['' => 0, 'k' => 1, 'm' => 2, 'g' => 3, 't' => 4, 'p' => 5, 'e' => 6];
    $bytes = $number * (1024 ** ($powers[$unit] ?? 0));
    return $bytes >= PHP_INT_MAX ? PHP_INT_MAX : (int)$bytes;
}

function byftp_upload_limit_bytes(): int
{
    return min(byftp_ini_bytes(ini_get('upload_max_filesize')), byftp_ini_bytes(ini_get('post_max_size')));
}

function byftp_upload_error_message(int $code, string $name = ''): string
{
    $label = $name !== '' ? ' (' . byftp_truncate($name, 100) . ')' : '';
    return match ($code) {
        UPLOAD_ERR_INI_SIZE, UPLOAD_ERR_FORM_SIZE => 'Datoteka je veća od dopuštenog upload limita' . $label . '.',
        UPLOAD_ERR_PARTIAL => 'Upload je samo djelomično dovršen' . $label . '.',
        UPLOAD_ERR_NO_FILE => 'Datoteka nije odabrana' . $label . '.',
        UPLOAD_ERR_NO_TMP_DIR => 'PHP privremeni direktorij nije dostupan.',
        UPLOAD_ERR_CANT_WRITE => 'Poslužitelj ne može zapisati privremenu upload datoteku.',
        UPLOAD_ERR_EXTENSION => 'PHP ekstenzija je zaustavila upload' . $label . '.',
        default => 'Upload nije uspio' . $label . '. PHP kod: ' . $code,
    };
}

function byftp_assert_temp_capacity(int $expectedBytes, int $reserveBytes = 16777216): void
{
    if ($expectedBytes <= 0) {
        return;
    }
    $free = @disk_free_space(BYFTP_STORAGE . '/tmp');
    if (!is_float($free) && !is_int($free)) {
        return; // Some shared hosts do not expose disk_free_space().
    }
    $needed = $expectedBytes + max(0, $reserveBytes);
    if ($free < $needed) {
        throw new RuntimeException('Nema dovoljno slobodnog prostora u storage/tmp za ovu operaciju. Oslobodi prostor ili koristi manju datoteku.');
    }
}

function byftp_cleanup_stale_temp_files(int $maxAgeSeconds = 86400, int $maxFiles = 100): int
{
    $directory = BYFTP_STORAGE . '/tmp';
    if (!is_dir($directory) || $maxAgeSeconds < 3600 || $maxFiles < 1) {
        return 0;
    }
    $cutoff = time() - $maxAgeSeconds;
    $removed = 0;
    $entries = @scandir($directory);
    if (!is_array($entries)) {
        return 0;
    }
    foreach ($entries as $entry) {
        if ($entry === '.' || $entry === '..' || $removed >= $maxFiles) {
            continue;
        }
        $path = $directory . '/' . $entry;
        if (!is_file($path) && !is_link($path)) {
            continue;
        }
        $modified = @filemtime($path);
        if (!is_int($modified) || $modified >= $cutoff) {
            continue;
        }
        if (@unlink($path)) {
            $removed++;
        }
    }
    return $removed;
}

function byftp_session_limits(?array $config = null): array
{
    $config ??= byftp_config();
    $idleSeconds = max(900, min(86400, (int)($config['session_idle_minutes'] ?? 120) * 60));
    $maxSeconds = max(3600, min(604800, (int)($config['session_max_hours'] ?? 12) * 3600));
    return ['idle' => $idleSeconds, 'max' => $maxSeconds, 'gc' => max($idleSeconds, $maxSeconds)];
}

function byftp_release_session_lock(): void
{
    if (session_status() === PHP_SESSION_ACTIVE) {
        session_write_close();
    }
}

function byftp_human_date(?string $iso): string
{
    if (!$iso) {
        return '—';
    }
    try {
        $date = new DateTimeImmutable($iso);
        return $date->setTimezone(new DateTimeZone(date_default_timezone_get()))->format('d.m.Y. H:i');
    } catch (Throwable) {
        return '—';
    }
}
