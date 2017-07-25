/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
*/

package reedsolomon

import (
	"errors"
	"sync"
)

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
		data    int
		parity  int
		gen     matrix
		inverse *matrixCache
	}
	matrixCache struct {
		_padding0 [8]uint64
		sync.RWMutex
		_padding1 [8]uint64
		size  uint32
		_padding2 [8]uint64
		cache map[uint64]matrix
	}
)

func New(data, parity int) (rs EncodeReconster, err error) {
	err = checkShards(data, parity)
	if err != nil {
		return
	}
	ins := getINS()
	g := genCauchyMatrix(data, parity)
	c := make(map[uint64]matrix)
	switch ins {
	case avx2:
		return &rsAVX2{data: data, parity: parity, gen: g, inverse:&matrixCache{cache:c}}, nil
	case ssse3:
		return &rsSSSE3{data: data, parity: parity, gen: g, inverse:&matrixCache{cache:c}}, nil
	default:
		return &rsBase{data: data, parity: parity, gen: g, inverse:&matrixCache{cache:c}}, nil
	}
}

// Instruction Extensions Flags
const (
	base      = iota
	avx2
	ssse3
)

func getINS() int {
	if hasAVX2() {
		return avx2
	} else if hasSSSE3() {
		return ssse3
	} else {
		return base
	}
}

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

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
