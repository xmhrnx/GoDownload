# GoDownload browser extension

Sends browser downloads, streaming media and page links to the **GoDownload**
desktop download manager running on the same computer.

- Manifest V3, plain ES2020 modules, no bundler and no npm dependencies.
- Chrome, Edge, Brave, Opera (Chromium 110+) and Firefox 109+.

## Building

```sh
./build.sh            # builds both targets
./build.sh chrome     # or just one: chrome | firefox
```

On Windows, double-click `build.bat` or run it from a Command Prompt
(`build.bat`, `build.bat chrome`). It needs nothing beyond Windows 10 or 11:
it copies with `robocopy` and zips with PowerShell.

This produces:

| Output | Contents |
| --- | --- |
| `dist/chrome/`, `dist/firefox/` | Unpacked extension, ready to load |
| `dist/godownload-<target>-<version>.zip` | Package for store upload |

The script copies `src/`, the matching `manifests/<target>.json` (as
`manifest.json`) and `icons/icon-{16,32,48,128}.png`. If any icon is missing it
prints a warning and removes the icon entries from the built manifest, because
Chrome refuses to load an extension whose manifest names a missing file. Add
the PNGs to `icons/` and rebuild before publishing.

`zip` is used when installed; otherwise the archive is written with `python3`.

## Installing for development

**Chrome / Edge / Brave / Opera**

1. Run `./build.sh chrome`.
2. Open `chrome://extensions` (`edge://extensions`, `brave://extensions`, `opera://extensions`).
3. Turn on **Developer mode**.
4. Click **Load unpacked** and choose `dist/chrome`.

**Firefox**

1. Run `./build.sh firefox`.
2. Open `about:debugging#/runtime/this-firefox`.
3. Click **Load Temporary Add-on…** and choose `dist/firefox/manifest.json`
   (or the zip file).
4. Firefox treats MV3 host permissions as opt-in. Open the GoDownload popup and
   click **Allow** in the yellow notice, or grant "Access your data for all
   websites" in `about:addons`. Without it, the extension can't call the local
   app (requests become CORS-restricted), read cookies, detect media or collect
   links.

Temporary add-ons are removed when Firefox restarts. For a permanent install the
package has to be signed through addons.mozilla.org.

The options page opens automatically on first install. There is no port to
set: the extension finds the app on its own. Only if the desktop app asks for
pairing, paste its token.

## Features

### Download capture
When the browser starts a download, the extension decides whether GoDownload
should take it instead:

- **Capture downloads** (popup and options) must be on, and the app must report
  `interceptEnabled: true`.
- Only `http`, `https` and `ftp` URLs. `blob:`, `data:`, `file:` and
  `filesystem:` downloads always stay in the browser, because no other program
  can fetch them.
- Neither the file's site nor the page it came from is on the **excluded
  sites** list. Subdomains are included.
- The file type is on the **capture list** (archives, installers, disk images,
  video, audio, documents, torrents; editable), **or** the browser knows the
  size and it is at least the **minimum size** (default 1 MB; 0 turns this rule
  off). A download of unknown size is captured only by file type.
- Downloads in private/incognito windows are never captured, so they don't end
  up in the app's history.

If the app isn't running (checked with a ping that times out after 800 ms and is
cached for 5 s), the browser downloads the file as usual. Otherwise the browser
download is cancelled and removed from download history. The URL, cookies,
referrer, user agent, MIME type and size are then sent to GoDownload. If that
request fails, the extension starts the download again in the browser and skips
that URL for 60 seconds so it doesn't capture its own retry. The popup explains
what happened.

Chromium uses `downloads.onDeterminingFilename`, which provides the final URL,
MIME type, size and suggested file name. Firefox uses `downloads.onCreated`.

**Send next download to browser** (popup) is a one-time pass. It applies to the
next download that would otherwise have been captured, then turns itself off.

### Right-click menu
- **Download with GoDownload** on links, images, video and audio. Video and
  audio send the media source. Other elements send the link if there is one,
  otherwise the source. Cookies for the URL and the page's referrer are
  included.
- **Send magnet link to GoDownload** on `magnet:` links. Chromium never reports
  magnets as downloads, so this needs no content script. If a browser rejects
  the `magnet:*` menu pattern, the generic item handles magnet links instead.
- **Download all links with GoDownload** (page menu) scans the page and all of
  its frames for `a[href]`, `area[href]` and `video`/`audio`/`source` sources.
  It keeps `http(s)` and `magnet` links, removes duplicates, and sends them as
  one batch. GoDownload shows its own picker for batches.
- **Download selected links with GoDownload** does the same for links inside
  the current selection. It also picks up URLs written as plain text.

The toolbar badge shows **OK** or **!** briefly to confirm each action.

### Media detection
The extension inspects network responses (`webRequest.onHeadersReceived`) and
records, for each tab:

- **HLS** playlists (`.m3u8`, `application/vnd.apple.mpegurl`,
  `application/x-mpegurl`, `audio/mpegurl`)
- **DASH** manifests (`.mpd`, `application/dash+xml`)
- **Progressive** video and audio (`video/*`, `audio/*`, or mp4/webm/mkv/mov/
  m4a/mp3/ogg/flac/… served as a generic type)

It skips stream segments: `.ts`, `.m4s` and CMAF fragments, `video/mp2t`, URLs
with byte-range or segment parameters, numbered segment/chunk/fragment paths,
and progressive responses under 512 KB. Duplicates are matched by URL with
volatile query parameters (signatures, expiry, cache busters) removed. The full
URL is kept for the download.

Each tab's list is cleared when the tab loads a new page, and dropped when the
tab closes. The lists are saved in `storage.session` so they survive the MV3
service worker being stopped. The badge shows how many items were found on the
active tab. You can turn the badge off in options.

### Popup
- Connection status: **Connected**, **App not running**, or **Token
  required**. Click it to check again.
- The two toggles described above.
- **Detected media on this page**, newest first, with type (HLS / DASH / Video /
  Audio), size when known, and a **Download** button for each item.
- **Options** and **Download all links on this page**.
- Light and dark themes follow the system setting. Everything works from the
  keyboard.

### Options
Port, pairing token (with a show/hide button), file types to capture (with
**Reset to defaults**), minimum size, excluded sites, **Add captured downloads
without asking** (sends `silent: true`), **Show detected-media badge**, and
**Test connection** (tests the values in the form, even unsaved ones).

Settings are saved in `storage.sync`, or `storage.local` if sync isn't
available. The **token is always stored in `storage.local` only**:
they belong to this computer's app, and the token is a secret that shouldn't be
uploaded to a browser-sync account.

## Security model

- **The extension only talks to the local app.** All API calls go to
  `http://127.0.0.1:<port>/api/v1/...` from the background script or the
  extension's own pages. They never come from content scripts, so the app can
  accept requests only from `chrome-extension://`, `moz-extension://` and
  `safari-web-extension://` origins. Requests send
  `X-GoDownload-Client: extension/<version>` and, when set,
  `Authorization: Bearer <token>`. They use `credentials: "omit"`, so the
  browser's own cookies for 127.0.0.1 are never attached, and they refuse
  redirects.
- **Only the app receives cookies.** Each download carries the Cookie header for
  that one URL so the app can download with your session. Nothing is sent to
  any other server, and there is no telemetry.
- **Web pages can't control the extension.** The background accepts messages
  only from the extension's own pages, checked by sender URL. The media
  download command looks the URL up in the tab's detected list instead of
  trusting the message. The link collector runs in the isolated world, only
  reads the DOM, and its results are filtered again in the background.
- **No remote or dynamic code.** No `eval`, no remote scripts, no inline scripts
  or event handlers. The extension-page CSP is `script-src 'self'; object-src
  'self'`. It is set explicitly to replace Firefox's MV3 default, which includes
  `upgrade-insecure-requests` and could rewrite the loopback `http://` calls.
- **Downloads fail safe.** If the app can't be reached, or refuses or fails the
  request, the browser downloads the file. Private-window downloads are never
  captured.
- **Why each permission is needed:** `downloads` to capture, cancel and restart
  downloads. `cookies` to forward session cookies. `webRequest` to detect
  media, using response headers only, non-blocking. `webNavigation` and `tabs`
  to reset each tab's list and read page titles. `scripting` to collect links
  on request. `contextMenus` and `storage` for the menus and settings.
  `<all_urls>` because downloads and media can come from any site.

## Source layout

```
src/
  background.js          entry point: registers every listener synchronously
  lib/
    browser.js           `browser ?? chrome` namespace, version, extension origin
    log.js               [GoDownload]-prefixed logging, safe async wrapper
    settings.js          defaults, normalisation, sync/local storage, change events
    session-store.js     storage.session wrapper (in-memory fallback on Firefox < 115)
    api.js               desktop-app HTTP client (ping cache, timeouts, errors)
    payload.js           builds the exact POST body for the API
    cookies.js           Cookie header for a URL (incl. partitioned cookies)
    url-utils.js         URL / file name helpers
    filters.js           pure capture rules
    bypass.js            loop-protection set and one-shot "skip next"
    activity.js          last error, shown in the popup
    interceptor.js       browser download capture
    media-detect.js      pure HLS / DASH / progressive classification
    media-store.js       per-tab detected media (memory + storage.session)
    media-sniffer.js     webRequest / navigation / tab listeners
    badge.js             toolbar badge count and feedback flashes
    link-collector.js    runs the link collector, builds batch requests
    context-menus.js     right-click menu entries and actions
    messages.js          message router for the popup and options page
  content/collect-links.js   injected on demand, returns [{url, text}]
  popup/                 popup.html / popup.css / popup.js
  options/               options.html / options.css / options.js
  ui/                    shared theme.css, messaging and formatting helpers
manifests/chrome.json, manifests/firefox.json
icons/                   icon-16/32/48/128.png (provided separately)
```
