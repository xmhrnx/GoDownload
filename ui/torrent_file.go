package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/godownload/godownload/ui/dialogs"
)

// torrentFile reads a .torrent file given by path.
func torrentFile(path string) (dialogs.Incoming, bool) {
	if !strings.EqualFold(filepath.Ext(path), ".torrent") {
		return dialogs.Incoming{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 32<<20 {
		return dialogs.Incoming{}, false
	}
	return dialogs.Incoming{URL: filepath.Base(path), MetaInfo: data}, true
}
