/**
 * Builds request bodies for POST /api/v1/downloads(/batch) in exactly the
 * shape the desktop app expects, so callers can't forget a field.
 */
import { getCookieHeader } from './cookies.js';

/**
 * @typedef {'file'|'media'|'hls'|'dash'|'torrent'|'magnet'} DownloadKind
 * @typedef {{url: string, filename: string, referrer: string, cookies: string,
 *   userAgent: string, mime: string, size: number, pageUrl: string,
 *   pageTitle: string, kind: DownloadKind, silent: boolean}} DownloadRequest
 */

const userAgent = globalThis.navigator?.userAgent ?? '';

/**
 * @param {Partial<DownloadRequest> & {url: string, storeId?: string}} input
 *   `cookies` may be passed precomputed; otherwise it is looked up for `url`.
 * @returns {Promise<DownloadRequest>}
 */
export async function buildDownloadRequest(input) {
  const kind = input.kind ?? 'file';
  // Magnet links have no host to send cookies to.
  const cookies = input.cookies ?? (kind === 'magnet' ? '' : await getCookieHeader(input.url, { storeId: input.storeId }));
  const size = Number(input.size);
  return {
    url: String(input.url),
    filename: String(input.filename ?? ''),
    referrer: String(input.referrer ?? ''),
    cookies,
    userAgent,
    mime: String(input.mime ?? ''),
    size: Number.isFinite(size) && size >= 0 ? Math.round(size) : -1,
    pageUrl: String(input.pageUrl ?? ''),
    pageTitle: String(input.pageTitle ?? ''),
    kind,
    silent: input.silent === true,
  };
}
