/**
 * Remembers the most recent hand-off failure so the popup can explain why a
 * download ended up in the browser (there is no other UI in the background).
 */
import { sessionGet, sessionRemove, sessionSet } from './session-store.js';

const KEY = 'lastError';
const SHOW_FOR_MS = 5 * 60_000;

export async function recordError(message, url = '') {
  await sessionSet(KEY, { message: String(message), url: String(url), at: Date.now() });
}

/** @returns {Promise<{message: string, url: string, at: number} | null>} */
export async function getRecentError() {
  const err = await sessionGet(KEY, null);
  return err && Date.now() - err.at < SHOW_FOR_MS ? err : null;
}

export async function clearError() {
  await sessionRemove(KEY);
}
