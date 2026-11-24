package bt

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

// Source is where a torrent comes from: a magnet link, a .torrent file's
// bytes, or a bare info hash.
type Source struct {
	Magnet   string
	MetaInfo []byte
	InfoHash string
}

// ParseSource recognizes magnet links and info hashes in text.
func ParseSource(s string) (Source, bool) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(strings.ToLower(s), "magnet:"):
		return Source{Magnet: s}, true
	case len(s) == 40:
		if _, err := hex.DecodeString(s); err == nil {
			return Source{InfoHash: s}, true
		}
	}
	return Source{}, false
}

func (e *Engine) spec(src Source) (*torrent.TorrentSpec, error) {
	switch {
	case len(src.MetaInfo) > 0:
		mi, err := metainfo.Load(bytes.NewReader(src.MetaInfo))
		if err != nil {
			return nil, fmt.Errorf("not a valid .torrent file: %w", err)
		}
		return torrent.TorrentSpecFromMetaInfoErr(mi)
	case src.Magnet != "":
		spec, err := torrent.TorrentSpecFromMagnetUri(src.Magnet)
		if err != nil {
			return nil, fmt.Errorf("not a valid magnet link: %w", err)
		}
		return e.withSavedInfo(spec), nil
	case src.InfoHash != "":
		var ih metainfo.Hash
		if err := ih.FromHexString(src.InfoHash); err != nil {
			return nil, err
		}
		spec := &torrent.TorrentSpec{}
		spec.InfoHash = ih
		return e.withSavedInfo(spec), nil
	}
	return nil, errors.New("no torrent given")
}

// withSavedInfo fills in the metadata saved by an earlier session.
func (e *Engine) withSavedInfo(spec *torrent.TorrentSpec) *torrent.TorrentSpec {
	data, err := os.ReadFile(e.torrentFile(spec.InfoHash))
	if err != nil {
		return spec
	}
	mi, err := metainfo.Load(bytes.NewReader(data))
	if err != nil {
		return spec
	}
	saved, err := torrent.TorrentSpecFromMetaInfoErr(mi)
	if err != nil {
		return spec
	}
	saved.Trackers = append(saved.Trackers, spec.Trackers...)
	saved.Webseeds = append(saved.Webseeds, spec.Webseeds...)
	saved.PeerAddrs = append(saved.PeerAddrs, spec.PeerAddrs...)
	return saved
}

// HashOf returns the info hash of a source without adding it.
func HashOf(src Source) (string, error) {
	e := &Engine{}
	spec, err := e.spec(Source{Magnet: src.Magnet, MetaInfo: src.MetaInfo, InfoHash: src.InfoHash})
	if err != nil {
		return "", err
	}
	return spec.InfoHash.HexString(), nil
}
