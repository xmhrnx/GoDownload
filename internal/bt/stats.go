package bt

import (
	"fmt"
	"time"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/speed"
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

// Peers lists connected peers with their current speeds (from SamplePeers).
func (h *Handle) Peers() []model.Peer {
	pieces := h.t.NumPieces()
	conns := h.t.PeerConns()
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]model.Peer, 0, len(conns))
	for _, c := range conns {
		st := c.Stats()
		addr := fmt.Sprint(c.RemoteAddr)
		p := model.Peer{Addr: addr}
		if pm := h.peerMeters[addr]; pm != nil {
			p.DownSpeed, p.UpSpeed = pm.down.Rate(), pm.up.Rate()
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

// SamplePeers measures each peer's speed from its byte counters. The
// library's own per-peer rate is an average over the whole connection,
// which doesn't match the torrent's current speed, so the speeds shown are
// measured here the same way as the torrent's. Call it about once a second.
func (h *Handle) SamplePeers(now time.Time) {
	conns := h.t.PeerConns()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.peerMeters == nil {
		h.peerMeters = map[string]*peerMeter{}
	}
	seen := make(map[string]bool, len(conns))
	for _, c := range conns {
		st := c.Stats()
		addr := fmt.Sprint(c.RemoteAddr)
		seen[addr] = true
		pm := h.peerMeters[addr]
		if pm == nil {
			pm = &peerMeter{}
			h.peerMeters[addr] = pm
		}
		pm.down.Update(st.BytesReadUsefulData.Int64(), now)
		pm.up.Update(st.BytesWrittenData.Int64(), now)
	}
	for addr := range h.peerMeters {
		if !seen[addr] {
			delete(h.peerMeters, addr)
		}
	}
}

type peerMeter struct {
	down, up speed.Meter
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
