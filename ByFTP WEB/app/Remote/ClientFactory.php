<?php
declare(strict_types=1);

namespace ByFTP\Remote;

use RuntimeException;

final class ClientFactory
{
    public static function make(array $profile): RemoteClientInterface
    {
        return match ((string)($profile['protocol'] ?? 'ftp')) {
            'ftp', 'ftps' => new FtpClient($profile),
            'sftp' => new SftpClient($profile),
            default => throw new RuntimeException('Nepodržani protokol.'),
        };
    }
}
