/**
 * Options page: edits settings directly through settings.js (extension pages
 * have storage access) and asks the background to test the connection.
 */
import { DEFAULT_EXTENSIONS, getSettings, onSettingsChanged, parseExtensionList, parseSiteList, saveSettings } from '../lib/settings.js';
import { callBackground } from '../ui/messaging.js';
import { installLogoFallback } from '../ui/format.js';

const $ = (id) => document.getElementById(id);
const el = {
  form: $('options-form'),
  token: $('token'),
  toggleToken: $('toggle-token'),
  enabled: $('enabled'),
  extensions: $('extensions'),
  resetExtensions: $('reset-extensions'),
  minSize: $('min-size'),
  excludedSites: $('excluded-sites'),
  showBadge: $('show-badge'),
  silentIntercept: $('silent-intercept'),
  testButton: $('test-connection'),
  testResult: $('test-result'),
  save: $('save'),
  saveStatus: $('save-status'),
};

let dirty = false;
let statusTimer = 0;

function fill(settings) {
  el.token.value = settings.token;
  el.enabled.checked = settings.enabled;
  el.extensions.value = settings.extensions.join(', ');
  el.minSize.value = String(settings.minSizeMB);
  el.excludedSites.value = settings.excludedSites.join('\n');
  el.showBadge.checked = settings.showBadge;
  el.silentIntercept.checked = settings.silentIntercept;
  dirty = false;
}

function setStatus(text, state = '') {
  clearTimeout(statusTimer);
  el.saveStatus.textContent = text;
  el.saveStatus.dataset.state = state;
  if (state === 'ok') statusTimer = setTimeout(() => setStatus(''), 2500);
}

function readMinSize() {
  const value = el.minSize.value.trim() === '' ? 0 : Number(el.minSize.value);
  const valid = Number.isFinite(value) && value >= 0;
  el.minSize.setAttribute('aria-invalid', String(!valid));
  return valid ? value : null;
}

/** Validates the form; returns the settings object or null (focusing the bad field). */
function collect() {
  const minSizeMB = readMinSize();
  if (minSizeMB === null) {
    setStatus('Minimum size must be 0 or more.', 'error');
    el.minSize.focus();
    return null;
  }
  return {
    token: el.token.value.trim(),
    enabled: el.enabled.checked,
    extensions: parseExtensionList(el.extensions.value),
    minSizeMB,
    excludedSites: parseSiteList(el.excludedSites.value),
    showBadge: el.showBadge.checked,
    silentIntercept: el.silentIntercept.checked,
  };
}

async function save(event) {
  event.preventDefault();
  const values = collect();
  if (!values) return;
  el.save.disabled = true;
  try {
    fill(await saveSettings(values)); // show the normalised values (deduped, cleaned)
    setStatus('Saved', 'ok');
  } catch (err) {
    setStatus(`Could not save: ${err.message}`, 'error');
  } finally {
    el.save.disabled = false;
  }
}

const TEST_TEXT = {
  connected: (r) => `Connected to GoDownload${r.appVersion ? ` ${r.appVersion}` : ''}${r.interceptEnabled === false ? ' (capturing is turned off in the app)' : ''}.`,
  unauthorized: () => 'GoDownload is running but rejected the pairing token.',
  offline: (r) => `Can’t reach GoDownload. Is the app running?${r.error && !/not running/i.test(r.error) ? ` (${r.error})` : ''}`,
};

async function testConnection() {
  el.testButton.disabled = true;
  el.testResult.dataset.state = '';
  el.testResult.textContent = 'Testing…';
  try {
    // Tests the values currently in the form, saved or not.
    const result = await callBackground('options:testConnection', { token: el.token.value.trim() });
    el.testResult.dataset.state = result.state;
    el.testResult.textContent = (TEST_TEXT[result.state] ?? TEST_TEXT.offline)(result);
  } catch (err) {
    el.testResult.dataset.state = 'offline';
    el.testResult.textContent = err.message;
  } finally {
    el.testButton.disabled = false;
  }
}

function bindEvents() {
  el.form.addEventListener('submit', save);
  el.form.addEventListener('input', () => {
    dirty = true;
    setStatus('Unsaved changes');
  });
  el.testButton.addEventListener('click', testConnection);

  el.toggleToken.addEventListener('click', () => {
    const show = el.token.type === 'password';
    el.token.type = show ? 'text' : 'password';
    el.toggleToken.textContent = show ? 'Hide' : 'Show';
    el.toggleToken.setAttribute('aria-pressed', String(show));
  });

  el.resetExtensions.addEventListener('click', () => {
    el.extensions.value = DEFAULT_EXTENSIONS.join(', ');
    el.extensions.dispatchEvent(new Event('input', { bubbles: true }));
    el.extensions.focus();
  });

  window.addEventListener('beforeunload', (event) => {
    if (!dirty) return;
    event.preventDefault();
    event.returnValue = ''; // older Chromium needs this to show the prompt
  });

  // Keep the form in sync with changes made elsewhere (popup toggle, sync),
  // unless the user is mid-edit.
  onSettingsChanged((settings) => {
    if (!dirty) fill(settings);
  });
}

async function init() {
  installLogoFallback();
  bindEvents();
  fill(await getSettings());
}

init().catch((err) => {
  console.error('[GoDownload] options init failed', err);
  setStatus(`Could not load settings: ${err.message}`, 'error');
});
