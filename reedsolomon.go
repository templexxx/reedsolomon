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

// Encode & Reconst receiver
type (
	encBase struct {
		data   int
		parity int
		em     matrix // encoding-matrix include a identity-matrix(upper) & generator-matrix(lower)
	}

	encAVX2  encSIMD
	encSSSE3 encSIMD
	encSIMD  struct {
		data   int
		parity int
		em     matrix      // encoding-matrix
		mc     matrixCache // inverse matrix's cache
		tbl    []byte
	}
	matrixCache struct {
		sync.RWMutex
		cache map[uint32]matrix
	}
)

type EncodeReconster interface {
	Encode(shards matrix) error
	Reconstruct(shards matrix) error
	ReconstructData(shards matrix) error
}

var errInvShards = errors.New("reedsolomon: data or parity shards must > 0")
var errMaxShards = errors.New("reedsolomon: shards must <= 256")

// limitRows : limit of data+parity
// should <= 32, I think that's enough for storage system or network package reconstruct
// usually, data <= 20; parity <= 5
const limitRow = 32

//

func checkShards(data, parity int) error {
	if (data <= 0) || (parity <= 0) {
		return errInvShards
	}
	if data+parity > limitRow {
		return errMaxShards
	}
	return nil
}

// New create an EncodeReconster use a vandermonde matrix as Encoding matrix
// concurrency-safety
// reusing-safety
func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkShards(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	return newRS(data, parity, e), nil
}

// New create an EncodeReconster use a cauchy matrix as Generator Matrix
func NewCauchy(data, parity int) (enc EncodeReconster, err error) {
	err = checkShards(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newRS(data, parity, e), nil
}

func (e *encBase) Encode(shards matrix) (err error) {
	err = checkEncodeShards(e.data, e.parity, shards)
	if err != nil {
		return
	}
	in := shards[:e.data]
	out := shards[e.data:]
	gen := e.em[e.data:]
	for i := 0; i < e.data; i++ {
		data := in[i]
		for oi := 0; oi < e.parity; oi++ {
			if i == 0 {
				vectMul(gen[oi][i], data, out[oi])
			} else {
				vectMulPlus(gen[oi][i], data, out[oi])
			}
		}
	}
	return
}

func vectMul(c byte, in, out []byte) {
	mt := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] = mt[in[i]]
	}
}

func vectMulPlus(c byte, in, out []byte) {
	mt := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= mt[in[i]]
	}
}

func (e *encBase) Reconstruct(shards matrix) (err error) {
	return e.reconst(shards, false)
}

func (e *encBase) ReconstructData(shards matrix) (err error) {
	return e.reconst(shards, true)
}

func (e *encBase) reconst(shards matrix, dataOnly bool) (err error) {
	err = checkMatrixRows(e.data, e.parity, shards)
	if err != nil {
		return
	}
	stat, err := reconstInfo(e.data, e.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := e.reconstData(shards, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		e.reconstParity(shards, stat.size, stat.parityLost)
	}
	return nil
}

var ErrNumShards = errors.New("reedsolomon: num of shards not match")

func checkMatrixRows(in, out int, shards matrix) error {
	if in+out != len(shards) {
		return ErrNumShards
	}
	return nil
}

type reconstStat struct {
	have       []int
	dataLost   []int
	parityLost []int
	size       int
}

var ErrTooFewShards = errors.New("reedsolomon: too few shards for repair")
var ErrNoNeedRepair = errors.New("reedsolomon: no shard need repair")

func reconstInfo(in, out int, shards matrix, dataOnly bool) (info reconstStat, err error) {
	err = checkMatrixRows(in, out, shards)
	if err != nil {
		return
	}
	size := 0
	var have, dataLost, parityLost []int
	for i, s := range shards {
		if s != nil {
			sSize := len(s)
			if sSize == 0 {
				err = ErrShardEmpty
				return
			}
			if size == 0 {
				size = sSize
				have = append(have, i)
			} else {
				if size != sSize {
					err = ErrShardSizeNoMatch
					return
				} else {
					have = append(have, i)
				}
			}
		} else {
			if i < in {
				dataLost = append(dataLost, i)
			} else {
				parityLost = append(parityLost, i)
			}
		}
	}
	if len(have) < in {
		err = ErrTooFewShards
		return
	}
	if len(dataLost)+len(parityLost) == 0 {
		err = ErrNoNeedRepair
		return
	}
	if len(dataLost)+len(parityLost) > out {
		err = ErrTooFewShards
		return
	}
	if len(have)+len(parityLost) == in+out && dataOnly {
		err = ErrNoNeedRepair
		return
	}
	info.have = have
	info.dataLost = dataLost
	info.parityLost = parityLost
	info.size = size
	return
}

func (r *encBase) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, r.data, r.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &encBase{data: r.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (r *encBase) reconstParity(shards matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.data, r.parity)
	numPL := len(parityLost)
	gen := newMatrix(numPL, r.data)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.data]
	}
	dpTmp := newMatrix(r.data+numPL, size)
	for i := 0; i < r.data; i++ {
		dpTmp[i] = shards[i]
	}
	for i, l := range parityLost {
		shards[l] = make([]byte, size)
		dpTmp[i+r.data] = shards[l]
	}
	e := &encBase{data: r.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}

// Check Encode Args
// TODO what will happend if shards[i] == nil
func checkEncodeShards(in, out int, shards matrix) error {
	err := checkMatrixRows(in, out, shards)
	if err != nil {
		return err
	}
	err = checkShardSize(shards)
	if err != nil {
		return err
	}
	return nil
}

var ErrShardEmpty = errors.New("reedsolomon: shards size equal 0")
var ErrShardSizeNoMatch = errors.New("reedsolomon: shards size not match")

func checkShardSize(shards matrix) error {
	size := len(shards[0])
	if size == 0 {
		return ErrShardEmpty
	}
	for i := 1; i < len(shards); i++ {
		if len(shards[i]) != size {
			return ErrShardSizeNoMatch
		}
	}
	return nil
}
