<?php
declare(strict_types=1);

namespace ByFTP\Operations;

use ByFTP\Remote\PathGuard;
use ByFTP\Remote\RemoteClientInterface;
use RuntimeException;
use ZipArchive;

final class RemoteOperations
{
    private const MAX_DEPTH = 40;
    private const MAX_TREE_ITEMS = 10000;
    private const MAX_SEARCH_RESULTS = 500;
    private const MAX_ARCHIVE_ITEMS = 5000;
    private const MAX_ARCHIVE_BYTES = 536870912;

    private int $treeItems = 0;

    public function __construct(private readonly RemoteClientInterface $client)
    {
    }

    public function createFile(string $path, string $content = ''): void
    {
        $path = PathGuard::ensureNotRoot($path);
        if ($this->exists($path)) {
            throw new RuntimeException('Datoteka ili direktorij s tim nazivom već postoji.');
        }
        $this->writeAtomic($path, $content);
    }

    public function ensureDirectory(string $path): void
    {
        $path = PathGuard::normalizeRelative($path);
        if ($path === '/') return;
        $built = '';
        foreach (array_filter(explode('/', $path), 'strlen') as $segment) {
            $built .= '/' . PathGuard::segment((string)$segment);
            if (!$this->exists($built, 'dir')) {
                if ($this->exists($built)) throw new RuntimeException('Datoteka blokira izradu potrebnog direktorija.');
                $this->client->makeDirectory($built);
            }
        }
    }

    public function stat(string $path): ?array
    {
        $path = PathGuard::normalizeRelative($path);
        if ($path === '/') return ['name'=>'/','type'=>'dir','size'=>0,'modified'=>null,'permissions'=>''];
        $parent = PathGuard::parent($path);
        $name = PathGuard::basename($path);
        foreach ($this->client->list($parent) as $item) {
            if ((string)($item['name'] ?? '') === $name) return $item;
        }
        return null;
    }

    public function exists(string $path, ?string $type = null): bool
    {
        $item = $this->stat($path);
        return $item !== null && ($type === null || ($item['type'] ?? null) === $type);
    }

    public function deleteRecursive(string $path, string $type): void
    {
        $path = PathGuard::ensureNotRoot($path);
        $this->resetBudget();
        if ($type !== 'dir') {
            $this->countItem();
            $this->client->delete($path, false);
            return;
        }
        $this->deleteDirectory($path, 0);
    }

    public function copy(string $source, string $destination): void
    {
        $source = PathGuard::ensureNotRoot($source);
        $destination = PathGuard::ensureNotRoot($destination);
        if ($source === $destination) throw new RuntimeException('Izvor i odredište su isti.');
        $item = $this->stat($source);
        if ($item === null) throw new RuntimeException('Izvorna stavka više ne postoji.');
        if (($item['type'] ?? 'file') === 'dir' && PathGuard::isDescendant($destination, $source)) {
            throw new RuntimeException('Direktorij nije moguće kopirati unutar njega samoga.');
        }
        $this->resetBudget();
        if (($item['type'] ?? 'file') === 'dir') $this->copyDirectory($source, $destination, 0);
        else $this->copyFile($source, $destination);
    }

    public function move(string $source, string $destination): void
    {
        $source = PathGuard::ensureNotRoot($source);
        $destination = PathGuard::ensureNotRoot($destination);
        if ($source === $destination) return;
        if (PathGuard::isDescendant($destination, $source)) throw new RuntimeException('Stavku nije moguće premjestiti unutar nje same.');
        try {
            $this->client->rename($source, $destination);
            return;
        } catch (\Throwable $renameError) {
            $item = $this->stat($source);
            if ($item === null) throw $renameError;
            $this->copy($source, $destination);
            try {
                $this->deleteRecursive($source, (string)($item['type'] ?? 'file'));
            } catch (\Throwable $deleteError) {
                throw new RuntimeException('Stavka je kopirana na odredište, ali izvor nije moguće obrisati: ' . $deleteError->getMessage(), 0, $deleteError);
            }
        }
    }

    public function duplicate(string $source): string
    {
        $source = PathGuard::ensureNotRoot($source);
        $item = $this->stat($source);
        if ($item === null) throw new RuntimeException('Stavka više ne postoji.');
        $name = PathGuard::basename($source);
        $parent = PathGuard::parent($source);
        [$base, $ext] = $this->splitName($name, ($item['type'] ?? 'file') === 'dir');
        for ($i = 1; $i <= 999; $i++) {
            $suffix = $i === 1 ? ' - kopija' : ' - kopija ' . $i;
            $candidate = PathGuard::child($parent, $base . $suffix . $ext);
            if (!$this->exists($candidate)) {
                $this->copy($source, $candidate);
                return $candidate;
            }
        }
        throw new RuntimeException('Nije moguće pronaći slobodan naziv za kopiju.');
    }

    public function uniquePath(string $path): string
    {
        $path = PathGuard::ensureNotRoot($path);
        if (!$this->exists($path)) return $path;
        $parent = PathGuard::parent($path);
        [$base, $ext] = $this->splitName(PathGuard::basename($path), false);
        for ($i = 1; $i <= 9999; $i++) {
            $candidate = PathGuard::child($parent, $base . ' (' . $i . ')' . $ext);
            if (!$this->exists($candidate)) return $candidate;
        }
        throw new RuntimeException('Nije moguće pronaći slobodan naziv datoteke.');
    }

    public function uploadWithConflict(string $localFile, string $remotePath, string $policy = 'overwrite'): ?string
    {
        $remotePath = PathGuard::ensureNotRoot($remotePath);
        if (!in_array($policy, ['overwrite','skip','rename'], true)) throw new RuntimeException('Nepoznata politika konflikta pri uploadu.');
        $existing = $this->stat($remotePath);
        if ($existing !== null) {
            if ($policy === 'skip') return null;
            if ($policy === 'rename') $remotePath = $this->uniquePath($remotePath);
            elseif (($existing['type'] ?? 'file') === 'dir') throw new RuntimeException('Upload ne može prepisati postojeći direktorij istog naziva.');
        }
        $this->uploadAtomic($localFile, $remotePath);
        return $remotePath;
    }

    public function writeAtomic(string $remotePath, string $content): void
    {
        $tmp = $this->tempFile('write-');
        try {
            if (file_put_contents($tmp, $content, LOCK_EX) === false) throw new RuntimeException('Nije moguće pripremiti sadržaj za spremanje.');
            $this->uploadAtomic($tmp, $remotePath);
        } finally {
            @unlink($tmp);
        }
    }

    public function uploadAtomic(string $localFile, string $remotePath): void
    {
        $remotePath = PathGuard::ensureNotRoot($remotePath);
        $this->ensureDirectory(PathGuard::parent($remotePath));
        $existing = $this->stat($remotePath);
        if (($existing['type'] ?? null) === 'dir') throw new RuntimeException('Datoteka ne može prepisati postojeći direktorij istog naziva.');
        $parent = PathGuard::parent($remotePath);
        $staging = $this->temporarySibling($parent, 'byftp-upload');
        $backup = null;
        $staged = false;
        try {
            $this->client->upload($localFile, $staging);
            $staged = true;
            if ($existing !== null) {
                $backup = $this->temporarySibling($parent, 'byftp-backup');
                $this->client->rename($remotePath, $backup);
            }
            try {
                $this->client->rename($staging, $remotePath);
                $staged = false;
            } catch (\Throwable $promoteError) {
                if ($backup !== null && !$this->exists($remotePath)) {
                    try { $this->client->rename($backup, $remotePath); $backup = null; } catch (\Throwable) {}
                }
                throw $promoteError;
            }
            if ($backup !== null) {
                try { $this->client->delete($backup, false); } catch (\Throwable) {}
            }
        } finally {
            if ($staged) {
                try { $this->client->delete($staging, false); } catch (\Throwable) {}
            }
        }
    }

    public function checksum(string $path, string $algorithm = 'sha256'): array
    {
        $path = PathGuard::ensureNotRoot($path);
        $algorithm = strtolower(trim($algorithm));
        if (!in_array($algorithm, ['sha256','sha1','md5'], true)) throw new RuntimeException('Nepodržan checksum algoritam.');
        $item = $this->stat($path);
        if ($item === null || ($item['type'] ?? 'file') !== 'file') throw new RuntimeException('Checksum je dostupan samo za datoteke.');
        \byftp_assert_temp_capacity(max(0, (int)($item['size'] ?? 0)));
        $tmp = $this->tempFile('hash-');
        try {
            $this->client->download($path, $tmp);
            $hash = hash_file($algorithm, $tmp);
            if (!is_string($hash)) throw new RuntimeException('Nije moguće izračunati checksum.');
            return ['algorithm'=>$algorithm,'hash'=>$hash,'bytes'=>(int)($item['size'] ?? 0)];
        } finally { @unlink($tmp); }
    }

    public function search(string $root, string $query): array
    {
        $root = PathGuard::normalizeRelative($root);
        $query = trim($query);
        if ($query === '') return [];
        $results = [];
        $visited = 0;
        $this->searchDirectory($root, $query, $results, $visited, 0);
        return $results;
    }

    public function analyze(string $path): array
    {
        $path = PathGuard::normalizeRelative($path);
        $item = $this->stat($path);
        if ($item === null) throw new RuntimeException('Stavka više ne postoji.');
        $stats = ['files'=>0,'directories'=>0,'bytes'=>0,'items'=>0,'truncated'=>false];
        $this->analyzeNode($path, (string)($item['type'] ?? 'file'), (int)($item['size'] ?? 0), $stats, 0);
        return $stats;
    }

    public function batchRename(array $items, string $find, string $replace, string $prefix, string $suffix): array
    {
        if ($find === '' && $prefix === '' && $suffix === '') throw new RuntimeException('Zadaj barem jednu promjenu naziva.');
        $plan = [];
        $sources = [];
        foreach ($items as $item) {
            if (!is_array($item)) continue;
            $source = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));
            $oldName = PathGuard::basename($source);
            $newName = $prefix . ($find !== '' ? str_replace($find, $replace, $oldName) : $oldName) . $suffix;
            $newName = PathGuard::segment($newName);
            $destination = PathGuard::child(PathGuard::parent($source), $newName);
            if ($source === $destination) continue;
            $sources[$source] = true;
            $plan[] = ['source'=>$source,'destination'=>$destination];
        }
        $destinations = [];
        foreach ($plan as $row) {
            $destination = $row['destination'];
            if (isset($destinations[$destination])) throw new RuntimeException('Više stavki bi dobilo isti novi naziv.');
            $destinations[$destination] = true;
            if (!isset($sources[$destination]) && $this->exists($destination)) throw new RuntimeException('Novi naziv već postoji: ' . PathGuard::basename($destination));
        }
        $staged = [];
        try {
            foreach ($plan as $row) {
                $tmp = $this->temporarySibling(PathGuard::parent($row['source']), 'byftp-rename');
                $this->client->rename($row['source'], $tmp);
                $staged[] = ['tmp'=>$tmp,'source'=>$row['source'],'destination'=>$row['destination']];
            }
            foreach ($staged as $index => $row) {
                $this->client->rename($row['tmp'], $row['destination']);
                $staged[$index]['done'] = true;
            }
        } catch (\Throwable $e) {
            // A partially promoted cyclic plan cannot be restored directly because a
            // destination can also be another row's source. Re-stage every promoted
            // destination first, then restore all staging paths back to their sources.
            foreach (array_reverse($staged, true) as $index => $row) {
                if (empty($row['done'])) continue;
                try {
                    $this->client->rename($row['destination'], $row['tmp']);
                    $staged[$index]['done'] = false;
                } catch (\Throwable) {
                    // Keep done=true so the second phase does not overwrite a path
                    // when this row could not be safely returned to staging.
                }
            }
            foreach (array_reverse($staged) as $row) {
                if (!empty($row['done'])) continue;
                try { $this->client->rename($row['tmp'], $row['source']); } catch (\Throwable) {}
            }
            throw $e;
        }
        return ['renamed'=>count($plan),'items'=>$plan];
    }

    public function chmodRecursive(string $path, int $fileMode, int $dirMode): void
    {
        $path = PathGuard::ensureNotRoot($path);
        $item = $this->stat($path);
        if ($item === null) throw new RuntimeException('Stavka više ne postoji.');
        $this->resetBudget();
        $this->chmodNode($path, (string)($item['type'] ?? 'file'), $fileMode, $dirMode, 0);
    }

    public function zipPaths(array $paths, string $destination): void
    {
        $destination = PathGuard::ensureNotRoot($destination);
        $tmp = $this->buildZip($paths);
        try { $this->uploadAtomic($tmp, $destination); } finally { @unlink($tmp); }
    }

    public function buildZipToLocal(array $paths, string $localFile): void
    {
        $tmp = $this->buildZip($paths);
        if (!@rename($tmp, $localFile)) {
            if (!@copy($tmp, $localFile)) { @unlink($tmp); throw new RuntimeException('Nije moguće pripremiti ZIP za preuzimanje.'); }
            @unlink($tmp);
        }
    }

    public function extractZip(string $path, string $destination): array
    {
        if (!class_exists(ZipArchive::class)) throw new RuntimeException('PHP ZIP ekstenzija nije dostupna.');
        $path = PathGuard::ensureNotRoot($path);
        $destination = PathGuard::normalizeRelative($destination);
        $item = $this->stat($path);
        if ($item === null || ($item['type'] ?? 'file') !== 'file') throw new RuntimeException('ZIP datoteka više ne postoji.');
        \byftp_assert_temp_capacity(max(0, (int)($item['size'] ?? 0)));
        $tmp = $this->tempFile('extract-');
        try {
            $this->client->download($path, $tmp);
            $zip = new ZipArchive();
            if ($zip->open($tmp) !== true) throw new RuntimeException('ZIP arhivu nije moguće otvoriti.');
            $files = 0;
            $bytes = 0;
            try {
                if ($zip->numFiles > self::MAX_ARCHIVE_ITEMS) throw new RuntimeException('ZIP sadrži previše stavki.');
                for ($i = 0; $i < $zip->numFiles; $i++) {
                    $stat = $zip->statIndex($i);
                    if (!is_array($stat)) continue;
                    $name = str_replace('\\', '/', (string)($stat['name'] ?? ''));
                    if ($name === '' || str_starts_with($name, '/') || preg_match('/^[A-Za-z]:\//', $name)) throw new RuntimeException('ZIP sadrži nesigurnu apsolutnu putanju.');
                    $parts = explode('/', rtrim($name, '/'));
                    foreach ($parts as $part) {
                        if ($part === '' || $part === '.' || $part === '..') throw new RuntimeException('ZIP sadrži nesigurnu relativnu putanju.');
                        PathGuard::segment($part);
                    }
                    $bytes += max(0, (int)($stat['size'] ?? 0));
                    if ($bytes > self::MAX_ARCHIVE_BYTES) throw new RuntimeException('ZIP prelazi sigurnosni limit od 512 MiB.');
                    $remote = $destination;
                    foreach ($parts as $part) $remote = PathGuard::child($remote, $part);
                    if (str_ends_with($name, '/')) {
                        $this->ensureDirectory($remote);
                        continue;
                    }
                    $parent = PathGuard::parent($remote);
                    $this->ensureDirectory($parent);
                    $stream = $zip->getStream((string)$stat['name']);
                    if (!is_resource($stream)) throw new RuntimeException('ZIP stavku nije moguće pročitati.');
                    $entryTmp = $this->tempFile('zipentry-');
                    $out = @fopen($entryTmp, 'wb');
                    if (!is_resource($out)) { fclose($stream); @unlink($entryTmp); throw new RuntimeException('Nije moguće pripremiti ZIP stavku.'); }
                    $copied = stream_copy_to_stream($stream, $out, self::MAX_ARCHIVE_BYTES + 1);
                    fclose($stream); fclose($out);
                    if ($copied === false || $copied > self::MAX_ARCHIVE_BYTES) { @unlink($entryTmp); throw new RuntimeException('ZIP stavka prelazi sigurnosni limit.'); }
                    try { $this->uploadAtomic($entryTmp, $remote); } finally { @unlink($entryTmp); }
                    $files++;
                }
            } finally { $zip->close(); }
            return ['files'=>$files,'bytes'=>$bytes];
        } finally { @unlink($tmp); }
    }

    private function buildZip(array $paths): string
    {
        if (!class_exists(ZipArchive::class)) throw new RuntimeException('PHP ZIP ekstenzija nije dostupna.');
        if ($paths === []) throw new RuntimeException('Odaberi barem jednu stavku za ZIP.');
        $tmpBase = $this->tempFile('zip-');
        @unlink($tmpBase);
        $tmp = $tmpBase . '.zip';
        $zip = new ZipArchive();
        if ($zip->open($tmp, ZipArchive::CREATE | ZipArchive::OVERWRITE) !== true) throw new RuntimeException('Nije moguće stvoriti ZIP arhivu.');
        $count = 0;
        $bytes = 0;
        $temps = [];
        $roots = [];
        try {
            foreach ($paths as $remote) {
                $remote = PathGuard::ensureNotRoot((string)$remote);
                $item = $this->stat($remote);
                if ($item === null) continue;
                $root = $this->uniqueArchiveName(PathGuard::basename($remote), ($item['type'] ?? 'file') === 'dir', $roots);
                $this->addZipNode($zip, $remote, $root, $count, $bytes, $temps, 0);
            }
            if ($count === 0) throw new RuntimeException('Nema dostupnih stavki za ZIP.');
        } catch (\Throwable $e) {
            $zip->close();
            foreach ($temps as $temp) @unlink($temp);
            @unlink($tmp);
            throw $e;
        }
        $zip->close();
        foreach ($temps as $temp) @unlink($temp);
        return $tmp;
    }

    private function addZipNode(ZipArchive $zip, string $remote, string $archivePath, int &$count, int &$bytes, array &$temps, int $depth): void
    {
        $this->guardDepth($depth);
        $item = $this->stat($remote);
        if ($item === null) return;
        $count++;
        if ($count > self::MAX_ARCHIVE_ITEMS) throw new RuntimeException('Odabir prelazi sigurnosni limit ZIP stavki.');
        if (($item['type'] ?? 'file') === 'dir') {
            $zip->addEmptyDir(rtrim($archivePath, '/'));
            foreach ($this->client->list($remote) as $child) {
                $name = PathGuard::segment((string)$child['name']);
                $this->addZipNode($zip, PathGuard::child($remote, $name), rtrim($archivePath,'/').'/'.$name, $count, $bytes, $temps, $depth + 1);
            }
            return;
        }
        $bytes += max(0, (int)($item['size'] ?? 0));
        if ($bytes > self::MAX_ARCHIVE_BYTES) throw new RuntimeException('Odabir prelazi sigurnosni limit ZIP veličine.');
        \byftp_assert_temp_capacity(max(0, (int)($item['size'] ?? 0)));
        $tmp = $this->tempFile('zipitem-');
        try {
            $this->client->download($remote, $tmp);
            if (!$zip->addFile($tmp, ltrim($archivePath,'/'))) throw new RuntimeException('Nije moguće dodati datoteku u ZIP.');
            $temps[] = $tmp;
        } catch (\Throwable $e) { @unlink($tmp); throw $e; }
    }

    private function deleteDirectory(string $path, int $depth): void
    {
        $this->guardDepth($depth);
        $this->countItem();
        foreach ($this->client->list($path) as $item) {
            $child = PathGuard::child($path, (string)$item['name']);
            if (($item['type'] ?? 'file') === 'dir') $this->deleteDirectory($child, $depth + 1);
            else { $this->countItem(); $this->client->delete($child, false); }
        }
        $this->client->delete($path, true);
    }

    private function copyDirectory(string $source, string $destination, int $depth): void
    {
        $this->guardDepth($depth);
        $this->countItem();
        if (!$this->exists($destination, 'dir')) {
            if ($this->exists($destination)) throw new RuntimeException('Datoteka blokira odredišni direktorij.');
            $this->ensureDirectory(PathGuard::parent($destination));
            $this->client->makeDirectory($destination);
        }
        foreach ($this->client->list($source) as $item) {
            $src = PathGuard::child($source, (string)$item['name']);
            $dst = PathGuard::child($destination, (string)$item['name']);
            if (($item['type'] ?? 'file') === 'dir') $this->copyDirectory($src, $dst, $depth + 1); else $this->copyFile($src, $dst);
        }
    }

    private function copyFile(string $source, string $destination): void
    {
        $this->countItem();
        $item = $this->stat($source);
        if ($item === null || ($item['type'] ?? 'file') !== 'file') throw new RuntimeException('Izvorna datoteka više ne postoji.');
        \byftp_assert_temp_capacity(max(0, (int)($item['size'] ?? 0)));
        $tmp = $this->tempFile('copy-');
        try { $this->client->download($source, $tmp); $this->uploadAtomic($tmp, $destination); } finally { @unlink($tmp); }
    }

    private function searchDirectory(string $path, string $query, array &$results, int &$visited, int $depth): void
    {
        if ($depth > self::MAX_DEPTH || count($results) >= self::MAX_SEARCH_RESULTS || $visited >= self::MAX_TREE_ITEMS) return;
        foreach ($this->client->list($path) as $item) {
            if (++$visited > self::MAX_TREE_ITEMS) return;
            $child = PathGuard::child($path, (string)$item['name']);
            if (stripos((string)$item['name'], $query) !== false) {
                $results[] = array_merge($item, ['path'=>$child]);
                if (count($results) >= self::MAX_SEARCH_RESULTS) return;
            }
            if (($item['type'] ?? 'file') === 'dir') $this->searchDirectory($child, $query, $results, $visited, $depth + 1);
        }
    }

    private function analyzeNode(string $path, string $type, int $size, array &$stats, int $depth): void
    {
        if ($stats['items'] >= self::MAX_TREE_ITEMS || $depth > self::MAX_DEPTH) { $stats['truncated'] = true; return; }
        $stats['items']++;
        if ($type !== 'dir') { $stats['files']++; $stats['bytes'] += max(0,$size); return; }
        $stats['directories']++;
        foreach ($this->client->list($path) as $item) {
            $this->analyzeNode(PathGuard::child($path,(string)$item['name']), (string)($item['type'] ?? 'file'), (int)($item['size'] ?? 0), $stats, $depth + 1);
            if ($stats['truncated']) return;
        }
    }

    private function chmodNode(string $path, string $type, int $fileMode, int $dirMode, int $depth): void
    {
        $this->guardDepth($depth);
        $this->countItem();
        if ($type !== 'dir') { $this->client->chmod($path, $fileMode); return; }
        $this->client->chmod($path, $dirMode);
        foreach ($this->client->list($path) as $item) {
            $this->chmodNode(PathGuard::child($path,(string)$item['name']), (string)($item['type'] ?? 'file'), $fileMode, $dirMode, $depth + 1);
        }
    }

    private function temporarySibling(string $parent, string $prefix): string
    {
        for ($i = 0; $i < 32; $i++) {
            $candidate = PathGuard::child($parent, $prefix . '-' . bin2hex(random_bytes(8)) . '.tmp');
            if (!$this->exists($candidate)) return $candidate;
        }
        throw new RuntimeException('Nije moguće pripremiti sigurnu privremenu remote datoteku.');
    }

    private function tempFile(string $prefix): string
    {
        $tmp = tempnam(BYFTP_STORAGE . '/tmp', $prefix);
        if ($tmp === false) throw new RuntimeException('Nije moguće stvoriti privremenu datoteku.');
        @chmod($tmp, 0600);
        return $tmp;
    }

    private function splitName(string $name, bool $directory): array
    {
        $dot = !$directory ? strrpos($name, '.') : false;
        return $dot !== false && $dot > 0 ? [substr($name,0,$dot), substr($name,$dot)] : [$name,''];
    }

    private function uniqueArchiveName(string $name, bool $directory, array &$used): string
    {
        if (!isset($used[$name])) { $used[$name] = true; return $name; }
        [$base,$ext] = $this->splitName($name,$directory);
        for ($i=2;$i<=9999;$i++) {
            $candidate = $base . ' (' . $i . ')' . $ext;
            if (!isset($used[$candidate])) { $used[$candidate]=true; return $candidate; }
        }
        throw new RuntimeException('Previše ZIP stavki s istim nazivom.');
    }

    private function resetBudget(): void { $this->treeItems = 0; }
    private function countItem(): void
    {
        if (++$this->treeItems > self::MAX_TREE_ITEMS) throw new RuntimeException('Operacija prelazi sigurnosni limit od ' . self::MAX_TREE_ITEMS . ' stavki.');
    }
    private function guardDepth(int $depth): void
    {
        if ($depth > self::MAX_DEPTH) throw new RuntimeException('Dosegnut je sigurnosni limit dubine direktorija.');
    }
}
