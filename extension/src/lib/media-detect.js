/**
 * Pure classification of network responses into downloadable media.
 * Given a URL and response headers, returns {kind, mime, size} or null.
 */
import { parseUrl, urlExtension } from './url-utils.js';

const MIN_PROGRESSIVE_BYTES = 512 * 1024;

const HLS_MIMES = new Set(['application/vnd.apple.mpegurl', 'application/x-mpegurl', 'audio/mpegurl', 'audio/x-mpegurl']);
const DASH_MIMES = new Set(['application/dash+xml']);
const VIDEO_EXTS = new Set(['mp4', 'webm', 'mkv', 'mov', 'm4v']);
const AUDIO_EXTS = new Set(['m4a', 'mp3', 'ogg', 'oga', 'opus', 'flac', 'wav', 'aac']);
// Fragment containers used by HLS/DASH; never useful on their own.
const SEGMENT_EXTS = new Set(['ts', 'm4s', 'cmfv', 'cmfa', 'm4f', 'fmp4']);
const SEGMENT_MIMES = new Set(['video/mp2t', 'video/iso.segment', 'audio/iso.segment']);
// Generic types servers use for media files with a proper extension.
const GENERIC_MIMES = new Set(['', 'application/octet-stream', 'binary/octet-stream', 'application/binary']);
// Query keys that mark a request for a byte range / numbered fragment of a stream.
const SEGMENT_QUERY_KEYS = ['range', 'bytes', 'byterange', 'sq', 'segment', 'frag', 'fragment'];
const SEGMENT_PATH = /(?:^|[/_.-])(?:seg(?:ment)?|chunk|frag(?:ment)?)[-_]?\d+(?:[/_.-]|$)|\/range\/\d+-\d+/i;

/** Lowercased header lookup over a webRequest responseHeaders array. */
export function headerValue(headers, name) {
  const lower = name.toLowerCase();
  const h = headers?.find((x) => x.name.toLowerCase() === lower);
  return h?.value ?? '';
}

/** Total resource size from Content-Range ("bytes 0-99/1234") or Content-Length. */
function resourceSize(headers, statusCode) {
  const range = /\/(\d+)\s*$/.exec(headerValue(headers, 'content-range'));
  if (range) return Number(range[1]);
  if (statusCode === 206) return -1; // partial body length isn't the file size
  const len = Number.parseInt(headerValue(headers, 'content-length'), 10);
  return Number.isFinite(len) && len >= 0 ? len : -1;
}

function looksLikeSegment(u) {
  for (const key of SEGMENT_QUERY_KEYS) if (u.searchParams.has(key)) return true;
  return SEGMENT_PATH.test(u.pathname);
}

/**
 * @param {{url: string, statusCode: number, responseHeaders?: {name: string, value?: string}[]}} details
 * @returns {{kind: 'hls'|'dash'|'video'|'audio', mime: string, size: number} | null}
 */
export function detectMedia(details) {
  const { url, statusCode, responseHeaders } = details;
  if (statusCode < 200 || statusCode >= 300) return null;
  const u = parseUrl(url);
  if (!u || (u.protocol !== 'http:' && u.protocol !== 'https:')) return null;

  const mime = headerValue(responseHeaders, 'content-type').split(';')[0].trim().toLowerCase();
  const ext = urlExtension(url);
  const size = resourceSize(responseHeaders, statusCode);

  // Manifests first: they are small, so no size threshold applies.
  if (ext === 'm3u8' || HLS_MIMES.has(mime)) return { kind: 'hls', mime: HLS_MIMES.has(mime) ? mime : 'application/vnd.apple.mpegurl', size: -1 };
  if (ext === 'mpd' || DASH_MIMES.has(mime)) return { kind: 'dash', mime: 'application/dash+xml', size: -1 };

  if (SEGMENT_EXTS.has(ext) || SEGMENT_MIMES.has(mime)) return null;

  let kind = null;
  if (mime.startsWith('video/')) kind = 'video';
  else if (mime.startsWith('audio/')) kind = 'audio';
  else if (GENERIC_MIMES.has(mime) && VIDEO_EXTS.has(ext)) kind = 'video';
  else if (GENERIC_MIMES.has(mime) && AUDIO_EXTS.has(ext)) kind = 'audio';
  if (!kind) return null;

  if (looksLikeSegment(u)) return null;
  if (size >= 0 && size < MIN_PROGRESSIVE_BYTES) return null;
  return { kind, mime, size };
}

// Query parameters that change between requests for the same media
// (signatures, expiry, cache busters, ranges) and must not break dedupe.
const VOLATILE_PARAMS = /^(?:_|t|ts|time|timestamp|rand|random|r|cb|cachebuster|nocache|v|expires?|exp|e|sig|signature|token|auth|key|hash|hmac|policy|key-pair-id|hdnts|hdnea|acl|x-amz-.*|x-goog-.*|validfrom|validto|range|bytes|rn|rbuf|session|sid)$/i;

/** A stable identity for a media URL, ignoring volatile query parameters. */
export function mediaDedupeKey(url) {
  const u = parseUrl(url);
  if (!u) return url;
  const kept = [...u.searchParams.entries()]
    .filter(([k]) => !VOLATILE_PARAMS.test(k))
    .sort(([a], [b]) => a.localeCompare(b));
  const query = new URLSearchParams(kept).toString();
  return `${u.origin}${u.pathname}${query ? `?${query}` : ''}`;
}

/** Maps a detected media kind to the API's "kind" field. */
export function apiKindForMedia(kind) {
  if (kind === 'hls' || kind === 'dash') return kind;
  return 'media';
}
