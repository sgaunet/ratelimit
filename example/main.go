package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	ratelimit "github.com/sgaunet/ratelimit"
)

func main() {
	// init rateLimit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	r, _ := ratelimit.New(ctx, 1*time.Second, 20)

	go recurse(ctx, r, 1000)
	time.Sleep(5 * time.Second)
	fmt.Println("=================>", time.Now())
	r.WaitIfLimitReached()
	fmt.Println("=================>", time.Now())
	r.WaitIfLimitReached()
	fmt.Println("=================>", time.Now())
	r.WaitIfLimitReached()
	fmt.Println("=================>", time.Now())
	r.WaitIfLimitReached()
	cancel()
	r.Stop()

	// time.Sleep(time.Second)
	fmt.Println(runtime.NumGoroutine())
	// buf := make([]byte, 1<<16)
	// runtime.Stack(buf, true)
	// fmt.Printf("%s", buf)

}

func recurse(ctx context.Context, r *ratelimit.RateLimit, i int) {
	if ctx.Err() != nil {
		return
	}
	r.WaitIfLimitReached() // Just call this function to check if rate limit has been reached or not
	i--
	fmt.Println(i, time.Now())
	if i == 0 {
		return
	}
	recurse(ctx, r, i)
}
