package manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/hls"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
)

// StartMode says when a new download begins.
type StartMode int

const (
	StartNow StartMode = iota
	StartPaused
	// StartScheduled waits for a schedule's window.
	StartScheduled
)

// AddRequest describes a new download.
type AddRequest struct {
	Kind    model.Kind
	URL     string
	Headers model.RequestHeaders
	Torrent *bt.Source

	Name     string
	Dir      string
	Category model.Category
	Size     int64

	Connections int
	SpeedLimit  int64
	Resumable   bool
	Checksum    string

	Files       []model.FilePriority
	Seed        bool
	UploadLimit int64
	SeedRatio   float64

	Start StartMode
}

// ErrDuplicate means the same file or torrent is already in the list.
var ErrDuplicate = errors.New("already in the list")

// Add puts a download in the list and, unless asked otherwise, in the queue.
func (m *Manager) Add(r AddRequest) (string, error) {
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		m.changed()
	}()
	d := model.Download{
		ID:             newID(),
		Kind:           r.Kind,
		URL:            r.URL,
		Name:           fileutil.Sanitize(r.Name),
		Dir:            r.Dir,
		Category:       r.Category,
		Size:           r.Size,
		AddedAt:        now(),
		MaxConnections: r.Connections,
		SpeedLimit:     r.SpeedLimit,
		Checksum:       r.Checksum,
		Request:        r.Headers,
		Resumable:      r.Resumable || r.Kind != model.KindHTTP,
		Status:         model.StatusQueued,
	}
	if d.Category == "" {
		d.Category = categorize.ByName(d.Name, "")
	}
	if d.Dir == "" {
		d.Dir = m.dirForLocked(d.Category)
	}
	if r.Kind == model.KindTorrent {
		if r.Torrent == nil {
			return "", errors.New("no torrent given")
		}
		hash, err := bt.HashOf(*r.Torrent)
		if err != nil {
			return "", err
		}
		for _, e := range m.entries {
			if e.d.Torrent != nil && e.d.Torrent.InfoHash == hash {
				return e.d.ID, ErrDuplicate
			}
		}
		if len(r.Torrent.MetaInfo) > 0 {
			if err := m.saveTorrentFileLocked(hash, r.Torrent.MetaInfo); err != nil {
				return "", err
			}
		}
		d.Torrent = &model.TorrentOptions{
			InfoHash: hash, Magnet: r.Torrent.Magnet, Seed: r.Seed,
			UploadLimit: r.UploadLimit, SeedRatio: r.SeedRatio, Files: r.Files,
		}
		if d.Name == "download" {
			d.Name = hash
		}
	} else {
		d.Name = filepath.Base(fileutil.UniquePath(d.Dir, d.Name, m.reservedLocked))
	}
	switch r.Start {
	case StartPaused:
		d.Status, d.Stopped = model.StatusPaused, true
	case StartScheduled:
		d.Status, d.Stopped, d.Scheduled = model.StatusPaused, true, true
	}
	e := newEntry(d)
	e.logf("Added %s", d.URL)
	m.entries[d.ID] = e
	m.hadWork = m.hadWork || r.Start == StartNow
	m.dirty = true
	m.scheduleLocked()
	return d.ID, nil
}

// reservedLocked reports whether another download will write to path.
func (m *Manager) reservedLocked(path string) bool {
	for _, e := range m.entries {
		if e.d.Kind != model.KindTorrent && e.path() == path {
			return true
		}
	}
	return false
}

// DirFor is the folder a category's downloads go to by default.
func (m *Manager) DirFor(c model.Category) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dirForLocked(c)
}

func (m *Manager) dirForLocked(c model.Category) string {
	if d := m.settings.CategoryDirs[c]; d != "" {
		return d
	}
	return filepath.Join(m.settings.BaseDir, c.Title())
}

func (m *Manager) saveTorrentFileLocked(hash string, data []byte) error {
	eng, err := m.torrentEngine()
	if err != nil {
		return err
	}
	_, err = eng.Inspect(context.Background(), bt.Source{MetaInfo: data})
	return err
}

// Probed is what the add dialog shows about a link before it's added.
type Probed struct {
	Kind      model.Kind
	URL       string
	Name      string
	Size      int64
	Resumable bool
	Category  model.Category
	Dir       string
	Duplicate string // ID of an unfinished download of the same URL
}

// Probe asks the server about a link.
func (m *Manager) Probe(ctx context.Context, rawURL string, h model.RequestHeaders) (Probed, error) {
	m.mu.Lock()
	client := m.client
	var dup string
	for _, e := range m.entries {
		if e.d.URL == rawURL && !e.d.Status.Finished() {
			dup = e.d.ID
		}
	}
	m.mu.Unlock()

	req := httpdl.Request{URL: rawURL, Headers: h}
	if looksLikeHLS(rawURL) {
		return m.probeHLS(ctx, client, req, dup)
	}
	info, err := httpdl.Probe(ctx, client, req)
	if err != nil {
		return Probed{}, err
	}
	if strings.Contains(strings.ToLower(info.ContentType), "mpegurl") {
		req.URL = info.URL
		return m.probeHLS(ctx, client, req, dup)
	}
	cat := categorize.ByName(info.Name, info.ContentType)
	return Probed{Kind: model.KindHTTP, URL: info.URL, Name: info.Name, Size: info.Size,
		Resumable: info.Ranges, Category: cat, Dir: m.DirFor(cat), Duplicate: dup}, nil
}

func (m *Manager) probeHLS(ctx context.Context, client *http.Client, req httpdl.Request, dup string) (Probed, error) {
	ext, err := hls.Extension(ctx, client, req)
	if err != nil {
		return Probed{}, err
	}
	base := fileutil.NameFromURL(req.URL)
	name := strings.TrimSuffix(base, filepath.Ext(base)) + ext
	return Probed{Kind: model.KindHLS, URL: req.URL, Name: name, Size: -1, Resumable: true,
		Category: model.CategoryVideo, Dir: m.DirFor(model.CategoryVideo), Duplicate: dup}, nil
}

func looksLikeHLS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
}

// InspectTorrent fetches a torrent's file list for the add dialog.
func (m *Manager) InspectTorrent(ctx context.Context, src bt.Source) (bt.Meta, error) {
	m.mu.Lock()
	eng, err := m.torrentEngine()
	m.mu.Unlock()
	if err != nil {
		return bt.Meta{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	meta, err := eng.Inspect(ctx, src)
	if errors.Is(err, context.DeadlineExceeded) {
		return meta, fmt.Errorf("no peers sent the torrent's details in time; it may have no seeders")
	}
	return meta, err
}
