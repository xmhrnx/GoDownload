/**
 * Cross-browser access to the WebExtension API.
 *
 * Firefox exposes the promise-based `browser` namespace; Chromium-based browsers
 * expose `chrome`, whose MV3 APIs also return promises. Every other module
 * imports `api` from here instead of touching the globals directly.
 */
export const api = globalThis.browser ?? globalThis.chrome;

/** The extension's own version, used in the X-GoDownload-Client header. */
export const extensionVersion = api.runtime.getManifest().version;

/** Origin prefix of the extension's own pages, e.g. "chrome-extension://abc/". */
export const extensionOrigin = api.runtime.getURL('');
