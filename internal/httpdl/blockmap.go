package httpdl

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/bits"
)

// BlockMap records which fixed-size blocks of a file are on disk, with a CRC-32C
// of each, so a resumed or suspect file can be checked block by block and only
// the damaged blocks fetched again.
type BlockMap struct {
	Size      int64
	BlockSize int64
	done      []uint64
	sums      []uint32
}

// blockSizeFor picks a block size giving at most ~8192 blocks, between 64 KiB and 4 MiB.
func blockSizeFor(size int64) int64 {
	bs := int64(64 << 10)
	for bs < 4<<20 && size/bs > 8192 {
		bs <<= 1
	}
	return bs
}

// NewBlockMap returns an empty map for a file of size bytes.
func NewBlockMap(size int64) *BlockMap {
	bs := blockSizeFor(size)
	n := (size + bs - 1) / bs
	return &BlockMap{
		Size:      size,
		BlockSize: bs,
		done:      make([]uint64, (n+63)/64),
		sums:      make([]uint32, n),
	}
}

// Len is the number of blocks.
func (m *BlockMap) Len() int { return len(m.sums) }

// Bounds is block i's byte range [start, end).
func (m *BlockMap) Bounds(i int) (start, end int64) {
	start = int64(i) * m.BlockSize
	return start, min(start+m.BlockSize, m.Size)
}

// Index is the block holding byte off.
func (m *BlockMap) Index(off int64) int { return int(off / m.BlockSize) }

// Done reports whether block i is on disk.
func (m *BlockMap) Done(i int) bool { return m.done[i/64]&(1<<(i%64)) != 0 }

// Sum is block i's checksum, valid when Done(i).
func (m *BlockMap) Sum(i int) uint32 { return m.sums[i] }

// Mark records block i as written with checksum sum.
func (m *BlockMap) Mark(i int, sum uint32) {
	m.done[i/64] |= 1 << (i % 64)
	m.sums[i] = sum
}

// Clear forgets block i, so it is downloaded again.
func (m *BlockMap) Clear(i int) {
	m.done[i/64] &^= 1 << (i % 64)
	m.sums[i] = 0
}

// Reset forgets every block.
func (m *BlockMap) Reset() {
	clear(m.done)
	clear(m.sums)
}

// Count is the number of blocks on disk.
func (m *BlockMap) Count() int {
	n := 0
	for _, w := range m.done {
		n += bits.OnesCount64(w)
	}
	return n
}

// Complete reports whether every block is on disk.
func (m *BlockMap) Complete() bool { return m.Count() == m.Len() }

// DoneBytes is how many bytes are on disk.
func (m *BlockMap) DoneBytes() int64 {
	n := int64(m.Count()) * m.BlockSize
	if last := m.Len() - 1; last >= 0 && m.Done(last) {
		n -= int64(m.Len())*m.BlockSize - m.Size
	}
	return n
}

// DoneFrom is how many contiguous bytes from off are on disk.
func (m *BlockMap) DoneFrom(off int64) int64 {
	if off >= m.Size {
		return 0
	}
	first := m.Index(off)
	i := first
	for i < m.Len() && m.Done(i) {
		i++
	}
	if i == first {
		return 0
	}
	_, end := m.Bounds(i - 1)
	return end - off
}

// Coarse summarizes the map into n buckets, each the fraction done.
func (m *BlockMap) Coarse(n int) []float32 {
	if m.Len() == 0 || n <= 0 {
		return nil
	}
	n = min(n, m.Len())
	out := make([]float32, n)
	for b := range n {
		lo, hi := b*m.Len()/n, (b+1)*m.Len()/n
		done := 0
		for i := lo; i < hi; i++ {
			if m.Done(i) {
				done++
			}
		}
		out[b] = float32(done) / float32(hi-lo)
	}
	return out
}

// blockState is the persisted form of a BlockMap.
type blockState struct {
	Size      int64  `json:"size"`
	BlockSize int64  `json:"blockSize"`
	Done      string `json:"done"`
	Sums      string `json:"sums"`
}

func (m *BlockMap) state() blockState {
	done := make([]byte, len(m.done)*8)
	for i, w := range m.done {
		binary.LittleEndian.PutUint64(done[i*8:], w)
	}
	sums := make([]byte, len(m.sums)*4)
	for i, s := range m.sums {
		binary.LittleEndian.PutUint32(sums[i*4:], s)
	}
	return blockState{
		Size:      m.Size,
		BlockSize: m.BlockSize,
		Done:      base64.StdEncoding.EncodeToString(done),
		Sums:      base64.StdEncoding.EncodeToString(sums),
	}
}

func blockMapFromState(s blockState) (*BlockMap, error) {
	if s.BlockSize <= 0 || s.Size < 0 {
		return nil, fmt.Errorf("corrupt resume data")
	}
	m := &BlockMap{Size: s.Size, BlockSize: s.BlockSize}
	n := (s.Size + s.BlockSize - 1) / s.BlockSize
	done, err := base64.StdEncoding.DecodeString(s.Done)
	if err != nil || len(done) != int((n+63)/64)*8 {
		return nil, fmt.Errorf("corrupt resume data")
	}
	sums, err := base64.StdEncoding.DecodeString(s.Sums)
	if err != nil || len(sums) != int(n)*4 {
		return nil, fmt.Errorf("corrupt resume data")
	}
	m.done = make([]uint64, (n+63)/64)
	for i := range m.done {
		m.done[i] = binary.LittleEndian.Uint64(done[i*8:])
	}
	m.sums = make([]uint32, n)
	for i := range m.sums {
		m.sums[i] = binary.LittleEndian.Uint32(sums[i*4:])
	}
	return m, nil
}
