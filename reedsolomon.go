/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
*/

package reedsolomon

import (
	"errors"
	"sync"
	"archive/tar"
)

// SIMD Instruction Extensions
const (
	none = iota
	avx2
	// SSSE3 was first introduced with Intel processors based on the Core microarchitecture
	// on 26 June 2006 with the "Woodcrest" Xeons.
	ssse3
)

var extension = none

type EncodeReconster interface {
	Encode(shards matrix) error
	Reconstruct(shards matrix) error
	ReconstructData(shards matrix) error
}

// the cap of inverse Matrix cache
const inverseCacheCap  = 1 << 14
// Encode & Reconst receiver
type (
	rsBase reedSolomon
	rsAVX2 reedSolomon
	rsSSSE3 reedSolomon
	reedSolomon struct {
		tables  []byte
		data    int
		parity  int
		gen     matrix
		inverse matrixCache
	}
	matrixCache struct {
		sync.RWMutex
		cnt  uint32
		// k = data+parity should < 64
		// I think it's enough
		cache map[uint64]matrix
	}
)

func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkShards(data, parity)
	if err != nil {
		return
	}
	g := genCauchyMatrix(data, parity)
	c := make(map[uint64]matrix)
	switch extension {
	case avx2:
		return &rsAVX2{data: data, parity: parity, gen: g, inverse:matrixCache{cache:c}}, nil
	case ssse3:
		return &rsSSSE3{data: data, parity: parity, gen: g, inverse:matrixCache{cache:c}}, nil
	default:
		return &rsBase{data: data, parity: parity, gen: g, inverse:matrixCache{cache:c}}, nil
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



