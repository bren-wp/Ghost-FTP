export const $ = (selector, root = document) => root.querySelector(selector);
export const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

export function escapeHtml(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
}

export function formatBytes(bytes) {
    let value = Number(bytes) || 0;
    if (value < 1024) return `${value} B`;
    const units = ['KB', 'MB', 'GB', 'TB'];
    let unit = -1;
    do {
        value /= 1024;
        unit += 1;
    } while (value >= 1024 && unit < units.length - 1);
    return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unit]}`;
}

export function formatDate(value) {
    if (!value) return '—';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return date.toLocaleString('hr-HR', { dateStyle: 'short', timeStyle: 'short' });
}

export function joinPath(parent, name) {
    const base = parent === '/' ? '' : parent.replace(/\/$/, '');
    return `${base}/${String(name).replace(/^\/+/, '')}` || '/';
}

export function parentPath(path) {
    if (!path || path === '/') return '/';
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    return `/${parts.join('/')}` || '/';
}

export function basename(path) {
    return path.split('/').filter(Boolean).pop() || '/';
}

export function isZip(path) {
    return /\.zip$/i.test(path);
}

export function fileIcon(item) {
    if (item.type === 'dir') return '📁';
    const ext = item.name.split('.').pop()?.toLowerCase();
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'ico'].includes(ext)) return '🖼';
    if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return '🗜';
    if (['php', 'js', 'ts', 'css', 'html', 'htm', 'json', 'xml', 'md', 'txt', 'yml', 'yaml'].includes(ext)) return '⌘';
    return '📄';
}

export function naturalCompare(a, b) {
    return String(a).localeCompare(String(b), 'hr', { numeric: true, sensitivity: 'base' });
}
