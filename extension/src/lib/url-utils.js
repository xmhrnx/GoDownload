/**
 * Small, dependency-free helpers for URLs and file names.
 */

/** @returns {URL | null} */
export function parseUrl(url) {
  try {
    return new URL(url);
  } catch {
    return null;
  }
}

export function hostnameOf(url) {
  return parseUrl(url)?.hostname.toLowerCase() ?? '';
}

/** Last path segment of a path or URL, without query/fragment. */
export function basename(pathOrUrl) {
  const s = String(pathOrUrl ?? '');
  const u = /^[a-z][a-z0-9+.-]*:\/\//i.test(s) ? parseUrl(s) : null;
  const path = u ? u.pathname : s.split(/[?#]/)[0];
  const name = path.split(/[\\/]/).filter(Boolean).pop() ?? '';
  try {
    return decodeURIComponent(name);
  } catch {
    return name;
  }
}

/** Lowercase file extension without the dot ("" when none). */
export function extensionOf(name) {
  const m = /\.([a-z0-9]{1,16})$/i.exec(String(name ?? ''));
  return m ? m[1].toLowerCase() : '';
}

/** Lowercase extension of a URL's path. */
export function urlExtension(url) {
  return extensionOf(basename(url));
}

/** Makes a string safe to use as a file name on Windows, macOS and Linux. */
export function sanitizeFilename(name, maxLength = 150) {
  let s = String(name ?? '')
    // eslint-disable-next-line no-control-regex
    .replace(/[\u0000-\u001f\u007f<>:"/\\|?*]+/g, ' ')
    .replace(/\s+/g, ' ')
    .replace(/ (?=\.[^.\s]+$)/, '')
    .trim()
    .replace(/^[.\s]+|[.\s]+$/g, '');
  if (/^(con|prn|aux|nul|com\d|lpt\d)(\.|$)/i.test(s)) s = `_${s}`;
  return s.slice(0, maxLength).trim();
}

/** Hostname equals `site` or is a subdomain of it. */
export function hostMatches(host, site) {
  return host === site || host.endsWith(`.${site}`);
}
