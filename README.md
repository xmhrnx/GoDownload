<p align="center"><img src="assets/icon-256.png" width="96" alt="GoDownload icon"></p>

# GoDownload

A fast, light, cross-platform download manager written in Go with
[Giode](https://github.com/xmhrnx/giode). Web downloads over many connections,
torrents that don't stall, video streams, playing files while they download,
schedules, power actions, self-repair of damaged files, and a browser
extension.

## Features

**Downloading**
- **Multi-connection downloads.** Each file is split across up to 32
  connections (8 by default). When a connection finishes its part it takes
  half of the biggest part still left, so all connections stay busy until the
  end. HTTP/2 is off on purpose: it would squeeze every connection into one.
- **Resume after anything.** Pausing, quitting, crashes and reboots all
  resume where they stopped. Resume data is written only after the file is
  flushed, so it never claims bytes that didn't reach the disk. If the
  server's file changed (size, ETag or date), the download starts over
  instead of mixing two files.
- **Expired links.** When a file host's link expires, *Refresh link* takes a
  fresh link from the page and continues the same download.
- **Video streams (HLS).** `.m3u8` streams, which the extension finds on
  pages, download at the best quality into one `.ts`/`.mp4`, including
  AES-128 encrypted ones.
- **Queue and limits.** A set number run at once and the rest wait their turn.
  There is a global speed limit, a per-download limit, and per-download
  connection counts.
- **Checksums.** Paste a SHA-256, SHA-1 or MD5 when adding; it's checked when
  the download finishes.

**Torrents** (magnet links, `.torrent` files, info hashes)
- DHT, peer exchange, UPnP/NAT-PMP port forwarding, encryption, and optional
  public trackers added to public torrents. These find peers for torrents
  whose own trackers are dead, which is the usual cause of "stuck at 0%".
- Pick which files to download, before or during the download.
- **Seeding per torrent:** turn it on or off for each torrent, give each its
  own upload limit and a ratio to stop at. There is also a global upload
  limit. Turning seeding off stops uploading at once.

**Play while downloading**
- *Play while downloading* opens the file in your player through a local
  stream URL. The start of the file is fetched first, and **seeking is safe**:
  jumping ahead makes that part download first, and the player waits for it
  instead of crashing or showing garbage. This works for web downloads and
  torrents alike.
- Players: **GoPlay** first when installed, then IINA, mpv, VLC, MPC-HC,
  PotPlayer, Celluloid and others, or any command you set. The protocol,
  including an optional status endpoint for drawing downloaded ranges on a seek
  bar, is in [docs/STREAMING.md](docs/STREAMING.md). It's written so GoPlay or
  any other player can use it.

**Repair**
- Every block of a web download is recorded with a CRC-32C checksum as it's
  written. *Check & repair* re-reads the file, finds blocks that don't match
  (a crash, a bad pause, a disk glitch, an edited file), and downloads just
  those blocks again from the link.
- For torrents, *Check & repair* rehashes every piece against the torrent and
  re-downloads the bad ones. Peers seen before are dialed again so the fix
  doesn't wait for a tracker.

**Organizing**
- Categories (Video, Music, Documents, Compressed, Programs, Images, Other),
  each saved by default to its own folder inside Downloads, e.g.
  `~/Downloads/Video`. Every folder can be changed.
- The list is sortable by name, size, progress, speed, time left and time
  added. The default is newest first by time added, so rows never move by
  themselves. Search, status filters, Ctrl/Shift multi-select and a
  right-click menu are included.
- *Remove from list* keeps the files on disk; *Delete files* also deletes
  them.

**Scheduling and power**
- Schedules have a start time, an optional stop time and weekdays. Each can
  set its own concurrency and speed limit.
- *When downloads finish* (toolbar power button) and each schedule can quit the
  app, sleep, hibernate, sign out, restart or shut down the computer. A
  60-second countdown lets someone at the computer cancel.
- The computer doesn't idle to sleep while downloads run.

**Everything else**
- Custom window frames: Windows-style caption buttons on Windows and Linux,
  traffic lights on macOS.
- Light and dark themes, which can follow the system.
- A tray icon (downloads and seeding continue with the window closed), start
  at login, and notifications.
- Clipboard watching (offers to download copied links), paste links with
  Ctrl+V, and drag `.torrent` files onto the window.
- Single instance: opening a magnet link or a `.torrent` file hands it to the
  running app.

## Browser extension

`extension/` holds a Manifest V3 extension for Chrome, Edge, Brave, Opera and
Firefox. It:
- captures downloads by file type or size, with the browser's cookies, referrer
  and user agent so logged-in downloads work;
- adds *Download with GoDownload* and *Download all links* to the right-click
  menu;
- detects videos and HLS streams on pages, with a badge count and a download
  button in the popup.

```sh
cd extension && ./build.sh     # dist/chrome and dist/firefox, plus zips
```

Then load `dist/chrome` as an unpacked extension (Developer mode in
`chrome://extensions`), or `dist/firefox/manifest.json` in Firefox's
`about:debugging`. The full guide and the security model are in
[extension/README.md](extension/README.md).

The app listens on `127.0.0.1:47821`. It accepts requests only from extension
origins (`chrome-extension://`, `moz-extension://`), never from web pages, and
can require a pairing token (Settings › Browser).

## Building

Go 1.26.6+. Linux also needs Gio's system libraries:

```sh
sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev
```

```sh
go run .                      # run from source
go test ./...                 # engine, manager, server and UI-model tests
giode build                   # this platform, into dist/ (needs the giode CLI)
giode build windows-installer dmg deb appimage
```

Plain `go build` works, but `giode build` adds the Gio patches, which are
needed for drag-and-drop and close-to-tray on Linux, and it embeds the icon and
registers `magnet:` and `.torrent` with the system.

## Memory

The big savings are in the engine. Torrent data goes through ordinary file
reads and writes, not memory-mapped files, which would count every touched page
of every download as the app's memory. HTTP connections stream through a 32 KB
buffer each. The Go heap is capped with a soft 128 MB limit and a low GC target.
In the development container, which has no GPU, the app used about 210 MB RSS
while seeding with the full UI open. An empty Giode window uses about 150 MB
there, almost all of it the Mesa software renderer, so the app's own share is
roughly 60 MB. On a machine with a GPU driver the totals are lower.

## Layout

```
main.go                 wiring only
assets/                 app icon, tray icon
internal/               the engine: no UI imports, unit tested
  httpdl/               multi-connection HTTP: block map, segments, resume, repair, stream reads
  bt/                   torrents (anacrolix/torrent): seeding control, upload caps, streaming readers
  hls/                  HLS playlists, AES-128, ordered segment writer
  manager/              queue, runners, schedules, repair, stream sources, events
  server/               127.0.0.1 API for the extension, stream and status endpoints
  player/               finding and launching media players
  power/                sleep / shut down / restart / sign out per OS
  model/ store/ format/ categorize/ fileutil/ links/ ratelimit/
ui/                     the window (Giode): one component per file
  theme/                palette, Inter font, icons, shared styles
  state/                UI model and actions
  listmodel/            filtering, sorting, selection (tested)
  components/           title bars, sidebar, toolbar, list rows, details panel, …
  dialogs/              add, batch links, remove, refresh link, play, settings, scheduler, power
extension/              the browser extension
docs/STREAMING.md       the player protocol (for GoPlay and others)
```

Data lives in the user config folder (`~/.config/GoDownload`,
`%AppData%\GoDownload`, `~/Library/Application Support/GoDownload`).

## Limitations

- Per-torrent *download* speed limits aren't offered. The torrent library has
  only a client-wide download limiter, which the global limit uses. Upload
  caps are per torrent.
- Live HLS streams, DASH (`.mpd`) and DRM-protected streams can't be downloaded.
- Downloads that come from a browser POST form are fetched with GET; the
  browser doesn't give extensions the original request body.
- On GNOME the tray icon needs the AppIndicator extension.
