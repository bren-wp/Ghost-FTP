<?php
declare(strict_types=1);

// Availability, not signing identity. Full hashes are verified at packaging
// and by deployment QA; HTTP discovery avoids hashing large installers on
// every request. Reject incomplete or inconsistent deployment metadata.
function local_release(): array {
    $root = __DIR__ . '/downloads/';
    $catalogPath = $root . 'catalog.json';
    $raw = !is_link(rtrim($root, '/')) && !is_link($catalogPath)
        ? @file_get_contents($catalogPath, false, null, 0, 4097) : false;
    $catalog = is_string($raw) && strlen($raw) <= 4096 ? json_decode($raw, true) : null;
    $version = is_array($catalog) && is_string($catalog['version'] ?? null)
        && preg_match('/^(0|[1-9]\d{0,9})\.(0|[1-9]\d{0,9})\.(0|[1-9]\d{0,9})$/D', $catalog['version'])
        ? $catalog['version'] : '';
    foreach (explode('.', $version) as $part) {
        if (strlen($part) === 10 && strcmp($part, '4294967295') > 0) {
            $version = '';
            break;
        }
    }
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
    $expected = array_merge(array_values($packages['Windows']), array_values($packages['Linux']));
    $metadata = $catalog['packages'] ?? null;
    $ready = $version !== '' && is_array($catalog) && count($catalog) === 2
        && is_array($metadata) && count($metadata) === count($expected);
    $sums = $root . 'SHA256SUMS.txt';
    $sumText = !is_link($sums) && is_file($sums)
        ? @file_get_contents($sums, false, null, 0, 4097) : false;
    $checksums = [];
    if (!is_string($sumText) || strlen($sumText) > 4096) {
        $ready = false;
    } else {
        foreach (explode("\n", rtrim($sumText, "\n")) as $line) {
            if (!preg_match('/^([a-f0-9]{64})  ([A-Za-z0-9._-]+)$/D', $line, $match)
                || !in_array($match[2], $expected, true) || isset($checksums[$match[2]])) {
                $ready = false;
                break;
            }
            $checksums[$match[2]] = $match[1];
        }
    }
    $ready = $ready && count($checksums) === count($expected);
    foreach ($expected as $name) {
        $file = $root . $name;
        $record = is_array($metadata) ? ($metadata[$name] ?? null) : null;
        $ready = $ready && is_array($record) && count($record) === 2
            && is_int($record['size'] ?? null) && $record['size'] > 0
            && is_string($record['sha256'] ?? null)
            && ($checksums[$name] ?? null) === $record['sha256']
            && !is_link($file) && is_file($file) && is_readable($file)
            && filesize($file) === $record['size'];
    }
    return [$version, $packages, $ready];
}
