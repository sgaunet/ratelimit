package ratelimit

import (
	"time"
)

type RateLimit struct {
	d     time.Duration
	limit int
	ch    chan interface{}
}

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

func (r *RateLimit) WaitIfLimitReached() {
	r.ch <- struct{}{}
}
