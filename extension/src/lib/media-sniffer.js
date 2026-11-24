/**
 * Watches network responses for streamable/downloadable media (HLS, DASH,
 * progressive audio/video) and records them per tab.
 */
import { api, extensionOrigin } from './browser.js';
import { log, safely } from './log.js';
import { detectMedia } from './media-detect.js';
import { addMedia, clearMedia, forgetTab, onMediaChanged } from './media-store.js';
import { updateBadge } from './badge.js';

const REQUEST_TYPES = ['media', 'xmlhttprequest', 'object', 'other'];

async function tabInfo(tabId) {
  const tab = await safely('tabs.get', () => api.tabs.get(tabId), null);
  return { title: tab?.title ?? '', pageUrl: tab?.url ?? '' };
}

async function onHeadersReceived(details) {
  if (details.tabId < 0) return; // not associated with a tab (service workers, prefetch)
  const initiator = details.initiator ?? details.originUrl ?? '';
  if (initiator.startsWith(extensionOrigin.replace(/\/$/, ''))) return;

  const media = detectMedia(details);
  if (!media) return;

  const { title, pageUrl } = await tabInfo(details.tabId);
  await addMedia(details.tabId, {
    url: details.url,
    kind: media.kind,
    mime: media.mime,
    size: media.size,
    title,
    pageUrl: pageUrl || details.documentUrl || initiator,
    firstSeen: Date.now(),
  });
}

/** Registers all listeners. Must run synchronously at startup. */
export function registerMediaSniffer() {
  onMediaChanged((tabId, list) => {
    updateBadge(tabId, list.length).catch((err) => log.error('badge update failed', err));
  });

  if (api.webRequest?.onHeadersReceived) {
    api.webRequest.onHeadersReceived.addListener(
      (details) => {
        onHeadersReceived(details).catch((err) => log.error('media sniffer:', err));
      },
      { urls: ['<all_urls>'], types: REQUEST_TYPES },
      ['responseHeaders'],
    );
  } else {
    log.warn('webRequest unavailable; media detection disabled');
  }

  // A new top-level document means a new page: forget the old page's media.
  if (api.webNavigation?.onCommitted) {
    api.webNavigation.onCommitted.addListener((details) => {
      if (details.frameId !== 0) return;
      clearMedia(details.tabId).catch((err) => log.error('clear media failed', err));
    });
  } else {
    api.tabs.onUpdated.addListener((tabId, changeInfo) => {
      if (changeInfo.status === 'loading' && changeInfo.url) {
        clearMedia(tabId).catch((err) => log.error('clear media failed', err));
      }
    });
  }

  api.tabs.onRemoved.addListener((tabId) => {
    forgetTab(tabId).catch((err) => log.error('forget tab failed', err));
  });
}
