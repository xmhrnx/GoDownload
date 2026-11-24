/**
 * Message router for the popup and options page. Only the extension's own
 * pages may call these handlers; content scripts (which run inside web pages)
 * are rejected by sender URL.
 */
import { api, extensionOrigin } from './browser.js';
import { log } from './log.js';
import { getSettings, saveSettings } from './settings.js';
import { ping, sendDownload } from './api.js';
import { getSkipNext, setSkipNext } from './bypass.js';
import { clearError, getRecentError } from './activity.js';
import { getMedia } from './media-store.js';
import { apiKindForMedia } from './media-detect.js';
import { buildDownloadRequest } from './payload.js';
import { storeIdForTab } from './cookies.js';
import { sendLinksFromTab } from './link-collector.js';
import { basename, extensionOf, sanitizeFilename } from './url-utils.js';

function isTrustedSender(sender) {
  return sender?.id === api.runtime.id && typeof sender.url === 'string' && sender.url.startsWith(extensionOrigin);
}

/** Suggested name for detected media: keep a real file name, else use the tab title. */
function mediaFilename(item) {
  if (item.kind === 'hls' || item.kind === 'dash') return ''; // the app picks the container
  const name = basename(item.url);
  if (extensionOf(name)) return sanitizeFilename(name);
  const title = sanitizeFilename(item.title);
  const ext = (item.mime.split('/')[1] || '').replace(/^x-/, '').replace('mpeg', 'mp3');
  return title && /^[a-z0-9]{2,5}$/.test(ext) ? `${title}.${ext}` : '';
}

const handlers = {
  async 'popup:getState'({ tabId, forcePing }) {
    const [connection, settings, skipNext, media, lastError] = await Promise.all([
      ping({ force: forcePing === true }),
      getSettings(),
      getSkipNext(),
      Number.isInteger(tabId) ? getMedia(tabId) : [],
      getRecentError(),
    ]);
    return {
      connection: { state: connection.state, appVersion: connection.info?.version ?? '', error: connection.error ?? '' },
      enabled: settings.enabled,
      skipNext,
      media: media.map(({ url, kind, mime, size, title, firstSeen }) => ({ url, kind, mime, size, title, firstSeen })),
      lastError,
    };
  },

  async 'popup:setEnabled'({ enabled }) {
    await saveSettings({ enabled: enabled === true });
    return { enabled: enabled === true };
  },

  async 'popup:setSkipNext'({ value }) {
    await setSkipNext(value === true);
    return { skipNext: value === true };
  },

  async 'popup:dismissError'() {
    await clearError();
    return {};
  },

  async 'popup:downloadMedia'({ tabId, url }) {
    // Look the item up instead of trusting the message, so only detected
    // media can be sent through this path.
    const item = (await getMedia(tabId)).find((m) => m.url === url);
    if (!item) throw new Error('This media is no longer available. Reload the page.');
    const tab = await api.tabs.get(tabId).catch(() => null);
    const request = await buildDownloadRequest({
      url: item.url,
      filename: mediaFilename(item),
      referrer: item.pageUrl,
      mime: item.mime,
      size: item.size,
      pageUrl: item.pageUrl,
      pageTitle: tab?.title || item.title,
      kind: apiKindForMedia(item.kind),
      storeId: storeIdForTab(tab),
    });
    const res = await sendDownload(request);
    await clearError();
    return { id: res.id ?? '' };
  },

  async 'popup:downloadAllLinks'({ tabId }) {
    const tab = await api.tabs.get(tabId);
    const result = await sendLinksFromTab(tab);
    await clearError();
    return result;
  },

  async 'options:testConnection'({ port, token }) {
    const result = await ping({ port, token, force: true });
    return { state: result.state, appVersion: result.info?.version ?? '', interceptEnabled: result.info?.interceptEnabled ?? null, error: result.error ?? '' };
  },
};

/** Registers the runtime.onMessage router. Must run synchronously at startup. */
export function registerMessageRouter() {
  api.runtime.onMessage.addListener((message, sender, sendResponse) => {
    const handler = message && Object.hasOwn(handlers, message.type) ? handlers[message.type] : null;
    if (!handler) return false;
    if (!isTrustedSender(sender)) {
      log.warn('rejected message from untrusted sender', sender?.url);
      return false;
    }
    // sendResponse + `return true` works in both Chrome and Firefox (Chrome
    // ignores promises returned from onMessage listeners).
    Promise.resolve()
      .then(() => handler(message.payload ?? {}))
      .then((data) => sendResponse({ ok: true, data }))
      .catch((err) => {
        log.warn(`${message.type} failed:`, err?.message ?? err);
        sendResponse({ ok: false, error: err?.message ?? String(err), code: err?.code ?? '' });
      });
    return true;
  });
}
