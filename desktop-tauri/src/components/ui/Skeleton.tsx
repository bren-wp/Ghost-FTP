import { cn } from "@/lib/cn";

// A single shimmering placeholder block. Animation lives in styles.css
// (@keyframes ghostftp-shimmer) so it tints with the active theme.
export function Skeleton({
  className,
  style,
}: {
  className?: string;
  style?: React.CSSProperties;
}) {
  return (
    <div
      className={cn("ghostftp-shimmer rounded bg-bg-hover", className)}
      style={style}
    />
  );
}

const WIDTHS = ["55%", "70%", "42%", "63%", "48%", "75%", "58%", "38%", "66%", "50%"];

// Placeholder list shaped like FilePane rows, shown while a directory loads so
// a latent SFTP / S3 listing never leaves the pane blank.
export function FileListSkeleton({ rows = 12 }: { rows?: number }) {
  return (
    <div className="px-3 py-1 anim-fade">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-2 py-[5px]">
          <Skeleton className="h-3.5 w-3.5" />
          <Skeleton
            className="h-3"
            style={{ width: WIDTHS[i % WIDTHS.length] }}
          />
          <div className="flex-1" />
          <Skeleton className="h-3 w-12" />
        </div>
      ))}
    </div>
  );
}
