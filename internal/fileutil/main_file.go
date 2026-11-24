package fileutil

import (
	"io/fs"
	"path/filepath"
)

// MainFile is the biggest file under dir: for a torrent folder, usually the
// film or album that "Open" should open rather than the folder itself.
func MainFile(dir string) (string, int64) {
	var best string
	var size int64 = -1
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // unreadable entries are skipped; the walk goes on
		}
		if info, err := d.Info(); err == nil && info.Size() > size {
			best, size = p, info.Size()
		}
		return nil
	})
	return best, size
}
