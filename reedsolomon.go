/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
*/

package reedsolomon

import (
	"errors"
	"fmt"
	"sync"
)

// SIMD Instruction Extensions
const (
	none = iota
	avx2
	ssse3
)

var extension = none

type (
	encBase struct {
		data         int
		parity       int
		encodeMatrix matrix
	}
	encAVX2  encSIMD
	encSSSE3 encSIMD
	encSIMD  struct {
		data               int
		parity             int
		encodeMatrix       matrix
		tbl                []byte //  multiply-tables of element in generator-matrix
		enableInverseCache bool
		inverseCache       matrixCache // inverse matrix's cache
	}
	matrixCache struct {
		sync.RWMutex
		cache map[uint32]matrix
	}
)

type EncodeReconster interface {
	Encode(vects [][]byte) error
	Reconstruct(vects [][]byte) error
	ReconstructData(vects [][]byte) error
}

func checkNumVects(data, parity int) error {
	if (data <= 0) || (parity <= 0) {
		return errors.New("rs.New: data or parity <= 0")
	}
	if data+parity >= 255 { //usually, data <= 20 & parity <= 6
		return errors.New(fmt.Sprintf("rs.New: data+parity >= 255"))
	}
	return nil
}

// New: vandermonde matrix as Encoding matrix
func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkNumVects(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	return newRS(data, parity, e), nil
}

// NewCauchy: cauchy matrix as Generator Matrix
func NewCauchy(data, parity int) (enc EncodeReconster, err error) {
	err = checkNumVects(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newRS(data, parity, e), nil
}

func checkVectsMatch(in, out int, vects [][]byte) error {
	v := len(vects)
	if in+out != v {
		return errors.New(fmt.Sprintf("rs.Enc: vects not match, in: %d out: %d vects: %d", in, out, v))
	}

	s := len(vects[0])
	if s == 0 {
		return errors.New("rs.Enc: vects size = 0")
	}
	for i := 1; i < v; i++ {
		if len(vects[i]) != s {
			return errors.New("rs.Enc: vects size not match")
		}
	}
	return nil
}

func checkEncVects(in, out int, vects [][]byte) error {
	err := checkVectsMatch(in, out, vects)
	if err != nil {
		return err
	}
	return nil
}

// Encode : multiply generator-matrix with data
func (e *encBase) Encode(vects [][]byte) (err error) {
	in := e.data
	out := e.parity
	err = checkEncVects(in, out, vects)
	if err != nil {
		return
	}
	inV := vects[:in]
	outV := vects[in:]
	gen := e.encodeMatrix[in*in:]
	for i := 0; i < out; i++ {
		coeffMulVect(gen[i*in], inV[0], outV[i])
	}
	for i := 1; i < in; i++ {
		for j := 0; j < out; j++ {
			coeffMulVectPlus(gen[j*in+i], inV[i], outV[j])
		}
	}
	return
}

func coeffMulVect(c byte, in, out []byte) {
	t := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] = t[in[i]]
	}
}

func coeffMulVectPlus(c byte, in, out []byte) {
	t := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= t[in[i]]
	}
}

// Reconstruct : reconstruct lost data & parity
func (e *encBase) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

// ReconstrcutData : reconstruct lost data
func (e *encBase) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func getLost(data int, vects [][]byte) (reconstMatrixPos, dataLost, parityLost []int) {
	cnt := 0
	for i, v := range vects {
		if v == nil {
			if i < data {
				dataLost = append(dataLost, i)
			} else {
				parityLost = append(parityLost, i)
			}
		} else {
			if cnt < data {
				reconstMatrixPos = append(reconstMatrixPos, i)
				cnt++
			}
		}
	}
	return reconstMatrixPos, dataLost, parityLost
}

func isMatchVectSize(size int, list []int, vects [][]byte) bool {
	for i := 1; i < len(list); i++ {
		if size != len(vects[list[i]]) {
			return false
		}
	}
	return true
}

type (
	reconstInfo struct {
		vectSize int
		data     reconstGen
		parity   reconstGen
	}
	reconstGen struct {
		posGen map[int]matrix
	}
)

func (e *encBase) getReconstInfo(vects [][]byte, dataOnly bool) (info reconstInfo, err error) {
	data := e.data
	parity := e.parity
	reconstMatrixPos, dataLost, parityLost := getLost(data, vects)
	if len(reconstMatrixPos) != data {
		err = errors.New(fmt.Sprintf("rs.Reconst: not enough vects, have: %d, data: %d", len(reconstMatrixPos), data))
		return
	}
	size := len(vects[reconstMatrixPos[0]])
	if size == 0 {
		err = errors.New("rs.Reconst: vects size = 0")
		return
	}
	if !isMatchVectSize(size, reconstMatrixPos, vects) {
		err = errors.New("rs.Reconst: vects size not match")
		return
	}

	em := e.encodeMatrix
	dm := newMatrix(data, data)
	for i, p := range reconstMatrixPos {
		dm[i] = em[p]
	}
	dgm, err := dm.invert(data)
	if err != nil {
		return
	}
	size := 0
	for i, v := range vects {
		if v != nil {
			s := len(v)
			if s != 0 {
				if size != 0 {
					if size != s {
						err = errors.New("rs.Reconst: vects size not match")
					}
				} else {
					size = s
				}
			}
			if s == 0 {
				err = ErrReconstVectEmpty
				return
			}
			if size == 0 {
				size = s
				have = append(have, i)
			} else {
				if size != s {
					err = ErrMatchVectSize
					return
				} else {
					have = append(have, i)
				}
			}
		} else {
			if i < data {
				dataLost = append(dataLost, i)
			} else {
				parityLost = append(parityLost, i)
			}
		}
	}
	if len(have) < data {
		err = ErrNoEnoughVects
		return
	}
	if len(dataLost)+len(parityLost) == 0 {
		err = ErrNoNeedRepair
		return
	}
	if len(dataLost)+len(parityLost) > parity {
		err = ErrNoEnoughVects
		return
	}
	if len(have)+len(parityLost) == data+parity && dataOnly {
		err = ErrNoNeedRepair
		return
	}
	info.have = have
	info.dataLost = dataLost
	info.parityLost = parityLost
	info.vectSize = size
	return
}

func checkReconstVects(data, parity int, vects [][]byte, dataOnly bool) (s reconstInfo, err error) {
	if data+parity != len(vects) {
		err = ErrMatchVects
		return
	}

	stat, err := getReconstInfo(data, parity, vects, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
}

func (e *encBase) reconst(vects [][]byte, dataOnly bool) (err error) {
	s, err := checkReconstVects(e.data, e.parity, vects, dataOnly)
	if err != nil {
		return
	}

	if len(stat.dataLost) > 0 {
		err := e.reconstData(vects, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		e.reconstParity(vects, stat.size, stat.parityLost)
	}
	return nil
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

func genReconstMatrix(shards matrix, data, parity, size int, have, dataLost []int) (dpTmp, gen matrix, err error) {
	e := genEncMatrixCauchy(data, parity)
	decodeM := newMatrix(data, data)
	numDL := len(dataLost)
	dpTmp = newMatrix(data+numDL, size)
	for i := 0; i < data; i++ {
		h := have[i]
		dpTmp[i] = shards[h]
		decodeM[i] = e[h]
	}
	for i, l := range dataLost {
		shards[l] = make([]byte, size)
		dpTmp[i+data] = shards[l]
	}
	decodeM, err = decodeM.invert()
	if err != nil {
		return
	}
	gen = newMatrix(numDL, data)
	for i, l := range dataLost {
		gen[i] = decodeM[l]
	}
	return
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
