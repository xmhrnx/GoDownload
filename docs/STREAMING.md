# Playing while downloading — the player protocol

GoDownload streams a download to any media player over plain HTTP on
`127.0.0.1`. Nothing here is GoDownload-specific for basic playback: a player
that can open a network URL and seek with `Range` requests works. This page is
for player authors (GoPlay first) who want more: showing what is downloaded on
the seek bar.

## The stream URL

When the user clicks **Play while downloading**, GoDownload launches the
player with one argument, the stream URL:

```
http://127.0.0.1:<port>/stream/<token>/<id>/<file>/<name>
```

- `<port>` is GoDownload's local port: the first free one of 47821, 18821,
  28821, 38821, 58821 and 8821. Use the URL as given rather than building it.
- `<token>` is random per GoDownload session, so other local pages can't guess URLs.
- `<id>` is the download, `<file>` the file index (torrents have several; 0 otherwise).
- `<name>` is the file name, for players that show it.

mpv and VLC also get the title (`--force-media-title=`, `--meta-title=`).

## Seeking

The stream answers `Range` requests with `206 Partial Content` and the full
`Content-Length`, as a file server would. A range that isn't downloaded yet
does **not** fail: GoDownload moves a connection (HTTP) or the piece priority
(torrents) to that position, and the response body starts as soon as the first
bytes arrive. Players see a slow network read, not an error, so seeking ahead
is safe; the read-ahead window (Settings › Streaming) keeps fetching just past
the play position.

Players should use a read timeout of at least 60 seconds for the first bytes
after a seek into a part that isn't downloaded.

## Download status (optional)

Every stream response carries a header pointing at its status:

```
X-GoDownload-Status: http://127.0.0.1:47821/status/<token>/<id>/<file>
```

`GET` it for JSON (CORS-enabled):

```json
{
  "name": "Big.Buck.Bunny.2160p.mkv",
  "size": 419430400,
  "done": 131072000,
  "complete": false,
  "speed": 3145728,
  "status": "downloading",
  "available": [[0, 41943040], [104857600, 125829120]]
}
```

- `available` lists byte ranges fully on disk, coarse (about 120 slices of
  the file), in order. Draw them on the seek bar as "buffered".
- `status` is one of `queued`, `connecting`, `downloading`, `paused`,
  `verifying`, `seeding`, `completed`, `failed`.
- Poll about once a second while playing; stop when `complete` is true.

## Making a player the default

GoDownload looks for these, in order, when the player setting is
*Automatic*: GoPlay, then IINA, mpv, VLC, MPC-HC, PotPlayer, Celluloid,
SMPlayer and Haruna, depending on the system. GoPlay is found as `goplay` on
the `PATH`, or in its usual install folder (`GoPlay.app` on macOS,
`%LOCALAPPDATA%\Programs\GoPlay` on Windows). Users can also enter any command
with `{url}` and `{title}` placeholders.
