<?php
declare(strict_types=1);

namespace ByFTP\Remote;

use ByFTP\Security\HostGuard;
use RuntimeException;

final class FtpClient implements RemoteClientInterface
{
    private mixed $connection = null;

    public function __construct(private array $profile)
    {
    }

    public function connect(): void
    {
        if (!extension_loaded('ftp')) {
            throw new RuntimeException('PHP FTP ekstenzija nije dostupna na hostingu.');
        }

        $host = (string)$this->profile['host'];
        $targets = HostGuard::connectionTargets($host, \byftp_private_hosts_allowed());
        $port = (int)$this->profile['port'];
        $protocol = (string)$this->profile['protocol'];
        $timeout = max(5, min(120, (int)($this->profile['timeout'] ?? 30)));

        if ($protocol === 'ftps' && !function_exists('ftp_ssl_connect')) {
            throw new RuntimeException('FTPS nije dostupan jer PHP nema ftp_ssl_connect().');
        }

        $conn = false;
        foreach ($targets as $target) {
            $conn = $protocol === 'ftps'
                ? @ftp_ssl_connect($target, $port, $timeout)
                : @ftp_connect($target, $port, $timeout);
            if ($conn !== false) {
                break;
            }
        }
        if ($conn === false) {
            throw new RuntimeException('Ne mogu se spojiti na FTP poslužitelj. Provjeri host, port i firewall.');
        }

        $password = (string)($this->profile['password'] ?? '');
        try {
            $authenticated = @ftp_login($conn, (string)$this->profile['username'], $password);
        } finally {
            $password = '';
            $this->profile['password'] = '';
        }
        if (!$authenticated) {
            @ftp_close($conn);
            throw new RuntimeException('FTP prijava nije uspjela. Provjeri korisničko ime i lozinku.');
        }

        @ftp_set_option($conn, FTP_TIMEOUT_SEC, $timeout);
        if ((bool)($this->profile['utf8'] ?? true)) {
            @ftp_raw($conn, 'OPTS UTF8 ON');
        }
        if (!@ftp_pasv($conn, (bool)($this->profile['passive'] ?? true))) {
            if ((bool)($this->profile['passive'] ?? true)) {
                @ftp_close($conn);
                throw new RuntimeException('Poslužitelj nije prihvatio pasivni FTP način rada. Pokušaj isključiti PASV u profilu.');
            }
        }
        $this->connection = $conn;
    }

    public function list(string $path): array
    {
        $this->ensureConnected();
        $remote = $this->full($path);
        $items = @ftp_mlsd($this->connection, $remote);
        if (is_array($items)) {
            $out = [];
            foreach ($items as $item) {
                $name = (string)($item['name'] ?? '');
                if ($name === '' || $name === '.' || $name === '..') continue;
                $type = strtolower((string)($item['type'] ?? 'file'));
                if (in_array($type, ['cdir', 'pdir'], true)) continue;
                $out[] = [
                    'name' => $name,
                    'type' => $type === 'dir' ? 'dir' : 'file',
                    'size' => (int)($item['size'] ?? 0),
                    'modified' => $this->mlsdDate((string)($item['modify'] ?? '')),
                    'permissions' => (string)($item['unix.mode'] ?? ''),
                ];
            }
            return $this->sortItems($out);
        }

        $raw = @ftp_rawlist($this->connection, $remote);
        if (!is_array($raw)) throw new RuntimeException('Ne mogu dohvatiti sadržaj direktorija.');
        return $this->sortItems($this->parseRawList($raw));
    }

    public function makeDirectory(string $path): void
    {
        $this->ensureConnected();
        if (@ftp_mkdir($this->connection, $this->full($path)) === false) throw new RuntimeException('Ne mogu stvoriti direktorij.');
    }

    public function rename(string $from, string $to): void
    {
        $this->ensureConnected();
        if (!@ftp_rename($this->connection, $this->full($from), $this->full($to))) throw new RuntimeException('Preimenovanje ili premještanje nije uspjelo.');
    }

    public function delete(string $path, bool $directory = false): void
    {
        $this->ensureConnected();
        $ok = $directory ? @ftp_rmdir($this->connection, $this->full($path)) : @ftp_delete($this->connection, $this->full($path));
        if (!$ok) throw new RuntimeException($directory ? 'Direktorij nije moguće obrisati.' : 'Datoteku nije moguće obrisati.');
    }

    public function upload(string $localFile, string $remotePath): void
    {
        $this->ensureConnected();
        $expected = @filesize($localFile);
        $fp = @fopen($localFile, 'rb');
        if (!is_resource($fp)) throw new RuntimeException('Ne mogu otvoriti lokalnu datoteku.');
        $remote = $this->full($remotePath);
        try {
            if (!@ftp_fput($this->connection, $remote, $fp, FTP_BINARY)) throw new RuntimeException('Upload nije uspio.');
        } finally {
            fclose($fp);
        }
        $actual = @ftp_size($this->connection, $remote);
        if (is_int($expected) && $expected >= 0 && is_int($actual) && $actual >= 0 && $actual !== $expected) {
            throw new RuntimeException('Upload je završio s neočekivanom veličinom datoteke. Prijenos nije pouzdan.');
        }
    }

    public function download(string $remotePath, string $localFile, ?int $maxBytes = null): int
    {
        $this->ensureConnected();
        $maxBytes = TransferLimiter::normalizeLimit($maxBytes);
        $remote = $this->full($remotePath);
        $expected = @ftp_size($this->connection, $remote);
        if ($maxBytes !== null && is_int($expected) && $expected > $maxBytes) {
            throw new RuntimeException('Download prelazi dopuštenu veličinu.');
        }

        $fp = @fopen($localFile, 'wb');
        if (!is_resource($fp)) throw new RuntimeException('Ne mogu otvoriti privremenu datoteku.');
        try {
            $status = @ftp_nb_fget($this->connection, $fp, $remote, FTP_BINARY);
            if (!is_int($status)) {
                throw new RuntimeException('Download nije uspio.');
            }
            while ($status === FTP_MOREDATA) {
                TransferLimiter::assertWithinLimit($fp, $maxBytes);
                $status = @ftp_nb_continue($this->connection);
                if (!is_int($status)) {
                    throw new RuntimeException('Download nije uspio.');
                }
            }
            if ($status !== FTP_FINISHED) {
                throw new RuntimeException('Download nije uspio.');
            }

            $actual = TransferLimiter::assertWithinLimit($fp, $maxBytes);
            if (is_int($expected) && $expected >= 0 && $actual >= 0 && $actual !== $expected) {
                throw new RuntimeException('Download je završio s neočekivanom veličinom datoteke. Prijenos nije pouzdan.');
            }
            if ($actual < 0) {
                throw new RuntimeException('Nije moguće potvrditi veličinu preuzete datoteke.');
            }
            return $actual;
        } catch (\Throwable $e) {
            @ftruncate($fp, 0);
            if ($this->connection) {
                @ftp_close($this->connection);
                $this->connection = null;
            }
            throw $e;
        } finally {
            fclose($fp);
        }
    }

    public function read(string $remotePath, int $maxBytes = 4194304): string
    {
        $this->ensureConnected();
        $remote = $this->full($remotePath);
        $expected = @ftp_size($this->connection, $remote);
        if (is_int($expected) && $expected > $maxBytes) throw new RuntimeException('Datoteka je prevelika za uređivanje u pregledniku.');
        \byftp_assert_temp_capacity($maxBytes);
        $tmp = tempnam(BYFTP_STORAGE . '/tmp', 'read-');
        if ($tmp === false) throw new RuntimeException('Ne mogu stvoriti privremenu datoteku.');
        try {
            $this->download($remotePath, $tmp, $maxBytes);
            $content = file_get_contents($tmp);
            if (!is_string($content) || str_contains(substr($content, 0, 8192), "\0")) throw new RuntimeException('Binarne datoteke nije moguće uređivati u editoru.');
            return $content;
        } finally {
            @unlink($tmp);
        }
    }

    public function write(string $remotePath, string $content): void
    {
        $tmp = tempnam(BYFTP_STORAGE . '/tmp', 'write-');
        if ($tmp === false) throw new RuntimeException('Ne mogu stvoriti privremenu datoteku.');
        try {
            if (file_put_contents($tmp, $content, LOCK_EX) === false) throw new RuntimeException('Ne mogu pripremiti sadržaj za spremanje.');
            $this->upload($tmp, $remotePath);
        } finally {
            @unlink($tmp);
        }
    }

    public function chmod(string $path, int $mode): void
    {
        $this->ensureConnected();
        if (!function_exists('ftp_chmod') || @ftp_chmod($this->connection, $mode, $this->full($path)) === false) throw new RuntimeException('CHMOD nije podržan ili nije uspio.');
    }

    public function disconnect(): void
    {
        $this->profile['password'] = '';
        if ($this->connection) {
            @ftp_close($this->connection);
            $this->connection = null;
        }
    }

    private function ensureConnected(): void
    {
        if (!$this->connection) $this->connect();
    }

    private function full(string $path): string
    {
        return PathGuard::join((string)($this->profile['base_path'] ?? '/'), $path);
    }

    private function mlsdDate(string $value): ?string
    {
        if (!preg_match('/^(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})/', $value, $m)) return null;
        return sprintf('%s-%s-%sT%s:%s:%sZ', $m[1], $m[2], $m[3], $m[4], $m[5], $m[6]);
    }

    private function parseRawList(array $lines): array
    {
        $items = [];
        foreach ($lines as $line) {
            if (!is_string($line) || trim($line) === '' || str_starts_with(trim($line), 'total ')) continue;
            if (preg_match('/^([dl-][rwxstST-]{9})\s+\d+\s+\S+\s+\S+\s+(\d+)\s+(\w+\s+\d+\s+[\d:]+|\w+\s+\d+\s+\d{4})\s+(.+)$/', $line, $m)) {
                $name = $m[4];
                if ($m[1][0] === 'l') {
                    $name = preg_replace('/\s+->\s+.*$/', '', $name) ?? $name;
                }
                if ($name === '' || $name === '.' || $name === '..') continue;
                $items[] = ['name'=>$name,'type'=>$m[1][0] === 'd' ? 'dir' : 'file','size'=>(int)$m[2],'modified'=>$m[3],'permissions'=>$m[1]];
                continue;
            }
            if (preg_match('/^(\d{2}-\d{2}-\d{2,4})\s+(\d{2}:\d{2}[AP]M)\s+(<DIR>|\d+)\s+(.+)$/i', trim($line), $m)) {
                $name = $m[4];
                if ($name === '.' || $name === '..') continue;
                $items[] = ['name'=>$name,'type'=>strtoupper($m[3]) === '<DIR>' ? 'dir' : 'file','size'=>is_numeric($m[3]) ? (int)$m[3] : 0,'modified'=>$m[1].' '.$m[2],'permissions'=>''];
            }
        }
        return $items;
    }

    private function sortItems(array $items): array
    {
        usort($items, static function (array $a, array $b): int {
            if ($a['type'] !== $b['type']) return $a['type'] === 'dir' ? -1 : 1;
            return strnatcasecmp((string)$a['name'], (string)$b['name']);
        });
        return $items;
    }
}
