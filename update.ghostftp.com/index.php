<?php
declare(strict_types=1);
header('Cache-Control: no-store');
header('Content-Type: text/html; charset=UTF-8');
$hr = ($_GET['lang'] ?? '') === 'hr';
function e(string $s): string { return htmlspecialchars($s, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8'); }
function t(string $en, string $hr): string { return e($GLOBALS['hr'] ? $hr : $en); }
$catalog = json_decode((string)@file_get_contents(__DIR__ . '/downloads/catalog.json'), true);
$version = is_array($catalog) && is_string($catalog['version'] ?? null) && preg_match('/^\d+\.\d+\.\d+$/D', $catalog['version']) ? $catalog['version'] : '';
$packages = ['Windows' => ['Setup (EXE)' => "GhostFTP-Windows-x64-Setup-v$version.exe", 'Setup (MSI)' => "GhostFTP-Windows-x64-Setup-v$version.msi"], 'Linux' => ['DEB' => "GhostFTP-Linux-amd64-v$version.deb", 'RPM' => "GhostFTP-Linux-x86_64-v$version.rpm"]];
$base = 'downloads/';
$ready = $version !== '';
foreach ($packages as $items) {
    foreach ($items as $name) {
        $ready = $ready && is_file(__DIR__ . '/' . $base . $name) && !is_link(__DIR__ . '/' . $base . $name) && filesize(__DIR__ . '/' . $base . $name) > 0;
    }
}
$ready = $ready && is_file(__DIR__ . '/downloads/SHA256SUMS.txt');
?>
<!doctype html>
<html lang="<?= $hr ? 'hr' : 'en' ?>"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Ghost FTP — <?= t('Updates','Ažuriranja') ?></title><link rel="icon" href="assets/icon.svg" type="image/svg+xml"><link rel="stylesheet" href="assets/site.css"></head><body>
<header><a class="brand" href="https://ghostftp.com"><img src="assets/icon.svg" alt="" width="48" height="48"><span>Ghost <b>FTP</b><small>by Brendigo</small></span></a><nav aria-label="Language"><a href="?lang=en" <?= !$hr ? 'aria-current="page"' : '' ?>>EN</a><a href="?lang=hr" <?= $hr ? 'aria-current="page"' : '' ?>>HR</a></nav></header>
<main><section class="hero"><p class="eyebrow">GHOST FTP / <?= t('UPDATE CENTER','CENTAR ZA AŽURIRANJE') ?></p><h1><?= t('Keep Ghost FTP up to date.','Redovito ažurirajte Ghost FTP.') ?></h1><p><?= t('Download the latest locally hosted installation packages for Windows and Linux.','Preuzmite najnovije lokalno dostupne instalacijske pakete za Windows i Linux.') ?></p><div class="version"><strong><?= $ready ? e("v" . $version) : t("Unavailable","Nedostupno") ?></strong><span> <?= t('Manual installation','Ručna instalacija') ?></span></div><p><?= t('This website offers manual installation packages. Automatic installation requires a verified update signature.','Ova stranica nudi pakete za ručnu instalaciju. Automatska instalacija zahtijeva verificirani potpis ažuriranja.') ?></p></section>
<?php if ($ready): ?><h2><?= t('Choose your platform','Odaberite platformu') ?></h2><div class="platforms"><?php foreach ($packages as $os => $items): ?><section class="panel"><h3><?= e($os) ?> <small>x64</small></h3><?php foreach ($items as $label => $asset): ?><a class="download" href="<?= e($base . $asset) ?>" download><span><?= e($label) ?></span><span aria-hidden="true">↓</span></a><?php endforeach; ?></section><?php endforeach; ?></div><?php else: ?><p role="status"><?= t("Installation packages are temporarily unavailable.","Instalacijski paketi trenutno nisu dostupni.") ?></p><?php endif; ?>
<section class="panel instructions"><h2><?= t('Update safely','Sigurno ažuriranje') ?></h2><p><?= t('Finish active transfers and close Ghost FTP before installing. Use the same package format as your current installation and keep a backup of your configuration. For DEB and RPM, use your distribution package manager.','Dovršite aktivne prijenose i zatvorite Ghost FTP prije instalacije. Koristite isti format paketa kao kod postojeće instalacije i sačuvajte sigurnosnu kopiju postavki. Za DEB i RPM koristite upravitelj paketa distribucije.') ?></p><p><?= t('In the app, open Help & About → Updates to check for a signed update.','U aplikaciji otvorite Help & About → Updates za provjeru potpisanog ažuriranja.') ?></p><?php if ($ready): ?><a href="downloads/SHA256SUMS.txt">SHA-256 checksums</a><?php endif; ?></section></main><footer><span>Ghost FTP by Brendigo</span><a href="https://ghostftp.com"><?= t('Official website','Službena stranica') ?></a></footer></body></html>
