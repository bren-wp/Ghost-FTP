<?php
declare(strict_types=1);

namespace GhostFTP\Security;

use GhostFTP\Storage\UserStore;

final class Auth
{
    private static ?array $cachedUser = null;

    public static function check(): bool
    {
        if (empty($_SESSION['authenticated']) || empty($_SESSION['user_id'])) {
            return false;
        }

        $user = self::loadUser();
        if (!$user || empty($user['active'])) {
            self::clearSession();
            return false;
        }

        $sessionVersion = (int)($_SESSION['user_session_version'] ?? 0);
        if ($sessionVersion !== (int)($user['session_version'] ?? 1)) {
            self::clearSession();
            return false;
        }
        return true;
    }

    public static function attempt(string $email, string $password): bool
    {
        if (strlen($password) > 4096) {
            return false;
        }
        try {
            $user = (new UserStore())->authenticate($email, $password);
        } catch (\Throwable) {
            return false;
        }
        if (!$user) {
            return false;
        }
        self::login($user);
        return true;
    }

    public static function login(array $user): void
    {
        session_regenerate_id(true);
        $now = time();
        $_SESSION['authenticated'] = true;
        $_SESSION['user_id'] = (string)$user['id'];
        $_SESSION['user_session_version'] = (int)($user['session_version'] ?? 1);
        $_SESSION['session_started_at'] = $now;
        $_SESSION['rotated_at'] = $now;
        $_SESSION['last_activity'] = $now;
        unset($_SESSION['csrf']);
        self::$cachedUser = $user;
        \GhostFTP_csrf_token();
    }

    public static function user(): ?array
    {
        return self::check() ? self::loadUser() : null;
    }

    public static function id(): string
    {
        return self::check() ? (string)($_SESSION['user_id'] ?? '') : '';
    }

    public static function isAdmin(): bool
    {
        $user = self::user();
        return $user !== null && ($user['role'] ?? '') === 'admin';
    }

    public static function refresh(): void
    {
        self::$cachedUser = null;
        if (!self::check()) {
            return;
        }
        $user = self::loadUser();
        if ($user) {
            $_SESSION['user_session_version'] = (int)($user['session_version'] ?? 1);
        }
    }

    public static function logout(): void
    {
        self::clearSession(true);
    }

    public static function requireAuth(): void
    {
        if (!self::check()) {
            \GhostFTP_redirect('login');
        }
    }

    public static function requireAdmin(): void
    {
        self::requireAuth();
        if (!self::isAdmin()) {
            http_response_code(403);
            exit('Pristup je dopušten samo administratoru.');
        }
    }

    private static function loadUser(): ?array
    {
        if (self::$cachedUser !== null) {
            return self::$cachedUser;
        }
        $id = (string)($_SESSION['user_id'] ?? '');
        if ($id === '') {
            return null;
        }
        try {
            $user = (new UserStore())->findById($id);
        } catch (\Throwable) {
            return null;
        }
        if (!$user) {
            return null;
        }
        unset($user['password_hash']);
        $user['active'] = !empty($user['active']);
        self::$cachedUser = $user;
        return self::$cachedUser;
    }

    private static function clearSession(bool $destroy = false): void
    {
        self::$cachedUser = null;
        $_SESSION = [];
        if ($destroy && ini_get('session.use_cookies')) {
            $params = session_get_cookie_params();
            setcookie(session_name(), '', [
                'expires' => time() - 42000,
                'path' => $params['path'] ?? '/',
                'domain' => $params['domain'] ?? '',
                'secure' => (bool)($params['secure'] ?? false),
                'httponly' => true,
                'samesite' => $params['samesite'] ?? 'Strict',
            ]);
        }
        if ($destroy && session_status() === PHP_SESSION_ACTIVE) {
            session_destroy();
        } elseif (session_status() === PHP_SESSION_ACTIVE) {
            session_regenerate_id(true);
        }
    }
}
