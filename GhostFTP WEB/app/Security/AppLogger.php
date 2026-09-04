<?php
declare(strict_types=1);

namespace ByFTP\Security;

final class AppLogger
{
    private const MAX_BYTES = 2097152;
    private const BACKUPS = 3;

    public static function event(string $action, array $context = []): void
    {
        $safe = [];
        foreach ($context as $key => $value) {
            if (in_array(strtolower((string)$key), ['password', 'secret', 'token', 'content', 'private_key', 'public_key', 'passphrase', 'key_passphrase'], true)) {
                continue;
            }
            if (is_scalar($value) || $value === null) {
                $safe[$key] = is_string($value) ? \byftp_truncate($value, 500) : $value;
            }
        }

        $row = [
            'time' => gmdate('c'),
            'ip' => \byftp_client_ip(),
            'user_id' => isset($_SESSION['user_id']) ? \byftp_truncate((string)$_SESSION['user_id'], 64) : null,
            'action' => \byftp_truncate($action, 80),
            'context' => $safe,
        ];

        $directory = BYFTP_STORAGE . '/logs';
        if (!is_dir($directory) && !@mkdir($directory, 0700, true) && !is_dir($directory)) {
            return;
        }
        $path = $directory . '/byftp.log';
        $lockPath = $directory . '/byftp.log.lock';
        $lock = @fopen($lockPath, 'c+');
        if (!is_resource($lock)) {
            return;
        }
        @chmod($lockPath, 0600);

        try {
            if (!flock($lock, LOCK_EX)) {
                return;
            }
            self::rotateIfNeeded($path);
            $line = json_encode($row, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE);
            if (is_string($line)) {
                @file_put_contents($path, $line . PHP_EOL, FILE_APPEND);
                @chmod($path, 0600);
            }
            flock($lock, LOCK_UN);
        } finally {
            fclose($lock);
        }
    }

    private static function rotateIfNeeded(string $path): void
    {
        if (!is_file($path)) {
            return;
        }
        $size = @filesize($path);
        if (!is_int($size) || $size < self::MAX_BYTES) {
            return;
        }

        for ($i = self::BACKUPS; $i >= 1; $i--) {
            $source = $i === 1 ? $path : $path . '.' . ($i - 1);
            $destination = $path . '.' . $i;
            if ($i === self::BACKUPS) {
                @unlink($destination);
            }
            if (is_file($source)) {
                @rename($source, $destination);
            }
        }
    }
}
