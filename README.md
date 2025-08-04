[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/ratelimit)](https://goreportcard.com/report/github.com/sgaunet/ratelimit)
[![GitHub release](https://img.shields.io/github/release/sgaunet/ratelimit.svg)](https://github.com/sgaunet/ratelimit/releases/latest)
![Coverage](https://raw.githubusercontent.com/wiki/sgaunet/ratelimit/coverage-badge.svg)
[![GoDoc](https://godoc.org/github.com/sgaunet/ratelimit?status.svg)](https://godoc.org/github.com/sgaunet/ratelimit)
[![License](https://img.shields.io/github/license/sgaunet/ratelimit.svg)](LICENSE)

# ratelimit

Just a little library to handle rate limit. Its use is very easy, an example can be found in the example folder.

> **⚠️ Important Note**: The Go ecosystem provides `golang.org/x/time/rate` as a production-ready rate limiting solution. This library is primarily educational. See the [comparison section](#go-standard-library-alternative) below.

**There are no tests, avoid to use it for now. I'm not working enough on it.**

## Installation

```bash
go get github.com/sgaunet/ratelimit
```

## Usage Example

Here's a simple example of how to use the rate limiter:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/sgaunet/ratelimit"
)

func main() {
    // Create a rate limiter that allows 10 operations per second
    ctx := context.Background()
    rl, err := ratelimit.New(ctx, 1*time.Second, 10)
    if err != nil {
        panic(err)
    }
    defer rl.Stop()
    
    // Example 1: Using WaitIfLimitReached (blocking)
    for i := 0; i < 15; i++ {
        rl.WaitIfLimitReached() // This will block after 10 calls until next second
        fmt.Printf("Operation %d executed at %v\n", i+1, time.Now())
    }
    
    // Example 2: Using IsLimitReached (non-blocking)
    for i := 0; i < 15; i++ {
        if rl.IsLimitReached() {
            fmt.Printf("Rate limit reached at operation %d\n", i+1)
            time.Sleep(100 * time.Millisecond)
            continue
        }
        fmt.Printf("Operation %d executed\n", i+1)
    }
}
```

### Key Methods

- `New(ctx, duration, limit)` - Creates a new rate limiter
  - `ctx`: Context for cancellation
  - `duration`: Time window for the rate limit
  - `limit`: Maximum number of operations allowed in the time window
  
- `WaitIfLimitReached()` - Blocks until an operation can proceed without exceeding the rate limit
  
- `IsLimitReached()` - Returns true if the rate limit has been reached (non-blocking)
  
- `Stop()` - Cleans up resources used by the rate limiter

## DEBUG

```
export RATELIMIT_LOGLEVEL=debug
```

## Go Standard Library Alternative

The Go standard library provides `golang.org/x/time/rate` package which offers more sophisticated rate limiting capabilities. Here's a comparison:

### Standard Library: `golang.org/x/time/rate`

```go
import "golang.org/x/time/rate"

// Create a rate limiter that allows 10 events per second
limiter := rate.NewLimiter(10, 1)

// Wait for permission (blocking)
err := limiter.Wait(ctx)

// Check if allowed (non-blocking)
if limiter.Allow() {
    // proceed with operation
}

// Reserve a future event
r := limiter.Reserve()
if !r.OK() {
    // rate limit exceeded
}
time.Sleep(r.Delay())
```

### Key Differences

| Feature | This Library | `golang.org/x/time/rate` |
|---------|--------------|-------------------------|
| Algorithm | Token bucket (simple) | Token bucket (advanced) |
| Burst support | No | Yes |
| Reserve future events | No | Yes |
| Performance | Basic | Highly optimized |
| Production ready | No | Yes |
| Testing | No tests | Well tested |

### When to Use This Library vs Standard Library

**Use this library when:**
- You need a very simple rate limiter for learning purposes
- You want to understand basic rate limiting concepts
- You're experimenting with Go channels and goroutines

**Use `golang.org/x/time/rate` when:**
- You need production-grade rate limiting
- You require burst capabilities
- You need to reserve future events
- Performance is critical
- You need a well-tested solution

### Recommendation

For any production use case, we strongly recommend using `golang.org/x/time/rate` instead of this library. This library serves primarily as an educational example of how rate limiting can be implemented using Go's channels and goroutines.

## Project Disclaimer

This software project is released under the MIT License and was created primarily for fun and testing purposes. While it may offer some interesting functionalities, please note:

* Intended Use
* This project is experimental in nature
* It serves as a playground for ideas and concepts
* The code may not be optimized or production-ready

## Recommendation

If you find the features provided by this project useful or intriguing, we strongly recommend exploring more mature and established solutions for your actual needs. This project is not intended to compete with or replace professional-grade software in its domain.

## Contributions

While we appreciate your interest, please understand that this project may not be actively maintained or developed further. Feel free to fork and experiment with the code as per the MIT License terms.
Thank you for your understanding and enjoy exploring!