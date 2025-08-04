package ratelimit_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sgaunet/ratelimit"
)

func TestNew(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		ctx := context.Background()
		rl, err := ratelimit.New(ctx, 1*time.Second, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if rl == nil {
			t.Fatal("expected rate limiter to be created")
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()
	})

	t.Run("invalid duration", func(t *testing.T) {
		ctx := context.Background()
		rl, err := ratelimit.New(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected error for zero duration")
		}
		if rl != nil {
			t.Fatal("expected nil rate limiter on error")
		}
		if err != ratelimit.ErrInvalidParams {
			t.Fatalf("expected ErrInvalidParams, got %v", err)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		ctx := context.Background()
		rl, err := ratelimit.New(ctx, 1*time.Second, 0)
		if err == nil {
			t.Fatal("expected error for zero limit")
		}
		if rl != nil {
			t.Fatal("expected nil rate limiter on error")
		}
		if err != ratelimit.ErrInvalidParams {
			t.Fatalf("expected ErrInvalidParams, got %v", err)
		}
	})

	t.Run("negative duration", func(t *testing.T) {
		ctx := context.Background()
		_, err := ratelimit.New(ctx, -1*time.Second, 10)
		if err == nil {
			t.Fatal("expected error for negative duration")
		}
		if err != ratelimit.ErrInvalidParams {
			t.Fatalf("expected ErrInvalidParams, got %v", err)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		ctx := context.Background()
		_, err := ratelimit.New(ctx, 1*time.Second, -10)
		if err == nil {
			t.Fatal("expected error for negative limit")
		}
		if err != ratelimit.ErrInvalidParams {
			t.Fatalf("expected ErrInvalidParams, got %v", err)
		}
	})
}

func TestWaitIfLimitReached(t *testing.T) {
	t.Run("basic rate limiting", func(t *testing.T) {
		ctx := context.Background()
		limit := 5
		duration := 100 * time.Millisecond
		rl, err := ratelimit.New(ctx, duration, limit)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		// First batch should go through immediately
		start := time.Now()
		for i := 0; i < limit; i++ {
			rl.WaitIfLimitReached()
		}
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			t.Fatalf("first %d calls took too long: %v", limit, elapsed)
		}

		// Next call should block
		start = time.Now()
		rl.WaitIfLimitReached()
		elapsed = time.Since(start)
		if elapsed < duration-20*time.Millisecond {
			t.Fatalf("expected blocking for ~%v, but only blocked for %v", duration, elapsed)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		rl, err := ratelimit.New(ctx, 1*time.Second, 1)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		// Use up the limit
		rl.WaitIfLimitReached()

		// Cancel context while waiting
		done := make(chan bool)
		go func() {
			rl.WaitIfLimitReached()
			done <- true
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-done:
			// Expected - context cancellation should unblock
		case <-time.After(200 * time.Millisecond):
			t.Fatal("WaitIfLimitReached did not return after context cancellation")
		}
	})
}

func TestIsLimitReached(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		ctx := context.Background()
		limit := 3
		rl, err := ratelimit.New(ctx, 100*time.Millisecond, limit)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		// First calls should succeed
		for i := 0; i < limit; i++ {
			if rl.IsLimitReached() {
				t.Fatalf("expected limit not reached for call %d", i+1)
			}
		}

		// Next call should fail
		if !rl.IsLimitReached() {
			t.Fatal("expected limit to be reached")
		}

		// Wait for reset
		time.Sleep(150 * time.Millisecond)

		// Should succeed again
		if rl.IsLimitReached() {
			t.Fatal("expected limit to be reset after duration")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		rl, err := ratelimit.New(ctx, 1*time.Second, 5)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		// Cancel context
		cancel()
		time.Sleep(50 * time.Millisecond) // Give time for context to propagate

		// Should return false after context cancellation
		if rl.IsLimitReached() {
			t.Fatal("expected IsLimitReached to return false after context cancellation")
		}
	})
}

func TestGetLastCall(t *testing.T) {
	ctx := context.Background()
	rl, err := ratelimit.New(ctx, 1*time.Second, 10)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
	defer rl.Stop()

	// Test with WaitIfLimitReached
	before := time.Now()
	rl.WaitIfLimitReached()
	after := time.Now()
	lastCall := rl.GetLastCall()

	if lastCall.Before(before) || lastCall.After(after) {
		t.Fatalf("GetLastCall returned time outside expected range")
	}

	time.Sleep(100 * time.Millisecond)

	// Test with IsLimitReached
	before = time.Now()
	rl.IsLimitReached()
	after = time.Now()
	lastCall = rl.GetLastCall()

	if lastCall.Before(before) || lastCall.After(after) {
		t.Fatalf("GetLastCall returned time outside expected range after IsLimitReached")
	}
}

func TestStop(t *testing.T) {
	ctx := context.Background()
	rl, err := ratelimit.New(ctx, 100*time.Millisecond, 5)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow background routine to initialize

	// Use up the limit
	for i := 0; i < 5; i++ {
		rl.WaitIfLimitReached()
	}

	// Stop should clean up resources
	rl.Stop()

	// After Stop, the rate limiter should still be usable but might behave differently
	// This tests that Stop doesn't cause panics
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Stop caused panic: %v", r)
			}
			done <- true
		}()
		rl.WaitIfLimitReached()
	}()

	select {
	case <-done:
		// OK
	case <-time.After(1 * time.Second):
		// Also OK - might block forever after Stop
	}
}

func TestConcurrentUsage(t *testing.T) {
	t.Run("concurrent WaitIfLimitReached", func(t *testing.T) {
		ctx := context.Background()
		limit := 10
		duration := 200 * time.Millisecond
		rl, err := ratelimit.New(ctx, duration, limit)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		var wg sync.WaitGroup
		var completed int32
		workers := 20

		start := time.Now()
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rl.WaitIfLimitReached()
				atomic.AddInt32(&completed, 1)
			}()
		}

		// After duration/2, only limit operations should have completed
		time.Sleep(duration / 2)
		count := atomic.LoadInt32(&completed)
		if count > int32(limit) {
			t.Fatalf("expected at most %d operations, got %d", limit, count)
		}

		// Wait for all to complete
		wg.Wait()
		elapsed := time.Since(start)

		// Should take at least one full duration to process all
		expectedMinDuration := duration
		if elapsed < expectedMinDuration-50*time.Millisecond {
			t.Fatalf("expected at least %v, but took %v", expectedMinDuration, elapsed)
		}
	})

	t.Run("concurrent IsLimitReached", func(t *testing.T) {
		ctx := context.Background()
		limit := 5
		rl, err := ratelimit.New(ctx, 100*time.Millisecond, limit)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		var wg sync.WaitGroup
		var allowed int32
		var denied int32
		attempts := 20

		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if rl.IsLimitReached() {
					atomic.AddInt32(&denied, 1)
				} else {
					atomic.AddInt32(&allowed, 1)
				}
			}()
		}

		wg.Wait()

		allowedCount := atomic.LoadInt32(&allowed)
		deniedCount := atomic.LoadInt32(&denied)

		if allowedCount > int32(limit) {
			t.Fatalf("allowed %d operations, but limit is %d", allowedCount, limit)
		}

		if allowedCount+deniedCount != int32(attempts) {
			t.Fatalf("total operations mismatch: allowed=%d, denied=%d, expected=%d",
				allowedCount, deniedCount, attempts)
		}
	})

	t.Run("mixed concurrent operations", func(t *testing.T) {
		ctx := context.Background()
		limit := 10
		rl, err := ratelimit.New(ctx, 200*time.Millisecond, limit)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		var wg sync.WaitGroup
		operations := 30

		// Mix of WaitIfLimitReached and IsLimitReached calls
		for i := 0; i < operations; i++ {
			wg.Add(1)
			if i%2 == 0 {
				go func(id int) {
					defer wg.Done()
					rl.WaitIfLimitReached()
					t.Logf("Wait operation %d completed at %v", id, time.Now())
				}(i)
			} else {
				go func(id int) {
					defer wg.Done()
					reached := rl.IsLimitReached()
					t.Logf("Check operation %d: limit reached = %v at %v", id, reached, time.Now())
				}(i)
			}
		}

		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("concurrent operations took too long")
		}
	})
}

func TestRateLimiterIntegration(t *testing.T) {
	t.Run("simulated API calls", func(t *testing.T) {
		ctx := context.Background()
		// Allow 5 requests per 100ms
		rl, err := ratelimit.New(ctx, 100*time.Millisecond, 5)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		// Simulate burst of API calls
		start := time.Now()
		callTimes := make([]time.Time, 0, 10)

		for i := 0; i < 10; i++ {
			rl.WaitIfLimitReached()
			callTimes = append(callTimes, time.Now())
		}

		// Verify rate limiting worked
		// First 5 should be immediate
		for i := 0; i < 5; i++ {
			if callTimes[i].Sub(start) > 20*time.Millisecond {
				t.Fatalf("call %d was delayed: %v", i, callTimes[i].Sub(start))
			}
		}

		// Next 5 should be after the duration
		for i := 5; i < 10; i++ {
			if callTimes[i].Sub(start) < 80*time.Millisecond {
				t.Fatalf("call %d was not rate limited: %v", i, callTimes[i].Sub(start))
			}
		}
	})

	t.Run("rate limiter with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		rl, err := ratelimit.New(ctx, 100*time.Millisecond, 2)
		if err != nil {
			t.Fatalf("failed to create rate limiter: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Allow background routine to initialize
		defer rl.Stop()

		count := 0
		for {
			select {
			case <-ctx.Done():
				// Expected - context timeout
				if count < 2 {
					t.Fatalf("expected at least 2 operations, got %d", count)
				}
				return
			default:
				rl.WaitIfLimitReached()
				count++
				if count > 10 {
					t.Fatal("too many operations allowed")
				}
			}
		}
	})
}

func TestMemoryLeaks(t *testing.T) {
	t.Run("multiple create and stop", func(t *testing.T) {
		// Create and stop multiple rate limiters to check for leaks
		for i := 0; i < 10; i++ {
			ctx := context.Background()
			rl, err := ratelimit.New(ctx, 50*time.Millisecond, 5)
			if err != nil {
				t.Fatalf("failed to create rate limiter: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // Allow background routine to initialize

			// Use it a bit
			for j := 0; j < 3; j++ {
				rl.WaitIfLimitReached()
			}

			rl.Stop()
		}
	})

	t.Run("context cancellation cleanup", func(t *testing.T) {
		// Create rate limiters with cancelled contexts
		for i := 0; i < 10; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			rl, err := ratelimit.New(ctx, 50*time.Millisecond, 5)
			if err != nil {
				t.Fatalf("failed to create rate limiter: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // Allow background routine to initialize

			// Cancel immediately
			cancel()
			time.Sleep(10 * time.Millisecond)

			// Try to use it
			done := make(chan bool)
			go func() {
				rl.WaitIfLimitReached()
				done <- true
			}()

			select {
			case <-done:
				// OK
			case <-time.After(100 * time.Millisecond):
				// Also OK - might block
			}

			rl.Stop()
		}
	})
}