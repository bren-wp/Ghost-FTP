const csrf = document.querySelector('meta[name="csrf-token"]')?.content || '';
const apiUrl = document.querySelector('meta[name="api-url"]')?.content || 'api';
const loginUrl = document.querySelector('meta[name="login-url"]')?.content || 'login';

async function parseJsonResponse(response) {
    try {
        return await response.json();
    } catch {
        throw new Error(`Neispravan odgovor servera (HTTP ${response.status}).`);
    }
}

function handleAuthFailure(status) {
    if (status === 401) {
        window.location.href = loginUrl;
        return true;
    }
    return false;
}

export async function api(action, data = {}, { signal } = {}) {
    const body = new FormData();
    body.append('action', action);
    Object.entries(data).forEach(([key, value]) => {
        if (value === undefined || value === null) return;
        if (Array.isArray(value)) {
            value.forEach((item) => body.append(`${key}[]`, item));
        } else {
            body.append(key, String(value));
        }
    });

    const response = await fetch(apiUrl, {
        method: 'POST',
        headers: { 'X-CSRF-Token': csrf },
        body,
        signal,
        credentials: 'same-origin',
        cache: 'no-store',
    });
    const payload = await parseJsonResponse(response);
    if (!response.ok || !payload.ok) {
        handleAuthFailure(response.status);
        throw new Error(payload.error || `Zahtjev nije uspio (HTTP ${response.status}).`);
    }
    return payload;
}

export function uploadRequest({ profileId, path, file, relativePath = '', conflict = 'overwrite', onProgress }) {
    const xhr = new XMLHttpRequest();
    const promise = new Promise((resolve, reject) => {
        const body = new FormData();
        body.append('action', 'upload');
        body.append('profile_id', profileId);
        body.append('path', path);
        body.append('conflict', conflict);
        body.append('files[]', file, file.name);
        body.append('relative_paths[]', relativePath);

        xhr.open('POST', apiUrl);
        xhr.setRequestHeader('X-CSRF-Token', csrf);
        xhr.responseType = 'json';
        xhr.upload.addEventListener('progress', (event) => {
            if (event.lengthComputable && onProgress) {
                onProgress(Math.round((event.loaded / event.total) * 100), event.loaded, event.total);
            }
        });
        xhr.addEventListener('load', () => {
            const data = xhr.response;
            if (handleAuthFailure(xhr.status)) {
                reject(new Error('Sesija je istekla.'));
                return;
            }
            if (xhr.status < 200 || xhr.status >= 300 || !data?.ok) {
                reject(new Error(data?.error || `Upload nije uspio (HTTP ${xhr.status}).`));
                return;
            }
            resolve(data);
        });
        xhr.addEventListener('error', () => reject(new Error('Mrežna greška tijekom uploada.')));
        xhr.addEventListener('abort', () => reject(new DOMException('Upload je otkazan.', 'AbortError')));
        xhr.send(body);
    });
    return { xhr, promise };
}

export function submitDownload(endpoint, fields) {
    const form = document.createElement('form');
    form.method = 'POST';
    form.action = endpoint;
    form.hidden = true;
    const values = { csrf, ...fields };
    Object.entries(values).forEach(([key, value]) => {
        const input = document.createElement('input');
        input.type = 'hidden';
        input.name = key;
        input.value = String(value);
        form.append(input);
    });
    document.body.append(form);
    form.submit();
    setTimeout(() => form.remove(), 1000);
}
