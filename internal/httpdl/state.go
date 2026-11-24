package httpdl

import (
	"github.com/godownload/godownload/internal/fileutil"
)

// resumeState is what survives a restart: which blocks are on disk, and how
// to tell whether the server still has the same file.
type resumeState struct {
	ETag         string      `json:"etag,omitempty"`
	LastModified string      `json:"lastModified,omitempty"`
	Size         int64       `json:"size"`
	Blocks       *blockState `json:"blocks,omitempty"`

	blocks *BlockMap
}

// matches reports whether info describes the file the state was saved for.
// A changed ETag or date means the bytes may differ, so the old blocks can't
// be mixed with new ones.
func (s *resumeState) matches(info Info) bool {
	if s.blocks == nil || s.Size != info.Size {
		return false
	}
	if s.ETag != "" && info.ETag != "" && s.ETag != info.ETag {
		return false
	}
	if s.LastModified != "" && info.LastModified != "" && s.LastModified != info.LastModified {
		return false
	}
	return true
}

// loadState reads saved resume data. Missing or unreadable data returns nil:
// the download then starts over, which is always safe.
func (d *Downloader) loadState() *resumeState {
	if d.cfg.StatePath == "" {
		return nil
	}
	var s resumeState
	if err := fileutil.ReadJSON(d.cfg.StatePath, &s); err != nil {
		d.logf("resume data unreadable, starting over: %v", err)
		return nil
	}
	if s.Blocks == nil {
		return nil
	}
	m, err := blockMapFromState(*s.Blocks)
	if err != nil {
		d.logf("resume data unreadable, starting over: %v", err)
		return nil
	}
	s.blocks = m
	return &s
}

// saveState flushes the file and then records which blocks are done. The
// order matters: a block is only recorded once its bytes are on disk, so a
// crash can't leave resume data claiming data that was never written.
func (d *Downloader) saveState() {
	if d.cfg.StatePath == "" {
		return
	}
	d.mu.Lock()
	if d.blocks == nil {
		d.mu.Unlock()
		return
	}
	bs := d.blocks.state()
	s := resumeState{ETag: d.info.ETag, LastModified: d.info.LastModified, Size: d.info.Size, Blocks: &bs}
	d.mu.Unlock()

	d.fileMu.RLock()
	var err error
	if d.file != nil {
		err = d.file.Sync()
	}
	d.fileMu.RUnlock()
	if err != nil {
		d.logf("flushing file: %v", err)
		return
	}
	if err := fileutil.WriteJSON(d.cfg.StatePath, s); err != nil {
		d.logf("saving resume data: %v", err)
	}
}

func (d *Downloader) logf(format string, args ...any) {
	if d.cfg.Log != nil {
		d.cfg.Log(format, args...)
	}
}
