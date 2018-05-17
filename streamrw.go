package reedsolomon

import (
	"errors"
	"io"
	"sync"
)

var ErrStreamSizeMismatch = errors.New("stream_size mismatch")

// read read bytes_streams to buf
func read(streams []io.Reader, buf [][]byte) error {
	if len(streams) != len(buf) {
		panic("stream&buf len mismatch")
	}
	size := -1
	for i := range buf {
		n, err := io.ReadFull(streams[i], buf[i]) // if streams[i] == nil but buf[i] != nil, it will panic
		switch err {
		// The error is EOF only if no bytes were read.
		// If an EOF happens after reading some but not all the bytes,
		// ReadFull returns ErrUnexpectedEOF.
		case io.ErrUnexpectedEOF: // len(streams[i]) < len(buf[i]) & n > 0
			if size < 0 {
				size = n
			} else if n != size {
				return ErrStreamSizeMismatch
			}
			buf[i] = buf[i][:n]
		case nil:
			continue // if streams[i] & buf[i] == nil, nothing happen
		default: // include EOF
			return err
		}
	}
	return nil
}

// write write buf to bytes_streams
func write(streams []io.Writer, buf [][]byte) error {
	if len(streams) != len(buf) {
		panic("stream&buf len mismatch")
	}
	for i := range buf {
		n, err := streams[i].Write(buf[i])
		if err != nil {
			return err
		}
		if n != len(buf[i]) {
			return io.ErrShortWrite
		}
	}
	return nil
}

type CReadResult struct {
	index        int
	err          error
	readDoneSize int
}

// concurrentRead reads streams concurrently
func concurrentRead(streams []io.Reader, buf [][]byte) error {
	streamCnt := len(streams)
	if len(buf) != streamCnt {
		panic("stream&buf len mismatch")
	}

	var wg sync.WaitGroup
	wg.Add(streamCnt)
	results := make(chan CReadResult, streamCnt)
	for i := range buf {
		go func(i int) {
			n, err := io.ReadFull(streams[i], buf[i])
			results <- CReadResult{index: i, err: err, readDoneSize: n}
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(results)

	size := -1
	for r := range results {
		i := r.index
		n := r.readDoneSize
		switch r.err {
		// The error is EOF only if no bytes were read.
		// If an EOF happens after reading some but not all the bytes,
		// ReadFull returns ErrUnexpectedEOF.
		case io.ErrUnexpectedEOF: // len(streams[i]) < len(buf[i])
			if size < 0 {
				size = n
			} else if n != size {
				return ErrStreamSizeMismatch
			}
			buf[i] = buf[i][:n]
		case nil: // if streams[i] & buf[i] == nil, nothing happen
			continue
		default: // include EOF
			return r.err
		}
	}
	return nil
}

// concurrentWrite write streams concurrently
func concurrentWrite(streams []io.Writer, buf [][]byte) error {
	streamCnt := len(streams)
	if len(buf) != streamCnt {
		panic("stream&buf len mismatch")
	}

	var wg sync.WaitGroup
	wg.Add(streamCnt)
	var errs = make(chan error, streamCnt)
	for i := range buf {
		go func(i int) {
			n, err := streams[i].Write(buf[i])
			if err != nil {
				errs <- err
			}
			if n != len(buf[i]) {
				errs <- io.ErrShortWrite
			}
			wg.Done()
		}(i)

	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
