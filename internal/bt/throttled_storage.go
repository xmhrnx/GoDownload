package bt

import (
	"context"
	"io"
	"sync"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"

	"github.com/godownload/godownload/internal/ratelimit"
)

// throttledStorage gives one torrent its own upload speed limit.
//
// The torrent library has a single upload limiter for the whole client. But
// every byte sent to a peer is first read through the storage's torrent
// reader, so a reader that waits on this torrent's limiter caps its upload
// exactly. Hash checks read pieces through another path and stay at full
// speed, and players read through raw(), which isn't limited either.
type throttledStorage struct {
	storage.ClientImplCloser
	limiter *ratelimit.Limiter

	mu  sync.Mutex
	pcs *pieceReader
}

func newThrottledStorage(inner storage.ClientImplCloser) *throttledStorage {
	return &throttledStorage{ClientImplCloser: inner, limiter: ratelimit.New(0)}
}

// OpenTorrent opens the files and adds the throttled reader.
func (s *throttledStorage) OpenTorrent(ctx context.Context, info *metainfo.Info, ih metainfo.Hash) (storage.TorrentImpl, error) {
	ti, err := s.ClientImplCloser.OpenTorrent(ctx, info, ih)
	if err != nil {
		return ti, err
	}
	pr := &pieceReader{ti: ti, info: info}
	s.mu.Lock()
	s.pcs = pr
	s.mu.Unlock()
	ti.NewReader = func() storage.TorrentReader { return &throttledReader{pr, s.limiter} }
	return ti, nil
}

// raw reads the torrent's data without the upload limit, once storage is open.
func (s *throttledStorage) raw() *pieceReader {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcs
}

// pieceReader reads torrent offsets from the per-piece storage.
type pieceReader struct {
	ti   storage.TorrentImpl
	info *metainfo.Info
}

func (r *pieceReader) piece(i int) storage.PieceImpl {
	p := r.info.Piece(i)
	if r.ti.Piece != nil {
		return r.ti.Piece(p)
	}
	return r.ti.PieceWithHash(p, g.None[[]byte]())
}

// ReadAt reads b at a torrent offset, across pieces.
func (r *pieceReader) ReadAt(b []byte, off int64) (n int, err error) {
	total := r.info.TotalLength()
	for len(b) > 0 {
		if off >= total {
			return n, io.EOF
		}
		i := int(off / r.info.PieceLength)
		p := r.info.Piece(i)
		within := off - p.Offset()
		chunk := b[:min(int64(len(b)), p.Length()-within)]
		m, rerr := r.piece(i).ReadAt(chunk, within)
		n += m
		off += int64(m)
		b = b[m:]
		if rerr != nil && !(rerr == io.EOF && m == len(chunk)) {
			return n, rerr
		}
		if m == 0 {
			return n, io.ErrUnexpectedEOF
		}
	}
	return n, nil
}

// Close does nothing: the pieces belong to the storage.
func (r *pieceReader) Close() error { return nil }

// throttledReader is the reader peers' requests are served from.
type throttledReader struct {
	r       *pieceReader
	limiter *ratelimit.Limiter
}

func (t *throttledReader) ReadAt(b []byte, off int64) (int, error) {
	n, err := t.r.ReadAt(b, off)
	if werr := t.limiter.Wait(context.Background(), n); werr != nil && err == nil {
		err = werr
	}
	return n, err
}

func (t *throttledReader) Close() error { return nil }
