<?php
declare(strict_types=1);

require __DIR__ . '/../app/bootstrap.php';

use GhostFTP\I18n;

function GhostFTP_i18n_assert(bool $condition, string $message): void
{
    if (!$condition) {
        fwrite(STDERR, "WEB_I18N_TEST_FAILED: {$message}\n");
        exit(1);
    }
}

$expected = [
    'en', 'hr', 'de', 'fr', 'es', 'tr', 'el', 'pt', 'zh', 'ru', 'hi', 'ja',
    'it', 'pl', 'nl', 'cs', 'uk', 'sv', 'ro', 'hu', 'da', 'fi', 'no', 'ko',
];
GhostFTP_i18n_assert(I18n::DEFAULT_LANGUAGE === 'en', 'English must remain the default language.');
GhostFTP_i18n_assert(I18n::supportedCodes() === $expected, 'Web supported-language registry drifted from the canonical 24-language order.');
GhostFTP_i18n_assert(I18n::normalize('zh-Hans') === 'zh', 'Simplified Chinese alias did not normalize to zh.');
GhostFTP_i18n_assert(I18n::normalize('zh_CN') === 'zh', 'Simplified Chinese underscore alias did not normalize to zh.');
GhostFTP_i18n_assert(I18n::normalize('nb-NO') === 'no', 'Norwegian Bokmål alias did not normalize to no.');
GhostFTP_i18n_assert(I18n::normalize('nn_NO') === 'no', 'Norwegian Nynorsk alias did not normalize to no.');
GhostFTP_i18n_assert(I18n::normalize('de-AT') === 'de', 'Regional German did not normalize to de.');
GhostFTP_i18n_assert(I18n::normalize('unsupported-locale') === 'en', 'Unknown locale did not fail safely to English.');

$english = I18n::catalog('en');
GhostFTP_i18n_assert(count($english) >= 12, 'English Web core catalog is unexpectedly small.');
GhostFTP_i18n_assert(($english['app.name'] ?? '') === 'Ghost FTP', 'Public product brand drifted in the Web catalog.');
$englishKeys = array_keys($english);

foreach ($expected as $code) {
    $catalog = I18n::catalog($code);
    GhostFTP_i18n_assert(array_keys($catalog) === $englishKeys, "Catalog key order/set drifted for {$code}.");
    foreach ($catalog as $key => $value) {
        GhostFTP_i18n_assert(is_string($value) && trim($value) !== '', "Empty value for {$code}:{$key}.");
    }
    if ($code !== 'en') {
        $translated = 0;
        foreach ($english as $key => $value) {
            if (($catalog[$key] ?? $value) !== $value) {
                $translated++;
            }
        }
        GhostFTP_i18n_assert($translated >= 8, "Locale {$code} translates only {$translated} Web core strings.");
    }
}

$languages = I18n::languages();
GhostFTP_i18n_assert(count($languages) === 24, 'Language metadata count must be 24.');
foreach ($languages as $language) {
    GhostFTP_i18n_assert(isset($language['code'], $language['englishName'], $language['nativeName']), 'Language metadata entry is incomplete.');
}

$preferenceSource = (string)file_get_contents(__DIR__ . '/../app/Storage/PreferenceStore.php');
GhostFTP_i18n_assert(str_contains($preferenceSource, "['language'] = I18n::normalize"), 'PreferenceStore does not sanitize the persisted language through I18n.');
GhostFTP_i18n_assert(str_contains($preferenceSource, "I18n::DEFAULT_LANGUAGE"), 'PreferenceStore does not use the English fallback.');

fwrite(STDOUT, "WEB_I18N_TESTS=PASS\nSUPPORTED_LANGUAGE_COUNT=24\nPRIMARY_LANGUAGE=en\n");
