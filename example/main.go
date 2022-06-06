package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sgaunet/ratelimit"
)

func main() {
	// init rateLimit
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	r, _ := ratelimit.New(ctx, 1*time.Second, 2)

	go recurse(r, 1000)
	time.Sleep(5 * time.Second)
	r.WaitIfLimitReached()
	r.WaitIfLimitReached()
	r.WaitIfLimitReached()
	r.WaitIfLimitReached()
}

func recurse(r *ratelimit.RateLimit, i int) {
	r.WaitIfLimitReached() // Just call this function to check if rate limit has been reached or not
	i--
	fmt.Println(i, time.Now())
	if i == 0 {
		return
	}
	recurse(r, i)
}
