package reedsolomon

import (
	"errors"
	"io"
	"sort"
	"sync"
)

type Stream struct {
	RS       *RS
	VectPool *sync.Pool
	Read     func(streams []io.Reader, buf [][]byte) (err error)
	Write    func(streams []io.Writer, buf [][]byte) (err error)
}

// Stream Encode/UpdateParity vects_buf size
const vectBufSize = 64 << 10

// NewStream create a new stream
func NewStream(data, parity int) (s *Stream, err error) {
	r, err := New(data, parity)
	if err != nil {
		return nil, err
	}
	return &Stream{
		RS: r,
		VectPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, vectBufSize*(data+parity))
			},
		},
		Read:  read,
		Write: write,
	}, nil
}

// NewStreamConcurrent create a new stream with concurrent read&write
func NewStreamConcurrent(data, parity int) (s *Stream, err error) {
	r, err := New(data, parity)
	if err != nil {
		return nil, err
	}
	return &Stream{
		RS: r,
		VectPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, vectBufSize*(data+parity))
			},
		},
		Read:  concurrentRead,
		Write: concurrentWrite,
	}, nil
}

var ErrNoData = errors.New("no data in data stream")

func (s *Stream) Encode(parity []io.Writer, data []io.Reader) error {
	if len(data) != s.RS.DataCnt {
		panic("data stream mismatch dataCnt")
	}
	if len(parity) != s.RS.ParityCnt {
		panic("parity stream mismatch parityCnt")
	}

	// step1: generate vects_buf
	buf := s.VectPool.Get().([]byte)
	defer s.VectPool.Put(buf)
	dataCnt := len(data)
	allVects := cutSlice(buf, s.RS.DataCnt+s.RS.ParityCnt)
	dataVects := allVects[:dataCnt]
	parityVect := allVects[dataCnt:]

	// step2: encode
	readDone := false
	for { // for until read all bytes in data reader
		err := s.Read(data, dataVects)
		switch err {
		case nil:
		case io.EOF:
			if readDone == false {
				return ErrNoData
			}
			return nil
		default:
			return err
		}
		for i := range allVects {
			allVects[i] = allVects[i][:len(dataVects[0])]
		}
		readDone = true
		err = s.RS.Encode(allVects)
		if err != nil {
			return err
		}
		err = s.Write(parity, parityVect)
		if err != nil {
			return err
		}
	}
}

func (s *Stream) Reconst(results []io.Writer, valid map[int]io.Reader, dpHas, needReconst []int) error {
	d, p := s.RS.DataCnt, s.RS.ParityCnt
	if len(valid) != d {
		panic("valid stream mismatch d")
	}
	if len(results) > p {
		panic("too many result streams")
	}
	if len(needReconst) != len(results) {
		panic("results streams mismatch needReconst index")
	}

	buf := s.VectPool.Get().([]byte)
	defer s.VectPool.Put(buf)
	allVects := cutSlice(buf, d+p)
	tmpValidVects := make([][]byte, d)
	validStreams := make([]io.Reader, d)
	for i, v := range dpHas {
		tmpValidVects[i] = allVects[v]
		validStreams[i] = valid[v]
	}
	sort.Ints(needReconst)
	resultsVects := make([][]byte, len(needReconst))
	for i, v := range needReconst {
		resultsVects[i] = allVects[v]
	}

	readDone := false
	for { // for until read all bytes in valid reader
		err := s.Read(validStreams, tmpValidVects)
		switch err {
		case nil:
		case io.EOF:
			if readDone == false {
				return ErrNoData
			}
			return nil
		default:
			return err
		}
		for i := range allVects {
			allVects[i] = allVects[i][:len(tmpValidVects[0])]
		}
		readDone = true
		err = s.RS.Reconst(allVects, dpHas, needReconst)
		if err != nil {
			return err
		}
		err = s.Write(results, resultsVects) // len(results) must equal with len(resultsVects)
		if err != nil {
			return err
		}
	}
}

func (s *Stream) UpdateParity(oldData, newData io.Reader, oldParity []io.Reader, newParity []io.Writer, updateRow int) error {
	if len(oldParity) != s.RS.ParityCnt {
		panic("old parity stream mismatch parityCnt")
	}
	if len(newParity) != s.RS.ParityCnt {
		panic("new parity stream mismatch parityCnt")
	}
	// step1: generate vects_buf
	buf := s.VectPool.Get().([]byte)
	defer s.VectPool.Put(buf)
	allVects := cutSlice(buf, 2+s.RS.ParityCnt)

	// step2: update
	r := make([]io.Reader, 2+s.RS.ParityCnt)
	r[0], r[1] = oldData, newData
	for i := 2; i < len(r); i++ {
		r[i] = oldParity[i-2]
	}
	readDone := false
	for { // for until read all bytes in data reader
		err := s.Read(r, allVects)
		switch err {
		case nil:
		case io.EOF:
			if readDone == false {
				return ErrNoData
			}
			return nil
		default:
			return err
		}
		readDone = true
		err = s.RS.UpdateParity(allVects[0], allVects[1], updateRow, allVects[2:])
		if err != nil {
			return err
		}
		err = s.Write(newParity, allVects[2:])
		if err != nil {
			return err
		}
	}
}

// cut []byte -> [][]byte
func cutSlice(raw []byte, n int) [][]byte {
	s := make([][]byte, n)
	for i := range s {
		offset := vectBufSize * i
		s[i] = raw[offset : offset+vectBufSize]
	}
	return s
}
