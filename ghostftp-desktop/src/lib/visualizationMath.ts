// Pure graph calculations. No DOM, backend writes, or fabricated telemetry.
export type BandwidthSample = { upload: number; download: number };
export const BANDWIDTH_WINDOW = 48;
export const BANDWIDTH_WIDTH = 600;
export const BANDWIDTH_TOP = 14;
export const BANDWIDTH_BOTTOM = 106;

function finiteRate(value: number): number {
  return Number.isFinite(value) && value > 0 ? value : 0;
}

export function prepareBandwidthGraph(history: readonly BandwidthSample[]) {
  const data = history.slice(-BANDWIDTH_WINDOW).map((sample) => ({
    upload: finiteRate(sample.upload),
    download: finiteRate(sample.download),
  }));
  const count = data.length;
  const latest = count ? data[count - 1] : null;
  const peak = data.reduce((largest, item) => Math.max(largest, item.upload, item.download), 0);
  const scale = Math.max(1, peak);
  // Incremental means stay within the non-negative sample range and avoid
  // overflow from adding large finite readings. Do not divide each tiny
  // sample up front, which can underflow an otherwise representable mean.
  const average = { upload: 0, download: 0 };
  for (let index = 0; index < count; index += 1) {
    const sample = data[index];
    const weight = index + 1;
    average.upload += (sample.upload - average.upload) / weight;
    average.download += (sample.download - average.download) / weight;
  }

  const plotY = (value: number) =>
    BANDWIDTH_BOTTOM - (value / scale) * (BANDWIDTH_BOTTOM - BANDWIDTH_TOP);
  // A single genuine sample draws a horizontal line; zero readings are not invented.
  const plotted = count === 1 ? [data[0], data[0]] : data;
  const points = (key: keyof BandwidthSample) =>
    plotted
      .map((sample, index) => {
        const x = plotted.length <= 1 ? 0 : (index / (plotted.length - 1)) * BANDWIDTH_WIDTH;
        return x.toFixed(1) + "," + plotY(sample[key]).toFixed(1);
      })
      .join(" ");

  return {
    hasData: count > 0,
    count,
    latest,
    average,
    peak,
    uploadPoints: points("upload"),
    downloadPoints: points("download"),
    lastUploadY: latest ? plotY(latest.upload) : BANDWIDTH_BOTTOM,
    lastDownloadY: latest ? plotY(latest.download) : BANDWIDTH_BOTTOM,
  };
}

export function safeDiskPercent(part: number, total: number): number {
  if (!Number.isFinite(part) || !Number.isFinite(total) || part <= 0 || total <= 0) return 0;
  return Math.max(0, Math.min(100, (part / total) * 100));
}

/** Rank a display copy by descending valid size, never mutate the scan tree. */
export function rankDiskItems<T extends { size: number }>(items: readonly T[]): T[] {
  const valid = (size: number) => Number.isFinite(size) && size >= 0;
  return items
    .map((node, index) => ({ node, index }))
    .sort((a, b) => {
      const aValid = valid(a.node.size);
      const bValid = valid(b.node.size);
      if (aValid !== bValid) return aValid ? -1 : 1;
      if (!aValid) return a.index - b.index;
      return b.node.size - a.node.size || a.index - b.index;
    })
    .map(({ node }) => node);
}
