/**
 * Pure decision logic for download interception. No browser APIs here, which
 * keeps the rules easy to reason about (and to unit test).
 */
import { basename, extensionOf, hostMatches, hostnameOf, urlExtension } from './url-utils.js';

const INTERCEPTABLE_SCHEMES = new Set(['http:', 'https:', 'ftp:']);
const TORRENT_MIMES = new Set(['application/x-bittorrent']);

/** True for http/https/ftp only (never blob:, data:, file:, filesystem: ...). */
export function isInterceptableUrl(url) {
  const m = /^([a-z][a-z0-9+.-]*:)/i.exec(String(url ?? ''));
  return !!m && INTERCEPTABLE_SCHEMES.has(m[1].toLowerCase());
}

/** True when any of the given URLs is on the excluded-sites list. */
export function isExcluded(urls, excludedSites) {
  if (!excludedSites?.length) return false;
  return urls.filter(Boolean).some((url) => {
    const host = hostnameOf(url);
    return !!host && excludedSites.some((site) => hostMatches(host, site));
  });
}

/** The API "kind" for a plain link URL (context menus, link batches). */
export function kindForUrl(url, mime = '') {
  const s = String(url ?? '');
  if (/^magnet:/i.test(s)) return 'magnet';
  const ext = urlExtension(s);
  if (ext === 'torrent' || TORRENT_MIMES.has(mime)) return 'torrent';
  if (ext === 'm3u8') return 'hls';
  if (ext === 'mpd') return 'dash';
  return 'file';
}

/**
 * Decides whether a browser download should go to GoDownload.
 *
 * @param {{url: string, finalUrl?: string, referrer?: string, filename?: string,
 *          mime?: string, size?: number}} dl
 * @param {{enabled: boolean, extensions: string[], minSizeMB: number,
 *          excludedSites: string[]}} settings
 * @returns {{intercept: boolean, reason: string, kind?: string, filename?: string}}
 */
export function evaluateDownload(dl, settings) {
  if (!settings.enabled) return { intercept: false, reason: 'disabled' };
  const urls = [dl.url, dl.finalUrl].filter(Boolean);
  if (!urls.length || !urls.every(isInterceptableUrl)) return { intercept: false, reason: 'scheme' };
  // Exclusion applies to the file's host and to the page it was started from.
  if (isExcluded([...urls, dl.referrer], settings.excludedSites)) return { intercept: false, reason: 'excluded' };

  const filename = basename(dl.filename || '') || basename(dl.finalUrl || dl.url);
  const ext = extensionOf(filename) || urlExtension(dl.finalUrl || dl.url);
  const isTorrentMime = TORRENT_MIMES.has(dl.mime);
  const extMatches = (!!ext && settings.extensions.includes(ext)) ||
    (isTorrentMime && settings.extensions.includes('torrent'));

  // Unknown size (-1/undefined) can only match through the extension rule.
  const size = Number.isFinite(dl.size) ? dl.size : -1;
  const minBytes = settings.minSizeMB * 1024 * 1024;
  const sizeMatches = settings.minSizeMB > 0 && size >= 0 && size >= minBytes;

  if (!extMatches && !sizeMatches) return { intercept: false, reason: 'filter' };

  const kind = ext === 'torrent' || isTorrentMime ? 'torrent' : 'file';
  return { intercept: true, reason: extMatches ? 'extension' : 'size', kind, filename };
}
