<?php
declare(strict_types=1);

namespace GhostFTP\Storage;

use GhostFTP\Remote\PathGuard;
use GhostFTP\Security\Crypto;
use RuntimeException;

final class ProfileStore
{
    private JsonStore $store;
    private Crypto $crypto;

    public function __construct(string $userId)
    {
        UserWorkspace::ensure($userId);
        // Saved connection profiles contain encrypted passwords/private keys and explicit
        // user deletion intent. Keep profiles.json.bak for manual recovery only; runtime
        // reads must never resurrect a deleted credential generation automatically.
        $this->store = new JsonStore(UserWorkspace::file($userId, 'profiles.json'), false);
        $this->crypto = new Crypto();
    }

    public function all(bool $withSecrets = false): array
    {
        $result = [];
        foreach ($this->read() as $row) {
            $result[] = $this->hydrate($row, $withSecrets);
        }
        usort($result, static fn(array $a, array $b): int => strnatcasecmp((string)$a['label'], (string)$b['label']));
        return $result;
    }

    public function find(string $id, bool $withSecrets = true): ?array
    {
        foreach ($this->read() as $row) {
            if (($row['id'] ?? '') === $id) {
                return $this->hydrate($row, $withSecrets);
            }
        }
        return null;
    }

    /**
     * Build and validate a connection profile from the current form without persisting it.
     * Blank secrets preserve stored values only while the endpoint/account binding remains
     * unchanged. Key passphrases are additionally bound to the same private key material.
     */
    public function connectionDraft(array $input): array
    {
        $draft = $this->normalizeInput($input);
        $old = null;
        if ($draft['id'] !== '') {
            $old = $this->find($draft['id'], true);
            if ($old === null) {
                throw new RuntimeException('Profil nije pronađen. Osvježi popis veza i pokušaj ponovno.');
            }
        }

        $sameAccount = is_array($old) && self::accountMatches($old, $draft);
        $sameSftpAccount = $sameAccount && $draft['protocol'] === 'sftp';
        $oldPrivateKey = $sameSftpAccount ? (string)($old['private_key'] ?? '') : '';

        $password = $this->resolvePlainSecret(
            $draft['password'],
            $sameAccount ? (string)($old['password'] ?? '') : '',
            $draft['clear_password']
        );
        $publicKey = $this->resolvePlainSecret(
            $draft['public_key'],
            $sameSftpAccount ? (string)($old['public_key'] ?? '') : '',
            $draft['clear_key_material']
        );
        $privateKey = $this->resolvePlainSecret(
            $draft['private_key'],
            $oldPrivateKey,
            $draft['clear_key_material']
        );
        $samePrivateKey = $sameSftpAccount
            && $oldPrivateKey !== ''
            && $privateKey !== ''
            && hash_equals($oldPrivateKey, $privateKey);
        $keyPassphrase = $this->resolvePlainSecret(
            $draft['key_passphrase'],
            $samePrivateKey ? (string)($old['key_passphrase'] ?? '') : '',
            $draft['clear_key_passphrase'] || $draft['clear_key_material']
        );

        if ($draft['auth_method'] === 'key' && ($publicKey === '' || $privateKey === '')) {
            throw new RuntimeException('Za SFTP autentikaciju ključem unesi javni i privatni ključ.');
        }

        return [
            'id' => $draft['id'],
            'label' => $draft['label'],
            'protocol' => $draft['protocol'],
            'host' => $draft['host'],
            'port' => $draft['port'],
            'base_path' => $draft['base_path'],
            'passive' => $draft['passive'],
            'utf8' => $draft['utf8'],
            'timeout' => $draft['timeout'],
            'host_fingerprint' => $draft['host_fingerprint'],
            'auth_method' => $draft['auth_method'],
            'username' => $draft['username'],
            'password' => $password,
            'public_key' => $publicKey,
            'private_key' => $privateKey,
            'key_passphrase' => $keyPassphrase,
        ];
    }

    public function save(array $input): array
    {
        $normalized = $this->normalizeInput($input);
        $id = $normalized['id'] !== '' ? $normalized['id'] : bin2hex(random_bytes(8));
        $saved = null;

        $this->store->update(function (array $rows) use ($id, $normalized, &$saved): array {
            $existingIndex = null;
            foreach ($rows as $idx => $row) {
                if (is_array($row) && ($row['id'] ?? '') === $id) {
                    $existingIndex = $idx;
                    break;
                }
            }

            if ($normalized['id'] !== '' && $existingIndex === null) {
                throw new RuntimeException('Profil nije pronađen. Osvježi popis veza i pokušaj ponovno.');
            }

            $old = $existingIndex !== null && is_array($rows[$existingIndex] ?? null) ? $rows[$existingIndex] : null;
            $oldPlain = is_array($old) ? $this->hydrate($old, true) : null;
            $sameAccount = is_array($oldPlain) && self::accountMatches($oldPlain, $normalized);
            $sameSftpAccount = $sameAccount && $normalized['protocol'] === 'sftp';
            $oldPrivateKey = $sameSftpAccount ? (string)($oldPlain['private_key'] ?? '') : '';

            $passwordEnc = $this->resolveEncryptedSecret(
                $normalized['password'],
                $sameAccount && is_array($old) ? (string)($old['password_enc'] ?? '') : '',
                $normalized['clear_password'],
                true
            );
            $publicKeyEnc = $this->resolveEncryptedSecret(
                $normalized['public_key'],
                $sameSftpAccount && is_array($old) ? (string)($old['public_key_enc'] ?? '') : '',
                $normalized['clear_key_material']
            );
            $privateKeyEnc = $this->resolveEncryptedSecret(
                $normalized['private_key'],
                $sameSftpAccount && is_array($old) ? (string)($old['private_key_enc'] ?? '') : '',
                $normalized['clear_key_material']
            );
            $finalPrivateKey = '';
            if (!$normalized['clear_key_material']) {
                $finalPrivateKey = $normalized['private_key'] !== '' ? $normalized['private_key'] : $oldPrivateKey;
            }
            $samePrivateKey = $sameSftpAccount
                && $oldPrivateKey !== ''
                && $finalPrivateKey !== ''
                && hash_equals($oldPrivateKey, $finalPrivateKey);
            $keyPassphraseEnc = $this->resolveEncryptedSecret(
                $normalized['key_passphrase'],
                $samePrivateKey && is_array($old) ? (string)($old['key_passphrase_enc'] ?? '') : '',
                $normalized['clear_key_passphrase'] || $normalized['clear_key_material']
            );

            if ($normalized['auth_method'] === 'key' && ($publicKeyEnc === '' || $privateKeyEnc === '')) {
                throw new RuntimeException('Za SFTP autentikaciju ključem unesi javni i privatni ključ.');
            }

            $row = [
                'id' => $id,
                'label' => $normalized['label'],
                'protocol' => $normalized['protocol'],
                'host' => $normalized['host'],
                'port' => $normalized['port'],
                'base_path' => $normalized['base_path'],
                'passive' => $normalized['passive'],
                'utf8' => $normalized['utf8'],
                'timeout' => $normalized['timeout'],
                'host_fingerprint' => $normalized['host_fingerprint'],
                'auth_method' => $normalized['auth_method'],
                'username_enc' => $this->crypto->encrypt($normalized['username']),
                'password_enc' => $passwordEnc,
                'public_key_enc' => $publicKeyEnc,
                'private_key_enc' => $privateKeyEnc,
                'key_passphrase_enc' => $keyPassphraseEnc,
                'updated_at' => gmdate('c'),
                'created_at' => is_array($old) ? (string)($old['created_at'] ?? gmdate('c')) : gmdate('c'),
            ];

            if ($existingIndex === null) {
                $rows[] = $row;
            } else {
                $rows[$existingIndex] = $row;
            }
            $saved = $row;
            return array_values($rows);
        });

        return is_array($saved) ? $this->hydrate($saved, false) : [];
    }

    public function duplicate(string $id): array
    {
        $profile = $this->find($id, true);
        if (!$profile) {
            throw new RuntimeException('Profil nije pronađen.');
        }
        unset($profile['id'], $profile['created_at'], $profile['updated_at'], $profile['has_private_key'], $profile['has_password'], $profile['has_key_passphrase']);
        $profile['label'] = \GhostFTP_truncate((string)$profile['label'] . ' – kopija', 80);
        return $this->save($profile);
    }

    public function delete(string $id): void
    {
        if ($id === '') {
            return;
        }
        $this->store->update(static fn(array $rows): array => array_values(array_filter(
            $rows,
            static fn($row): bool => !is_array($row) || ($row['id'] ?? '') !== $id
        )));
    }

    private function normalizeInput(array $input): array
    {
        $rawId = (string)($input['id'] ?? '');
        $requestedId = preg_replace('/[^a-zA-Z0-9_-]/', '', $rawId) ?? '';
        if ($rawId !== '' && !hash_equals($rawId, $requestedId)) {
            throw new RuntimeException('Identifikator profila nije valjan.');
        }

        $rawHost = (string)($input['host'] ?? '');
        if ($rawHost === '' || $rawHost !== trim($rawHost) || preg_match('/[\x00-\x20\x7F]/', $rawHost)) {
            throw new RuntimeException('Host nije valjan. Ne koristi rubne razmake ili kontrolne znakove.');
        }
        $host = $rawHost;
        if (str_starts_with($host, '[') || str_ends_with($host, ']')) {
            if (!(str_starts_with($host, '[') && str_ends_with($host, ']'))) {
                throw new RuntimeException('IPv6 host ima neispravne uglate zagrade.');
            }
            $host = substr($host, 1, -1);
        }

        $label = trim((string)($input['label'] ?? ''));
        $protocol = (string)($input['protocol'] ?? 'ftp');
        if (!in_array($protocol, ['ftp', 'ftps', 'sftp'], true)) {
            throw new RuntimeException('Nepodržani ili nekanonski protokol.');
        }

        $rawPort = $input['port'] ?? ($protocol === 'sftp' ? '22' : '21');
        if (is_int($rawPort)) {
            $port = $rawPort;
        } elseif (is_string($rawPort) && preg_match('/^[0-9]{1,5}$/', $rawPort)) {
            $port = (int)$rawPort;
        } else {
            throw new RuntimeException('Port mora sadržavati samo znamenke bez razmaka ili dodatnih znakova.');
        }

        $basePath = (string)($input['base_path'] ?? '/');
        if ($basePath === '') {
            $basePath = '/';
        }
        $username = (string)($input['username'] ?? '');
        $password = (string)($input['password'] ?? '');
        $timeoutRaw = $input['timeout'] ?? 30;
        if (is_int($timeoutRaw)) {
            $timeout = $timeoutRaw;
        } elseif (is_string($timeoutRaw) && preg_match('/^[0-9]{1,3}$/', $timeoutRaw)) {
            $timeout = (int)$timeoutRaw;
        } else {
            throw new RuntimeException('Timeout mora biti cijeli broj bez dodatnih znakova.');
        }

        $fingerprint = (string)($input['host_fingerprint'] ?? '');
        $rawAuthMethod = (string)($input['auth_method'] ?? 'password');
        if (!in_array($rawAuthMethod, ['password', 'key'], true)) {
            throw new RuntimeException('SFTP metoda autentikacije nije valjana.');
        }
        $authMethod = $protocol === 'sftp' ? $rawAuthMethod : 'password';
        $publicKey = (string)($input['public_key'] ?? '');
        $privateKey = (string)($input['private_key'] ?? '');
        $keyPassphrase = (string)($input['key_passphrase'] ?? '');
        $passive = !isset($input['passive']) || filter_var($input['passive'], FILTER_VALIDATE_BOOLEAN);
        $utf8 = !isset($input['utf8']) || filter_var($input['utf8'], FILTER_VALIDATE_BOOLEAN);

        if ($protocol !== 'sftp') {
            // SFTP-only trust/key state must never survive a protocol switch. Keeping it
            // hidden in an FTP/FTPS profile could resurrect stale credentials later.
            $fingerprint = '';
            $publicKey = '';
            $privateKey = '';
            $keyPassphrase = '';
        }

        if ($label === '' || $host === '' || $username === '') {
            throw new RuntimeException('Naziv, host i korisničko ime su obavezni.');
        }
        if (str_contains($host, '://')) {
            throw new RuntimeException('U polje host unesi samo naziv servera ili IP adresu, bez ftp://, ftps:// ili sftp:// prefiksa.');
        }
        if (strlen($host) > 255 || strlen($username) > 512 || strlen($password) > 4096) {
            throw new RuntimeException('Host, korisničko ime ili lozinka prelaze dopuštenu veličinu.');
        }
        if (preg_match('/[\r\n\x00]/', $username) || preg_match('/[\r\n\x00]/', $password)) {
            throw new RuntimeException('Korisničko ime ili lozinka sadrže nedopuštene protokolarne kontrolne znakove.');
        }
        if ($port < 1 || $port > 65535) {
            throw new RuntimeException('Port nije valjan.');
        }
        if ($timeout < 5 || $timeout > 120) {
            throw new RuntimeException('Timeout mora biti između 5 i 120 sekundi.');
        }
        if (strlen($publicKey) > 8192 || strlen($privateKey) > 65536 || strlen($keyPassphrase) > 2048) {
            throw new RuntimeException('SFTP ključ ili passphrase prelazi dopuštenu veličinu.');
        }
        if (str_contains($publicKey, "\0") || str_contains($privateKey, "\0") || str_contains($keyPassphrase, "\0")) {
            throw new RuntimeException('SFTP ključ ili passphrase sadrži NUL znak.');
        }

        if ($fingerprint !== '') {
            $validFingerprint = false;
            if (preg_match('/^SHA256:([A-Za-z0-9+\/]{43})$/', $fingerprint, $match)) {
                $decoded = base64_decode($match[1] . '=', true);
                $validFingerprint = is_string($decoded) && strlen($decoded) === 32;
            } elseif (preg_match('/^[A-Fa-f0-9]{64}$/', $fingerprint)) {
                $validFingerprint = true;
            }
            if (!$validFingerprint) {
                throw new RuntimeException('SFTP host fingerprint mora biti kanonski OpenSSH SHA256: ili 64-znamenkasti SHA-256 HEX.');
            }
        }

        return [
            'id' => $requestedId,
            'label' => \GhostFTP_truncate($label, 80),
            'protocol' => $protocol,
            'host' => \GhostFTP_truncate($host, 255),
            'port' => $port,
            'base_path' => PathGuard::normalizeRelative($basePath),
            'username' => $username,
            'password' => $password,
            'passive' => $passive,
            'utf8' => $utf8,
            'timeout' => $timeout,
            'host_fingerprint' => \GhostFTP_truncate($fingerprint, 200),
            'auth_method' => $authMethod,
            'public_key' => $publicKey,
            'private_key' => $privateKey,
            'key_passphrase' => $keyPassphrase,
            'clear_password' => self::boolInput($input['clear_password'] ?? false),
            'clear_key_passphrase' => self::boolInput($input['clear_key_passphrase'] ?? false),
            'clear_key_material' => self::boolInput($input['clear_key_material'] ?? false),
        ];
    }

    private function resolveEncryptedSecret(string $newValue, string $oldEncrypted, bool $clear, bool $encryptEmpty = false): string
    {
        if ($clear) {
            return $encryptEmpty ? $this->crypto->encrypt('') : '';
        }
        if ($newValue !== '') {
            return $this->crypto->encrypt($newValue);
        }
        if ($oldEncrypted !== '') {
            return $oldEncrypted;
        }
        return $encryptEmpty ? $this->crypto->encrypt('') : '';
    }

    private function resolvePlainSecret(string $newValue, string $oldValue, bool $clear): string
    {
        if ($clear) {
            return '';
        }
        return $newValue !== '' ? $newValue : $oldValue;
    }

    private static function accountMatches(array $old, array $next): bool
    {
        return self::endpointMatches($old, $next)
            && hash_equals((string)($old['username'] ?? ''), (string)($next['username'] ?? ''));
    }

    private static function endpointMatches(array $old, array $next): bool
    {
        return (string)($old['protocol'] ?? '') === (string)($next['protocol'] ?? '')
            && (int)($old['port'] ?? 0) === (int)($next['port'] ?? 0)
            && hash_equals(
                self::canonicalBindingHost((string)($old['host'] ?? '')),
                self::canonicalBindingHost((string)($next['host'] ?? ''))
            );
    }

    private static function canonicalBindingHost(string $host): string
    {
        return strtolower(rtrim($host, '.'));
    }

    private function hydrate(array $row, bool $withSecrets): array
    {
        $row['username'] = $this->crypto->decrypt((string)($row['username_enc'] ?? ''));
        $row['auth_method'] = (string)($row['auth_method'] ?? 'password');
        $row['has_password'] = !empty($row['password_enc']) && $this->crypto->decrypt((string)$row['password_enc']) !== '';
        $row['has_private_key'] = !empty($row['private_key_enc']);
        $row['has_key_passphrase'] = !empty($row['key_passphrase_enc']) && $this->crypto->decrypt((string)$row['key_passphrase_enc']) !== '';
        if ($withSecrets) {
            $row['password'] = !empty($row['password_enc']) ? $this->crypto->decrypt((string)$row['password_enc']) : '';
            $row['public_key'] = !empty($row['public_key_enc']) ? $this->crypto->decrypt((string)$row['public_key_enc']) : '';
            $row['private_key'] = !empty($row['private_key_enc']) ? $this->crypto->decrypt((string)$row['private_key_enc']) : '';
            $row['key_passphrase'] = !empty($row['key_passphrase_enc']) ? $this->crypto->decrypt((string)$row['key_passphrase_enc']) : '';
        }
        unset($row['username_enc'], $row['password_enc'], $row['public_key_enc'], $row['private_key_enc'], $row['key_passphrase_enc']);
        $row['timeout'] = (int)($row['timeout'] ?? 30);
        $row['utf8'] = !array_key_exists('utf8', $row) || (bool)$row['utf8'];
        $row['host_fingerprint'] = (string)($row['host_fingerprint'] ?? '');
        return $row;
    }

    private function read(): array
    {
        return array_values(array_filter($this->store->read(), 'is_array'));
    }

    private static function boolInput(mixed $value): bool
    {
        return filter_var($value, FILTER_VALIDATE_BOOLEAN);
    }
}
