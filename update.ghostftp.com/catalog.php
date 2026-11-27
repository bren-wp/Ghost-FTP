<?php
declare(strict_types=1);

// Availability, not signing identity. All expected local files must exist.
function local_release(): array {
    $root = __DIR__ . '/downloads/';
    $raw = @file_get_contents($root . 'catalog.json', false, null, 0, 4097);
    $catalog = is_string($raw) && strlen($raw) <= 4096 ? json_decode($raw, true) : null;
    $version = is_array($catalog) && is_string($catalog['version'] ?? null)
        && preg_match('/^(0|[1-9]\d{0,9})\.(0|[1-9]\d{0,9})\.(0|[1-9]\d{0,9})$/D', $catalog['version'])
        ? $catalog['version'] : '';
    $packages = [
        'Windows' => [
            'Setup (EXE)' => "GhostFTP-Windows-x64-Setup-v$version.exe",
            'Setup (MSI)' => "GhostFTP-Windows-x64-Setup-v$version.msi",
        ],
        'Linux' => [
            'DEB' => "GhostFTP-Linux-amd64-v$version.deb",
            'RPM' => "GhostFTP-Linux-x86_64-v$version.rpm",
        ],
    ];
    $ready = $version !== '';
    foreach ($packages as $items) {
        foreach ($items as $name) {
            $file = $root . $name;
            $ready = $ready && !is_link($file) && is_file($file) && filesize($file) > 0;
        }
    }
    $sums = $root . 'SHA256SUMS.txt';
    $ready = $ready && !is_link($sums) && is_file($sums) && filesize($sums) > 0;
    return [$version, $packages, $ready];
}
