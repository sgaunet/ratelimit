// Package ratelimit provides a simple rate limiting implementation.
package ratelimit

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Define constants for magic numbers.
const (
	waitSleepDuration = 10 * time.Millisecond
	stopSleepDuration = 100 * time.Millisecond
)

// Define static errors.
var (
	ErrInvalidParams = errors.New("ratelimit: duration or limit cannot be <= 0")
)

// RateLimit represents a rate limiter that allows a certain number of operations within a given duration.
type RateLimit struct {
	d        time.Duration
	limit    int
	ch       chan struct{}
	t        *time.Ticker
	lastCall time.Time
	log      *logrus.Logger

	// done channel to signal context cancellation
	done chan struct{}
	// cancelFunc is used to cancel the background routines
	cancelFunc context.CancelFunc
}

// New returns a Ratelimit instance and initialize it.
func New(ctx context.Context, d time.Duration, limit int) (*RateLimit, error) {
	if limit <= 0 || d <= 0 {
		return nil, ErrInvalidParams
	}

	// Create a new context with cancel function
	rctx, cancel := context.WithCancel(ctx)

	r := RateLimit{
		d:          d,
		limit:      limit,
		ch:         make(chan struct{}, limit),
		cancelFunc: cancel,
		done:       make(chan struct{}),
		log:        initLog(os.Getenv("RATELIMIT_LOGLEVEL")),
		lastCall:   time.Now(),
	}
	
	// Setup context monitoring
	go func() {
		<-rctx.Done()
		close(r.done)
	}()
	
	r.backgroundRoutine(rctx)
	r.handleCtx(rctx)
	return &r, nil
}

// WaitIfLimitReached wait if limit has been reached.
// do not use IsLimitReached and WaitIFLimitReached in the same algo.
func (r *RateLimit) WaitIfLimitReached() {
	r.lastCall = time.Now()

	for {
		select {
		case <-r.done:
			r.log.Debugln("End WaitIfLimitReached")
			return
		case r.ch <- struct{}{}:
			return
		default:
			time.Sleep(waitSleepDuration)
		}
	}
}

// IsLimitReached returns true if limit has been reached.
// do not use IsLimitReached and WaitIFLimitReached in the same algo.
func (r *RateLimit) IsLimitReached() bool {
	r.lastCall = time.Now()
	
	select {
	case <-r.done:
		// program is going to be terminated
		return false
	default:
		// continue
	}
	
	select {
	case r.ch <- struct{}{}:
		return false
	default:
		return true
	}
}

// GetLastCall returns the time of the last call to WaitIfLimitReached or IsLimitReached.
func (r *RateLimit) GetLastCall() time.Time {
	return r.lastCall
}

// Stop close background Goroutine.
func (r *RateLimit) Stop() {
	r.log.Debugln("Stop Ticker")
	r.t.Stop()
	r.log.Debugln("Empty chan")
	r.emptyChan()
	time.Sleep(stopSleepDuration)
}

// backgroundRoutine launches a goroutine to empty the channel every r.d duration.
func (r *RateLimit) backgroundRoutine(ctx context.Context) {
	r.log.Debugln("Start backgroundRoutine")
	go func() {
		r.t = time.NewTicker(r.d)
	loop:
		for {
			select {
			case <-r.t.C:
				r.emptyChan()
			case <-ctx.Done():
				break loop
			}
		}
		r.log.Debugln("Stop backgroundRoutine")
	}()
}

func (r *RateLimit) handleCtx(ctx context.Context) {
	go func() {
		<-ctx.Done()
		r.log.Debugln("Stop Ticker")
		r.t.Stop()
		r.log.Debugln("Empty chan")
		r.emptyChan()
		r.log.Debugln("End of handleCtx")
	}()
}

func (r *RateLimit) emptyChan() {
	select {
	case <-r.done:
		return
	default:
		// continue
		length := len(r.ch)
		for range length {
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
	// log.SetFormatter(&log.JSONFormatter{})
	l.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		FullTimestamp:    false,
		DisableTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	l.SetOutput(os.Stdout)

	switch debugLevel {
	case "debug":
		l.SetLevel(logrus.DebugLevel)
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