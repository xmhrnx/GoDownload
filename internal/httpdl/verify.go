package httpdl

import (
	"context"
	"errors"
	"hash/crc32"
	"io"
	"os"

	"github.com/godownload/godownload/internal/fileutil"
)

// ErrNoChecksums means there is no block data to check the file against.
var ErrNoChecksums = errors.New("no block checksums recorded for this file")

// Check rereads every block recorded as done and forgets the ones whose bytes
// no longer match their checksum (or are missing), so the next Run fetches
// just those again. It returns how many blocks were damaged. The download must
// not be running.
func (d *Downloader) Check(ctx context.Context, progress func(done, total int64)) (int, error) {
	saved := d.loadState()
	if saved == nil {
		return 0, ErrNoChecksums
	}
	m := saved.blocks
	path := d.cfg.Path + fileutil.PartSuffix
	if _, err := os.Stat(path); err != nil {
		path = d.cfg.Path
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			bad := m.Count()
			m.Reset()
			return bad, d.writeBlocks(saved)
		}
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, m.BlockSize)
	bad := 0
	for i := range m.Len() {
		if err := ctx.Err(); err != nil {
			return bad, err
		}
		if !m.Done(i) {
			continue
		}
		start, end := m.Bounds(i)
		n, err := f.ReadAt(buf[:end-start], start)
		if (err != nil && err != io.EOF) || int64(n) != end-start ||
			crc32.Checksum(buf[:n], crcTable) != m.Sum(i) {
			m.Clear(i)
			bad++
		}
		if progress != nil {
			progress(end, m.Size)
		}
	}
	return bad, d.writeBlocks(saved)
}

// writeBlocks saves checked resume data back.
func (d *Downloader) writeBlocks(s *resumeState) error {
	bs := s.blocks.state()
	s.Blocks = &bs
	return fileutil.WriteJSON(d.cfg.StatePath, s)
}

// Forget deletes the resume data, so the next Run downloads the whole file again.
func (d *Downloader) Forget() error {
	if d.cfg.StatePath == "" {
		return nil
	}
	err := os.Remove(d.cfg.StatePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
