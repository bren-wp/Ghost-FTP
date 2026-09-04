<?php
declare(strict_types=1);

namespace GhostFTP\Security;

use RuntimeException;

/** Prevents authenticated users from turning GhostFTP WEB into an internal-network probe by default. */
final class HostGuard
{
    public static function assertAllowed(string $host, bool $allowPrivate = false): void
    {
        self::connectionTargets($host, $allowPrivate);
    }

    /**
     * Resolve once and return the exact validated IP targets clients should connect to.
     * This closes the DNS rebinding/TOCTOU gap created by validating a hostname and then
     * letting the protocol library resolve the same hostname a second time.
     *
     * @return list<string>
     */
    public static function connectionTargets(string $host, bool $allowPrivate = false): array
    {
        if ($host === '' || $host !== trim($host) || preg_match('/[\x00-\x20\x7F]/', $host)) {
            throw new RuntimeException('Host nije kanonski zadan. Ukloni razmake i kontrolne znakove.');
        }

        if (str_starts_with($host, '[') || str_ends_with($host, ']')) {
            if (!(str_starts_with($host, '[') && str_ends_with($host, ']'))) {
                throw new RuntimeException('IPv6 host ima neispravne uglate zagrade.');
            }
            $host = substr($host, 1, -1);
            if ($host === '') {
                throw new RuntimeException('Host nije zadan.');
            }
        }

        $lower = strtolower(rtrim($host, '.'));
        if (!$allowPrivate && ($lower === 'localhost' || str_ends_with($lower, '.localhost'))) {
            throw new RuntimeException('Veze prema localhost adresama su blokirane sigurnosnom politikom.');
        }

        if (filter_var($host, FILTER_VALIDATE_IP)) {
            if (!$allowPrivate) {
                self::assertPublicIp($host);
            }
            return [$host];
        }

        if (!preg_match('/^(?=.{1,253}$)(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*\.?$/', $host)) {
            throw new RuntimeException('DNS naziv hosta nije valjan.');
        }

        $ips = self::resolve($host);
        if ($ips === []) {
            if ($allowPrivate) {
                // Internal DNS/mDNS may only be resolvable by the protocol library.
                // This fallback exists only behind the explicit admin policy switch.
                return [$host];
            }
            throw new RuntimeException('Host nije moguće sigurno razriješiti u IP adresu. Provjeri DNS naziv servera.');
        }

        if (!$allowPrivate) {
            foreach ($ips as $ip) {
                self::assertPublicIp($ip);
            }
        }
        return $ips;
    }

    private static function assertPublicIp(string $ip): void
    {
        $public = filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_NO_PRIV_RANGE | FILTER_FLAG_NO_RES_RANGE);
        if ($public === false) {
            throw new RuntimeException('Veze prema privatnim, lokalnim ili rezerviranim IP adresama su blokirane sigurnosnom politikom.');
        }
    }

    /** @return list<string> */
    private static function resolve(string $host): array
    {
        $ips = [];
        if (function_exists('dns_get_record')) {
            $records = @dns_get_record($host, DNS_A | DNS_AAAA);
            if (is_array($records)) {
                foreach ($records as $record) {
                    foreach (['ip', 'ipv6'] as $field) {
                        $ip = (string)($record[$field] ?? '');
                        if ($ip !== '' && filter_var($ip, FILTER_VALIDATE_IP)) {
                            $ips[] = $ip;
                        }
                    }
                }
            }
        }
        if ($ips === [] && function_exists('gethostbynamel')) {
            $rows = @gethostbynamel($host);
            if (is_array($rows)) {
                foreach ($rows as $ip) {
                    if (is_string($ip) && filter_var($ip, FILTER_VALIDATE_IP)) {
                        $ips[] = $ip;
                    }
                }
            }
        }
        return array_values(array_unique($ips));
    }
}
