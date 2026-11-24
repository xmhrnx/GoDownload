/**
 * Builds a Cookie header value for a URL so the desktop app can download with
 * the user's session (logged-in downloads, CDN tokens bound to cookies...).
 */
import { api } from './browser.js';
import { log } from './log.js';

async function getAllSafe(details) {
  try {
    return (await api.cookies.getAll(details)) ?? [];
  } catch (err) {
    // Expected on browsers that don't know `partitionKey` or the store id.
    log.debug('cookies.getAll rejected', Object.keys(details), err?.message);
    return [];
  }
}

/**
 * Returns the cookie store to use for a tab: Firefox reports it directly
 * (containers, private windows); Chrome's incognito store is always "1".
 * @param {{cookieStoreId?: string, incognito?: boolean} | undefined} tab
 */
export function storeIdForTab(tab) {
  if (!tab) return undefined;
  if (tab.cookieStoreId) return tab.cookieStoreId;
  return tab.incognito ? '1' : undefined;
}

/**
 * @param {string} url
 * @param {{storeId?: string}} [opts]
 * @returns {Promise<string>} e.g. "a=1; b=2" ("" when none or not http(s))
 */
export async function getCookieHeader(url, opts = {}) {
  if (!/^https?:/i.test(url) || !api.cookies) return '';
  const base = { url };
  if (opts.storeId) base.storeId = opts.storeId;

  // `partitionKey: {}` additionally returns partitioned (CHIPS / first-party
  // isolated) cookies. Older browsers throw on the unknown key, hence the
  // separate guarded call.
  const [plain, partitioned] = await Promise.all([
    getAllSafe(base),
    getAllSafe({ ...base, partitionKey: {} }),
  ]);

  const seen = new Set();
  const cookies = [];
  for (const c of [...plain, ...partitioned]) {
    const key = [c.name, c.domain, c.path, c.partitionKey?.topLevelSite ?? ''].join('\u0000');
    if (seen.has(key)) continue;
    seen.add(key);
    cookies.push(c);
  }
  // Same ordering browsers use: longer (more specific) paths first.
  cookies.sort((a, b) => (b.path?.length ?? 0) - (a.path?.length ?? 0));
  return cookies.map((c) => `${c.name}=${c.value}`).join('; ');
}

/**
 * Memoised variant for batches, where many URLs often share an origin+path.
 * @returns {(url: string) => Promise<string>}
 */
export function cookieHeaderCache(opts = {}) {
  const memo = new Map();
  return (url) => {
    if (!memo.has(url)) memo.set(url, getCookieHeader(url, opts));
    return memo.get(url);
  };
}
