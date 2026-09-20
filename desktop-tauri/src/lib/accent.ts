// Accent colors. A chosen accent overrides the four --accent* tokens on :root
// (inline styles win over the [data-theme] block), so it layers on top of any
// light/dark mode or named palette. Derived shades are computed so white text on
// the "strong" button fill always clears WCAG AA (4.5:1).

export interface AccentColor {
  name: string;
  hex: string;
}

// Named accents spanning the wheel, in spectral order (reds → oranges, then a
// few neutrals) so the swatch grid reads like a palette.
export const ACCENTS: AccentColor[] = [
  // reds
  { name: "Red", hex: "#ef4444" },
  { name: "Crimson", hex: "#dc2626" },
  { name: "Brick", hex: "#b91c1c" },
  { name: "Coral", hex: "#ff6b6b" },
  { name: "Salmon", hex: "#fb7185" },
  // roses / pinks
  { name: "Rose", hex: "#f43f5e" },
  { name: "Wine", hex: "#9f1239" },
  { name: "Pink", hex: "#ec4899" },
  { name: "Hot Pink", hex: "#f472b6" },
  // magentas / purples
  { name: "Fuchsia", hex: "#d946ef" },
  { name: "Magenta", hex: "#c026d3" },
  { name: "Mauve", hex: "#c084fc" },
  { name: "Purple", hex: "#a855f7" },
  { name: "Grape", hex: "#9333ea" },
  { name: "Plum", hex: "#7e22ce" },
  { name: "Violet", hex: "#8b5cf6" },
  // indigos
  { name: "Royal", hex: "#6366f1" },
  { name: "Indigo", hex: "#4f46e5" },
  { name: "Periwinkle", hex: "#818cf8" },
  { name: "Lavender", hex: "#a78bfa" },
  // blues
  { name: "Blue", hex: "#3b82f6" },
  { name: "Cobalt", hex: "#2563eb" },
  { name: "Cerulean", hex: "#0284c7" },
  { name: "Azure", hex: "#38bdf8" },
  { name: "Sky", hex: "#0ea5e9" },
  // cyans / teals
  { name: "Ocean", hex: "#0891b2" },
  { name: "Cyan", hex: "#06b6d4" },
  { name: "Turquoise", hex: "#22d3ee" },
  { name: "Teal", hex: "#14b8a6" },
  { name: "Aqua", hex: "#2dd4bf" },
  // greens
  { name: "Spring", hex: "#34d399" },
  { name: "Emerald", hex: "#10b981" },
  { name: "Jade", hex: "#059669" },
  { name: "Green", hex: "#22c55e" },
  { name: "Mint", hex: "#4ade80" },
  { name: "Forest", hex: "#16a34a" },
  // yellow-greens
  { name: "Olive", hex: "#65a30d" },
  { name: "Lime", hex: "#84cc16" },
  { name: "Chartreuse", hex: "#a3e635" },
  // yellows
  { name: "Lemon", hex: "#facc15" },
  { name: "Yellow", hex: "#eab308" },
  { name: "Gold", hex: "#ca8a04" },
  // oranges
  { name: "Amber", hex: "#f59e0b" },
  { name: "Tangerine", hex: "#fb923c" },
  { name: "Orange", hex: "#f97316" },
  { name: "Cinnamon", hex: "#d97706" },
  // neutrals
  { name: "Slate", hex: "#64748b" },
  { name: "Steel", hex: "#475569" },
  { name: "Stone", hex: "#78716c" },
];

type RGB = [number, number, number];

function hexToRgb(hex: string): RGB {
  const h = hex.replace("#", "");
  return [
    parseInt(h.slice(0, 2), 16),
    parseInt(h.slice(2, 4), 16),
    parseInt(h.slice(4, 6), 16),
  ];
}

function channelLin(c: number): number {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
}

function relLuminance([r, g, b]: RGB): number {
  return 0.2126 * channelLin(r) + 0.7152 * channelLin(g) + 0.0722 * channelLin(b);
}

/** Contrast ratio of white text (#fff) against the given background. */
function contrastWithWhite(rgb: RGB): number {
  return 1.05 / (relLuminance(rgb) + 0.05);
}

function darken([r, g, b]: RGB, f: number): RGB {
  return [r * (1 - f), g * (1 - f), b * (1 - f)];
}

function triple([r, g, b]: RGB): string {
  return `${Math.round(r)} ${Math.round(g)} ${Math.round(b)}`;
}

/** Ghost FTP stores colors as space-separated RGB triples (used via rgb(var(--x))). */
export function deriveAccent(hex: string): {
  accent: string;
  hover: string;
  strong: string;
  glow: string;
} {
  const base = hexToRgb(hex);
  const hover = darken(base, 0.12);
  let strong: RGB = base;
  for (let i = 0; i < 24 && contrastWithWhite(strong) < 4.5; i++) {
    strong = darken(strong, 0.1);
  }
  return {
    accent: triple(base),
    hover: triple(hover),
    strong: triple(strong),
    glow: triple(base),
  };
}

const ACCENT_VARS = [
  "--accent",
  "--accent-hover",
  "--accent-strong",
  "--accent-glow",
] as const;

/** Apply an accent override (or clear it, falling back to the theme's accent). */
export function applyAccent(hex: string | null | undefined): void {
  const el = document.documentElement;
  if (!hex) {
    ACCENT_VARS.forEach((v) => el.style.removeProperty(v));
    return;
  }
  const d = deriveAccent(hex);
  el.style.setProperty("--accent", d.accent);
  el.style.setProperty("--accent-hover", d.hover);
  el.style.setProperty("--accent-strong", d.strong);
  el.style.setProperty("--accent-glow", d.glow);
}
