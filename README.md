# go-singleflight-streams
Go library to return a dedicated reader to each singleflight consumer. Useful for reading a source once, 
but sharing the result with many consumers.

Example usage:

```go
package main

import (
	"io"
	"github.com/t2bot/go-singleflight-streams"
)

g := new(sfstreams.Group)

workFn := func() (io.ReadCloser, error) {
	// do your file download, thumbnailing, whatever here
	return src, nil
}

// in your various goroutines...
r, err, shared := g.Do("string key", workFn)
// do something with r (it'll be a unique instance)
```
