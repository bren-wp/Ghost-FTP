<?php
declare(strict_types=1);

namespace GhostFTP\Web;

interface Transport
{
    public function list(string $path): array;
    public function mkdir(string $path): void;
    public function rename(string $from, string $to): void;
    public function delete(string $path, bool $directory): void;
    public function chmod(string $path, int $mode): void;
    public function upload(string $localFile, string $remotePath): void;
    public function download(string $remotePath, string $localFile, int $maxBytes): int;
    public function read(string $remotePath, int $maxBytes): string;
    public function write(string $remotePath, string $content): void;
    public function close(): void;
}
