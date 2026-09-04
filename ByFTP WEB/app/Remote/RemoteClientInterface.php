<?php
declare(strict_types=1);

namespace ByFTP\Remote;

interface RemoteClientInterface
{
    public function connect(): void;
    public function list(string $path): array;
    public function makeDirectory(string $path): void;
    public function rename(string $from, string $to): void;
    public function delete(string $path, bool $directory = false): void;
    public function upload(string $localFile, string $remotePath): void;
    public function download(string $remotePath, string $localFile): void;
    public function read(string $remotePath, int $maxBytes = 2097152): string;
    public function write(string $remotePath, string $content): void;
    public function chmod(string $path, int $mode): void;
    public function disconnect(): void;
}
