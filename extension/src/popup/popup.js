/**
 * Popup UI: connection status, capture toggles, detected media for the
 * active tab, and quick actions. All privileged work happens in the
 * background via messages; this file only renders and forwards clicks.
 */
import { api } from '../lib/browser.js';
import { callBackground } from '../ui/messaging.js';
import { KIND_LABELS, formatBytes, hostOf, installLogoFallback } from '../ui/format.js';

const REFRESH_MS = 2000;
const ALL_URLS = { origins: ['<all_urls>'] };

const $ = (id) => document.getElementById(id);
const el = {
  root: document.querySelector('.popup'),
  connection: $('connection'),
  enabled: $('enabled'),
  skipNext: $('skip-next'),
  notices: $('notices'),
  permissionNotice: $('permission-notice'),
  tokenNotice: $('token-notice'),
  errorNotice: $('error-notice'),
  errorText: $('error-text'),
  okNotice: $('ok-notice'),
  okText: $('ok-text'),
  mediaList: $('media-list'),
  mediaEmpty: $('media-empty'),
  mediaCount: $('media-count'),
  downloadAll: $('download-all'),
  template: $('media-item-template'),
};

const CONNECTION_TEXT = {
  checking: 'Checking…',
  connected: 'Connected',
  offline: 'App not running',
  unauthorized: 'Token required',
};

let tabId = null;
let lastMediaSignature = '';
let okTimer = 0;
/** url -> 'sending' | 'sent', so polling re-renders keep button feedback. */
const sendState = new Map();

/* ---------- Notices ---------- */

function syncNoticeContainer() {
  el.notices.hidden = [...el.notices.children].every((n) => n.hidden);
}

function showError(message) {
  el.errorText.textContent = message;
  el.errorNotice.hidden = !message;
  syncNoticeContainer();
}

function showOk(message) {
  clearTimeout(okTimer);
  el.okText.textContent = message;
  el.okNotice.hidden = false;
  syncNoticeContainer();
  okTimer = setTimeout(() => {
    el.okNotice.hidden = true;
    syncNoticeContainer();
  }, 3500);
}

/* ---------- Rendering ---------- */

function renderConnection(connection) {
  const state = connection?.state ?? 'checking';
  el.connection.dataset.state = state;
  el.connection.textContent = CONNECTION_TEXT[state] ?? state;
  el.connection.title = state === 'connected' && connection.appVersion
    ? `GoDownload ${connection.appVersion} — click to re-check`
    : 'Click to check the connection again';
  el.tokenNotice.hidden = state !== 'unauthorized';
}

function renderMediaItem(item) {
  const node = el.template.content.firstElementChild.cloneNode(true);
  const title = item.title || hostOf(item.url) || item.url;
  node.querySelector('.media-title').textContent = title;
  node.querySelector('.media-title').title = item.url;
  const chip = node.querySelector('.chip');
  chip.dataset.kind = item.kind;
  chip.textContent = KIND_LABELS[item.kind] ?? item.kind;
  node.querySelector('.media-size').textContent = formatBytes(item.size);
  node.querySelector('.media-host').textContent = hostOf(item.url);

  const button = node.querySelector('.media-download');
  button.setAttribute('aria-label', `Download ${KIND_LABELS[item.kind] ?? ''} “${title}” with GoDownload`);
  applyButtonState(button, sendState.get(item.url));
  button.addEventListener('click', () => downloadMedia(item.url, button));
  return node;
}

function applyButtonState(button, state) {
  button.dataset.state = state ?? '';
  button.disabled = state === 'sending';
  button.textContent = state === 'sending' ? 'Sending…' : state === 'sent' ? 'Sent ✓' : 'Download';
}

function renderMedia(media) {
  // Rebuilding on every poll would steal keyboard focus; only redraw on change.
  const signature = JSON.stringify(media.map((m) => [m.url, m.size, m.title]));
  if (signature === lastMediaSignature) return;
  lastMediaSignature = signature;

  const focusedUrl = document.activeElement?.closest?.('.media-item')?.dataset.url;
  // Newest first: the stream the user just started is the likeliest target.
  const items = [...media].sort((a, b) => b.firstSeen - a.firstSeen);
  el.mediaList.replaceChildren(...items.map((item) => {
    const node = renderMediaItem(item);
    node.dataset.url = item.url;
    return node;
  }));
  if (focusedUrl) {
    [...el.mediaList.children].find((n) => n.dataset.url === focusedUrl)?.querySelector('button')?.focus();
  }
  el.mediaEmpty.hidden = media.length > 0;
  el.mediaList.hidden = media.length === 0;
  el.mediaCount.textContent = media.length ? String(media.length) : '';
}

function render(state) {
  renderConnection(state.connection);
  el.enabled.checked = state.enabled;
  el.skipNext.checked = state.skipNext;
  el.skipNext.disabled = !state.enabled;
  renderMedia(state.media ?? []);
  if (state.lastError?.message) showError(state.lastError.message);
  el.root.setAttribute('aria-busy', 'false');
}

/* ---------- Data ---------- */

async function refresh({ forcePing = false } = {}) {
  try {
    render(await callBackground('popup:getState', { tabId, forcePing }));
  } catch (err) {
    renderConnection({ state: 'offline' });
    showError(err.message);
  }
}

async function checkHostPermission() {
  // Firefox MV3 treats host permissions as opt-in; Chrome grants them at install.
  if (!api.permissions?.contains) return;
  try {
    el.permissionNotice.hidden = await api.permissions.contains(ALL_URLS);
  } catch {
    el.permissionNotice.hidden = true;
  }
  syncNoticeContainer();
}

/* ---------- Actions ---------- */

async function downloadMedia(url, button) {
  sendState.set(url, 'sending');
  applyButtonState(button, 'sending');
  try {
    await callBackground('popup:downloadMedia', { tabId, url });
    sendState.set(url, 'sent');
    applyButtonState(button, 'sent');
    showError('');
  } catch (err) {
    sendState.delete(url);
    applyButtonState(button, undefined);
    showError(err.code === 'unauthorized' ? 'GoDownload rejected the pairing token. Check it in Options.' : err.message);
    refresh();
  }
}

async function toggle(input, type, key) {
  const value = input.checked;
  input.disabled = true;
  try {
    await callBackground(type, { [key]: value });
  } catch (err) {
    input.checked = !value;
    showError(err.message);
  } finally {
    input.disabled = false;
    refresh();
  }
}

function openOptions() {
  api.runtime.openOptionsPage().catch((err) => showError(err.message));
  window.close();
}

function bindEvents() {
  el.enabled.addEventListener('change', () => toggle(el.enabled, 'popup:setEnabled', 'enabled'));
  el.skipNext.addEventListener('change', () => toggle(el.skipNext, 'popup:setSkipNext', 'value'));

  el.connection.addEventListener('click', () => {
    renderConnection({ state: 'checking' });
    refresh({ forcePing: true });
  });

  $('open-options').addEventListener('click', openOptions);
  $('open-options-token').addEventListener('click', openOptions);

  $('dismiss-error').addEventListener('click', () => {
    showError('');
    callBackground('popup:dismissError').catch(() => {});
  });

  // permissions.request must run directly inside the click handler (user gesture).
  $('grant-permission').addEventListener('click', () => {
    api.permissions.request(ALL_URLS)
      .then((granted) => {
        el.permissionNotice.hidden = granted;
        syncNoticeContainer();
      })
      .catch((err) => showError(err.message));
  });

  el.downloadAll.addEventListener('click', async () => {
    if (tabId === null) return;
    el.downloadAll.disabled = true;
    try {
      const { count } = await callBackground('popup:downloadAllLinks', { tabId });
      showError('');
      showOk(`Sent ${count} link${count === 1 ? '' : 's'} to GoDownload.`);
    } catch (err) {
      showError(err.code === 'unauthorized' ? 'GoDownload rejected the pairing token. Check it in Options.' : err.message);
    } finally {
      el.downloadAll.disabled = false;
    }
  });
}

async function init() {
  installLogoFallback();
  syncNoticeContainer();
  bindEvents();
  try {
    const [tab] = await api.tabs.query({ active: true, currentWindow: true });
    tabId = tab?.id ?? null;
  } catch {
    tabId = null;
  }
  el.downloadAll.disabled = tabId === null;
  await Promise.all([refresh(), checkHostPermission()]);
  setInterval(refresh, REFRESH_MS);
}

init().catch((err) => {
  console.error('[GoDownload] popup init failed', err);
  showError(err.message);
});
