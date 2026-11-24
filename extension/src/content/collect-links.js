/**
 * Injected with scripting.executeScript (never declared as a content script).
 * Returns [{url, text}] for the links and media sources in this frame, or only
 * those inside the current selection when the background set the mode to
 * "selection" just before injecting this file.
 *
 * It runs in the isolated world, reads the DOM only, and talks to nothing:
 * the background receives the value of the final expression.
 */
(() => {
  const mode = globalThis.__godownloadCollectMode === 'selection' ? 'selection' : 'all';
  delete globalThis.__godownloadCollectMode;

  const ALLOWED = /^(https?|magnet):/i;
  const URL_IN_TEXT = /\b(?:https?:\/\/[^\s<>"'`]+|magnet:\?[^\s<>"'`]+)/gi;
  const MAX_TEXT = 200;

  const selection = mode === 'selection' ? window.getSelection() : null;
  if (mode === 'selection' && (!selection || selection.isCollapsed)) return [];

  const seen = new Set();
  const out = [];

  const clean = (s) => String(s ?? '').replace(/\s+/g, ' ').trim().slice(0, MAX_TEXT);

  const add = (rawUrl, text) => {
    if (!rawUrl) return;
    let url;
    try {
      url = new URL(rawUrl, document.baseURI);
    } catch {
      return;
    }
    if (!ALLOWED.test(url.protocol)) return;
    if (url.protocol !== 'magnet:') url.hash = ''; // fragments never change the file
    const href = url.href;
    if (seen.has(href)) return;
    seen.add(href);
    out.push({ url: href, text: clean(text) });
  };

  const inScope = (el) => !selection || selection.containsNode(el, true);

  const linkText = (a) => clean(a.textContent) ||
    clean(a.getAttribute('title')) ||
    clean(a.getAttribute('aria-label')) ||
    clean(a.querySelector('img[alt]')?.getAttribute('alt'));

  for (const a of document.querySelectorAll('a[href], area[href]')) {
    if (inScope(a)) add(a.href, linkText(a));
  }

  for (const media of document.querySelectorAll('video, audio')) {
    if (!inScope(media)) continue;
    const label = media.getAttribute('title') || media.getAttribute('aria-label') || '';
    // currentSrc covers <source> selection; blob: (MSE) URLs are filtered by ALLOWED.
    add(media.currentSrc || media.getAttribute('src'), label);
    for (const source of media.querySelectorAll('source[src]')) add(source.src, label);
  }
  for (const source of document.querySelectorAll('source[src]')) {
    if (inScope(source) && source.closest('video, audio')) add(source.src, '');
  }

  // Plain-text URLs typed in the selection (not wrapped in <a>).
  if (selection) {
    for (const match of selection.toString().matchAll(URL_IN_TEXT)) {
      add(match[0].replace(/[),.;:!?\]]+$/, ''), '');
    }
  }

  return out;
})();
