import { useId } from "react";

/** Exact shape and geometry of the approved premium Ghost FTP brand icon. */
export function GhostMark({ size = 28, className = "" }: { size?: number; className?: string }) {
  const id = useId().replace(/:/g, "");
  return (
    <svg width={size} height={size} viewBox="0 0 512 512" className={className}
      role="img" aria-label="Ghost FTP">
      <defs>
        <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#173D65" /><stop offset=".56" stopColor="#0A1B32" />
          <stop offset="1" stopColor="#050D19" />
        </linearGradient>
        <linearGradient id={`${id}-ghost`} x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#FFFFFF" /><stop offset=".48" stopColor="#C7F0FF" />
          <stop offset="1" stopColor="#38ABFF" />
        </linearGradient>
        <linearGradient id={`${id}-stripe`} x1="0" y1="0" x2="1" y2="0">
          <stop stopColor="#38ABFF" /><stop offset="1" stopColor="#7EDEFF" />
        </linearGradient>
        <radialGradient id={`${id}-glow`}>
          <stop stopColor="#38ABFF" stopOpacity=".38" />
          <stop offset="1" stopColor="#38ABFF" stopOpacity="0" />
        </radialGradient>
      </defs>
      <rect x="6" y="6" width="500" height="500" rx="116" fill={`url(#${id}-bg)`} />
      <rect x="12" y="12" width="488" height="488" rx="111" fill="none" stroke="#4EBBFC"
        strokeOpacity=".35" strokeWidth="3" />
      <circle cx="270" cy="235" r="207" fill={`url(#${id}-glow)`} />
      <g transform="translate(120 115) scale(1.39)">
        <path d="M24 155c16-4 28-17 30-33l7-57C65 24 78 8 96 8s31 16 35 57l7 57c2 16 14 29 30 33-9 9-22 15-35 11-10-3-19-11-24-21-6 13-16 21-29 21s-23-8-29-21c-5 10-14 18-24 21-13 4-26-2-35-11Z"
          transform="translate(16 12) scale(.83)" fill={`url(#${id}-ghost)`} />
        <ellipse cx="82" cy="83" rx="8" ry="13" fill="#0A2B51" />
        <ellipse cx="111" cy="83" rx="8" ry="13" fill="#0A2B51" />
      </g>
      <path d="M105 403H408" stroke={`url(#${id}-stripe)`} opacity=".5"
        strokeWidth="4" strokeLinecap="round" />
    </svg>
  );
}

export function GhostWordmark({ compact = false }: { compact?: boolean }) {
  return (
    <div className="ghost-wordmark" aria-label="Ghost FTP">
      <GhostMark size={compact ? 34 : 38} />
      <span className={compact ? "text-[20px]" : "text-[21px]"}>Ghost <strong>FTP</strong></span>
    </div>
  );
}
