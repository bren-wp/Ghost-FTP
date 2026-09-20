// Shared formatting helpers. Consolidated here so panes, footers, dialogs and
// the transfer queue all render sizes / dates / modes identically.

export function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

// Unix seconds -> "2024-05-21 14:32". Empty string when unknown.
export function fmtMtime(secs?: number): string {
  if (!secs) return "";
  const d = new Date(secs * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(
    d.getHours()
  )}:${pad(d.getMinutes())}`;
}

// POSIX mode bits -> "rwxr-xr-x". Empty when the backend doesn't expose a mode.
export function formatMode(mode?: number): string {
  if (mode == null) return "";
  const bits = mode & 0o777;
  const part = (n: number) =>
    `${n & 4 ? "r" : "-"}${n & 2 ? "w" : "-"}${n & 1 ? "x" : "-"}`;
  return part((bits >> 6) & 7) + part((bits >> 3) & 7) + part(bits & 7);
}

// Compact relative age, e.g. "12s", "4m", "3h", "2d".
export function relTime(ms: number): string {
  const s = Math.max(0, Math.floor((Date.now() - ms) / 1000));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d`;
}

// Last path segment regardless of separator style.
export function baseName(p: string): string {
  const parts = p.split(/[/\\]/).filter(Boolean);
  return parts.length ? parts[parts.length - 1] : p;
}

// A 1-2 letter monogram for a server name, e.g. "prod-db" -> "PD",
// "web 01" -> "W0", "x" -> "X". Splits on whitespace/-/_ and is code-point safe
// (Array.from) so emoji and non-ASCII names never render a broken half-glyph.
// Falls back to "?" so a bubble is never blank.
export function monogram(name: string): string {
  const words = name.trim().split(/[\s\-_]+/).filter(Boolean);
  if (words.length === 0) return "?";
  const first = Array.from(words[0]);
  if (words.length >= 2) {
    return (first[0] + Array.from(words[1])[0]).toUpperCase();
  }
  return first.slice(0, 2).join("").toUpperCase() || "?";
}
