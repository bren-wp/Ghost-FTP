import { useEffect, useRef, useState, type ReactNode } from "react";
import type { DirEntry, SessionId } from "../types";
import { imageMime } from "../lib/fileIcons";

interface Props {
  sessionId: SessionId;
  entry: DirEntry;
  /** Fetch raw image bytes (the adapter's `thumbnail`). */
  load: (
    sessionId: SessionId,
    entry: DirEntry,
    signal?: AbortSignal
  ) => Promise<Blob | null>;
  /** Square box size in CSS px. */
  size?: number;
  /** Icon shown until a preview loads, or if none is available. */
  fallback: ReactNode;
  className?: string;
}

// A lazy image preview for the grid view. It only fetches once scrolled near the
// viewport (IntersectionObserver), downscales the decoded bitmap to a small
// square via canvas, and owns the resulting object URL's lifecycle (revoked on
// unmount / entry change). Any failure — no bytes, unsupported codec, decode
// error — falls back to the type icon, so it never shows a broken image.
export function Thumbnail({
  sessionId,
  entry,
  load,
  size = 72,
  fallback,
  className,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [url, setUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setUrl(null);
    setFailed(false);
    const el = ref.current;
    if (!el) return;

    let cancelled = false;
    let objectUrl: string | null = null;
    // Aborts a remote fetch the moment this row scrolls back out / unmounts, so
    // a fast scroll never spends network on rows nobody ends up looking at.
    const ac = new AbortController();

    const run = async () => {
      try {
        const blob = await load(sessionId, entry, ac.signal);
        if (cancelled) return;
        if (!blob) {
          setFailed(true);
          return;
        }
        const out = await downscale(blob, entry.name, size);
        if (cancelled) {
          if (out) URL.revokeObjectURL(out);
          return;
        }
        if (!out) {
          setFailed(true);
          return;
        }
        objectUrl = out;
        setUrl(out);
      } catch {
        if (!cancelled) setFailed(true);
      }
    };

    const io = new IntersectionObserver(
      (entries, obs) => {
        if (entries.some((e) => e.isIntersecting)) {
          obs.disconnect();
          run();
        }
      },
      { rootMargin: "200px" }
    );
    io.observe(el);

    return () => {
      cancelled = true;
      ac.abort();
      io.disconnect();
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [sessionId, entry.path, entry.name, size, load]);

  return (
    <div
      ref={ref}
      className={className}
      style={{ width: size, height: size }}
    >
      {url && !failed ? (
        <img
          src={url}
          alt={entry.name}
          loading="lazy"
          draggable={false}
          className="h-full w-full rounded-md object-cover shadow-elev-1"
          onError={() => setFailed(true)}
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center">
          {fallback}
        </div>
      )}
    </div>
  );
}

// Decode → scale to fit a `size`-square (×DPR) → re-encode a small PNG object URL.
async function downscale(
  blob: Blob,
  name: string,
  size: number
): Promise<string | null> {
  // SVG is vector — canvas decode of SVG blobs is inconsistent across webviews,
  // so just hand back the blob's own object URL and let the <img> scale it.
  if (imageMime(name) === "image/svg+xml") {
    return URL.createObjectURL(blob);
  }

  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(blob);
  } catch {
    // Unsupported codec in this webview (some HEIC/TIFF) — fall back to icon.
    return null;
  }

  const dpr = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
  const target = Math.round(size * dpr);
  const scale = Math.min(1, target / Math.max(bitmap.width, bitmap.height));
  const w = Math.max(1, Math.round(bitmap.width * scale));
  const h = Math.max(1, Math.round(bitmap.height * scale));

  const canvas = document.createElement("canvas");
  canvas.width = w;
  canvas.height = h;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    bitmap.close();
    return null;
  }
  ctx.drawImage(bitmap, 0, 0, w, h);
  bitmap.close();

  const out = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob((b) => resolve(b), "image/png")
  );
  return out ? URL.createObjectURL(out) : null;
}
