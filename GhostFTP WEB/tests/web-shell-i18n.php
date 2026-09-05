<?php
declare(strict_types=1);

$root = dirname(__DIR__);
$failed = false;

function shell_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function shell_source(string $root, string $relative): string
{
    $value = @file_get_contents($root . '/' . $relative);
    if (!is_string($value)) {
        fwrite(STDERR, "FAIL: missing {$relative}\n");
        exit(1);
    }
    return $value;
}

$index = shell_source($root, 'index.php');
$app = shell_source($root, 'assets/js/app.js');
$api = shell_source($root, 'assets/js/api.js');
$pwa = shell_source($root, 'assets/js/pwa.js');
$settings = shell_source($root, 'assets/js/settings.js');
$language = shell_source($root, 'assets/js/language.js');

foreach ([
    'use GhostFTP\\I18n;',
    'use GhostFTP\\Storage\\PreferenceStore;',
    'name="ghostftp-language"',
    'name="ghostftp-i18n"',
    'id="languageSelect"',
    'I18n::languages()',
] as $marker) {
    shell_check(str_contains($index, $marker), "index.php contains {$marker}");
}

shell_check(!str_contains($index, '<html lang="hr">'), 'web shell no longer hardcodes Croatian as document language');
shell_check(str_contains($index, '<html lang="<?= GhostFTP_e($htmlLanguage) ?>">'), 'document language is server-normalized');
shell_check(str_contains($app, "const pageLanguage = document.querySelector('meta[name=\"ghostftp-language\"]')"), 'app.js consumes server language');
shell_check(str_contains($app, "readMetaJson('ghostftp-i18n')"), 'app.js consumes server translation catalog');
shell_check(str_contains($app, 'function browserLocale()'), 'app.js maps canonical language to browser locale');
shell_check(!str_contains($app, "toLocaleLowerCase('hr')"), 'file filtering is no longer Croatian-hardcoded');
shell_check(str_contains($app, "name:'Ghost-FTP-download.zip'"), 'download archive uses public Ghost FTP naming');
shell_check(!str_contains($app, "name:'GhostFTP-download.zip'"), 'legacy compact public archive name is gone');
shell_check(str_contains($api, 'Invalid server response'), 'API JavaScript fallback is English-first');
shell_check(str_contains($pwa, 'On iPhone or iPad'), 'PWA install fallback is English-first');
shell_check(str_contains($settings, 'saved Ghost FTP data'), 'settings destructive confirmation uses public brand');
shell_check(str_contains($language, "api('save_preferences'"), 'language selector persists through existing preference API');
shell_check(str_contains($language, "api('me')"), 'language selector preserves existing preference state');

if ($failed) {
    fwrite(STDERR, "WEB_SHELL_I18N_TEST=FAIL\n");
    exit(1);
}

echo "WEB_SHELL_I18N_TEST=PASS\n";
