package reedsolomon

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
)

const streamVerifySize = vectBufSize*6 + 1 // 6 = parityCnt+2

func TestStream_Encode(t *testing.T) {
	s, err := NewStream(testDataCnt, testParityCnt)
	if err != nil {
		t.Fatal(err)
	}
	streamSize := []int{1, streamVerifySize}
	testStreamEncode(s, streamSize, t)
}

func testStreamEncode(s *Stream, streamSize []int, t *testing.T) {
	d, p := s.RS.DataCnt, s.RS.ParityCnt
	for _, size := range streamSize {
		expect := make([][]byte, d+p)
		for j := range expect {
			expect[j] = make([]byte, size)
		}
		expectData := expect[:d]
		for j := range expectData {
			fillRandom(expectData[j])
		}
		dataStream := bufToReader(byteToBuf(expectData))
		resultParity := genEmptyBuffer(p)
		parityStream := bufToWriter(resultParity)
		err := s.Encode(parityStream, dataStream)
		if err != nil {
			t.Fatal(err)
		}
		err = s.RS.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		expectParity := expect[d:]
		for j := range expectParity {
			if !bytes.Equal(resultParity[j].Bytes()[:size], expectParity[j]) {
				t.Fatal("stream_encode & encode mismatch", size)
			}
		}
	}
}

func TestStream_Reconst(t *testing.T) {
	s, err := NewStream(testDataCnt, testParityCnt)
	if err != nil {
		t.Fatal(err)
	}
	streamSize := []int{1, streamVerifySize}
	testStreamReconst(s, streamSize, t)
}

func testStreamReconst(s *Stream, streamSize []int, t *testing.T) {
	d, p := s.RS.DataCnt, s.RS.ParityCnt
	for _, size := range streamSize {
		expect := make([][]byte, d+p)
		for j := range expect {
			expect[j] = make([]byte, size)
		}
		expectData := expect[:d]
		for j := range expectData {
			fillRandom(expectData[j])
		}
		err := s.RS.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		result := genEmptyBuffer(len(testNeedReconst))
		resultStream := bufToWriter(result)
		validStream := make(map[int]io.Reader)
		for _, v := range testDPHas {
			validStream[v] = io.Reader(bytes.NewBuffer(expect[v]))
		}

		err = s.Reconst(resultStream, validStream, testDPHas, testNeedReconst)
		if err != nil {
			t.Fatal(err)
		}

		for j, v := range testNeedReconst {
			if !bytes.Equal(result[j].Bytes()[:size], expect[v]) {
				t.Fatal("stream_reconst & reconst mismatch", j)
			}
		}
	}
}

func TestStream_UpdateParity(t *testing.T) {
	s, err := NewStream(testDataCnt, testParityCnt)
	if err != nil {
		t.Fatal(err)
	}
	streamSize := []int{1, streamVerifySize}
	testStreamUpdateParity(s, streamSize, testUpdateRow, t)
}

func testStreamUpdateParity(s *Stream, streamSize []int, updateRow int, t *testing.T) {
	d, p := s.RS.DataCnt, s.RS.ParityCnt
	for _, size := range streamSize {
		allVects := make([][]byte, d+p)
		for i := range allVects {
			allVects[i] = make([]byte, size)
			if i < d {
				fillRandom(allVects[i])
			}
		}
		err := s.RS.Encode(allVects)
		if err != nil {
			t.Fatal(err)
		}
		oldParity := make([][]byte, p)
		for i := range oldParity {
			oldParity[i] = make([]byte, size)
			copy(oldParity[i], allVects[i+d])
		}
		newData, oldData := make([]byte, size), make([]byte, size)
		fillRandom(newData)
		copy(oldData, allVects[updateRow])
		copy(allVects[updateRow], newData)
		err = s.RS.Encode(allVects)
		if err != nil {
			t.Fatal(err)
		}
		oldDataStream := io.Reader(bytes.NewBuffer(oldData))
		newDataStream := io.Reader(bytes.NewBuffer(newData))
		oldParityStream := bufToReader(byteToBuf(oldParity))
		resultParity := genEmptyBuffer(testParityCnt)
		resultParityStream := bufToWriter(resultParity)
		err = s.UpdateParity(oldDataStream, newDataStream, oldParityStream, resultParityStream, testUpdateRow)
		if err != nil {
			t.Fatal(err)
		}
		for j := range oldParity {
			if !bytes.Equal(resultParity[j].Bytes()[:size], allVects[j+d]) {
				t.Fatal("stream_encode & encode mismatch", size)
			}
		}
	}
}

func BenchmarkEncodeStream(b *testing.B) {
	sizes := []int{4 * mb}
	b.Run("", benchEncStreamRun(benchEncStream, testDataCnt, testParityCnt, sizes, false))
}

func BenchmarkEncodeStreamConcurrent(b *testing.B) {
	sizes := []int{4 * mb}
	b.Run("", benchEncStreamRun(benchEncStream, testDataCnt, testParityCnt, sizes, true))
}

func benchEncStreamRun(f func(*testing.B, int, int, int, bool), dataCnt, parityCnt int, sizes []int, c bool) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("%d+%d_%dKB", dataCnt, parityCnt, s/kb), func(b *testing.B) {
				f(b, dataCnt, parityCnt, s, c)
			})
		}
	}
}

func benchEncStream(b *testing.B, dataCnt, parityCnt, size int, c bool) {
	data := make([][]byte, dataCnt)
	for i := 0; i < dataCnt; i++ {
		data[i] = make([]byte, size)
		rand.Seed(int64(i))
		fillRandom(data[i])
	}
	s := &Stream{}
	var err error
	if c == true {
		s, err = NewStreamConcurrent(dataCnt, parityCnt)
		if err != nil {
			b.Fatal(err)
		}
	} else {
		s, err = NewStream(dataCnt, parityCnt)
		if err != nil {
			b.Fatal()
		}
	}
	parityStream := make([]io.Writer, parityCnt)
	for i := range parityStream {
		parityStream[i] = ioutil.Discard
	}
	err = s.Encode(parityStream, bufToReader(byteToBuf(data)))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(dataCnt * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err2 := s.Encode(parityStream, bufToReader(byteToBuf(data)))
		if err2 != nil {
			b.Fatal(err2)
		}
	}
}

func genEmptyBuffer(n int) []*bytes.Buffer {
	w := make([]*bytes.Buffer, n)
	for i := range w {
		w[i] = &bytes.Buffer{}
	}
	return w
}

func bufToWriter(buf []*bytes.Buffer) []io.Writer {
	w := make([]io.Writer, len(buf))
	for i := range buf {
		w[i] = buf[i]
	}
	return w
}

func bufToReader(buf []*bytes.Buffer) []io.Reader {
	r := make([]io.Reader, len(buf))
	for i := range buf {
		r[i] = buf[i]
	}
	return r
}
