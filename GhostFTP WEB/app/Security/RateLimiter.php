<?php
declare(strict_types=1);

namespace GhostFTP\Security;

use GhostFTP\Storage\JsonStore;

final class RateLimiter
{
    public function __construct(private readonly int $maxAttempts = 5, private readonly int $window = 900)
    {
    }

    private function path(string $key): string
    {
        return GhostFTP_STORAGE . '/logs/rl-' . hash('sha256', $key) . '.json';
    }

    private function store(string $key): JsonStore
    {
        // Login/registration attempt counters are security state. Never roll them back
        // to an older .bak generation after primary corruption or loss because that can
        // reduce the observed attempt count and weaken brute-force protection.
        return new JsonStore($this->path($key), false);
    }

    /**
     * Atomically reserve one authentication/registration attempt.
     *
     * Returns true while the caller is still within the configured limit and false once
     * the limit was already exhausted. JsonStore::update() holds the exclusive store lock
     * across both the threshold check and increment, so concurrent requests cannot all
     * observe the same pre-increment count.
     */
    public function consume(string $key): bool
    {
        $allowed = false;
        $now = time();
        $this->store($key)->update(function (array $data) use (&$allowed, $now): array {
            if ((int)($data['first'] ?? 0) + $this->window < $now) {
                $data = ['first' => $now, 'count' => 0];
            }
            $data['first'] = (int)($data['first'] ?? $now);
            $count = max(0, (int)($data['count'] ?? 0));
            if ($count >= $this->maxAttempts) {
                $allowed = false;
                $data['count'] = $count;
                return $data;
            }
            $data['count'] = $count + 1;
            $allowed = true;
            return $data;
        }, ['first' => $now, 'count' => 0]);
        return $allowed;
    }

    public function blocked(string $key): bool
    {
        $data = $this->store($key)->read(['first' => time(), 'count' => 0]);
        if ((int)($data['first'] ?? 0) + $this->window < time()) {
            $this->clear($key);
            return false;
        }
        return (int)($data['count'] ?? 0) >= $this->maxAttempts;
    }

    public function hit(string $key): void
    {
        $this->store($key)->update(function (array $data): array {
            if ((int)($data['first'] ?? 0) + $this->window < time()) {
                $data = ['first' => time(), 'count' => 0];
            }
            $data['first'] = (int)($data['first'] ?? time());
            $data['count'] = (int)($data['count'] ?? 0) + 1;
            return $data;
        }, ['first' => time(), 'count' => 0]);
    }

    public function clear(string $key): void
    {
        // Reset through JsonStore instead of unlinking the primary and lock files.
        // JsonStore keeps a last-known-good .bak generation; deleting only the
        // primary would make the next read restore stale failed-login attempts.
        // Reusing the same lock file also preserves cross-request serialization.
        $this->store($key)->write(['first' => time(), 'count' => 0]);
    }
}
