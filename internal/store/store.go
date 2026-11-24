// Package store keeps the download list and settings on disk.
package store

import (
	"path/filepath"
	"sync"

	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/model"
)

// Store reads and writes GoDownload's files in its data directory.
type Store struct {
	dir string
	mu  sync.Mutex
}

// Open uses dir for every file.
func Open(dir string) *Store { return &Store{dir: dir} }

// Dir is the data directory.
func (s *Store) Dir() string { return s.dir }

// StatePath is where a download's resume data lives.
func (s *Store) StatePath(id string) string {
	return filepath.Join(s.dir, "state", id+".json")
}

// TorrentDir holds the torrent client's piece records and .torrent files.
func (s *Store) TorrentDir() string { return filepath.Join(s.dir, "bt") }

type listFile struct {
	Version   int              `json:"version"`
	Downloads []model.Download `json:"downloads"`
}

// LoadDownloads reads the list; a missing file is an empty list.
func (s *Store) LoadDownloads() ([]model.Download, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var f listFile
	err := fileutil.ReadJSON(filepath.Join(s.dir, "downloads.json"), &f)
	return f.Downloads, err
}

// SaveDownloads writes the list atomically.
func (s *Store) SaveDownloads(list []model.Download) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fileutil.WriteJSON(filepath.Join(s.dir, "downloads.json"), listFile{Version: 1, Downloads: list})
}

// LoadSettings reads the settings over the defaults, so settings added in a
// newer version get their default values.
func (s *Store) LoadSettings() (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := model.DefaultSettings(fileutil.DownloadsDir())
	err := fileutil.ReadJSON(filepath.Join(s.dir, "settings.json"), &st)
	return st, err
}

// SaveSettings writes the settings atomically.
func (s *Store) SaveSettings(st model.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fileutil.WriteJSON(filepath.Join(s.dir, "settings.json"), st)
}
