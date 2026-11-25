/**
 * User settings: defaults, normalisation, persistence and change notification.
 *
 * Most settings live in storage.sync (falling back to storage.local when sync
 * is unavailable or fails). The pairing token is deliberately kept in
 * storage.local only: it belongs to *this* machine's desktop app, and is a
 * secret that should never be uploaded to a browser-sync account. There is no
 * port setting: the extension finds the app itself (see api.js).
 */
import { api } from './browser.js';
import { log } from './log.js';


export const DEFAULT_EXTENSIONS = Object.freeze([
  // archives
  'zip', 'rar', '7z', 'tar', 'gz', 'tgz', 'bz2', 'xz', 'zst', 'lz', 'lzma', 'cab',
  // executables / installers / packages
  'exe', 'msi', 'msix', 'msixbundle', 'appx', 'appxbundle', 'deb', 'rpm', 'apk', 'xapk',
  'pkg', 'appimage', 'flatpak', 'snap', 'jar', 'bin', 'run',
  // disk images
  'iso', 'img', 'dmg', 'vhd', 'vhdx', 'vmdk', 'qcow2', 'ova',
  // video
  'mp4', 'm4v', 'mkv', 'webm', 'avi', 'mov', 'wmv', 'flv', 'mpg', 'mpeg', '3gp', 'ogv',
  // audio
  'mp3', 'm4a', 'flac', 'wav', 'aac', 'ogg', 'oga', 'opus', 'wma', 'aiff',
  // documents
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods', 'odp', 'epub', 'mobi', 'djvu',
  // torrents
  'torrent',
]);

export const DEFAULTS = Object.freeze({
  enabled: true,              // global "Capture downloads" switch
  token: '',
  extensions: DEFAULT_EXTENSIONS,
  minSizeMB: 1,               // 0 disables the size rule
  excludedSites: [],
  showBadge: true,
  silentIntercept: false,     // forwarded as "silent" for intercepted downloads
});

/** Keys stored only on this device (never synced). */
const LOCAL_ONLY = new Set(['token']);

let cache = null;

/** Splits a free-form list ("zip, .rar  7z") into clean lowercase extensions. */
export function parseExtensionList(text) {
  const items = String(text ?? '')
    .split(/[\s,;]+/)
    .map((s) => s.trim().toLowerCase().replace(/^\*?\./, ''))
    .filter((s) => /^[a-z0-9][a-z0-9_+-]{0,15}$/.test(s));
  return [...new Set(items)];
}

/**
 * Normalises one excluded-site entry to a bare hostname.
 * Accepts "example.com", "*.example.com", "https://example.com/path".
 */
export function normalizeSite(entry) {
  let s = String(entry ?? '').trim().toLowerCase();
  if (!s || s.startsWith('#')) return '';
  s = s.replace(/^\*\./, '').replace(/^\.+/, '');
  try {
    const host = new URL(s.includes('://') ? s : `http://${s}`).hostname;
    return host.replace(/\.$/, '');
  } catch {
    return '';
  }
}

export function parseSiteList(text) {
  const lines = Array.isArray(text) ? text : String(text ?? '').split(/[\r\n]+/);
  return [...new Set(lines.map(normalizeSite).filter(Boolean))];
}

/** Coerces arbitrary stored/user input into a valid settings object. */
export function normalizeSettings(raw) {
  const s = { ...DEFAULTS, ...(raw ?? {}) };
  const minSize = Number(s.minSizeMB);
  return {
    enabled: s.enabled !== false,
    token: String(s.token ?? '').trim(),
    extensions: Array.isArray(s.extensions) ? parseExtensionList(s.extensions.join(' ')) : parseExtensionList(s.extensions),
    minSizeMB: Number.isFinite(minSize) && minSize >= 0 ? minSize : DEFAULTS.minSizeMB,
    excludedSites: parseSiteList(s.excludedSites),
    showBadge: s.showBadge !== false,
    silentIntercept: s.silentIntercept === true,
  };
}

async function readArea(area, keys) {
  try {
    return (await api.storage[area].get(keys)) ?? {};
  } catch (err) {
    log.warn(`storage.${area} read failed`, err);
    return {};
  }
}

/** Returns the current settings (cached; the cache is dropped on any change). */
export async function getSettings() {
  if (cache) return cache;
  const syncedKeys = Object.keys(DEFAULTS).filter((k) => !LOCAL_ONLY.has(k));
  const [local, synced] = await Promise.all([
    readArea('local', Object.keys(DEFAULTS)),
    api.storage.sync ? readArea('sync', syncedKeys) : Promise.resolve({}),
  ]);
  // A synced key only exists in local storage when the last sync write for it
  // failed (see saveSettings), so the local copy is the newest one and wins.
  const merged = { ...synced, ...local };
  cache = normalizeSettings(merged);
  return cache;
}

/** Persists a partial update and returns the full normalised settings. */
export async function saveSettings(partial) {
  const next = normalizeSettings({ ...(await getSettings()), ...partial });
  const localPart = {};
  const syncPart = {};
  for (const key of Object.keys(partial)) {
    if (!(key in next)) continue;
    (LOCAL_ONLY.has(key) ? localPart : syncPart)[key] = next[key];
  }
  if (Object.keys(syncPart).length) {
    try {
      if (!api.storage.sync) throw new Error('storage.sync unavailable');
      await api.storage.sync.set(syncPart);
      // Drop any stale fallback copies so the synced value is authoritative again.
      await api.storage.local.remove(Object.keys(syncPart)).catch(() => {});
    } catch (err) {
      log.warn('storage.sync write failed, using local', err);
      Object.assign(localPart, syncPart);
    }
  }
  if (Object.keys(localPart).length) await api.storage.local.set(localPart);
  cache = next;
  return next;
}

/**
 * Calls `listener(settings, changedKeys)` whenever settings change in any
 * context (popup, options page, other devices via sync).
 */
export function onSettingsChanged(listener) {
  api.storage.onChanged.addListener((changes, areaName) => {
    if (areaName !== 'sync' && areaName !== 'local') return;
    const keys = Object.keys(changes).filter((k) => k in DEFAULTS);
    if (!keys.length) return;
    cache = null;
    getSettings()
      .then((settings) => listener(settings, keys))
      .catch((err) => log.error('settings listener failed', err));
  });
}
