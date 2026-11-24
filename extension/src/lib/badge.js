/**
 * Toolbar badge: number of media items detected on each tab. Badge text is set
 * per tab, so the browser automatically shows the active tab's count.
 */
import { api } from './browser.js';
import { safely } from './log.js';
import { getSettings } from './settings.js';
import { getMedia, tabsWithMedia } from './media-store.js';

const BADGE_COLOR = '#6D5EF5';
const action = api.action ?? api.browserAction;

function formatCount(n) {
  if (n <= 0) return '';
  return n > 99 ? '99+' : String(n);
}

/** Sets the badge for one tab, honouring the "Show detected-media badge" setting. */
export async function updateBadge(tabId, count) {
  if (!action || tabId < 0) return;
  const { showBadge } = await getSettings();
  const text = showBadge ? formatCount(count) : '';
  await safely('setBadgeText', () => action.setBadgeText({ tabId, text }));
}

/** Repaints every tab we know about (after the badge setting changes or on startup). */
export async function refreshAllBadges() {
  if (!action) return;
  await safely('setBadgeBackgroundColor', () => action.setBadgeBackgroundColor({ color: BADGE_COLOR }));
  if (action.setBadgeTextColor) {
    await safely('setBadgeTextColor', () => action.setBadgeTextColor({ color: '#FFFFFF' }));
  }
  const tabIds = await tabsWithMedia();
  await Promise.all(tabIds.map(async (tabId) => updateBadge(tabId, (await getMedia(tabId)).length)));
}

/**
 * Briefly shows "OK"/"!" on a tab's badge as feedback for context-menu actions,
 * which otherwise have no visible UI, then restores the media count.
 */
export async function flashBadge(tabId, ok) {
  if (!action || !Number.isInteger(tabId) || tabId < 0) return;
  await safely('flash badge', async () => {
    await action.setBadgeBackgroundColor({ tabId, color: ok ? '#16A34A' : '#DC2626' });
    await action.setBadgeText({ tabId, text: ok ? 'OK' : '!' });
  });
  setTimeout(() => {
    safely('restore badge', async () => {
      await action.setBadgeBackgroundColor({ tabId, color: BADGE_COLOR });
      await updateBadge(tabId, (await getMedia(tabId)).length);
    });
  }, 2500);
}
