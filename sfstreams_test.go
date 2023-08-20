package sfstreams

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func makeStream() (key string, expectedBytes int64, src io.ReadCloser) {
	key = "fake file"

	b := make([]byte, 16*1024) // 16kb
	_, _ = rand.Read(b)
	src = io.NopCloser(bytes.NewBuffer(b))

	expectedBytes = int64(len(b))
	return
}

func TestDo(t *testing.T) {
	key, expectedBytes, src := makeStream()

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return src, nil
	}

	g := new(Group)
	r, err, shared := g.Do(key, workFn)
	if err != nil {
		t.Fatal(err)
	}
	if shared {
		t.Error("Expected a non-shared result")
	}
	if r == src {
		t.Error("Reader and source are the same")
	}

	//goland:noinspection GoUnhandledErrorResult
	defer r.Close()
	c, _ := io.Copy(io.Discard, r)
	if c != expectedBytes {
		t.Errorf("Read %d bytes but expected %d", c, expectedBytes)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDoError(t *testing.T) {
	expectedErr := errors.New("this is expected")
	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return nil, expectedErr
	}

	g := new(Group)
	r, err, shared := g.Do("test", workFn)
	if err != nil && err != expectedErr {
		t.Fatal(err)
	}
	if shared {
		t.Error("Expected a non-shared result")
	}
	if err == nil || r != nil {
		t.Error("Expected an error; Expected no reader")
	}
}

func TestDoDuplicates(t *testing.T) {
	key, expectedBytes, src := makeStream()

	workWg1 := new(sync.WaitGroup)
	workWg2 := new(sync.WaitGroup)
	workCh := make(chan int, 1)
	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		if callCount == 1 {
			workWg1.Done()
		}
		v := <-workCh
		workCh <- v
		time.Sleep(10 * time.Millisecond)
		return src, nil
	}

	g := &Group{}
	readFn := func() {
		defer workWg2.Done()
		workWg1.Done()
		r, err, _ := g.Do(key, workFn)
		if err != nil {
			t.Error(err)
			return
		}
		c, err := io.Copy(io.Discard, r)
		if err != nil {
			t.Error(err)
			return
		}
		if c != expectedBytes {
			t.Errorf("Read %d bytes instead of %d", c, expectedBytes)
		}
	}

	const max = 10
	workWg1.Add(1)
	for i := 0; i < max; i++ {
		workWg1.Add(1)
		workWg2.Add(1)
		go readFn()
	}
	workWg1.Wait()
	workCh <- 1
	workWg2.Wait()
	if callCount <= 0 || callCount >= max {
		t.Errorf("Expected between 1 and %d calls, got %d", max-1, callCount)
	}
}

func TestDoNilReturn(t *testing.T) {
	key, _, _ := makeStream()

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return nil, nil
	}

	g := new(Group)
	r, err, shared := g.Do(key, workFn)
	if err != nil {
		t.Fatal(err)
	}
	if shared {
		t.Error("Expected a non-shared result")
	}
	if r != nil {
		t.Error("Expected a nil result")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDoErrorAndStream(t *testing.T) {
	key, expectedBytes, src := makeStream()
	expectedErr := errors.New("this is an error")

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return src, expectedErr
	}

	g := new(Group)
	r, err, shared := g.Do(key, workFn)
	if err != expectedErr {
		t.Error("Expected a different error")
	}
	if shared {
		t.Error("Expected a non-shared result")
	}
	if r == src {
		t.Error("Reader and source are the same")
	}

	//goland:noinspection GoUnhandledErrorResult
	defer r.Close()
	c, _ := io.Copy(io.Discard, r)
	if c != expectedBytes {
		t.Errorf("Read %d bytes but expected %d", c, expectedBytes)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDoChan(t *testing.T) {
	key, expectedBytes, src := makeStream()

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return src, nil
	}

	g := new(Group)
	ch := g.DoChan(key, workFn)
	res := <-ch
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Shared {
		t.Error("Expected a non-shared result")
	}
	if res.Reader == src {
		t.Error("Reader and source are the same")
	}

	//goland:noinspection GoUnhandledErrorResult
	defer res.Reader.Close()
	c, _ := io.Copy(io.Discard, res.Reader)
	if c != expectedBytes {
		t.Errorf("Read %d bytes but expected %d", c, expectedBytes)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDoChanError(t *testing.T) {
	expectedErr := errors.New("this is expected")
	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return nil, expectedErr
	}

	g := new(Group)
	ch := g.DoChan("key", workFn)
	res := <-ch
	if res.Err != nil && res.Err != expectedErr {
		t.Fatal(res.Err)
	}
	if res.Shared {
		t.Error("Expected a non-shared result")
	}
	if res.Err == nil || res.Reader != nil {
		t.Error("Expected an error; Expected no reader")
	}
}

func TestDoChanNilReturn(t *testing.T) {
	key, _, _ := makeStream()

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return nil, nil
	}

	g := new(Group)
	ch := g.DoChan(key, workFn)
	res := <-ch
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Shared {
		t.Error("Expected a non-shared result")
	}
	if res.Reader != nil {
		t.Error("Expected a nil result")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDoChanErrorAndStream(t *testing.T) {
	key, expectedBytes, src := makeStream()
	expectedError := errors.New("this is an error")

	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		return src, expectedError
	}

	g := new(Group)
	ch := g.DoChan(key, workFn)
	res := <-ch
	if res.Err != expectedError {
		t.Error("Expected a different error")
	}
	if res.Shared {
		t.Error("Expected a non-shared result")
	}
	if res.Reader == src {
		t.Error("Reader and source are the same")
	}

	//goland:noinspection GoUnhandledErrorResult
	defer res.Reader.Close()
	c, _ := io.Copy(io.Discard, res.Reader)
	if c != expectedBytes {
		t.Errorf("Read %d bytes but expected %d", c, expectedBytes)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestStallOnRead(t *testing.T) {
	key, expectedBytes, src := makeStream()

	workWg1 := new(sync.WaitGroup)
	workWg2 := new(sync.WaitGroup)
	workCh := make(chan int, 1)
	callCount := 0
	workFn := func() (io.ReadCloser, error) {
		callCount++
		if callCount == 1 {
			workWg1.Done()
		}
		v := <-workCh
		workCh <- v
		time.Sleep(10 * time.Millisecond)
		return src, nil
	}

	g := &Group{}
	readFn := func(i int) {
		defer workWg2.Done()
		workWg1.Done()
		r, err, _ := g.Do(key, workFn)
		if r != nil {
			defer func(r io.ReadCloser) {
				err = r.Close()
				if err != nil {
					t.Error(err)
				}
			}(r)
		}
		if err != nil {
			t.Error(err)
			return
		}
		if i > 0 {
			return // intentionally don't read from the stream
		}
		c, err := io.Copy(io.Discard, r)
		if err != nil {
			t.Error(err)
			return
		}
		if c != expectedBytes {
			t.Errorf("Read %d bytes instead of %d", c, expectedBytes)
		}
	}

	const max = 10
	workWg1.Add(1)
	for i := 0; i < max; i++ {
		workWg1.Add(1)
		workWg2.Add(1)
		go readFn(i)
	}
	workWg1.Wait()
	workCh <- 1
	workWg2.Wait()
	if callCount <= 0 || callCount >= max {
		t.Errorf("Expected between 1 and %d calls, got %d", max-1, callCount)
	}
}
