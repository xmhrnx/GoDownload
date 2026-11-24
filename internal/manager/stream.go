package manager

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
)

// Media is a file the stream server can serve while it downloads.
type Media struct {
	Name string
	Size int64
	// Open returns a reader whose reads wait for data that hasn't arrived,
	// and make the bytes being read the most urgent download.
	Open func(ctx context.Context) (io.ReadSeekCloser, error)
}

// ErrNotStreamable means the download can't be played until it finishes.
var ErrNotStreamable = errors.New("this download can only be opened once it finishes")

// Media prepares file (a torrent's file index; 0 otherwise) of a download for
// streaming, starting the download if it isn't running.
func (m *Manager) Media(ctx context.Context, id string, file int) (Media, error) {
	m.mu.Lock()
	e, err := m.get(id)
	if err != nil {
		m.mu.Unlock()
		return Media{}, err
	}
	kind, status, path := e.d.Kind, e.d.Status, e.path()
	m.mu.Unlock()

	if kind != model.KindTorrent && status == model.StatusCompleted {
		return fileMedia(path)
	}
	switch kind {
	case model.KindHTTP:
		return m.httpMedia(ctx, e)
	case model.KindTorrent:
		return m.torrentMedia(ctx, e, file)
	}
	return Media{}, ErrNotStreamable
}

func fileMedia(path string) (Media, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Media{}, err
	}
	return Media{Name: st.Name(), Size: st.Size(), Open: func(context.Context) (io.ReadSeekCloser, error) {
		return os.Open(path)
	}}, nil
}

// acquireStream counts a reader on e and makes sure e is downloading.
func (m *Manager) acquireStream(e *entry) {
	m.mu.Lock()
	e.streams++
	e.d.Stopped = false
	if !e.running() && !e.busy {
		if e.d.Status != model.StatusSeeding && e.d.Status != model.StatusCompleted {
			e.d.Status = model.StatusQueued
		}
		if !e.d.Status.Finished() {
			m.startLocked(e)
		}
	}
	m.mu.Unlock()
	m.changed()
}

func (m *Manager) releaseStream(e *entry) {
	m.mu.Lock()
	e.streams--
	if e.streams == 0 && !e.running() && e.torrent != nil {
		e.torrent.Close()
		e.torrent = nil
	}
	m.scheduleLocked()
	m.mu.Unlock()
}

func (m *Manager) httpMedia(ctx context.Context, e *entry) (Media, error) {
	m.acquireStream(e)
	m.mu.Lock()
	dl := m.httpDownloaderLocked(e)
	name := e.d.Name
	m.mu.Unlock()
	size, err := waitSize(ctx, dl)
	if err != nil {
		m.releaseStream(e)
		return Media{}, err
	}
	m.releaseStream(e) // each Open holds its own count
	return Media{Name: name, Size: size, Open: func(ctx context.Context) (io.ReadSeekCloser, error) {
		m.acquireStream(e)
		return &httpStream{ctx: ctx, dl: dl, size: size, release: func() { m.releaseStream(e) }}, nil
	}}, nil
}

func waitSize(ctx context.Context, dl *httpdl.Downloader) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if s := dl.Size(); s >= 0 {
			return s, nil
		}
		select {
		case <-ctx.Done():
			return 0, errors.New("the server didn't say how big the file is, so it can't be streamed")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// httpStream reads an HTTP download as it arrives.
type httpStream struct {
	ctx     context.Context
	dl      *httpdl.Downloader
	off     int64
	size    int64
	release func()
	closed  bool
}

func (s *httpStream) Read(p []byte) (int, error) {
	if s.off >= s.size {
		return 0, io.EOF
	}
	avail, err := s.dl.WaitAvailable(s.ctx, s.off)
	if err != nil {
		return 0, err
	}
	n, err := s.dl.ReadAt(p[:min(int64(len(p)), avail)], s.off)
	s.off += int64(n)
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}

func (s *httpStream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		offset += s.off
	case io.SeekEnd:
		offset += s.size
	}
	if offset < 0 {
		return 0, errors.New("negative position")
	}
	s.off = offset
	return offset, nil
}

func (s *httpStream) Close() error {
	if !s.closed {
		s.closed = true
		s.release()
	}
	return nil
}

func (m *Manager) torrentMedia(ctx context.Context, e *entry, file int) (Media, error) {
	m.acquireStream(e)
	m.mu.Lock()
	h, err := m.openTorrentLocked(e)
	running := e.running()
	window := int64(max(m.settings.StreamBufferMB, 4)) << 20
	m.mu.Unlock()
	if err != nil {
		m.releaseStream(e)
		return Media{}, err
	}
	if !running {
		h.SetSeed(false) // opened just to read finished data
	}
	if err := h.WaitInfo(ctx); err != nil {
		m.releaseStream(e)
		return Media{}, err
	}
	files := h.Files()
	if file < 0 || file >= len(files) {
		m.releaseStream(e)
		return Media{}, errors.New("no such file in the torrent")
	}
	m.mu.Lock()
	if p := e.d.Torrent.Files; file < len(p) && p[file] == model.PrioritySkip {
		p[file] = model.PriorityNormal
		h.SetPriorities(p)
	}
	m.mu.Unlock()
	h.PrioritizeEnds(file)
	m.releaseStream(e)
	return Media{Name: files[file].Path, Size: files[file].Size, Open: func(ctx context.Context) (io.ReadSeekCloser, error) {
		m.acquireStream(e)
		r := h.Reader(ctx, file, window)
		return &releasingReader{ReadSeekCloser: r, release: func() { m.releaseStream(e) }}, nil
	}}, nil
}

type releasingReader struct {
	io.ReadSeekCloser
	release func()
	closed  bool
}

func (r *releasingReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	err := r.ReadSeekCloser.Close()
	r.release()
	return err
}

// StreamFiles lists the files a download can stream: a torrent's files (its
// metadata is fetched if needed), or the one file of anything else.
func (m *Manager) StreamFiles(ctx context.Context, id string) ([]model.TorrentFile, error) {
	m.mu.Lock()
	e, err := m.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if e.d.Kind != model.KindTorrent {
		f := model.TorrentFile{Path: e.d.Name, Size: e.d.Size, Done: e.d.Done}
		hlsBusy := e.d.Kind == model.KindHLS && e.d.Status != model.StatusCompleted
		m.mu.Unlock()
		if hlsBusy {
			return nil, ErrNotStreamable
		}
		return []model.TorrentFile{f}, nil
	}
	m.mu.Unlock()
	m.acquireStream(e)
	defer m.releaseStream(e)
	m.mu.Lock()
	h, err := m.openTorrentLocked(e)
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if err := h.WaitInfo(ctx); err != nil {
		return nil, err
	}
	return h.Files(), nil
}
