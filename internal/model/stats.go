package model

// Stats is the live, unpersisted state of a running download.
type Stats struct {
	DownSpeed int64 // bytes per second
	UpSpeed   int64
	// ETA is seconds left; -1 when unknown.
	ETA int64
	// Connections is the number of open HTTP connections or connected peers.
	Connections int
	Seeders     int
	Peers       int
	// Blocks is a coarse map of the file for the progress bar: each entry is
	// how complete that slice of the file is, from 0 to 1.
	Blocks []float32
	// Segments are the byte ranges active HTTP connections are working on.
	Segments []Segment
	// Streaming is true while a player reads the download through the stream server.
	Streaming bool
}

// Segment is one connection's share of an HTTP download.
type Segment struct {
	Start, Pos, End int64
}

// Item is a download with its live stats, as the UI shows it.
type Item struct {
	Download
	Stats
	Files    []TorrentFile
	PeerList []Peer
}

// TorrentFile is a file inside a torrent.
type TorrentFile struct {
	Path     string
	Size     int64
	Done     int64
	Priority FilePriority
}

// Peer is a connected torrent peer.
type Peer struct {
	Addr      string
	Client    string
	DownSpeed int64
	UpSpeed   int64
	Progress  float32
}
