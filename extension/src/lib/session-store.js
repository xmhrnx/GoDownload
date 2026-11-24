/**
 * Small key/value store for state that must survive MV3 service-worker
 * restarts but not a browser restart (detected media, bypass list, one-shot
 * toggles).
 *
 * Uses storage.session when available. Firefox only gained storage.session in
 * 115, so on 109-114 we fall back to an in-memory map; Firefox's background
 * script lives as long as an event page, which is acceptable for this data.
 */
import { api } from './browser.js';
import { log } from './log.js';

const area = api.storage?.session ?? null;
const memory = new Map();

if (!area) log.info('storage.session unavailable; using in-memory session state');

/** @returns {Promise<any>} the stored value or `fallback` */
export async function sessionGet(key, fallback = undefined) {
  if (!area) return memory.has(key) ? memory.get(key) : fallback;
  try {
    const result = await area.get(key);
    return key in result ? result[key] : fallback;
  } catch (err) {
    log.warn('session get failed', key, err);
    return memory.has(key) ? memory.get(key) : fallback;
  }
}

export async function sessionSet(key, value) {
  memory.set(key, value);
  if (!area) return;
  try {
    await area.set({ [key]: value });
  } catch (err) {
    log.warn('session set failed', key, err);
  }
}

export async function sessionRemove(key) {
  memory.delete(key);
  if (!area) return;
  try {
    await area.remove(key);
  } catch (err) {
    log.warn('session remove failed', key, err);
  }
}

/** Returns every stored key starting with `prefix`, as {key: value}. */
export async function sessionGetByPrefix(prefix) {
  let all;
  if (area) {
    try {
      all = await area.get(null);
    } catch (err) {
      log.warn('session list failed', err);
    }
  }
  all ??= Object.fromEntries(memory);
  return Object.fromEntries(Object.entries(all).filter(([k]) => k.startsWith(prefix)));
}
