/**
 * Runs content/collect-links.js in a tab and turns the result into a batch
 * request for the desktop app. Shared by the context menu and the popup.
 */
import { api } from './browser.js';
import { log, safely } from './log.js';
import { sendBatch } from './api.js';
import { buildDownloadRequest } from './payload.js';
import { cookieHeaderCache, storeIdForTab } from './cookies.js';
import { kindForUrl } from './filters.js';

const SCRIPT_FILE = 'content/collect-links.js';
const MAX_ITEMS = 2000;

/**
 * @param {number} tabId
 * @param {{selectionOnly?: boolean, frameId?: number}} opts
 * @returns {Promise<{url: string, text: string}[]>}
 */
export async function collectLinks(tabId, { selectionOnly = false, frameId } = {}) {
  const target = selectionOnly && Number.isInteger(frameId)
    ? { tabId, frameIds: [frameId] }
    : { tabId, allFrames: !selectionOnly };

  // executeScript({files}) can't take arguments, so the mode is passed via a
  // global in the isolated world that the collector reads and deletes.
  await api.scripting.executeScript({
    target,
    func: (mode) => { globalThis.__godownloadCollectMode = mode; },
    args: [selectionOnly ? 'selection' : 'all'],
  });
  const results = await api.scripting.executeScript({ target, files: [SCRIPT_FILE] });

  const seen = new Set();
  const links = [];
  for (const frame of results ?? []) {
    if (!Array.isArray(frame?.result)) continue;
    for (const link of frame.result) {
      if (typeof link?.url !== 'string' || seen.has(link.url)) continue;
      if (!/^(https?|magnet):/i.test(link.url)) continue; // never trust page-side filtering
      seen.add(link.url);
      links.push({ url: link.url, text: String(link.text ?? '').slice(0, 300) });
      if (links.length >= MAX_ITEMS) return links;
    }
  }
  return links;
}

/**
 * Collects links in a tab and sends them to the app's batch picker.
 * @returns {Promise<{count: number}>}
 * @throws when the page can't be scripted, nothing was found, or the app fails
 */
export async function sendLinksFromTab(tab, opts = {}) {
  let links;
  try {
    links = await collectLinks(tab.id, opts);
  } catch (err) {
    log.warn('collect links failed', err);
    throw new Error('This page can’t be scanned for links (browser or store pages are protected).');
  }
  if (!links.length) throw new Error(opts.selectionOnly ? 'No links in the selection.' : 'No links found on this page.');

  const storeId = storeIdForTab(tab);
  const cookiesFor = cookieHeaderCache({ storeId });
  const pageUrl = tab.url ?? '';
  const pageTitle = tab.title ?? '';
  const items = await Promise.all(links.map(async (link) => {
    const kind = kindForUrl(link.url);
    return buildDownloadRequest({
      url: link.url,
      // The contract has no field for link text, so it is not forwarded;
      // the app derives names from the URL / response headers.
      filename: '',
      referrer: pageUrl,
      cookies: kind === 'magnet' ? '' : await safely('cookies', () => cookiesFor(link.url), ''),
      pageUrl,
      pageTitle,
      kind,
    });
  }));
  await sendBatch({ items, pageUrl, pageTitle });
  return { count: items.length };
}
