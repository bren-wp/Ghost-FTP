import { cn } from "@/lib/cn";

export type BadgeVariant =
  | "default"
  | "accent"
  | "success"
  | "danger"
  | "warning"
  | "muted";

const VARIANTS: Record<BadgeVariant, string> = {
  default: "bg-bg-hover text-text-muted",
  accent: "bg-accent/15 text-accent",
  success: "bg-success/15 text-success",
  danger: "bg-danger/15 text-danger",
  warning: "bg-warning/15 text-warning",
  muted: "bg-bg-subtle text-text-dim",
};

export function Badge({
  children,
  variant = "default",
  className,
}: {
  children: React.ReactNode;
  variant?: BadgeVariant;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium leading-none",
        VARIANTS[variant],
        className
      )}
    >
      {children}
    </span>
  );
}
