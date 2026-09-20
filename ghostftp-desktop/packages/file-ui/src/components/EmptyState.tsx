import { type ReactNode } from "react";

// Designed empty / placeholder state so dead pane real estate never looks
// broken. Used for no-connection, empty folders, and no-filter-matches.
export function EmptyState({
  icon,
  title,
  hint,
  action,
}: {
  icon?: ReactNode;
  title: ReactNode;
  hint?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center px-3 py-12 text-center anim-fade">
      {icon && (
        <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-bg-subtle text-text-dim">
          {icon}
        </div>
      )}
      <div className="text-xs text-text-muted">{title}</div>
      {hint && <div className="mt-1 max-w-[15rem] text-[11px] text-text-dim">{hint}</div>}
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}
