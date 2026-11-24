/**
 * Per-tab list of detected media, kept in memory and mirrored to session
 * storage (MV3 service workers are killed when idle; the popup must still see
 * what was detected before that).
 */
import { sessionGet, sessionGetByPrefix, sessionRemove, sessionSet } from './session-store.js';
import { log } from './log.js';
import { mediaDedupeKey } from './media-detect.js';

const PREFIX = 'media:';
const MAX_PER_TAB = 100;
const WRITE_DELAY_MS = 300;

/**
 * @typedef {{url: string, key: string, kind: 'hls'|'dash'|'video'|'audio',
 *   mime: string, size: number, title: string, pageUrl: string, firstSeen: number}} MediaItem
 */

/** @type {Map<number, Promise<MediaItem[]>>} tabId -> list (loaded lazily) */
const lists = new Map();
const pendingWrites = new Map();
const listeners = new Set();

function load(tabId) {
  if (!lists.has(tabId)) {
    lists.set(tabId, sessionGet(PREFIX + tabId, []).then((v) => (Array.isArray(v) ? v : [])));
  }
  return lists.get(tabId);
}

/** Debounced write: a page can produce dozens of media responses per second. */
function scheduleWrite(tabId) {
  clearTimeout(pendingWrites.get(tabId));
  pendingWrites.set(tabId, setTimeout(() => {
    pendingWrites.delete(tabId);
    load(tabId)
      .then((list) => (list.length ? sessionSet(PREFIX + tabId, list) : sessionRemove(PREFIX + tabId)))
      .catch((err) => log.error('media persist failed', err));
  }, WRITE_DELAY_MS));
}

function notify(tabId, list) {
  for (const fn of listeners) {
    try {
      fn(tabId, list);
    } catch (err) {
      log.error('media listener failed', err);
    }
  }
}

/** Subscribes to list changes: `fn(tabId, list)`. */
export function onMediaChanged(fn) {
  listeners.add(fn);
}

/** @returns {Promise<MediaItem[]>} */
export async function getMedia(tabId) {
  return [...(await load(tabId))];
}

/**
 * Adds an item unless an equivalent URL is already listed.
 * When a duplicate has new information (size learned later), it is updated.
 * @returns {Promise<boolean>} whether the list changed
 */
export async function addMedia(tabId, item) {
  const list = await load(tabId);
  const key = mediaDedupeKey(item.url);
  const existing = list.find((m) => m.key === key);
  if (existing) {
    if (existing.size < 0 && item.size >= 0) {
      existing.size = item.size;
      scheduleWrite(tabId);
      notify(tabId, list);
      return true;
    }
    return false;
  }
  if (list.length >= MAX_PER_TAB) return false;
  list.push({ ...item, key });
  scheduleWrite(tabId);
  notify(tabId, list);
  return true;
}

export async function clearMedia(tabId) {
  const had = (await load(tabId)).length > 0;
  lists.set(tabId, Promise.resolve([]));
  clearTimeout(pendingWrites.get(tabId));
  pendingWrites.delete(tabId);
  await sessionRemove(PREFIX + tabId);
  if (had) notify(tabId, []);
}

/** Drops all state for a closed tab. */
export async function forgetTab(tabId) {
  clearTimeout(pendingWrites.get(tabId));
  pendingWrites.delete(tabId);
  lists.delete(tabId);
  await sessionRemove(PREFIX + tabId);
}

/** Tab ids that currently have stored media (used to repaint badges). */
export async function tabsWithMedia() {
  const stored = await sessionGetByPrefix(PREFIX);
  const ids = new Set(Object.keys(stored).map((k) => Number(k.slice(PREFIX.length))));
  for (const id of lists.keys()) ids.add(id);
  return [...ids].filter(Number.isInteger);
}
