// Package hls downloads HTTP Live Streaming videos (.m3u8) into one file.
package hls

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// Segment is one media chunk of the stream.
type Segment struct {
	URL      string
	Duration float64
	Key      *Key
	Seq      int64
}

// Key is how a segment is encrypted.
type Key struct {
	Method string // "AES-128"
	URL    string
	IV     []byte // nil: derived from the sequence number
}

// Playlist is a parsed media playlist.
type Playlist struct {
	Segments []Segment
	// Init is the fMP4 initialization segment (EXT-X-MAP), when there is one.
	Init string
	// Ended is false for live streams, which never finish.
	Ended bool
}

// Variant is one quality level of a master playlist.
type Variant struct {
	URL       string
	Bandwidth int64
	Height    int
}

// ErrLive means the playlist is a live stream with no end.
var ErrLive = errors.New("live streams can't be downloaded yet")

// parse reads a playlist. A master playlist returns its variants and no media.
func parse(r io.Reader, base *url.URL) (*Playlist, []Variant, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	if !sc.Scan() || !strings.HasPrefix(strings.TrimSpace(sc.Text()), "#EXTM3U") {
		return nil, nil, errors.New("not an HLS playlist")
	}
	pl := &Playlist{}
	var variants []Variant
	var key *Key
	var seq int64
	var dur float64
	var pending *Variant
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "":
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:"):
			a := attrs(line[len("#EXT-X-STREAM-INF:"):])
			v := Variant{}
			v.Bandwidth, _ = strconv.ParseInt(a["BANDWIDTH"], 10, 64)
			if res := a["RESOLUTION"]; res != "" {
				if i := strings.IndexByte(res, 'x'); i > 0 {
					v.Height, _ = strconv.Atoi(res[i+1:])
				}
			}
			pending = &v
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			seq, _ = strconv.ParseInt(line[len("#EXT-X-MEDIA-SEQUENCE:"):], 10, 64)
		case strings.HasPrefix(line, "#EXTINF:"):
			v := strings.TrimSuffix(strings.SplitN(line[len("#EXTINF:"):], ",", 2)[0], ",")
			dur, _ = strconv.ParseFloat(v, 64)
		case strings.HasPrefix(line, "#EXT-X-KEY:"):
			k, err := parseKey(attrs(line[len("#EXT-X-KEY:"):]), base)
			if err != nil {
				return nil, nil, err
			}
			key = k
		case strings.HasPrefix(line, "#EXT-X-MAP:"):
			u, err := resolve(base, attrs(line[len("#EXT-X-MAP:"):])["URI"])
			if err != nil {
				return nil, nil, err
			}
			pl.Init = u
		case line == "#EXT-X-ENDLIST":
			pl.Ended = true
		case strings.HasPrefix(line, "#"):
		default:
			u, err := resolve(base, line)
			if err != nil {
				return nil, nil, err
			}
			if pending != nil {
				pending.URL = u
				variants = append(variants, *pending)
				pending = nil
				continue
			}
			pl.Segments = append(pl.Segments, Segment{URL: u, Duration: dur, Key: key, Seq: seq})
			seq++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	return pl, variants, nil
}

func parseKey(a map[string]string, base *url.URL) (*Key, error) {
	switch a["METHOD"] {
	case "NONE", "":
		return nil, nil
	case "AES-128":
	default:
		return nil, fmt.Errorf("%s encryption isn't supported (DRM-protected streams can't be downloaded)", a["METHOD"])
	}
	u, err := resolve(base, a["URI"])
	if err != nil {
		return nil, err
	}
	k := &Key{Method: "AES-128", URL: u}
	if iv := a["IV"]; iv != "" {
		iv = strings.TrimPrefix(strings.TrimPrefix(iv, "0x"), "0X")
		b := make([]byte, 16)
		for i := 0; i < 16 && 2*i+2 <= len(iv); i++ {
			v, err := strconv.ParseUint(iv[2*i:2*i+2], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("bad IV")
			}
			b[i] = byte(v)
		}
		k.IV = b
	}
	return k, nil
}

// attrs parses KEY=VALUE,KEY="quoted, value" attribute lists.
func attrs(s string) map[string]string {
	out := map[string]string{}
	for len(s) > 0 {
		eq := strings.IndexByte(s, '=')
		if eq < 0 {
			break
		}
		k := strings.TrimSpace(s[:eq])
		s = s[eq+1:]
		var v string
		if strings.HasPrefix(s, `"`) {
			end := strings.IndexByte(s[1:], '"')
			if end < 0 {
				end = len(s) - 1
			}
			v, s = s[1:end+1], s[min(end+2, len(s)):]
		} else if comma := strings.IndexByte(s, ','); comma >= 0 {
			v, s = s[:comma], s[comma:]
		} else {
			v, s = s, ""
		}
		out[k] = v
		s = strings.TrimPrefix(s, ",")
	}
	return out
}

func resolve(base *url.URL, ref string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return "", err
	}
	return base.ResolveReference(u).String(), nil
}

// best picks the highest-bandwidth variant.
func best(vs []Variant) Variant {
	b := vs[0]
	for _, v := range vs[1:] {
		if v.Bandwidth > b.Bandwidth || (v.Bandwidth == b.Bandwidth && v.Height > b.Height) {
			b = v
		}
	}
	return b
}
