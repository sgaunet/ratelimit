package ratelimit

import (
	"time"
)

type RateLimit struct {
	d     time.Duration
	limit int
	ch    chan interface{}
}

// New return a Ratelimit instance and initialize it
func New(d time.Duration, limit int) *RateLimit {
	if limit == 0 || d == 0 {
		panic("limit/duration cannot be 0")
	}
	r := RateLimit{
		d:     d,
		limit: limit,
		ch:    make(chan interface{}, limit),
	}
	r.backgroundRoutine()
	return &r
}

// backgroundRoutine launches a goroutine to empty the channel every r.d duration
func (r *RateLimit) backgroundRoutine() {
	go func() {
		t := time.NewTicker(r.d)
		for range t.C {
			// fmt.Println(tick, len(r.ch))
			length := len(r.ch)
			for i := 0; i < length; i++ {
				<-r.ch
				// fmt.Println("empty ch", len(r.ch), i)
			}
		}
	}()
}

// WaitIfLimitReached wait if limit has been reached
func (r *RateLimit) WaitIfLimitReached() {
	r.ch <- struct{}{}
}

// IsLimitReached returns true if limit hasbeen reached
func (r *RateLimit) IsLimitReached() bool {
	select {
	case r.ch <- struct{}{}:
		return false
	default:
		return true
	}
}
