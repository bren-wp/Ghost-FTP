<?php
declare(strict_types=1);

namespace ByFTP\Security;

use ByFTP\Storage\JsonStore;

final class RateLimiter
{
    public function __construct(private readonly int $maxAttempts = 5, private readonly int $window = 900)
    {
    }

    private function path(string $key): string
    {
        return BYFTP_STORAGE . '/logs/rl-' . hash('sha256', $key) . '.json';
    }

    public function blocked(string $key): bool
    {
        $data = (new JsonStore($this->path($key)))->read(['first' => time(), 'count' => 0]);
        if ((int)($data['first'] ?? 0) + $this->window < time()) {
            $this->clear($key);
            return false;
        }
        return (int)($data['count'] ?? 0) >= $this->maxAttempts;
    }

    public function hit(string $key): void
    {
        (new JsonStore($this->path($key)))->update(function (array $data): array {
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
        (new JsonStore($this->path($key)))->write(['first' => time(), 'count' => 0]);
    }
}
