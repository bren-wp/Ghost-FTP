<?php
declare(strict_types=1);

namespace ByFTP\Remote;

use RuntimeException;

/** Canonical fail-closed remote path validation shared by FTP and SFTP operations. */
final class PathGuard
{
    public static function normalizeRelative(string $path): string
    {
        if (str_contains($path, "\0") || preg_match('/[\x01-\x1F\x7F]/', $path)) {
            throw new RuntimeException('Putanja sadrži nedopuštene kontrolne znakove.');
        }
        if (str_contains($path, '\\')) {
            throw new RuntimeException('Putanja ne smije sadržavati obrnute kose crte.');
        }
        if (str_contains($path, '//')) {
            throw new RuntimeException('Putanja nije kanonska: dvostruki separator nije dopušten.');
        }

        if ($path === '' || $path === '/') {
            return '/';
        }

        $parts = [];
        foreach (explode('/', $path) as $part) {
            if ($part === '') {
                continue; // leading slash
            }
            if ($part === '.' || $part === '..') {
                throw new RuntimeException('Putanja sadrži nedopuštenu relativnu komponentu.');
            }
            if (strlen($part) > 255) {
                throw new RuntimeException('Dio putanje je predugačak.');
            }
            $parts[] = $part;
        }

        $normalized = '/' . implode('/', $parts);
        if (strlen($normalized) > 4096) {
            throw new RuntimeException('Putanja je predugačka.');
        }
        return $normalized;
    }

    public static function join(string $base, string $relative): string
    {
        $base = rtrim(self::normalizeRelative($base), '/');
        $relative = self::normalizeRelative($relative);
        if ($base === '') {
            $base = '/';
        }
        return ($base === '/' ? '' : $base) . ($relative === '/' ? '/' : $relative);
    }

    public static function basename(string $path): string
    {
        if (str_contains($path, '\\')) {
            throw new RuntimeException('Naziv datoteke ne smije sadržavati obrnutu kosu crtu.');
        }
        $parts = explode('/', rtrim($path, '/'));
        $name = (string)end($parts);
        return self::segment($name);
    }

    public static function segment(string $name): string
    {
        if ($name === '' || $name === '.' || $name === '..' || str_contains($name, "\0")) {
            throw new RuntimeException('Naziv datoteke nije valjan.');
        }
        if (str_contains($name, '/') || str_contains($name, '\\')) {
            throw new RuntimeException('Naziv datoteke ne smije sadržavati znak / ili \\.');
        }
        if (preg_match('/[\x00-\x1F\x7F]/', $name)) {
            throw new RuntimeException('Naziv datoteke sadrži nedopuštene kontrolne znakove.');
        }
        if (strlen($name) > 255) {
            throw new RuntimeException('Naziv datoteke je predugačak.');
        }
        return $name;
    }

    public static function parent(string $path): string
    {
        $path = self::normalizeRelative($path);
        if ($path === '/') {
            return '/';
        }
        $parts = explode('/', ltrim($path, '/'));
        array_pop($parts);
        return $parts === [] ? '/' : '/' . implode('/', $parts);
    }

    public static function child(string $parent, string $name): string
    {
        $parent = self::normalizeRelative($parent);
        $name = self::segment($name);
        return self::normalizeRelative(($parent === '/' ? '' : $parent) . '/' . $name);
    }

    public static function ensureNotRoot(string $path): string
    {
        $path = self::normalizeRelative($path);
        if ($path === '/') {
            throw new RuntimeException('Korijenski direktorij nije dopušten za ovu operaciju.');
        }
        return $path;
    }

    public static function isDescendant(string $candidate, string $parent): bool
    {
        $candidate = self::normalizeRelative($candidate);
        $parent = rtrim(self::normalizeRelative($parent), '/');
        if ($parent === '') {
            return $candidate !== '/';
        }
        return $candidate !== $parent && str_starts_with($candidate . '/', $parent . '/');
    }
}
