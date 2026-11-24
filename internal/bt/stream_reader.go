package bt

import (
	"context"
	"errors"
	"io"

	"github.com/anacrolix/torrent"
)

// StreamReader reads one file of a torrent for a player, waiting for pieces
// that haven't arrived and fetching the ones at the read position first.
//
// The library's reader decides what to fetch and waits for each piece to be
// verified; the bytes themselves are read from storage directly, so a
// torrent's upload limit never slows down playback.
type StreamReader struct {
	h      *Handle
	f      *torrent.File
	r      torrent.Reader
	pos    int64
	synced bool
}

// Reader opens a stream of file i, fetching readahead bytes ahead of the
// read position first. Reads end when ctx does.
func (h *Handle) Reader(ctx context.Context, i int, readahead int64) *StreamReader {
	f := h.t.Files()[i]
	r := f.NewReader()
	r.SetReadahead(readahead)
	r.SetContext(ctx)
	return &StreamReader{h: h, f: f, r: r}
}

// Read waits until the piece at the position is verified, then reads up to
// its end.
func (s *StreamReader) Read(p []byte) (int, error) {
	size := s.f.Length()
	if s.pos >= size {
		return 0, io.EOF
	}
	if !s.synced {
		if _, err := s.r.Seek(s.pos, io.SeekStart); err != nil {
			return 0, err
		}
		s.synced = true
	}
	// One byte through the library's reader: it waits for the piece and
	// keeps the pieces ahead prioritized.
	var one [1]byte
	if _, err := s.r.Read(one[:]); err != nil {
		return 0, err
	}
	raw := s.h.store.raw()
	if raw == nil {
		return 0, errors.New("torrent storage isn't open")
	}
	off := s.f.Offset() + s.pos
	pieceLen := raw.info.PieceLength
	pieceEnd := (off/pieceLen + 1) * pieceLen
	n := min(int64(len(p)), pieceEnd-off, size-s.pos)
	m, err := raw.ReadAt(p[:n], off)
	s.pos += int64(m)
	// The library's reader advanced one byte; keep it at our position.
	s.synced = false
	if err == io.EOF && m > 0 {
		err = nil
	}
	return m, err
}

// Seek moves the read position within the file.
func (s *StreamReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		offset += s.pos
	case io.SeekEnd:
		offset += s.f.Length()
	}
	if offset < 0 {
		return 0, errors.New("negative position")
	}
	s.pos, s.synced = offset, false
	return offset, nil
}

// Close ends the stream.
func (s *StreamReader) Close() error { return s.r.Close() }
