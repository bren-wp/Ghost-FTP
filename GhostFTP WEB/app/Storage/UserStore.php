<?php
declare(strict_types=1);

namespace ByFTP\Storage;

use RuntimeException;

final class UserStore
{
    private JsonStore $store;

    public function __construct()
    {
        // Authentication/authorization state must never silently roll back to the
        // previous generation after primary corruption. Keep users.json.bak for
        // explicit operator recovery, but fail closed for runtime reads/updates.
        $this->store = new JsonStore(BYFTP_STORAGE . '/users.json', false);
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

        // Prepare the isolated workspace before publishing the account in users.json.
        // This prevents a failed workspace creation from ever becoming a valid registry
        // generation (or JsonStore backup) that could later resurrect a ghost account.
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
        $createdId = (string)$created['id'];
        UserWorkspace::ensure($createdId);

        try {
            $this->store->update(function (array $rows) use ($email, $created): array {
                foreach ($rows as $row) {
                    if (is_array($row) && hash_equals((string)($row['email'] ?? ''), $email)) {
                        throw new RuntimeException('Korisnički račun s tom e-mail adresom već postoji.');
                    }
                }
                $rows[] = $created;
                return array_values($rows);
            });
        } catch (\Throwable $e) {
            // The workspace is still empty at this point because the caller cannot use
            // the account until the registry write succeeds. Remove only that new path.
            @rmdir(UserWorkspace::directory($createdId));
            throw $e;
        }

        return $this->publicUser($created);
    }

    public function authenticate(string $email, string $password): ?array
    {
        $user = $this->findByEmail($email);
        $verifiedHash = is_array($user) ? (string)($user['password_hash'] ?? '') : '';
        if (!$user || empty($user['active']) || !password_verify($password, $verifiedHash)) {
            return null;
        }

        $rehash = null;
        if (password_needs_rehash($verifiedHash, self::passwordAlgorithm())) {
            $candidate = password_hash($password, self::passwordAlgorithm());
            if (!is_string($candidate) || $candidate === '') {
                throw new RuntimeException('Nije moguće sigurno obnoviti hash lozinke.');
            }
            $rehash = $candidate;
        }

        // Verification and session publication are two separate CPU/storage phases. Before
        // returning an authenticated user, atomically prove the verified password generation
        // is still current. This prevents an old password from winning a race with a reset.
        $fresh = $this->completeAuthentication((string)$user['id'], $verifiedHash, $rehash);
        return $this->publicUser($fresh);
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
                    if (!empty($row['deleting'])) {
                        throw new RuntimeException('Brisanje korisničkog računa nije dovršeno. Ponovi brisanje prije drugih izmjena.');
                    }
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
        if (!empty($user['deleting'])) {
            throw new RuntimeException('Brisanje korisničkog računa nije dovršeno. Lozinku nije moguće mijenjati.');
        }
        $expectedCurrentHash = (string)($user['password_hash'] ?? '');
        if ($requireCurrent && !password_verify($currentPassword, $expectedCurrentHash)) {
            throw new RuntimeException('Trenutačna lozinka nije točna.');
        }
        $hash = password_hash($newPassword, self::passwordAlgorithm());
        if (!is_string($hash)) {
            throw new RuntimeException('Nije moguće sigurno spremiti novu lozinku.');
        }

        // Current-password verification happens outside the file lock because Argon2/bcrypt
        // can be intentionally expensive. The write itself is compare-and-swap bound to the
        // exact verified hash, so a concurrent password change invalidates this request.
        $this->replacePasswordHash($id, $hash, $requireCurrent ? $expectedCurrentHash : null);
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
                if (!empty($row['deleting'])) {
                    throw new RuntimeException('Brisanje korisničkog računa nije dovršeno. Ponovi brisanje prije promjene prava.');
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
        $markedForDeletion = false;
        $this->store->update(function (array $rows) use ($id, &$markedForDeletion): array {
            $targetIndex = null;
            $target = null;
            $activeAdmins = 0;
            foreach ($rows as $index => $row) {
                if (!is_array($row)) {
                    continue;
                }
                if (($row['role'] ?? '') === 'admin' && !empty($row['active'])) {
                    $activeAdmins++;
                }
                if (hash_equals((string)($row['id'] ?? ''), $id)) {
                    $targetIndex = $index;
                    $target = $row;
                }
            }
            if ($targetIndex === null || !is_array($target)) {
                return $rows;
            }
            if (($target['role'] ?? '') === 'admin' && !empty($target['active']) && $activeAdmins <= 1) {
                throw new RuntimeException('Nije moguće obrisati posljednjeg aktivnog administratora.');
            }

            $row = $rows[$targetIndex];
            if (empty($row['deleting'])) {
                $row['deleting'] = true;
                if (!empty($row['active'])) {
                    $row['session_version'] = (int)($row['session_version'] ?? 1) + 1;
                }
                $row['active'] = false;
                $row['updated_at'] = gmdate('c');
                $rows[$targetIndex] = $row;
            }
            $markedForDeletion = true;
            return array_values($rows);
        });

        if (!$markedForDeletion) {
            return;
        }

        // Keep the inactive registry row until workspace cleanup is fully verified. If a
        // filesystem operation fails, the administrator can safely retry deletion instead
        // of losing the only registry reference to orphaned private data.
        $directory = UserWorkspace::directory($id);
        try {
            if (is_link($directory)) {
                // Never recurse through a workspace-root symlink. Only unlink the symlink
                // itself so a manipulated workspace cannot delete an external directory.
                if (!@unlink($directory) && is_link($directory)) {
                    throw new RuntimeException('Nije moguće ukloniti simboličku poveznicu korisničkog workspacea.');
                }
            } elseif (is_dir($directory)) {
                $this->deleteDirectory($directory);
            } elseif (file_exists($directory)) {
                if (!@unlink($directory) && file_exists($directory)) {
                    throw new RuntimeException('Korisnički workspace nije direktorij i nije ga moguće ukloniti.');
                }
            }

            if (is_link($directory) || is_dir($directory) || file_exists($directory)) {
                throw new RuntimeException('Korisnički workspace nije u potpunosti uklonjen.');
            }
        } catch (\Throwable $e) {
            throw new RuntimeException(
                'Korisnički račun je deaktiviran, ali workspace nije moguće u potpunosti obrisati. Provjeri dozvole i ponovi brisanje: ' . $e->getMessage(),
                0,
                $e
            );
        }

        // Final registry removal happens only after the private workspace is gone. If this
        // write fails, the inactive deleting row remains and a retry can finish safely.
        $this->store->update(static fn(array $rows): array => array_values(array_filter(
            $rows,
            static fn($row): bool => !is_array($row) || !hash_equals((string)($row['id'] ?? ''), $id)
        )));
    }

    public function count(): int
    {
        return count(array_filter($this->store->read(), 'is_array'));
    }

    private function completeAuthentication(string $id, string $verifiedHash, ?string $rehash): array
    {
        $authenticated = null;
        $this->store->update(function (array $rows) use ($id, $verifiedHash, $rehash, &$authenticated): array {
            foreach ($rows as &$row) {
                if (!is_array($row) || !hash_equals((string)($row['id'] ?? ''), $id)) {
                    continue;
                }
                if (empty($row['active']) || !empty($row['deleting'])) {
                    throw new RuntimeException('Korisnički račun više nije aktivan.');
                }
                if (!hash_equals($verifiedHash, (string)($row['password_hash'] ?? ''))) {
                    throw new RuntimeException('Lozinka je promijenjena tijekom prijave. Ponovi prijavu s aktualnom lozinkom.');
                }
                if ($rehash !== null) {
                    if ($rehash === '') {
                        throw new RuntimeException('Nije moguće sigurno obnoviti hash lozinke.');
                    }
                    $row['password_hash'] = $rehash;
                    $row['session_version'] = (int)($row['session_version'] ?? 1) + 1;
                }
                $row['last_login_at'] = gmdate('c');
                $row['updated_at'] = gmdate('c');
                $authenticated = $row;
                return $rows;
            }
            unset($row);
            throw new RuntimeException('Korisnik nije pronađen.');
        });
        if (!is_array($authenticated)) {
            throw new RuntimeException('Prijava nije dovršena.');
        }
        return $authenticated;
    }

    private function replacePasswordHash(string $id, string|false $hash, ?string $expectedCurrentHash = null): void
    {
        if (!is_string($hash) || $hash === '') {
            throw new RuntimeException('Nije moguće sigurno hashirati lozinku.');
        }
        $this->store->update(function (array $rows) use ($id, $hash, $expectedCurrentHash): array {
            foreach ($rows as &$row) {
                if (is_array($row) && hash_equals((string)($row['id'] ?? ''), $id)) {
                    if (!empty($row['deleting'])) {
                        throw new RuntimeException('Brisanje korisničkog računa nije dovršeno. Lozinku nije moguće mijenjati.');
                    }
                    if ($expectedCurrentHash !== null
                        && !hash_equals($expectedCurrentHash, (string)($row['password_hash'] ?? ''))) {
                        throw new RuntimeException('Lozinka je u međuvremenu promijenjena. Ponovi radnju s aktualnom lozinkom.');
                    }
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
        $user['deleting'] = !empty($user['deleting']);
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
        if (is_link($directory)) {
            if (!@unlink($directory) && is_link($directory)) {
                throw new RuntimeException('Nije moguće ukloniti simboličku poveznicu iz korisničkog workspacea.');
            }
            return;
        }

        $items = @scandir($directory);
        if (!is_array($items)) {
            throw new RuntimeException('Nije moguće pročitati korisnički workspace radi brisanja.');
        }
        foreach ($items as $item) {
            if ($item === '.' || $item === '..') {
                continue;
            }
            $path = $directory . '/' . $item;
            if (is_dir($path) && !is_link($path)) {
                $this->deleteDirectory($path);
            } else {
                if (!@unlink($path) && (file_exists($path) || is_link($path))) {
                    throw new RuntimeException('Nije moguće ukloniti stavku iz korisničkog workspacea.');
                }
            }
        }
        if (!@rmdir($directory) && is_dir($directory)) {
            throw new RuntimeException('Nije moguće ukloniti korisnički workspace direktorij.');
        }
    }
}
