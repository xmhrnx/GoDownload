/**
 * Right-click menu entries: send a link/media/magnet, all links on the page,
 * or the links inside the current selection.
 */
import { api } from './browser.js';
import { log } from './log.js';
import { sendDownload } from './api.js';
import { buildDownloadRequest } from './payload.js';
import { storeIdForTab } from './cookies.js';
import { kindForUrl } from './filters.js';
import { sendLinksFromTab } from './link-collector.js';
import { recordError, clearError } from './activity.js';
import { flashBadge } from './badge.js';

const menus = api.contextMenus ?? api.menus;

const ID = Object.freeze({
  ITEM: 'godownload-item',
  MAGNET: 'godownload-magnet',
  ALL_LINKS: 'godownload-all-links',
  SELECTION: 'godownload-selection',
});

const WEB_PATTERNS = ['http://*/*', 'https://*/*', 'ftp://*/*'];

/** contextMenus.create reports errors through lastError (or throws); normalise to a boolean. */
function tryCreate(props) {
  return new Promise((resolve) => {
    try {
      menus.create(props, () => {
        const err = api.runtime.lastError;
        if (err) {
          log.debug(`menu "${props.id}" not created:`, err.message);
          resolve(false);
        } else {
          resolve(true);
        }
      });
    } catch (err) {
      log.debug(`menu "${props.id}" threw:`, err?.message);
      resolve(false);
    }
  });
}

let setupChain = Promise.resolve();

/**
 * (Re)creates all menu items. Safe to call repeatedly (onInstalled and
 * onStartup may both fire); calls are serialised and start with removeAll.
 */
export function setupContextMenus() {
  if (!menus) return Promise.resolve();
  setupChain = setupChain.then(async () => {
    try {
      await menus.removeAll();
    } catch (err) {
      log.warn('menus.removeAll failed', err);
    }

    // Not every browser accepts a magnet: match pattern. If it is rejected, the
    // generic link item stays unrestricted so it still handles magnet links.
    const magnetOk = await tryCreate({
      id: ID.MAGNET,
      title: 'Send magnet link to GoDownload',
      contexts: ['link'],
      targetUrlPatterns: ['magnet:*'],
    });

    const item = { id: ID.ITEM, title: 'Download with GoDownload', contexts: ['link', 'image', 'video', 'audio'] };
    const created = magnetOk && (await tryCreate({ ...item, targetUrlPatterns: WEB_PATTERNS }));
    if (!created) await tryCreate(item);

    await tryCreate({ id: ID.ALL_LINKS, title: 'Download all links with GoDownload', contexts: ['page'] });
    await tryCreate({ id: ID.SELECTION, title: 'Download selected links with GoDownload', contexts: ['selection'] });
  }).catch((err) => log.error('context menu setup failed', err));
  return setupChain;
}

/** Picks the URL the user meant: the media itself for video/audio, else the link. */
function targetUrl(info) {
  const mediaType = info.mediaType;
  if ((mediaType === 'video' || mediaType === 'audio') && info.srcUrl) return info.srcUrl;
  return info.linkUrl || info.srcUrl || '';
}

function kindFor(info, url) {
  const kind = kindForUrl(url);
  if (kind === 'file' && (info.mediaType === 'video' || info.mediaType === 'audio')) return 'media';
  return kind;
}

async function sendSingle(info, tab) {
  const url = targetUrl(info);
  if (!/^(https?|ftp|magnet):/i.test(url)) {
    throw new Error(url.startsWith('blob:')
      ? 'This media is streamed (blob: URL). Use “Detected media” in the GoDownload popup instead.'
      : 'This link type can’t be downloaded.');
  }
  const referrer = info.frameUrl || info.pageUrl || tab?.url || '';
  const request = await buildDownloadRequest({
    url,
    referrer,
    pageUrl: info.pageUrl || tab?.url || '',
    pageTitle: tab?.title ?? '',
    kind: kindFor(info, url),
    storeId: storeIdForTab(tab),
  });
  await sendDownload(request);
}

async function handleClick(info, tab) {
  const tabId = tab?.id ?? -1;
  try {
    switch (info.menuItemId) {
      case ID.ITEM:
      case ID.MAGNET:
        await sendSingle(info, tab);
        break;
      case ID.ALL_LINKS:
        if (!tab) throw new Error('No tab to collect links from.');
        await sendLinksFromTab(tab);
        break;
      case ID.SELECTION:
        if (!tab) throw new Error('No tab to collect links from.');
        await sendLinksFromTab(tab, { selectionOnly: true, frameId: info.frameId });
        break;
      default:
        return;
    }
    await clearError();
    await flashBadge(tabId, true);
  } catch (err) {
    log.warn('context menu action failed:', err?.message ?? err);
    await recordError(err?.message ?? String(err), info.linkUrl || info.srcUrl || info.pageUrl || '');
    await flashBadge(tabId, false);
  }
}

/** Registers the click handler. Must run synchronously at startup. */
export function registerContextMenuHandler() {
  if (!menus?.onClicked) return;
  menus.onClicked.addListener((info, tab) => {
    handleClick(info, tab).catch((err) => log.error('menu click:', err));
  });
}
