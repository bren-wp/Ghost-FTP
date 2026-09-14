<?php
declare(strict_types=1);

namespace GhostFTP\Web;

use RuntimeException;

final class Security
{
    public static function sendHeaders(): void
    {
        header('X-Content-Type-Options: nosniff');
        header('X-Frame-Options: DENY');
        header('Referrer-Policy: no-referrer');
        header('Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=(), usb=()');
        header("Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'");
        header('Cache-Control: no-store, max-age=0');
        header('Pragma: no-cache');
    }

    public static function publicTarget(string $host): array
    {
        $host = trim($host);
        if ($host === '' || strlen($host) > 253 || preg_match('/[\x00-\x20\x7f]/', $host)) {
            throw new RuntimeException('Invalid host.');
        }
        $host = trim($host, '[]');
        if (filter_var($host, FILTER_VALIDATE_IP)) {
            self::assertAllowedIp($host);
            return [$host, $host];
        }
        if (!preg_match('/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i', $host)) {
            throw new RuntimeException('Invalid host name.');
        }
        $ips = [];
        foreach (dns_get_record($host, DNS_A | DNS_AAAA) ?: [] as $record) {
            $ip = $record['ip'] ?? $record['ipv6'] ?? null;
            if (is_string($ip) && filter_var($ip, FILTER_VALIDATE_IP)) {
                $ips[$ip] = true;
            }
        }
        if (!$ips) {
            throw new RuntimeException('Host could not be resolved.');
        }
        foreach (array_keys($ips) as $ip) {
            self::assertAllowedIp($ip);
        }
        return [$host, array_key_first($ips)];
    }

    private static function assertAllowedIp(string $ip): void
    {
        if (\ghostftp_allow_private_hosts()) {
            return;
        }
        $public = filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_NO_PRIV_RANGE | FILTER_FLAG_NO_RES_RANGE);
        if ($public === false) {
            throw new RuntimeException('Private, loopback, link-local and reserved destinations are blocked by default.');
        }
    }

    public static function port(mixed $value, int $fallback): int
    {
        $port = filter_var($value, FILTER_VALIDATE_INT, ['options' => ['min_range' => 1, 'max_range' => 65535]]);
        return $port === false ? $fallback : (int)$port;
    }

    public static function remotePath(string $path, string $root = '/'): string
    {
        foreach ([$path, $root] as $value) {
            if (preg_match('/[\x00-\x1f\x7f]/', $value)) {
                throw new RuntimeException('Remote path contains forbidden control characters.');
            }
        }
        $normalize = static function (string $value): string {
            $parts = [];
            foreach (explode('/', str_replace('\\', '/', $value)) as $part) {
                if ($part === '' || $part === '.') continue;
                if ($part === '..') {
                    array_pop($parts);
                    continue;
                }
                $parts[] = $part;
            }
            return '/' . implode('/', $parts);
        };
        $rootN = rtrim($normalize($root), '/');
        if ($rootN === '') $rootN = '/';
        $pathN = $normalize($path);
        if ($rootN !== '/' && $pathN !== $rootN && !str_starts_with($pathN, $rootN . '/')) {
            throw new RuntimeException('Remote path escapes the configured root.');
        }
        return $pathN;
    }

    public static function fingerprint(string $value): string
    {
        $value = trim($value);
        if (!preg_match('/^SHA256:[A-Za-z0-9+\/=]{32,64}$/', $value)) {
            throw new RuntimeException('SFTP requires an expected SHA-256 host-key fingerprint.');
        }
        return rtrim($value, '=');
    }
}
