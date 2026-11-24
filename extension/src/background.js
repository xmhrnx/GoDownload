/**
 * GoDownload background entry point (Chrome MV3 service worker / Firefox
 * background script, loaded as an ES module).
 *
 * All event listeners are registered synchronously at the top level: MV3 only
 * delivers the event that woke a terminated service worker to listeners that
 * exist after the first turn of the script.
 */
import { api } from './lib/browser.js';
import { log } from './lib/log.js';
import { onSettingsChanged } from './lib/settings.js';
import { registerInterceptor } from './lib/interceptor.js';
import { registerMediaSniffer } from './lib/media-sniffer.js';
import { registerContextMenuHandler, setupContextMenus } from './lib/context-menus.js';
import { registerMessageRouter } from './lib/messages.js';
import { refreshAllBadges } from './lib/badge.js';

// Last-resort guard so nothing ever surfaces as an unhandled rejection.
globalThis.addEventListener?.('unhandledrejection', (event) => {
  log.error('unhandled rejection:', event.reason);
  event.preventDefault();
});

registerInterceptor();
registerMediaSniffer();
registerContextMenuHandler();
registerMessageRouter();

onSettingsChanged((_settings, keys) => {
  if (keys.includes('showBadge')) refreshAllBadges().catch((err) => log.error('badge refresh failed', err));
});

api.runtime.onInstalled.addListener((details) => {
  log.info(`installed (${details.reason}), version ${api.runtime.getManifest().version}`);
  setupContextMenus();
  refreshAllBadges().catch((err) => log.error('badge refresh failed', err));
  if (details.reason === 'install') {
    api.runtime.openOptionsPage().catch((err) => log.warn('could not open options', err));
  }
});

api.runtime.onStartup.addListener(() => {
  setupContextMenus();
  refreshAllBadges().catch((err) => log.error('badge refresh failed', err));
});
