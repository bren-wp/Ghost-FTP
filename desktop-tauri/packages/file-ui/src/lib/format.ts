// Formatting helpers used by the file list. Kept local to the package so it has
// no dependency on a host app. (Host apps that already have these can ignore
// them — the package only uses its own copies internally.)

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
