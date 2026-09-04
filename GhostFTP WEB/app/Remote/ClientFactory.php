<?php
declare(strict_types=1);

namespace GhostFTP\Remote;

use RuntimeException;

final class ClientFactory
{
    public static function make(array $profile): RemoteClientInterface
    {
        $protocol = (string)($profile['protocol'] ?? 'ftp');
        if ($protocol === 'sftp') {
            // PHP ssh2 does not provide an OpenSSH known_hosts trust decision for us.
            // Never allow password/key authentication before GhostFTP has a pinned server key.
            if ((string)($profile['host_fingerprint'] ?? '') === '') {
                throw new RuntimeException('SFTP zahtijeva SHA-256 host fingerprint prije povezivanja. Provjeri fingerprint servera iz pouzdanog izvora i spremi ga u profil.');
            }
            return new SftpClient($profile);
        }

        return match ($protocol) {
            'ftp', 'ftps' => new FtpClient($profile),
            default => throw new RuntimeException('Nepodržani protokol.'),
        };
    }
}
