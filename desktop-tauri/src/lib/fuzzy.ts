// Tiny subsequence fuzzy matcher with contiguous / word-boundary bonuses and
// match ranges for highlighting. Good enough for a command palette of dozens
// of entries — no dependency, no index.

export interface FuzzyResult {
  matched: boolean;
  score: number;
  ranges: [number, number][]; // [start, end) char ranges in `text` that matched
}

const BOUNDARY = /[\s/_\-.:\\]/;

export function fuzzyMatch(query: string, text: string): FuzzyResult {
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  if (q === "") return { matched: true, score: 0, ranges: [] };

  let qi = 0;
  let score = 0;
  let prev = -2;
  const idxs: number[] = [];

  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) {
      idxs.push(ti);
      score += ti === prev + 1 ? 6 : 1; // contiguous run bonus
      if (ti === 0 || BOUNDARY.test(t[ti - 1])) score += 8; // word start bonus
      prev = ti;
      qi++;
    }
  }

  if (qi < q.length) return { matched: false, score: 0, ranges: [] };

  score -= idxs[0]; // prefer earlier matches

  const ranges: [number, number][] = [];
  for (const i of idxs) {
    const last = ranges[ranges.length - 1];
    if (last && i === last[1]) last[1] = i + 1;
    else ranges.push([i, i + 1]);
  }
  return { matched: true, score, ranges };
}
