package reedsolomon

import (
	"bytes"
	"io"
	"testing"
)

const (
	testBufSize   = 16
	bigVerifySize = 4 * 16 // bigVerifySize 循环读取 reader 直到读尽
)

// read len(stream) <= bufSize
func TestReadSmall(t *testing.T) {
	r := read
	testReadSmall(r, t)
}

func TestConcurrentReadSmall(t *testing.T) {
	r := concurrentRead
	testReadSmall(r, t)
}

func testReadSmall(read func(streams []io.Reader, buf [][]byte) (err error), t *testing.T) {
	bs, dn := testBufSize, testDataCnt
	for i := 1; i <= bs; i++ {
		streamData := make([][]byte, dn)
		buf := make([][]byte, dn)
		for j := range streamData {
			streamData[j] = make([]byte, i)
			fillRandom(streamData[j])
			buf[j] = make([]byte, bs)
		}
		streams := toReader(byteToBuf(streamData))
		err := read(streams, buf)
		if err != nil {
			t.Fatal(err)
		}

		for j := 0; j < dn; j++ {
			if !bytes.Equal(streamData[j], buf[j]) {
				t.Fatalf("streamData[%d] mismatch buf[%d] after read", j, j)
			}
		}
	}
}

// read len(stream) n*bufSize
func TestReadBig(t *testing.T) {
	r := read
	testReadBig(r, t)
}

func TestConcurrentReadBig(t *testing.T) {
	r := concurrentRead
	testReadBig(r, t)
}

func testReadBig(read func(streams []io.Reader, buf [][]byte) (err error), t *testing.T) {
	dn, bvs, bs := testDataCnt, bigVerifySize, testBufSize
	streamData := make([][]byte, dn)
	buf := make([][]byte, dn)
	for j := range streamData {
		streamData[j] = make([]byte, bvs)
		fillRandom(streamData[j])
		buf[j] = make([]byte, bs)
	}
	streams := toReader(byteToBuf(streamData))

	for i := 0; i < bvs/bs; i++ {
		err := read(streams, buf)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < dn; j++ {
			if !bytes.Equal(streamData[j][i*bs:(i+1)*bs], buf[j]) {
				t.Fatalf("streamData[%d] mismatch buf[%d] after read", j, j)
			}
		}
	}
}

func TestWrite(t *testing.T) {
	w := write
	testWrite(w, t)
}

func TestConcurrentWrite(t *testing.T) {
	w := concurrentWrite
	testWrite(w, t)
}

func testWrite(write func(streams []io.Writer, buf [][]byte) (err error), t *testing.T) {
	bvs, dn := bigVerifySize, testDataCnt
	for i := 1; i <= bvs; i++ {
		buf := make([][]byte, dn)
		for j := range buf {
			buf[j] = make([]byte, i)
			fillRandom(buf[j])
		}
		streamsBuf := genEmptyBuf(dn)
		streams := toWriter(streamsBuf)
		err := write(streams, buf)
		if err != nil {
			t.Fatal(err)
		}

		for j := 0; j < dn; j++ {
			if !bytes.Equal(streamsBuf[j].Bytes(), buf[j]) {
				t.Fatalf("streamData[%d] mismatch buf[%d] after write", j, j)
			}
		}
	}
}

func toWriter(b []*bytes.Buffer) []io.Writer {
	w := make([]io.Writer, len(b))
	for i := range b {
		w[i] = b[i]
	}
	return w
}

func genEmptyBuf(n int) []*bytes.Buffer {
	b := make([]*bytes.Buffer, n)
	for i := range b {
		b[i] = &bytes.Buffer{}
	}
	return b
}

func toReader(b []*bytes.Buffer) []io.Reader {
	r := make([]io.Reader, len(b))
	for i := range b {
		r[i] = io.Reader(b[i])
	}
	return r
}

func byteToBuf(data [][]byte) []*bytes.Buffer {
	buf := make([]*bytes.Buffer, len(data))
	for i := range data {
		buf[i] = bytes.NewBuffer(data[i])
	}
	return buf
}
