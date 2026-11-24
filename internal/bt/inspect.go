package bt

import (
	"context"

	"github.com/anacrolix/torrent"

	"github.com/godownload/godownload/internal/model"
)

// Meta describes a torrent's contents.
type Meta struct {
	InfoHash string
	Name     string
	Size     int64
	Private  bool
	Files    []model.TorrentFile
}

// Inspect fetches a torrent's metadata (from peers, for a magnet link) without
// touching the disk, so the user can pick files and a folder first.
func (e *Engine) Inspect(ctx context.Context, src Source) (Meta, error) {
	spec, err := e.spec(src)
	if err != nil {
		return Meta{}, err
	}
	e.mu.Lock()
	if h, ok := e.handles[spec.InfoHash]; ok {
		e.mu.Unlock()
		if err := h.WaitInfo(ctx); err != nil {
			return Meta{}, err
		}
		return metaOf(h.t), nil
	}
	spec.DeferStorageOpen = true
	spec.DisallowDataDownload = true
	spec.DisallowDataUpload = true
	t, _, err := e.cl.AddTorrentSpec(spec)
	e.mu.Unlock()
	if err != nil {
		return Meta{}, err
	}
	defer t.Drop()
	if e.cfg.ExtraTrackers {
		t.AddTrackers(publicTrackers)
	}
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return Meta{}, ctx.Err()
	}
	e.writeMetainfo(t)
	return metaOf(t), nil
}

func metaOf(t *torrent.Torrent) Meta {
	m := Meta{InfoHash: t.InfoHash().HexString(), Name: t.Name(), Size: t.Length()}
	if info := t.Info(); info != nil && info.Private != nil {
		m.Private = *info.Private
	}
	for _, f := range t.Files() {
		m.Files = append(m.Files, model.TorrentFile{Path: f.DisplayPath(), Size: f.Length(), Priority: model.PriorityNormal})
	}
	return m
}
