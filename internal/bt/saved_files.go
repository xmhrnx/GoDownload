package bt

import (
	"path/filepath"
	"strings"

	"github.com/anacrolix/torrent/metainfo"

	"github.com/godownload/godownload/internal/model"
)

// SavedFiles lists a torrent's files from the .torrent saved in dataDir,
// without starting the torrent client, for torrents that aren't running.
// Done is -1 (unknown) for every file.
func SavedFiles(dataDir, infoHash string) ([]model.TorrentFile, error) {
	mi, err := metainfo.LoadFromFile(filepath.Join(dataDir, "torrents", infoHash+".torrent"))
	if err != nil {
		return nil, err
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []model.TorrentFile{{Path: info.BestName(), Size: info.TotalLength(), Done: -1, Priority: model.PriorityNormal}}, nil
	}
	files := info.UpvertedFiles()
	out := make([]model.TorrentFile, 0, len(files))
	for _, f := range files {
		out = append(out, model.TorrentFile{
			Path: strings.Join(f.BestPath(), "/"), Size: f.Length, Done: -1, Priority: model.PriorityNormal,
		})
	}
	return out, nil
}
