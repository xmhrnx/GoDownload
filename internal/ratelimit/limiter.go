// Package ratelimit throttles byte streams with token buckets that can be
// chained (a download's own limit and the global one) and changed while running.
package ratelimit

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// chunk is the most bytes taken from a bucket at once, so a slow limit still
// lets data trickle instead of arriving in bursts.
const chunk = 16 * 1024

// Limiter is a byte-rate limit. The zero limit means unlimited.
type Limiter struct {
	l *rate.Limiter
}

// New returns a limiter at bytesPerSec (0 for unlimited).
func New(bytesPerSec int64) *Limiter {
	l := &Limiter{l: rate.NewLimiter(rate.Inf, chunk)}
	l.Set(bytesPerSec)
	return l
}

// Set changes the rate; 0 removes the limit.
func (l *Limiter) Set(bytesPerSec int64) {
	if bytesPerSec <= 0 {
		l.l.SetLimit(rate.Inf)
		return
	}
	burst := int(bytesPerSec / 4)
	if burst < chunk {
		burst = chunk
	}
	l.l.SetBurst(burst)
	l.l.SetLimit(rate.Limit(bytesPerSec))
}

// Rate is the current limit, 0 when unlimited.
func (l *Limiter) Rate() int64 {
	if l.l.Limit() == rate.Inf {
		return 0
	}
	return int64(l.l.Limit())
}

// Raw returns the underlying x/time/rate limiter, for libraries that take one.
func (l *Limiter) Raw() *rate.Limiter { return l.l }

func (l *Limiter) wait(ctx context.Context, n int) error {
	if l == nil || l.l.Limit() == rate.Inf {
		return nil
	}
	for n > 0 {
		take := min(n, l.l.Burst())
		if err := l.l.WaitN(ctx, take); err != nil {
			return err
		}
		n -= take
	}
	return nil
}

// Reader returns r throttled by every limiter in chain (nil entries are skipped).
func Reader(ctx context.Context, r io.Reader, chain ...*Limiter) io.Reader {
	return &reader{ctx: ctx, r: r, chain: chain}
}

type reader struct {
	ctx   context.Context
	r     io.Reader
	chain []*Limiter
}

func (r *reader) Read(p []byte) (int, error) {
	if len(p) > chunk {
		p = p[:chunk]
	}
	n, err := r.r.Read(p)
	for _, l := range r.chain {
		if werr := l.wait(r.ctx, n); werr != nil {
			return n, werr
		}
	}
	return n, err
}
