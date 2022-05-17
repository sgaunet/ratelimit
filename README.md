
# ratelimit

Just a little library to handle rate limit. Its use is very easy, an example can be found in the example folder.

# Example

```
package main

import (
	"fmt"
	"time"

	"github.com/sgaunet/ratelimit"
)

// init rateLimit
var r *ratelimit.RateLimit = ratelimit.New(1*time.Second, 2)

func main() {
	recurse(1000)
}

func recurse(i int) {
	r.WaitIfLimitReached() // Just call this function to check if rate limit has been reached or not
	i--
	fmt.Println(i, time.Now())
	if i == 0 {
		return
	}
	recurse(i)
}
```

