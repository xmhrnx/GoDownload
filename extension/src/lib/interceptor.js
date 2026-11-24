/**
 * Browser-download interception: decides for each new download whether to
 * cancel it and hand it to GoDownload instead.
 *
 * Chromium: downloads.onDeterminingFilename (has finalUrl, mime, fileSize and
 * the suggested filename, and holds the download until we answer).
 * Firefox: downloads.onCreated (no onDeterminingFilename there).
 */
import { api } from './browser.js';
import { log, safely } from './log.js';
import { getSettings } from './settings.js';
import { evaluateDownload } from './filters.js';
import { Connection, ping, sendDownload } from './api.js';
import { buildDownloadRequest } from './payload.js';
import { addBypass, consumeBypass, consumeSkipNext } from './bypass.js';
import { clearError, recordError } from './activity.js';
import { basename, sanitizeFilename } from './url-utils.js';

function knownSize(item) {
  if (item.fileSize > 0) return item.fileSize;
  if (item.totalBytes > 0) return item.totalBytes;
  return -1;
}

/** Best-effort page title: the active tab, if it is the page that started the download. */
async function pageTitleFor(referrer) {
  if (!referrer) return '';
  const [tab] = await safely('tabs.query', () => api.tabs.query({ active: true, lastFocusedWindow: true }), []);
  return tab?.url === referrer ? tab.title ?? '' : '';
}

/** Starts the download again in the browser after a failed hand-off. */
async function restartInBrowser(item, filename) {
  const urls = [item.url, item.finalUrl].filter(Boolean);
  // Mark first: the new download fires our listener almost immediately.
  await addBypass(...urls);
  const name = sanitizeFilename(filename);
  try {
    await api.downloads.download({ url: item.url, ...(name ? { filename: name } : {}), saveAs: false });
  } catch (err) {
    // An odd filename can be rejected by the browser; retry with its default.
    if (!name) throw err;
    log.warn('re-download with filename failed, retrying without', err);
    await api.downloads.download({ url: item.url, saveAs: false });
  }
}

async function takeOver(item, decision, settings) {
  const url = item.finalUrl || item.url;
  // Stop the browser transfer as early as possible, then hide it from history.
  await safely('downloads.cancel', () => api.downloads.cancel(item.id));
  await safely('downloads.erase', () => api.downloads.erase({ id: item.id }));

  const referrer = item.referrer || '';
  try {
    const request = await buildDownloadRequest({
      url,
      filename: decision.filename,
      referrer,
      mime: item.mime,
      size: knownSize(item),
      pageUrl: referrer,
      pageTitle: await pageTitleFor(referrer),
      kind: decision.kind,
      silent: settings.silentIntercept,
    });
    await sendDownload(request);
    await clearError();
    log.info('handed off', url);
  } catch (err) {
    const name = decision.filename || basename(url) || url;
    log.warn('hand-off failed, restarting in browser:', name, err?.message);
    await recordError(`Couldn't send “${name}” to GoDownload (${err?.message ?? err}). It was downloaded by the browser instead.`, url);
    await safely('restart download', () => restartInBrowser(item, decision.filename));
  }
}

async function handleDownload(item) {
  // Our own re-downloads, finished/restored items and private windows are left alone.
  if (item.byExtensionId && item.byExtensionId === api.runtime.id) return;
  if (item.state && item.state !== 'in_progress') return;
  if (item.incognito) return;
  if (await consumeBypass(item.url, item.finalUrl)) return;

  const settings = await getSettings();
  const decision = evaluateDownload({
    url: item.url,
    finalUrl: item.finalUrl,
    referrer: item.referrer,
    filename: item.filename,
    mime: item.mime,
    size: knownSize(item),
  }, settings);
  if (!decision.intercept) return;

  // Only cancel once we know the app can take it; otherwise the browser keeps it.
  const conn = await ping();
  if (conn.state !== Connection.CONNECTED) return;
  if (conn.info?.interceptEnabled === false) return;

  // The one-shot toggle is consumed only by a download we'd otherwise take.
  if (await consumeSkipNext()) return;

  await takeOver(item, decision, settings);
}

/** Registers the download listener. Must run synchronously at startup. */
export function registerInterceptor() {
  const downloads = api.downloads;
  if (!downloads) {
    log.warn('downloads API unavailable; interception disabled');
    return;
  }
  if (downloads.onDeterminingFilename) {
    downloads.onDeterminingFilename.addListener((item, suggest) => {
      handleDownload(item)
        .catch((err) => log.error('interceptor:', err))
        .finally(() => {
          // Always answer, otherwise Chrome stalls the download. Calling it
          // for a cancelled download is harmless.
          try {
            suggest();
          } catch (err) {
            log.debug('suggest() after cancel:', err?.message);
          }
        });
      return true; // we call suggest asynchronously
    });
  } else {
    downloads.onCreated.addListener((item) => {
      handleDownload(item).catch((err) => log.error('interceptor:', err));
    });
  }
}
