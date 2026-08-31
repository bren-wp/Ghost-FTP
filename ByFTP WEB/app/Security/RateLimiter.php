<?php
declare(strict_types=1);

namespace ByFTP\Security;

use ByFTP\Storage\JsonStore;

final class RateLimiter
{
    public function __construct(
        private readonly string $root,
        private readonly int $maxHits = 8,
        private readonly int $windowSeconds = 900,
    ) {
    }

    public function blocked(string $key): bool
    {
        $cutoff = time() - $this->windowSeconds;
        $store = new JsonStore($this->pathFor($key));
        $data = $store->read(['hits' => []]);
        $hits = array_values(array_filter((array)($data['hits'] ?? []), static fn($time): bool => is_int($time) && $time >= $cutoff));
        return count($hits) >= $this->maxHits;
    }

    public function hit(string $key): void
    {
        $cutoff = time() - $this->windowSeconds;
        $store = new JsonStore($this->pathFor($key));
        $store->update(static function (array $data) use ($cutoff): array {
            $hits = array_values(array_filter((array)($data['hits'] ?? []), static fn($time): bool => is_int($time) && $time >= $cutoff));
            $hits[] = time();
            return ['hits' => $hits];
        }, ['hits' => []]);
    }

    public function clear(string $key): void
    {
        // Reset through JsonStore instead of unlinking the primary/lock files.
        // JsonStore may have a last-known-good .bak file, so deleting only the
        // primary can resurrect stale hits on the next read. Keeping the same
        // lock file also preserves cross-request serialization while clearing.
        $store = new JsonStore($this->pathFor($key));
        $store->write(['hits' => []]);
    }

    private function pathFor(string $key): string
    {
        return rtrim($this->root, '/\\') . '/rate-' . hash('sha256', $key) . '.json';
    }
}
