package sfstreams

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"
)

type bufCloser struct {
	io.ReadSeeker
	io.Closer
}

func (b *bufCloser) Close() error {
	return nil // no-op
}

func createSource(length int64, t *testing.T) (io.ReadSeekCloser, []byte) {
	buf := make([]byte, length)
	i, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	if int64(i) != length {
		t.Fatal("did not read enough random bytes")
	}
	return &bufCloser{ReadSeeker: bytes.NewReader(buf)}, buf
}

func TestDuplicateReads(t *testing.T) {
	rsc, b := createSource(1024, t)
	ps := newParentSeeker(rsc, 2)
	s1 := newSyncSeeker(ps)
	s2 := newSyncSeeker(ps)

	// Seek to different places
	_, err := s1.Seek(512, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s2.Seek(128, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	// Read a segment of bytes from each
	b1 := make([]byte, 128)
	b2 := make([]byte, 128)
	_, err = s1.Read(b1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s2.Read(b2)
	if err != nil {
		t.Fatal(err)
	}

	// Compare each segment to ensure the correct thing was read
	for i := 512; i < 640; i++ {
		if b1[i-512] != b[i] {
			t.Fatalf("byte %d in segment 1 is incorrect", i)
		}
	}
	for i := 128; i < 256; i++ {
		if b2[i-128] != b[i] {
			t.Fatalf("byte %d in segment 2 is incorrect", i)
		}
	}
}

func TestOverRead(t *testing.T) {
	rsc, _ := createSource(1024, t)
	ps := newParentSeeker(rsc, 1)
	s1 := newSyncSeeker(ps)

	// Discard the whole stream
	_, err := io.Copy(io.Discard, s1)
	if err != nil {
		t.Fatal(err)
	}

	// Read from it again
	b := make([]byte, 128)
	i, err := s1.Read(b)
	if !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if i > 0 {
		t.Fatalf("expected to read zero bytes, got %d", i)
	}
}

type badStream struct {
	io.ReadSeekCloser
	pos    int64
	source *bytes.Reader
}

func (b *badStream) Read(buf []byte) (int, error) {
	if b.pos >= int64(b.source.Len()) {
		return 0, errors.New("the requested range cannot be satisfied")
	}
	i, err := b.source.Read(buf)
	if !errors.Is(err, io.EOF) && (int64(i)+b.pos) >= int64(b.source.Len()) {
		return i, io.EOF
	}
	return i, err
}

func (b *badStream) Seek(offset int64, whence int) (int64, error) {
	p, err := b.source.Seek(offset, whence)
	b.pos = p
	return p, err
}

func (b *badStream) Close() error {
	return nil // no-op
}

func TestImproperSourceOverRead(t *testing.T) {
	_, b := createSource(1024, t)
	bs := &badStream{source: bytes.NewReader(b)}
	ps := newParentSeeker(bs, 1)
	s1 := newSyncSeeker(ps)

	// Discard the whole stream
	_, err := io.Copy(io.Discard, s1)
	if err != nil {
		t.Fatal(err)
	}

	// Read from it again
	rb := make([]byte, 128)
	i, err := s1.Read(rb)
	if !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if i > 0 {
		t.Fatalf("expected to read zero bytes, got %d", i)
	}
}

func TestUseAfterClose(t *testing.T) {
	rsc, _ := createSource(1024, t)
	ps := newParentSeeker(rsc, 1)
	s1 := newSyncSeeker(ps)

	// Close the whole thing
	err := s1.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Now try to read/seek from it
	b := make([]byte, 128)
	_, err = s1.Read(b)
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	_, err = s1.Seek(12, io.SeekStart)
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}
