// Ghost FTP by Brendigo
import { ChevronDown, ChevronUp } from "lucide-react";

import type { SiteSortField } from "@/lib/siteViewModel";
export type { SiteSortField } from "@/lib/siteViewModel";

export function FilterChip({ active, label, count, onClick }: { active: boolean; label: string; count: number; onClick: () => void }) {
  return <button type="button" className={`ghost-site-filter-chip ${active ? "active" : ""}`} aria-pressed={active} onClick={onClick}><span>{label}</span><b>{count}</b></button>;
}

export function SiteSortHeader({
  label,
  field,
  activeField,
  direction,
  onSort,
}: {
  label: string;
  field: SiteSortField;
  activeField: SiteSortField;
  direction: "asc" | "desc";
  onSort: (field: SiteSortField) => void;
}) {
  const active = activeField === field;
  return (
    <button
      type="button"
      className={`flex min-w-0 items-center gap-1 text-left hover:text-white focus-visible:outline-none focus-visible:text-white ${active ? "text-text" : "text-text-dim"}`}
      aria-label={`Sort by ${label}${active ? `, ${direction === "asc" ? "ascending" : "descending"}` : ""}`}
      aria-pressed={active}
      onClick={() => onSort(field)}
    >
      <span className="truncate">{label}</span>
      {active ? direction === "asc" ? <ChevronUp size={12}/> : <ChevronDown size={12}/> : null}
    </button>
  );
}
