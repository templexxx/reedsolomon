/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
*/

package reedsolomon

import (
	"errors"
	"sync"
)

// SIMD Instruction Extensions
const (
	none = iota
	avx2
	ssse3
)

var extension = none

type EncodeReconster interface {
	Encode(shards matrix) error
	Reconstruct(shards matrix) error
	ReconstructData(shards matrix) error
}

const inverseCacheCap = 1 << 14               // the cap of inverse Matrix cache
const limitInverseMatrixCacheRows = 1<<64 - 1 //data+parity should < 64, I think that's enough

// Encode & Reconst receiver
type (
	encBase   encNoSIMD
	encAVX2   encSIMD
	encSSSE3  encSIMD
	encNoSIMD struct {
		data    int
		parity  int
		encM    matrix // encode matrix include a identity_matrix & cauchy_matrix
		inverse matrixCache
	}
	encSIMD struct {
		data    int
		parity  int
		encM    matrix
		inverse matrixCache
		tables  []byte
	}
	matrixCache struct {
		sync.RWMutex
		cache map[uint64]matrix
	}
)

func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkShards(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrix(data, parity)
	return newRS(data, parity, e), nil
	g := genCauchyMatrix(data, parity)
	c := make(map[uint64]matrix)
	switch extension {
	// 分别两个 newrs 以使用加速的gentables
	case avx2:
		return &encAVX2{data, parity, gen: g, inverse: matrixCache{cache: c}}, nil
	case ssse3:
		return &encSSSE3{data: data, parity: parity, gen: g, inverse: matrixCache{cache: c}}, nil
	default:
		return &encBase{data: data, parity: parity, gen: g, inverse: matrixCache{cache: c}}, nil
	}
}

// Check EC Shards
var errInvShards = errors.New("reedsolomon: data or parity shards must > 0")
var errMaxShards = errors.New("reedsolomon: shards must <= 256")

func checkShards(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return errInvShards
	}
	if d+p >= 255 {
		return errMaxShards
	}
	return nil
}

func genTables(gen matrix) []byte {
	rows := len(gen)
	cols := len(gen[0])
	tables := make([]byte, 32*rows*cols)
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j][i]
			offset := (i*rows + j) * 32
			l := mulTableLow[c][:]
			copy(tables[offset:offset+16], l)
			h := mulTableHigh[c][:]
			copy(tables[offset+16:offset+32], h)
		}
	}
	return tables
}

//func genTables(gen matrix) []byte {
//	rows := len(gen)
//	cols := len(gen[0])
//	tables := make([]byte, 32*rows*cols)
//	for i := 0; i < cols; i++ {
//		for j := 0; j < rows; j++ {
//			c := gen[j][i]
//			offset := (i*rows + j) * 32
//			l := mulTableLow[c][:]
//			copy(tables[offset:offset+16], l)
//			h := mulTableHigh[c][:]
//			copy(tables[offset+16:offset+32], h)
//		}
//	}
//	return tables
//}
