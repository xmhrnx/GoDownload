package bt

import (
	"sync"
	"time"
)

// uploadCap holds one torrent's upload rate under a limit. The torrent library
// only has a client-wide upload limiter, so this works as a token bucket over
// the torrent's own upload counter: when the torrent has sent more than its
// share, uploading is switched off until the bucket refills.
type uploadCap struct {
	h *Handle

	mu      sync.Mutex
	limit   int64
	tokens  float64
	last    int64
	lastAt  time.Time
	allowed bool
	quit    chan struct{}
	running bool
}

func newUploadCap(h *Handle) *uploadCap {
	return &uploadCap{h: h, allowed: true, quit: make(chan struct{})}
}

func (c *uploadCap) set(limit int64) {
	c.mu.Lock()
	c.limit = limit
	start := limit > 0 && !c.running
	if start {
		c.running = true
		c.last = c.h.Counters().Uploaded
		c.lastAt = time.Now()
		c.tokens = float64(limit)
	}
	c.mu.Unlock()
	if start {
		go c.loop()
	}
	c.apply()
}

func (c *uploadCap) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		close(c.quit)
		c.running = false
	}
}

func (c *uploadCap) loop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-c.quit:
			return
		case now := <-t.C:
			c.tick(now)
		}
	}
}

func (c *uploadCap) tick(now time.Time) {
	up := c.h.Counters().Uploaded
	c.mu.Lock()
	if c.limit <= 0 {
		c.allowed = true
		c.mu.Unlock()
		c.apply()
		return
	}
	dt := now.Sub(c.lastAt).Seconds()
	c.tokens += dt*float64(c.limit) - float64(up-c.last)
	c.tokens = min(c.tokens, float64(c.limit)) // at most a second of burst
	c.last, c.lastAt = up, now
	c.allowed = c.tokens > 0
	c.mu.Unlock()
	c.apply()
}

// apply switches uploading on or off from the seed setting and the cap.
func (c *uploadCap) apply() {
	c.mu.Lock()
	allowed := c.allowed || c.limit <= 0
	c.mu.Unlock()
	if allowed && c.h.seeding() {
		c.h.t.AllowDataUpload()
	} else {
		c.h.t.DisallowDataUpload()
	}
}
