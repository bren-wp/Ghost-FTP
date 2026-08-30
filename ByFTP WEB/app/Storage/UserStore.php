<?php
declare(strict_types=1);

namespace ByFTP\Storage;

use RuntimeException;

final class UserStore
{
    private JsonStore $store;

    public function __construct()
    {
        $this->store = new JsonStore(BYFTP_STORAGE . '/users.json');
    }

    public function all(): array
    {
        $rows = array_values(array_filter($this->store->read(), 'is_array'));
        $safe = array_map([$this, 'publicUser'], $rows);
        usort($safe, static fn(array $a, array $b): int => strnatcasecmp((string)$a['name'], (string)$b['name']));
        return $safe;
    }

    public function findById(string $id): ?array
    {
        foreach ($this->store->read() as $row) {
            if (is_array($row) && hash_equals((string)($row['id'] ?? ''), $id)) {
                return $row;
            }
        }
        return null;
    }

    public function findByEmail(string $email): ?array
    {
        $normalized = self::normalizeEmail($email);
        foreach ($this->store->read() as $row) {
            if (is_array($row) && hash_equals((string)($row['email'] ?? ''), $normalized)) {
                return $row;
            }
        }
        return null;
    }

    public function create(string $name, string $email, string $password, string $role = 'user'): array
    {
        $name = self::validateName($name);
        $email = self::normalizeEmail($email);
        self::validatePassword($password);
        $role = in_array($role, ['admin', 'user'], true) ? $role : 'user';
        $hash = password_hash($password, self::passwordAlgorithm());
        if (!is_string($hash)) {
            throw new RuntimeException('Nije moguće sigurno hashirati lozinku.');
        }

        $created = null;
        $this->store->update(function (array $rows) use ($name, $email, $hash, $role, &$created): array {
            foreach ($rows as $row) {
                if (is_array($row) && hash_equals((string)($row['email'] ?? ''), $email)) {
                    throw new RuntimeException('Korisnički račun s tom e-mail adresom već postoji.');
                }
            }
            $created = [
                'id' => bin2hex(random_bytes(12)),
                'name' => $name,
                'email' => $email,
                'password_hash' => $hash,
                'role' => $role,
                'active' => true,
                'created_at' => gmdate('c'),
                'updated_at' => gmdate('c'),
                'last_login_at' => null,
                'session_version' => 1,
            ];
            $rows[] = $created;
            return array_values($rows);
        });

        if (!is_array($created)) {
            throw new RuntimeException('Korisnički račun nije izrađen.');
        }
        try {
            UserWorkspace::ensure((string)$created['id']);
        } catch (\Throwable $e) {
            // Keep account persistence and its isolated workspace transactional enough
            // for shared hosting: a failed workspace must not leave a ghost account.
            try {
                $createdId = (string)$created['id'];
                $this->store->update(static fn(array $rows): array => array_values(array_filter(
                    $rows,
                    static fn($row): bool => !is_array($row) || !hash_equals((string)($row['id'] ?? ''), $createdId)
                )));
                @rmdir(UserWorkspace::directory($createdId));
            } catch (\Throwable) {
                // Preserve the original storage failure; the admin can retry after fixing permissions.
            }
            throw $e;
        }
        return $this->publicUser($created);
    }

    public function authenticate(string $email, string $password): ?array
    {
        $user = $this->findByEmail($email);
        if (!$user || empty($user['active']) || !password_verify($password, (string)($user['password_hash'] ?? ''))) {
            return null;
        }
        if (password_needs_rehash((string)$user['password_hash'], self::passwordAlgorithm())) {
            $this->replacePasswordHash((string)$user['id'], password_hash($password, self::passwordAlgorithm()));
        }
        $this->touchLogin((string)$user['id']);
        $fresh = $this->findById((string)$user['id']);
        return $fresh ? $this->publicUser($fresh) : null;
    }

    public function updateProfile(string $id, string $name, string $email): array
    {
        $name = self::validateName($name);
        $email = self::normalizeEmail($email);
        $updated = null;
        $this->store->update(function (array $rows) use ($id, $name, $email, &$updated): array {
            foreach ($rows as $row) {
                if (is_array($row) && ($row['id'] ?? '') !== $id && hash_equals((string)($row['email'] ?? ''), $email)) {
                    throw new RuntimeException('Drugi račun već koristi tu e-mail adresu.');
                }
            }
            foreach ($rows as &$row) {
                if (is_array($row) && hash_equals((string)($row['id'] ?? ''), $id)) {
                    $row['name'] = $name;
                    $row['email'] = $email;
                    $row['updated_at'] = gmdate('c');
                    $updated = $row;
                    break;
                }
            }
            unset($row);
            if (!is_array($updated)) {
                throw new RuntimeException('Korisnik nije pronađen.');
            }
            return $rows;
        });
        return $this->publicUser($updated);
    }

    public function changePassword(string $id, string $currentPassword, string $newPassword, bool $requireCurrent = true): void
    {
        self::validatePassword($newPassword);
        $user = $this->findById($id);
        if (!$user) {
            throw new RuntimeException('Korisnik nije pronađen.');
        }
        if ($requireCurrent && !password_verify($currentPassword, (string)($user['password_hash'] ?? ''))) {
            throw new RuntimeException('Trenutačna lozinka nije točna.');
        }
        $hash = password_hash($newPassword, self::passwordAlgorithm());
        if (!is_string($hash)) {
            throw new RuntimeException('Nije moguće sigurno spremiti novu lozinku.');
        }
        $this->replacePasswordHash($id, $hash);
    }

    public function updateAdminFields(string $id, string $role, bool $active): array
    {
        $role = in_array($role, ['admin', 'user'], true) ? $role : 'user';
        $updated = null;
        $this->store->update(function (array $rows) use ($id, $role, $active, &$updated): array {
            $admins = 0;
            foreach ($rows as $row) {
                if (is_array($row) && ($row['role'] ?? '') === 'admin' && !empty($row['active'])) {
                    $admins++;
                }
            }
            foreach ($rows as &$row) {
                if (!is_array($row) || !hash_equals((string)($row['id'] ?? ''), $id)) {
                    continue;
                }
                $wouldRemoveLastAdmin = ($row['role'] ?? '') === 'admin' && !empty($row['active']) && $admins <= 1 && ($role !== 'admin' || !$active);
                if ($wouldRemoveLastAdmin) {
                    throw new RuntimeException('Mora ostati barem jedan aktivan administrator.');
                }
                $row['role'] = $role;
                if ((bool)($row['active'] ?? false) !== $active) {
                    $row['session_version'] = (int)($row['session_version'] ?? 1) + 1;
                }
                $row['active'] = $active;
                $row['updated_at'] = gmdate('c');
                $updated = $row;
                break;
            }
            unset($row);
            if (!is_array($updated)) {
                throw new RuntimeException('Korisnik nije pronađen.');
            }
            return $rows;
        });
        return $this->publicUser($updated);
    }

    public function delete(string $id): void
    {
        $this->store->update(function (array $rows) use ($id): array {
            $target = null;
            $activeAdmins = 0;
            foreach ($rows as $row) {
                if (!is_array($row)) {
                    continue;
                }
                if (($row['role'] ?? '') === 'admin' && !empty($row['active'])) {
                    $activeAdmins++;
                }
                if (hash_equals((string)($row['id'] ?? ''), $id)) {
                    $target = $row;
                }
            }
            if (!$target) {
                return $rows;
            }
            if (($target['role'] ?? '') === 'admin' && !empty($target['active']) && $activeAdmins <= 1) {
                throw new RuntimeException('Nije moguće obrisati posljednjeg aktivnog administratora.');
            }
            return array_values(array_filter($rows, static fn($row): bool => !is_array($row) || !hash_equals((string)($row['id'] ?? ''), $id)));
        });

        // User data deletion is explicit and complete; storage is never shared between users.
        $directory = UserWorkspace::directory($id);
        if (is_dir($directory)) {
            $this->deleteDirectory($directory);
        }
    }

    public function count(): int
    {
        return count(array_filter($this->store->read(), 'is_array'));
    }

    private function touchLogin(string $id): void
    {
        $this->store->update(function (array $rows) use ($id): array {
            foreach ($rows as &$row) {
                if (is_array($row) && hash_equals((string)($row['id'] ?? ''), $id)) {
                    $row['last_login_at'] = gmdate('c');
                    $row['updated_at'] = gmdate('c');
                    break;
                }
            }
            unset($row);
            return $rows;
        });
    }

    private function replacePasswordHash(string $id, string|false $hash): void
    {
        if (!is_string($hash) || $hash === '') {
            throw new RuntimeException('Nije moguće sigurno hashirati lozinku.');
        }
        $this->store->update(function (array $rows) use ($id, $hash): array {
            foreach ($rows as &$row) {
                if (is_array($row) && hash_equals((string)($row['id'] ?? ''), $id)) {
                    $row['password_hash'] = $hash;
                    $row['session_version'] = (int)($row['session_version'] ?? 1) + 1;
                    $row['updated_at'] = gmdate('c');
                    return $rows;
                }
            }
            unset($row);
            throw new RuntimeException('Korisnik nije pronađen.');
        });
    }

    private function publicUser(array $user): array
    {
        unset($user['password_hash']);
        $user['active'] = !empty($user['active']);
        $user['role'] = ($user['role'] ?? 'user') === 'admin' ? 'admin' : 'user';
        return $user;
    }

    private static function normalizeEmail(string $email): string
    {
        $email = strtolower(trim($email));
        if (strlen($email) > 254 || filter_var($email, FILTER_VALIDATE_EMAIL) === false) {
            throw new RuntimeException('Unesi valjanu e-mail adresu.');
        }
        return $email;
    }

    private static function validateName(string $name): string
    {
        $name = trim(preg_replace('/\s+/u', ' ', $name) ?? '');
        $length = function_exists('mb_strlen') ? mb_strlen($name) : strlen($name);
        if ($length < 2 || $length > 80) {
            throw new RuntimeException('Ime mora imati između 2 i 80 znakova.');
        }
        return $name;
    }

    private static function passwordAlgorithm(): string|int
    {
        return defined('PASSWORD_ARGON2ID') ? PASSWORD_ARGON2ID : PASSWORD_DEFAULT;
    }

    private static function validatePassword(string $password): void
    {
        if (strlen($password) < 12 || strlen($password) > 4096) {
            throw new RuntimeException('Lozinka mora imati između 12 i 4096 znakova.');
        }
    }

    private function deleteDirectory(string $directory): void
    {
        $items = @scandir($directory);
        if (!is_array($items)) {
            return;
        }
        foreach ($items as $item) {
            if ($item === '.' || $item === '..') {
                continue;
            }
            $path = $directory . '/' . $item;
            if (is_dir($path) && !is_link($path)) {
                $this->deleteDirectory($path);
            } else {
                @unlink($path);
            }
        }
        @rmdir($directory);
    }
}
