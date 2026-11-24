/**
 * Display formatting helpers shared by the extension pages.
 */

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

/** 1536 -> "1.5 KB"; negative/unknown -> "". */
export function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes < 0) return '';
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const digits = value >= 100 || unit === 0 ? 0 : 1;
  return `${value.toFixed(digits)} ${UNITS[unit]}`;
}

export const KIND_LABELS = Object.freeze({ hls: 'HLS', dash: 'DASH', video: 'Video', audio: 'Audio' });

export function hostOf(url) {
  try {
    return new URL(url).hostname.replace(/^www\./, '');
  } catch {
    return '';
  }
}

/** Swaps the brand <img> for the gradient "G" mark when the icon file is missing. */
export function installLogoFallback(root = document) {
  const img = root.querySelector('.brand-logo');
  const mark = root.querySelector('.brand-mark');
  if (!img || !mark) return;
  const fallback = () => {
    img.hidden = true;
    mark.hidden = false;
  };
  if (img.complete && img.naturalWidth === 0) fallback();
  else img.addEventListener('error', fallback, { once: true });
}
