// Package model holds the data types shared by the engine, the store and the UI.
// It has no behavior beyond small helpers and imports nothing from the UI.
package model

import "time"

// Kind is the protocol a download uses.
type Kind string

const (
	KindHTTP    Kind = "http"
	KindTorrent Kind = "torrent"
	KindHLS     Kind = "hls"
)

// Status is where a download is in its life cycle.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusConnecting  Status = "connecting"
	StatusDownloading Status = "downloading"
	StatusPaused      Status = "paused"
	StatusVerifying   Status = "verifying"
	StatusSeeding     Status = "seeding"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

// Active reports whether the status holds a transfer slot.
func (s Status) Active() bool {
	return s == StatusConnecting || s == StatusDownloading || s == StatusVerifying
}

// Finished reports whether all wanted data is on disk.
func (s Status) Finished() bool {
	return s == StatusCompleted || s == StatusSeeding
}

// Download is the persisted description of one item in the list.
type Download struct {
	ID       string   `json:"id"`
	Kind     Kind     `json:"kind"`
	URL      string   `json:"url,omitempty"`
	Name     string   `json:"name"`
	Dir      string   `json:"dir"`
	Category Category `json:"category"`
	Size     int64    `json:"size"`
	Done     int64    `json:"done"`
	Status   Status   `json:"status"`
	Error    string   `json:"error,omitempty"`

	AddedAt     time.Time `json:"addedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`

	// MaxConnections is the most parallel connections an HTTP download opens.
	MaxConnections int `json:"connections,omitempty"`
	// SpeedLimit caps the download rate in bytes per second; 0 is unlimited.
	SpeedLimit int64 `json:"speedLimit,omitempty"`
	// Checksum is "algo:hex" (md5, sha1, sha256, sha512), checked on completion.
	Checksum string `json:"checksum,omitempty"`

	// Resumable is false when the server can't continue a broken transfer.
	Resumable bool `json:"resumable,omitempty"`

	Request RequestHeaders `json:"request,omitempty"`

	Torrent *TorrentOptions `json:"torrent,omitempty"`

	// Scheduled marks downloads started by a schedule rather than the user,
	// so the schedule's stop time pauses only those.
	Scheduled bool `json:"scheduled,omitempty"`
	// Stopped marks a download the user paused, which the queue must not start.
	Stopped bool `json:"stopped,omitempty"`
}

// RequestHeaders carries what the browser knew about an HTTP request, so the
// download is made as the browser would have made it.
type RequestHeaders struct {
	Referrer  string `json:"referrer,omitempty"`
	Cookies   string `json:"cookies,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// TorrentOptions are the per-torrent settings.
type TorrentOptions struct {
	InfoHash string `json:"infoHash"`
	Magnet   string `json:"magnet,omitempty"`
	// Seed allows uploading to other peers. Off makes the torrent leech only.
	Seed bool `json:"seed"`
	// UploadLimit caps this torrent's upload rate in bytes per second; 0 is unlimited.
	UploadLimit int64 `json:"uploadLimit,omitempty"`
	// SeedRatio stops seeding once uploaded/size reaches it; 0 seeds forever.
	SeedRatio float64 `json:"seedRatio,omitempty"`
	// SeedMinutes stops seeding after this long; 0 seeds forever.
	SeedMinutes int `json:"seedMinutes,omitempty"`
	// Uploaded is the total uploaded over the torrent's lifetime.
	Uploaded int64 `json:"uploaded,omitempty"`
	// SeedingSince is when the torrent finished downloading.
	SeedingSince time.Time `json:"seedingSince,omitempty"`
	// Files holds each file's priority, in metainfo order.
	Files []FilePriority `json:"files,omitempty"`
	// Sequential downloads pieces in order, for playing while downloading.
	Sequential bool `json:"sequential,omitempty"`
}

// FilePriority is how much a torrent file is wanted.
type FilePriority int

const (
	PrioritySkip FilePriority = iota
	PriorityNormal
	PriorityHigh
)
