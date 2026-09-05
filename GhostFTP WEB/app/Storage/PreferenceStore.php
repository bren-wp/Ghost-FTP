<?php
declare(strict_types=1);

namespace GhostFTP\Storage;

use GhostFTP\I18n;
use GhostFTP\Remote\PathGuard;

// PreferenceStore is also exercised by standalone CLI recovery tools that do not
// bootstrap the application autoloader. Keep this dependency explicit and local
// so language normalization remains available in those fail-closed code paths.
if (!class_exists(I18n::class, false)) {
    require_once dirname(__DIR__) . '/I18n.php';
}

final class PreferenceStore
{
    private JsonStore $store;

    public function __construct(string $userId)
    {
        UserWorkspace::ensure($userId);
        // Preferences contain explicit user state such as favorites and recently visited
        // remote paths. Keep preferences.json.bak for manual recovery only; runtime reads
        // must not silently restore data the user already removed from the current state.
        $this->store = new JsonStore(UserWorkspace::file($userId, 'preferences.json'), false);
    }

    public function favorites(string $profileId): array
    {
        $data = $this->store->read();
        $items = $data['favorites'][$profileId] ?? [];
        return is_array($items) ? array_values(array_filter($items, 'is_string')) : [];
    }

    public function toggleFavorite(string $profileId, string $path): array
    {
        $path = PathGuard::normalizeRelative($path);
        $result = [];
        $this->store->update(function (array $data) use ($profileId, $path, &$result): array {
            $items = is_array($data['favorites'][$profileId] ?? null) ? array_values($data['favorites'][$profileId]) : [];
            $idx = array_search($path, $items, true);
            if ($idx === false) {
                $items[] = $path;
            } else {
                array_splice($items, (int)$idx, 1);
            }
            $items = array_values(array_unique(array_slice(array_filter($items, 'is_string'), 0, 100)));
            $data['favorites'][$profileId] = $items;
            $result = $items;
            return $data;
        });
        return $result;
    }

    public function clientState(): array
    {
        $data = $this->store->read();
        $current = is_array($data['client_state'] ?? null) ? $data['client_state'] : [];
        // Always return the same object-shaped schema, including for a brand-new account.
        return $this->sanitizeClientState($current);
    }

    public function saveClientState(array $input): array
    {
        $result = [];
        $this->store->update(function (array $data) use ($input, &$result): array {
            $current = is_array($data['client_state'] ?? null) ? $data['client_state'] : [];
            if (!array_key_exists('language', $input)) {
                $input['language'] = (string)($current['language'] ?? I18n::DEFAULT_LANGUAGE);
            }
            $result = $this->sanitizeClientState($input);
            $data['client_state'] = $result;
            $data['updated_at'] = gmdate('c');
            return $data;
        });
        return $result;
    }

    private function sanitizeClientState(array $input): array
    {
        $out = [];
        $out['language'] = I18n::normalize((string)($input['language'] ?? I18n::DEFAULT_LANGUAGE));
        $out['lastProfile'] = preg_replace('/[^a-zA-Z0-9_-]/', '', (string)($input['lastProfile'] ?? '')) ?? '';
        $out['showHidden'] = !array_key_exists('showHidden', $input) || (bool)$input['showHidden'];
        $out['compactRows'] = !empty($input['compactRows']);
        $out['uploadConflict'] = in_array(($input['uploadConflict'] ?? ''), ['overwrite', 'rename', 'skip'], true) ? $input['uploadConflict'] : 'rename';

        $sort = is_array($input['sort'] ?? null) ? $input['sort'] : [];
        $key = in_array(($sort['key'] ?? ''), ['name', 'size', 'modified'], true) ? $sort['key'] : 'name';
        $direction = (int)($sort['direction'] ?? 1) < 0 ? -1 : 1;
        $out['sort'] = ['key' => $key, 'direction' => $direction];

        foreach (['lastPaths' => 50, 'recentPaths' => 50] as $field => $limitProfiles) {
            $rows = is_array($input[$field] ?? null) ? $input[$field] : [];
            $safe = [];
            foreach (array_slice($rows, 0, $limitProfiles, true) as $profileId => $value) {
                $profileId = preg_replace('/[^a-zA-Z0-9_-]/', '', (string)$profileId) ?? '';
                if ($profileId === '') {
                    continue;
                }
                if ($field === 'lastPaths') {
                    if (is_string($value)) {
                        $safe[$profileId] = PathGuard::normalizeRelative($value);
                    }
                } elseif (is_array($value)) {
                    $paths = [];
                    foreach (array_slice($value, 0, 12) as $path) {
                        if (is_string($path)) {
                            $paths[] = PathGuard::normalizeRelative($path);
                        }
                    }
                    $safe[$profileId] = array_values(array_unique($paths));
                }
            }
            $out[$field] = $safe;
        }
        return $out;
    }
}
