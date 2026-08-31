<?php
declare(strict_types=1);

namespace ByFTP\Storage;

use InvalidArgumentException;
use RuntimeException;

final class UserWorkspace
{
    public static function id(string $userId): string
    {
        $clean = preg_replace('/[^a-zA-Z0-9_-]/', '', $userId) ?? '';
        if ($clean === '' || !hash_equals($clean, $userId)) {
            throw new InvalidArgumentException('Neispravan korisnički identifikator.');
        }
        return $clean;
    }

    public static function directory(string $userId): string
    {
        return BYFTP_STORAGE . '/users/' . self::id($userId);
    }

    public static function file(string $userId, string $name): string
    {
        if (!preg_match('/^[a-z0-9_-]+\.json$/i', $name)) {
            throw new InvalidArgumentException('Neispravan naziv korisničke datoteke.');
        }
        return self::directory($userId) . '/' . $name;
    }

    public static function migrateLegacy(string $userId): void
    {
        $directory = self::ensure($userId);
        foreach (['profiles.json', 'preferences.json'] as $name) {
            $source = BYFTP_STORAGE . '/' . $name;
            $target = $directory . '/' . $name;
            $sourceExists = is_file($source) || is_file($source . '.bak');
            if (!$sourceExists || is_file($target) || is_file($target . '.bak')) {
                continue;
            }

            // Legacy profiles/preferences contain credential/privacy state and deletion intent.
            // Keep .bak available for explicit operator recovery, but never migrate from an
            // older generation automatically when the primary file is corrupt or missing.
            $data = (new JsonStore($source, false))->read([]);
            (new JsonStore($target))->write($data);

            foreach ([$source, $source . '.bak'] as $legacyPath) {
                if (!is_file($legacyPath)) {
                    continue;
                }
                $archive = $legacyPath . '.migrated.bak';
                if (!@rename($legacyPath, $archive)) {
                    // Migration itself already succeeded. Leaving the source in place is
                    // safer than deleting it when the hosting filesystem rejects rename.
                    @chmod($legacyPath, 0600);
                }
            }
            @unlink($source . '.lock');
        }
    }

    public static function ensure(string $userId): string
    {
        $directory = self::directory($userId);
        if (!is_dir($directory) && !@mkdir($directory, 0700, true) && !is_dir($directory)) {
            throw new RuntimeException('Nije moguće izraditi korisnički storage direktorij. Provjeri dozvole storage/users direktorija.');
        }
        if (!is_writable($directory)) {
            throw new RuntimeException('Korisnički storage direktorij nije zapisiv. Provjeri dozvole storage/users direktorija.');
        }
        @chmod($directory, 0700);
        return $directory;
    }
}
