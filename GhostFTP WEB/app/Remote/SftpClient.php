<?php
declare(strict_types=1);

namespace GhostFTP\Remote;

use GhostFTP\Security\HostGuard;
use RuntimeException;

final class SftpClient implements RemoteClientInterface, BoundedDownloadInterface
{
    private mixed $connection = null;
    private mixed $sftp = null;

    /** @var array<string, int> */
    private array $listedFileSizes = [];

    public function __construct(private array $profile)
    {
    }

    public function connect(): void
    {
        if (!extension_loaded('ssh2')) {
            throw new RuntimeException('SFTP zahtijeva PHP ssh2 ekstenziju na ovom hostingu.');
        }

        $host = (string)$this->profile['host'];
        $targets = HostGuard::connectionTargets($host, \GhostFTP_private_hosts_allowed());
        $conn = false;
        foreach ($targets as $target) {
            $conn = @ssh2_connect($target, (int)$this->profile['port']);
            if ($conn) break;
        }
        if (!$conn) throw new RuntimeException('Ne mogu se spojiti na SFTP/SSH poslužitelj.');

        $configuredFingerprint = (string)($this->profile['host_fingerprint'] ?? '');
        if ($configuredFingerprint !== '') $this->verifyHostFingerprint($conn, $configuredFingerprint);

        $username = (string)$this->profile['username'];
        $authMethod = (string)($this->profile['auth_method'] ?? 'password');
        try {
            if ($authMethod === 'key') {
                $this->authenticateWithKey($conn, $username);
            } elseif (!@ssh2_auth_password($conn, $username, (string)($this->profile['password'] ?? ''))) {
                throw new RuntimeException('SFTP prijava nije uspjela. Provjeri korisničko ime i lozinku.');
            }
        } finally {
            $this->profile['password'] = '';
            $this->profile['private_key'] = '';
            $this->profile['key_passphrase'] = '';
        }
        $sftp = @ssh2_sftp($conn);
        if (!$sftp) throw new RuntimeException('SFTP podsustav nije dostupan.');
        $this->connection = $conn;
        $this->sftp = $sftp;
    }

    public function list(string $path): array
    {
        $this->ensureConnected();
        $full = $this->full($path);
        $handle = @opendir($this->uri($full));
        if (!$handle) throw new RuntimeException('Ne mogu dohvatiti sadržaj direktorija.');
        $items = [];
        while (($name = readdir($handle)) !== false) {
            if ($name === '.' || $name === '..') continue;
            $remote = rtrim($full, '/') . '/' . $name;
            $stat = @ssh2_sftp_lstat($this->sftp, $remote) ?: [];
            $mode = (int)($stat['mode'] ?? 0);
            $isDir = ($mode & 0040000) === 0040000;
            $sizeKnown = array_key_exists('size', $stat) && is_numeric($stat['size']);
            $size = $sizeKnown ? max(0, (int)$stat['size']) : 0;
            $items[] = [
                'name' => $name,
                'type' => $isDir ? 'dir' : 'file',
                'size' => $size,
                'modified' => isset($stat['mtime']) ? gmdate('c', (int)$stat['mtime']) : null,
                'permissions' => substr(sprintf('%o', $mode), -4),
            ];
            if (!$isDir && $sizeKnown) {
                $this->listedFileSizes[PathGuard::normalizeRelative($remote)] = $size;
            }
        }
        closedir($handle);
        usort($items, static fn(array $a, array $b): int => $a['type'] !== $b['type'] ? ($a['type'] === 'dir' ? -1 : 1) : strnatcasecmp((string)$a['name'], (string)$b['name']));
        return $items;
    }

    public function makeDirectory(string $path): void
    {
        $this->ensureConnected();
        if (!@ssh2_sftp_mkdir($this->sftp, $this->full($path), 0755, false)) throw new RuntimeException('Ne mogu stvoriti direktorij.');
    }

    public function rename(string $from, string $to): void
    {
        $this->ensureConnected();
        if (!@ssh2_sftp_rename($this->sftp, $this->full($from), $this->full($to))) throw new RuntimeException('Preimenovanje ili premještanje nije uspjelo.');
    }

    public function delete(string $path, bool $directory = false): void
    {
        $this->ensureConnected();
        $ok = $directory ? @ssh2_sftp_rmdir($this->sftp, $this->full($path)) : @ssh2_sftp_unlink($this->sftp, $this->full($path));
        if (!$ok) throw new RuntimeException($directory ? 'Direktorij nije moguće obrisati.' : 'Datoteku nije moguće obrisati.');
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $this->ensureConnected();
        $in = @fopen($localFile, 'rb');
        $out = @fopen($this->uri($this->full($remotePath)), 'wb');
        if (!is_resource($in) || !is_resource($out)) {
            if (is_resource($in)) fclose($in);
            if (is_resource($out)) fclose($out);
            throw new RuntimeException('Upload nije moguće pokrenuti.');
        }
        $sourceStat = fstat($in);
        $expected = is_array($sourceStat) ? (int)($sourceStat['size'] ?? -1) : -1;
        try {
            $copied = stream_copy_to_stream($in, $out);
            if ($copied === false) throw new RuntimeException('Upload nije uspio.');
            if ($expected >= 0 && $copied !== $expected) throw new RuntimeException('Upload nije dovršen u cijelosti.');
        } finally {
            fclose($in);
            fclose($out);
        }
        if ($expected >= 0) {
            $remoteStat = @ssh2_sftp_lstat($this->sftp, $this->full($remotePath));
            $remoteSize = is_array($remoteStat) && array_key_exists('size', $remoteStat) && is_numeric($remoteStat['size'])
                ? (int)$remoteStat['size']
                : -1;
            if ($remoteSize >= 0 && $remoteSize !== $expected) {
                throw new RuntimeException('Upload je završio s neočekivanom veličinom datoteke. Prijenos nije pouzdan.');
            }
        }
    }

    public function download(string $remotePath, string $localFile): void
    {
        $this->downloadBounded($remotePath, $localFile);
    }

    public function downloadBounded(string $remotePath, string $localFile, ?int $maxBytes = null): int
    {
        $this->ensureConnected();
        $remote = $this->full($remotePath);
        $in = @fopen($this->uri($remote), 'rb');
        if (!is_resource($in)) {
            throw new RuntimeException('Download nije moguće pokrenuti.');
        }

        $sourceStat = fstat($in);
        $expected = is_array($sourceStat) && array_key_exists('size', $sourceStat) && is_numeric($sourceStat['size'])
            ? max(0, (int)$sourceStat['size'])
            : -1;
        $maxBytes = $this->effectiveDownloadLimit($remote, $maxBytes, $expected);
        $maxBytes = TransferLimiter::limitForDestination($localFile, $maxBytes);
        if ($expected >= 0 && $expected > $maxBytes) {
            fclose($in);
            throw new RuntimeException('Download prelazi dopuštenu veličinu.');
        }

        $out = @fopen($localFile, 'wb');
        if (!is_resource($out)) {
            fclose($in);
            throw new RuntimeException('Download nije moguće pokrenuti.');
        }
        try {
            $copied = TransferLimiter::copy($in, $out, $maxBytes);
            if ($expected >= 0 && $copied !== $expected) {
                throw new RuntimeException('Download nije dovršen u cijelosti.');
            }
            return $copied;
        } catch (\Throwable $e) {
            @ftruncate($out, 0);
            throw $e;
        } finally {
            fclose($in);
            fclose($out);
        }
    }

    public function read(string $remotePath, int $maxBytes = 4194304): string
    {
        $this->ensureConnected();
        $maxBytes = TransferLimiter::normalizeLimit($maxBytes) ?? 4194304;
        $fp = @fopen($this->uri($this->full($remotePath)), 'rb');
        if (!is_resource($fp)) throw new RuntimeException('Ne mogu otvoriti datoteku.');
        $content = stream_get_contents($fp, $maxBytes === PHP_INT_MAX ? PHP_INT_MAX : $maxBytes + 1);
        fclose($fp);
        if (!is_string($content) || strlen($content) > $maxBytes) throw new RuntimeException('Datoteka je prevelika za uređivanje.');
        if (str_contains(substr($content, 0, 8192), "\0")) throw new RuntimeException('Binarne datoteke nije moguće uređivati.');
        return $content;
    }

    public function write(string $remotePath, string $content): void
    {
        $this->ensureConnected();
        $fp = @fopen($this->uri($this->full($remotePath)), 'wb');
        if (!is_resource($fp)) throw new RuntimeException('Ne mogu otvoriti datoteku za spremanje.');
        $remaining = $content;
        while ($remaining !== '') {
            $written = fwrite($fp, $remaining);
            if ($written === false || $written === 0) {
                fclose($fp);
                throw new RuntimeException('Spremanje nije uspjelo.');
            }
            $remaining = substr($remaining, $written);
        }
        fclose($fp);
    }

    public function chmod(string $path, int $mode): void
    {
        $this->ensureConnected();
        if (!@ssh2_sftp_chmod($this->sftp, $this->full($path), $mode)) throw new RuntimeException('CHMOD nije uspio.');
    }

    public function disconnect(): void
    {
        $this->profile['password'] = '';
        $this->profile['private_key'] = '';
        $this->profile['key_passphrase'] = '';
        $this->listedFileSizes = [];
        $this->sftp = null;
        $this->connection = null;
    }

    private function effectiveDownloadLimit(string $remote, ?int $maxBytes, int $expectedSize): int
    {
        $snapshot = $this->listedFileSizes[$remote] ?? null;
        unset($this->listedFileSizes[$remote]);
        return TransferLimiter::effectiveLimit($maxBytes, is_int($snapshot) ? $snapshot : null, $expectedSize);
    }

    private function verifyHostFingerprint(mixed $conn, string $expected): void
    {
        if (!defined('SSH2_FINGERPRINT_SHA256')) throw new RuntimeException('Ova verzija PHP ssh2 ekstenzije ne podržava SHA-256 host fingerprint provjeru.');
        if (str_starts_with(strtoupper($expected), 'SHA256:')) {
            if (!defined('SSH2_FINGERPRINT_RAW')) throw new RuntimeException('Hosting ne podržava OpenSSH SHA256: fingerprint format; koristi SHA-256 HEX vrijednost.');
            $raw = @ssh2_fingerprint($conn, SSH2_FINGERPRINT_SHA256 | SSH2_FINGERPRINT_RAW);
            if (!is_string($raw) || $raw === '') throw new RuntimeException('Nije moguće pročitati SFTP host fingerprint.');
            $actual = rtrim(base64_encode($raw), '=');
            $wanted = rtrim(trim(substr($expected, 7)), '=');
            if ($wanted === '' || !hash_equals($wanted, $actual)) throw new RuntimeException('SFTP host fingerprint se ne podudara. Veza je prekinuta radi sigurnosti.');
            return;
        }
        if (!defined('SSH2_FINGERPRINT_HEX')) throw new RuntimeException('Hosting ne podržava HEX host fingerprint provjeru.');
        $actual = @ssh2_fingerprint($conn, SSH2_FINGERPRINT_SHA256 | SSH2_FINGERPRINT_HEX);
        $actualHex = is_string($actual) ? strtolower(preg_replace('/[^a-f0-9]/i', '', $actual) ?? '') : '';
        $wantedHex = strtolower(preg_replace('/[^a-f0-9]/i', '', $expected) ?? '');
        if (strlen($wantedHex) !== 64 || $actualHex === '' || !hash_equals($wantedHex, $actualHex)) throw new RuntimeException('SFTP host fingerprint se ne podudara ili nije valjan SHA-256 fingerprint.');
    }

    private function authenticateWithKey(mixed $conn, string $username): void
    {
        if (!function_exists('ssh2_auth_pubkey_file')) throw new RuntimeException('PHP ssh2 ekstenzija na ovom hostingu ne podržava autentikaciju javnim ključem.');
        $publicKey = (string)($this->profile['public_key'] ?? '');
        $privateKey = (string)($this->profile['private_key'] ?? '');
        if ($publicKey === '' || $privateKey === '') throw new RuntimeException('SFTP profil nema spremljen javni i privatni ključ.');
        $pub = tempnam(GhostFTP_STORAGE . '/tmp', 'sftp-pub-');
        $priv = tempnam(GhostFTP_STORAGE . '/tmp', 'sftp-key-');
        if ($pub === false || $priv === false) {
            if (is_string($pub)) @unlink($pub);
            if (is_string($priv)) @unlink($priv);
            throw new RuntimeException('Nije moguće pripremiti privremene SFTP ključeve.');
        }
        try {
            if (DIRECTORY_SEPARATOR === '/') {
                if (!@chmod($pub, 0600) || !@chmod($priv, 0600)) {
                    throw new RuntimeException('Nije moguće zaštititi privremene SFTP ključeve dozvolama 0600.');
                }
            } else {
                @chmod($pub, 0600);
                @chmod($priv, 0600);
            }
            $publicMaterial = $publicKey . (str_ends_with($publicKey, "\n") ? '' : "\n");
            $privateMaterial = $privateKey . (str_ends_with($privateKey, "\n") ? '' : "\n");
            $publicWritten = file_put_contents($pub, $publicMaterial, LOCK_EX);
            $privateWritten = file_put_contents($priv, $privateMaterial, LOCK_EX);
            if (!is_int($publicWritten) || $publicWritten !== strlen($publicMaterial)
                || !is_int($privateWritten) || $privateWritten !== strlen($privateMaterial)) {
                throw new RuntimeException('Nije moguće zapisati cijele privremene SFTP ključeve.');
            }
            if (DIRECTORY_SEPARATOR === '/') {
                if (!@chmod($pub, 0600) || !@chmod($priv, 0600)) {
                    throw new RuntimeException('Nije moguće zadržati zaštitu privremenih SFTP ključeva dozvolama 0600.');
                }
            } else {
                @chmod($pub, 0600);
                @chmod($priv, 0600);
            }
            $passphrase = (string)($this->profile['key_passphrase'] ?? '');
            $ok = $passphrase === '' ? @ssh2_auth_pubkey_file($conn, $username, $pub, $priv) : @ssh2_auth_pubkey_file($conn, $username, $pub, $priv, $passphrase);
            if (!$ok) throw new RuntimeException('SFTP prijava ključem nije uspjela. Provjeri ključ, korisničko ime i passphrase.');
        } finally {
            @unlink($pub);
            @unlink($priv);
        }
    }

    private function ensureConnected(): void
    {
        if (!$this->sftp) $this->connect();
    }

    private function full(string $path): string
    {
        return PathGuard::join((string)($this->profile['base_path'] ?? '/'), $path);
    }

    private function uri(string $path): string
    {
        return 'ssh2.sftp://' . intval($this->sftp) . $path;
    }
}
