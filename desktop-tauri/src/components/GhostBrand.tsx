import { useId } from "react";

export function GhostMark({ size = 28, className = "" }: { size?: number; className?: string }) {
  const gid = useId().replace(/:/g, "");
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className={className}
      role="img"
      aria-label="Ghost FTP"
    >
      <defs>
        <linearGradient id={`${gid}-body`} x1="14" y1="8" x2="50" y2="57" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F4FDFF" />
          <stop offset="0.45" stopColor="#BEEBFF" />
          <stop offset="1" stopColor="#42AEFF" />
        </linearGradient>
        <filter id={`${gid}-glow`} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <path
        d="M10 51.5c4.2-1.2 7.3-4.6 7.8-8.7l1.7-15.2C20.6 17.2 25.4 10 32 10s11.4 7.2 12.5 17.6l1.7 15.2c.5 4.1 3.6 7.5 7.8 8.7-2.2 2.3-5.5 3.6-8.7 2.7-2.5-.7-4.8-2.7-6.1-5.2-1.4 3.2-4.1 5.4-7.2 5.4s-5.8-2.2-7.2-5.4c-1.3 2.5-3.6 4.5-6.1 5.2-3.2.9-6.5-.4-8.7-2.7Z"
        fill={`url(#${gid}-body)`}
        filter={`url(#${gid}-glow)`}
      />
      <ellipse cx="27" cy="30" rx="3.1" ry="5.1" fill="#0A2B51" />
      <ellipse cx="38.5" cy="30" rx="3.1" ry="5.1" fill="#0A2B51" />
    </svg>
  );
}

export function GhostWordmark({ compact = false }: { compact?: boolean }) {
  return (
    <div className="ghost-wordmark" aria-label="Ghost FTP">
      <GhostMark size={compact ? 25 : 34} />
      <span className={compact ? "text-[15px]" : "text-[19px]"}>Ghost <strong>FTP</strong></span>
    </div>
  );
}
