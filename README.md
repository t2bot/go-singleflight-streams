# go-singleflight-streams
Go library to return a dedicated reader to each singleflight consumer. Useful for reading a source once, 
but sharing the result with many consumers.

Example usage ([Go Playground](https://go.dev/play/p/29wwkiHODU1)):

```go
package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	sfstreams "github.com/t2bot/go-singleflight-streams"
)

func main() {
	g := new(sfstreams.Group)

	workFn := func() (io.ReadCloser, error) {
		// Call your resource or otherwise create a stream. Here we create an
		// example stream made of random bytes.
		b := make([]byte, 16*1024) // 16kb
		_, err := rand.Read(b)
		if err != nil {
			return nil, err
		}
		time.Sleep(1 * time.Second) // for example purposes, we add a delay
		return io.NopCloser(bytes.NewBuffer(b)), nil
	}

	key := "my_resource" // deduplication key

	// This loop simulates multiple requests, such as incoming HTTP requests
	wg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// `r` is guaranteed to be unique for this call
			r, err, shared := g.Do(key, workFn)
			if r != nil {
				defer r.Close()
			}
			if err != nil {
				panic(err) // or whatever other error handling
			}

			if shared {
				fmt.Println("Response was shared!")
				// When true, the workFn was only called once and its output used
				// multiple times (to distinct readers).
			} else {
				// This shouldn't happen in this example
				fmt.Println("WARN: Response was not shared!")
			}

			// We discard here, but a more realistic handling might be to stream
			// the response to a user.
			c, err := io.Copy(io.Discard, r)
			if err != nil {
				panic(err) // or whatever other error handling
			}

			fmt.Println("Read bytes from stream: ", c)
		}()
	}
	
	fmt.Println("Waiting for all requests to finish")
	wg.Wait()
	fmt.Println("Done!")
}
```
