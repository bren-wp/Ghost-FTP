<?php
declare(strict_types=1);

namespace ByFTP\Storage;

use RuntimeException;

/**
 * Atomic JSON persistence for shared hosting.
 *
 * - adjacent flock file coordinates readers/writers
 * - temp + rename prevents partial primary writes
 * - one last-known-good .bak generation improves recovery from external corruption
 * - security-sensitive stores may disable automatic backup recovery and fail closed
 * - bounded reads/writes prevent abnormal state files from exhausting PHP memory
 */
final class JsonStore
{
    private const MAX_JSON_BYTES = 8 * 1024 * 1024;

    public function __construct(
        private readonly string $path,
        private readonly bool $recoverFromBackup = true
    ) {
    }

    public function read(array $fallback = []): array
    {
        if (!is_file($this->path) && !is_file($this->backupPath())) {
            return $fallback;
        }

        $lock = $this->openLock();
        try {
            if (!flock($lock, LOCK_SH)) {
                throw new RuntimeException('Nije moguće zaključati podatke za čitanje.');
            }
            [$data] = $this->readBestAvailable($fallback);
            flock($lock, LOCK_UN);
            return $data;
        } finally {
            fclose($lock);
        }
    }

    public function write(array $data): void
    {
        $this->ensureDirectory();
        $json = $this->encode($data);

        $lock = $this->openLock();
        try {
            if (!flock($lock, LOCK_EX)) {
                throw new RuntimeException('Nije moguće zaključati podatke za spremanje.');
            }
            $this->backupPrimaryIfValid();
            $this->replacePrimary($json);
            flock($lock, LOCK_UN);
        } finally {
            fclose($lock);
        }
    }

    public function update(callable $callback, array $fallback = []): array
    {
        $this->ensureDirectory();
        $lock = $this->openLock();
        try {
            if (!flock($lock, LOCK_EX)) {
                throw new RuntimeException('Nije moguće zaključati podatke za izmjenu.');
            }

            [$current, $fromBackup] = $this->readBestAvailable($fallback);
            $next = $callback($current);
            if (!is_array($next)) {
                throw new RuntimeException('Interna greška prilikom spremanja podataka.');
            }

            $json = $this->encode($next);
            // Never replace a good backup with a known-corrupt primary.
            if (!$fromBackup) {
                $this->backupPrimaryIfValid();
            }
            $this->replacePrimary($json);
            flock($lock, LOCK_UN);
            return $next;
        } finally {
            fclose($lock);
        }
    }

    /** @return array{0: array, 1: bool} */
    private function readBestAvailable(array $fallback): array
    {
        if (!is_file($this->path)) {
            if (is_file($this->backupPath())) {
                if (!$this->recoverFromBackup) {
                    throw new RuntimeException('Primarna JSON datoteka nedostaje. Automatski oporavak iz sigurnosne kopije nije dopušten za ove podatke.');
                }
                return [$this->decodeFile($this->backupPath()), true];
            }
            return [$fallback, false];
        }

        try {
            return [$this->decodeFile($this->path), false];
        } catch (RuntimeException $primaryError) {
            if (!$this->recoverFromBackup || !is_file($this->backupPath())) {
                throw $primaryError;
            }
            try {
                return [$this->decodeFile($this->backupPath()), true];
            } catch (RuntimeException) {
                throw $primaryError;
            }
        }
    }

    private function decodeFile(string $path): array
    {
        $handle = @fopen($path, 'rb');
        if (!is_resource($handle)) {
            throw new RuntimeException('Nije moguće pročitati spremljene podatke.');
        }
        try {
            $raw = stream_get_contents($handle, self::MAX_JSON_BYTES + 1);
        } finally {
            fclose($handle);
        }
        if (!is_string($raw)) {
            throw new RuntimeException('Nije moguće pročitati spremljene podatke.');
        }
        if (strlen($raw) > self::MAX_JSON_BYTES) {
            throw new RuntimeException('Spremljena JSON datoteka je prevelika.');
        }
        $decoded = json_decode($raw, true);
        if (!is_array($decoded)) {
            throw new RuntimeException('Spremljena JSON datoteka je oštećena.');
        }
        return $decoded;
    }

    private function encode(array $data): string
    {
        $json = json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE);
        if (!is_string($json)) {
            throw new RuntimeException('Nije moguće pretvoriti podatke u JSON.');
        }
        if (strlen($json) > self::MAX_JSON_BYTES) {
            throw new RuntimeException('JSON podaci su preveliki za sigurno spremanje.');
        }
        return $json;
    }

    private function replacePrimary(string $json): void
    {
        $tmp = $this->path . '.tmp.' . bin2hex(random_bytes(5));
        $handle = @fopen($tmp, 'xb');
        if (!is_resource($handle)) {
            throw new RuntimeException('Nije moguće otvoriti privremenu datoteku za spremanje.');
        }

        $offset = 0;
        $length = strlen($json);
        try {
            while ($offset < $length) {
                $written = fwrite($handle, substr($json, $offset));
                if ($written === false || $written === 0) {
                    throw new RuntimeException('Nije moguće spremiti sve podatke. Provjeri slobodan prostor i dozvole storage direktorija.');
                }
                $offset += $written;
            }
            if (!fflush($handle)) {
                throw new RuntimeException('Nije moguće dovršiti zapis podataka na disk.');
            }
            // fsync improves durability where the hosting filesystem supports it, but
            // some shared/NFS filesystems expose the function and still reject it. A
            // complete fflush + atomic rename remains the compatibility baseline.
            if (function_exists('fsync')) {
                @fsync($handle);
            }
        } catch (\Throwable $e) {
            fclose($handle);
            @unlink($tmp);
            throw $e;
        }
        fclose($handle);

        if ($offset !== $length) {
            @unlink($tmp);
            throw new RuntimeException('Spremanje podataka je prekinuto prije dovršetka.');
        }

        @chmod($tmp, 0600);
        if (!@rename($tmp, $this->path)) {
            @unlink($tmp);
            throw new RuntimeException('Nije moguće dovršiti atomsko spremanje podataka.');
        }
        @chmod($this->path, 0600);
    }

    private function backupPrimaryIfValid(): void
    {
        if (!is_file($this->path)) {
            return;
        }
        try {
            $this->decodeFile($this->path);
        } catch (RuntimeException) {
            return;
        }

        $sourceHash = @hash_file('sha256', $this->path);
        if (!is_string($sourceHash) || $sourceHash === '') {
            return;
        }

        $backup = $this->backupPath();
        $tmpBackup = $backup . '.tmp.' . bin2hex(random_bytes(4));
        if (!@copy($this->path, $tmpBackup)) {
            @unlink($tmpBackup);
            return;
        }
        $backupHash = @hash_file('sha256', $tmpBackup);
        if (!is_string($backupHash) || !hash_equals($sourceHash, $backupHash)) {
            @unlink($tmpBackup);
            return;
        }
        @chmod($tmpBackup, 0600);
        if (!@rename($tmpBackup, $backup)) {
            @unlink($tmpBackup);
            return;
        }
        @chmod($backup, 0600);
    }

    private function ensureDirectory(): void
    {
        $directory = dirname($this->path);
        if (!is_dir($directory) && !@mkdir($directory, 0700, true) && !is_dir($directory)) {
            throw new RuntimeException('Nije moguće izraditi direktorij za spremanje podataka.');
        }
        @chmod($directory, 0700);
    }

    private function openLock()
    {
        $this->ensureDirectory();
        $lockPath = $this->path . '.lock';
        $handle = @fopen($lockPath, 'c+');
        if ($handle === false) {
            throw new RuntimeException('Nije moguće otvoriti lock datoteku.');
        }
        @chmod($lockPath, 0600);
        return $handle;
    }

    private function backupPath(): string
    {
        return $this->path . '.bak';
    }
}
