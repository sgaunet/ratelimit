package ratelimit

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type RateLimit struct {
	d     time.Duration
	limit int
	ch    chan interface{}
	ctx   context.Context
	t     *time.Ticker
	log   *logrus.Logger
}

// New returns a Ratelimit instance and initialize it
func New(ctx context.Context, d time.Duration, limit int) (*RateLimit, error) {
	if limit <= 0 || d <= 0 {
		return nil, errors.New("ratelimit: duration or limit cannot be <= 0")
	}

	r := RateLimit{
		d:     d,
		limit: limit,
		ch:    make(chan interface{}, limit),
		ctx:   ctx,
		log:   initLog(os.Getenv("RATELIMIT_LOGLEVEL")),
	}
	r.backgroundRoutine()
	r.handleCtx()
	return &r, nil
}

// backgroundRoutine launches a goroutine to empty the channel every r.d duration
func (r *RateLimit) backgroundRoutine() {
	r.log.Debugln("Start backgroundRoutine")
	go func() {
		r.t = time.NewTicker(r.d)
	loop:
		for {
			select {
			case <-r.t.C:
				r.emptyChan()
			case <-r.ctx.Done():
				break loop
			}
		}
		r.log.Debugln("Stop backgroundRoutine")
	}()
}

func (r *RateLimit) handleCtx() {
	go func() {
		<-r.ctx.Done()
		r.log.Debugln("Stop Ticker")
		r.t.Stop()
		r.log.Debugln("Empty chan")
		r.emptyChan()
		r.log.Debugln("End of handleCtx")
	}()
}

// WaitIfLimitReached wait if limit has been reached
// do not use IsLimitReached and WaitIFLimitReached in the same algo
func (r *RateLimit) WaitIfLimitReached() {
	select {
	case <-r.ctx.Done():
		r.log.Debugln("End WaitIfLimitReached")
		return
	default:
		r.ch <- struct{}{}
	}
}

// IsLimitReached returns true if limit has been reached
// do not use IsLimitReached and WaitIFLimitReached in the same algo
func (r *RateLimit) IsLimitReached() bool {
	if r.ctx.Err() != nil {
		// program is going to be terminated
		return false
	}
	select {
	case r.ch <- struct{}{}:
		return false
	default:
		return true
	}
}

func (r *RateLimit) emptyChan() {
	if r.ctx.Err() == nil {
		length := len(r.ch)
		for i := 0; i < length; i++ {
			_, ok := <-r.ch
			if !ok {
				break // channel is closed
			}
		}
	}
}

func initLog(debugLevel string) *logrus.Logger {
	l := logrus.New()
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	l.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		FullTimestamp:    false,
		DisableTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	l.SetOutput(os.Stdout)

	switch debugLevel {
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "warn":
		l.SetLevel(logrus.WarnLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	default:
		l.SetLevel(logrus.InfoLevel)
	}
	return l
}
