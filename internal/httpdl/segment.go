package httpdl

import "sync/atomic"

// segment is one connection's claim on the file: it fetches [pos, end) and
// moves pos forward as data lands. end only ever shrinks, when another
// connection takes over the tail (a split) or a stream needs the connection.
type segment struct {
	start int64
	pos   atomic.Int64
	end   atomic.Int64
}

func newSegment(start, end int64) *segment {
	s := &segment{start: start}
	s.pos.Store(start)
	s.end.Store(end)
	return s
}

func (s *segment) remaining() int64 { return s.end.Load() - s.pos.Load() }

// claimedMask marks every block some segment still has to fetch.
func (d *Downloader) claimedMask() []bool {
	mask := make([]bool, d.blocks.Len())
	for s := range d.segs {
		end := s.end.Load()
		for i := d.blocks.Index(s.pos.Load()); i < d.blocks.Len(); i++ {
			if start, _ := d.blocks.Bounds(i); start >= end {
				break
			}
			mask[i] = true
		}
	}
	return mask
}

// claim hands a connection its next piece of work, or nil when there is none.
// Called with d.mu held.
func (d *Downloader) claim() *segment {
	claimed := d.claimedMask()
	if s := d.claimPriority(claimed); s != nil {
		return s
	}
	runStart, runLen := d.largestFreeRun(claimed)
	victim, victimRem := d.largestSegment()
	bs := d.blocks.BlockSize
	// Take a free run when it's at least as big as half a split would give.
	if runLen > 0 && int64(runLen)*bs >= victimRem/2 {
		return d.addSegment(runStart, runStart+runLen)
	}
	if victim != nil && victimRem >= 2*minSplit {
		if s := d.split(victim); s != nil {
			return s
		}
	}
	if runLen > 0 {
		return d.addSegment(runStart, runStart+runLen)
	}
	return nil
}

// minSplit is the smallest share worth a new connection.
const minSplit = 512 << 10

// claimPriority claims a short run at the stream's read position, so a player
// gets its next bytes first.
func (d *Downloader) claimPriority(claimed []bool) *segment {
	if d.priority < 0 {
		return nil
	}
	first := d.blocks.Index(d.priority)
	limit := d.blocks.Index(min(d.priority+d.streamWindow, d.blocks.Size-1)) + 1
	for i := first; i < limit; i++ {
		if d.blocks.Done(i) || claimed[i] {
			continue
		}
		n := 0
		for i+n < d.blocks.Len() && n < d.streamRunBlocks() && !d.blocks.Done(i+n) && !claimed[i+n] {
			n++
		}
		return d.addSegment(i, i+n)
	}
	return nil
}

// streamRunBlocks keeps each connection's share near the play position small,
// so the blocks a player needs next are fetched in parallel, not one after another.
func (d *Downloader) streamRunBlocks() int {
	return max(1, int((1<<20)/d.blocks.BlockSize))
}

func (d *Downloader) largestFreeRun(claimed []bool) (start, length int) {
	for i := 0; i < d.blocks.Len(); {
		if d.blocks.Done(i) || claimed[i] {
			i++
			continue
		}
		j := i
		for j < d.blocks.Len() && !d.blocks.Done(j) && !claimed[j] {
			j++
		}
		if j-i > length {
			start, length = i, j-i
		}
		i = j
	}
	return start, length
}

func (d *Downloader) largestSegment() (*segment, int64) {
	var best *segment
	var rem int64
	for s := range d.segs {
		if r := s.remaining(); r > rem {
			best, rem = s, r
		}
	}
	return best, rem
}

// split gives the back half of s to a new segment, at a block boundary past
// the block s is writing now.
func (d *Downloader) split(s *segment) *segment {
	pos, end := s.pos.Load(), s.end.Load()
	_, curEnd := d.blocks.Bounds(d.blocks.Index(pos))
	mid := pos + (end-pos)/2
	mid = (mid + d.blocks.BlockSize - 1) / d.blocks.BlockSize * d.blocks.BlockSize
	if mid <= curEnd || mid >= end {
		return nil
	}
	s.end.Store(mid)
	ns := newSegment(mid, end)
	d.segs[ns] = struct{}{}
	return ns
}

func (d *Downloader) addSegment(firstBlock, endBlock int) *segment {
	start, _ := d.blocks.Bounds(firstBlock)
	_, end := d.blocks.Bounds(endBlock - 1)
	s := newSegment(start, end)
	d.segs[s] = struct{}{}
	return s
}

// truncateAt stops s at the end of the block it is writing, freeing it for
// other work without throwing away a half-written block.
func (d *Downloader) truncateAt(s *segment, off int64) {
	_, curEnd := d.blocks.Bounds(d.blocks.Index(s.pos.Load()))
	off = max(off, curEnd)
	if off < s.end.Load() {
		s.end.Store(off)
	}
}
