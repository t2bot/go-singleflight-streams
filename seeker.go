package sfstreams

import (
	"errors"
	"io"
	"sync"
)

type parentSeeker struct {
	io.ReadSeekCloser
	underlying io.ReadSeekCloser
	mutex      *sync.Mutex
	closeWg    *sync.WaitGroup
}

func newParentSeeker(src io.ReadSeekCloser, downstreamReaders int) *parentSeeker {
	wg := new(sync.WaitGroup)
	wg.Add(downstreamReaders)
	go func() {
		wg.Wait()
		_ = src.Close()
	}()
	return &parentSeeker{
		underlying: src,
		mutex:      new(sync.Mutex),
		closeWg:    wg,
	}
}

func (p *parentSeeker) Read(b []byte) (int, error) {
	return p.underlying.Read(b)
}

func (p *parentSeeker) Seek(offset int64, whence int) (int64, error) {
	return p.underlying.Seek(offset, whence)
}

func (p *parentSeeker) Close() error {
	return p.underlying.Close()
}

type downstreamSeeker struct {
	io.ReadSeekCloser
	parent *parentSeeker
	pos    int64
	eof    bool
	eofPos int64
	closed bool
}

func newSyncSeeker(parent *parentSeeker) *downstreamSeeker {
	return &downstreamSeeker{
		parent: parent,
		pos:    0,
		eof:    false,
		eofPos: 0,
		closed: false,
	}
}

func (s *downstreamSeeker) Read(b []byte) (int, error) {
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	s.parent.mutex.Lock()
	defer s.parent.mutex.Unlock()
	if s.eof && s.pos == s.eofPos {
		return 0, io.EOF
	}
	offset, err := s.parent.Seek(s.pos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	i, err := s.parent.Read(b)
	s.pos = offset + int64(i)
	if err != nil && errors.Is(err, io.EOF) {
		s.eof = true
		s.eofPos = s.pos
	}
	return i, err
}

func (s *downstreamSeeker) Seek(offset int64, whence int) (int64, error) {
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	s.parent.mutex.Lock()
	defer s.parent.mutex.Unlock()
	offset, err := s.parent.Seek(offset, whence)
	if err != nil {
		return s.pos, err
	}
	s.pos = offset
	return s.pos, nil
}

func (s *downstreamSeeker) Close() error {
	s.parent.closeWg.Done()
	s.closed = true
	return nil
}
