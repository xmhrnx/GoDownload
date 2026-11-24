package bt

import (
	"fmt"

	"github.com/godownload/godownload/internal/model"
)

// Counters are the torrent's running byte totals for this session.
type Counters struct {
	Downloaded int64 // useful payload received
	Uploaded   int64 // payload sent
	Peers      int
	Seeders    int
}

// Counters samples the torrent's totals; rates come from differences between samples.
func (h *Handle) Counters() Counters {
	st := h.t.Stats()
	return Counters{
		Downloaded: st.BytesReadUsefulData.Int64(),
		Uploaded:   st.BytesWrittenData.Int64(),
		Peers:      st.ActivePeers,
		Seeders:    st.ConnectedSeeders,
	}
}

// Peers lists connected peers.
func (h *Handle) Peers() []model.Peer {
	pieces := h.t.NumPieces()
	conns := h.t.PeerConns()
	out := make([]model.Peer, 0, len(conns))
	for _, c := range conns {
		st := c.Stats()
		p := model.Peer{
			Addr:      fmt.Sprint(c.RemoteAddr),
			DownSpeed: int64(st.DownloadRate),
			UpSpeed:   int64(st.LastWriteUploadRate),
		}
		if name, ok := c.PeerClientName.Load().(string); ok {
			p.Client = name
		}
		if pieces > 0 {
			p.Progress = float32(st.RemotePieceCount) / float32(pieces)
		}
		out = append(out, p)
	}
	return out
}

// Coarse summarizes piece completion into n buckets for the progress bar.
func (h *Handle) Coarse(n int) []float32 {
	pieces := h.t.NumPieces()
	if pieces == 0 || n <= 0 {
		return nil
	}
	n = min(n, pieces)
	done := make([]bool, pieces)
	i := 0
	for _, run := range h.t.PieceStateRuns() {
		for j := 0; j < run.Length; j++ {
			done[i+j] = run.Complete
		}
		i += run.Length
	}
	out := make([]float32, n)
	for b := range n {
		lo, hi := b*pieces/n, (b+1)*pieces/n
		c := 0
		for p := lo; p < hi; p++ {
			if done[p] {
				c++
			}
		}
		out[b] = float32(c) / float32(hi-lo)
	}
	return out
}
