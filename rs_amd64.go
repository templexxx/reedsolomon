package reedsolomon

import (
	"errors"
	"fmt"
)

func init() {
	getEXT()
}

// get CPU Instruction Extensions
func getEXT() {
	if hasAVX2() {
		extension = avx2
	} else if hasSSSE3() {
		extension = ssse3
	} else {
		extension = none
	}
	return
}

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

// limitShardsMC : data+parity must < limitShardsMC for having inverse matrix cache
// there is at most 38760 inverse matrix (data: 14, parity: 6, calculated by mathtool/cntinverse)
const (
	limitShardsMC = 33
	limitParityMC = 5
)
const (
	limitSmallShardsMC = 21
	limitSmallParityMC = 7
)

func cacheInverse(data, parity int) bool {
	shards := data + parity
	if shards < limitSmallShardsMC && parity < limitSmallParityMC {
		return true
	}
	if shards < limitShardsMC && parity < limitParityMC {
		return true
	}
	return false
}

// make generator_matrix's low&high tbl
func makeTbl(gen matrix, rows, cols int) []byte {
	tbl := make([]byte, 32*len(gen))
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j*rows+i]
			tbl := lowhighTbl[c][:]
			copy32B(tbl[off:off+32], tbl)
		}
		off += 32
	}
	return tbl
}

//go:noescape
func copy32B(dst, src []byte) // need SSE2, SSE2 introduced in 2001. So assume all amd64 has sse2

func newRS(data, parity int, encodeMatrix matrix) (enc EncodeReconster) {
	gen := encodeMatrix[data*data:]
	if extension == none {
		return &encBase{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen}
	}
	c := make(map[uint64]matrix)
	t := makeTbl(gen, parity, data)
	if extension == avx2 {
		if cacheInverse(data, parity) {
			return &encAVX2{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: true, inverseCache: matrixCache{cache: c}}
		} else {
			return &encAVX2{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: false}
		}
	} else {
		if cacheInverse(data, parity) {
			return &encSSSE3{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: true, inverseCache: matrixCache{cache: c}}
		} else {
			return &encSSSE3{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: false}
		}
	}
}

// size of sub-vector
const unitSize int = 16 * 1024

func (e *encAVX2) Encode(vects [][]byte) (err error) {
	err = checkEncVects(e.data, e.parity, vects)
	if err != nil {
		return
	}
	inVS := vects[:e.data]
	outVS := vects[e.data:]
	size := len(inVS[0])
	start, end := 0, 0
	do := unitSize
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, inVS, outVS)
			start = end
		} else {
			e.matrixMulRemain(start, size, inVS, outVS)
			start = size
		}
	}
	return
}

//go:noescape
func vectMulAVX2(tbl, inV, outV []byte)

//go:noescape
func vectMulPlusAVX2(tbl, inV, outV []byte)

func (e *encAVX2) matrixMul(start, end int, inVS, outVS [][]byte) {
	off := 0
	in := e.data
	out := e.parity
	for i := 0; i < out; i++ {
		t := e.tbl[off : off+32]
		vectMulAVX2(t, inVS[0][start:end], outVS[i][start:end])
		off += 32
	}
	for i := 1; i < in; i++ {
		for j := 0; j < out; j++ {
			t := e.tbl[off : off+32]
			vectMulPlusAVX2(t, inVS[i][start:end], outVS[j][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulAVX2_32B(tbl, inV, outV []byte)

//go:noescape
func vectMulPlusAVX2_32B(tbl, inV, outV []byte)

func (e *encAVX2) matrixMul32B(start, end int, inVS, outVS [][]byte) {
	in := e.data
	out := e.parity
	off := 0
	for i := 0; i < out; i++ {
		t := e.tbl[off : off+32]
		vectMulAVX2_32B(t, inVS[0][start:end], outVS[i][start:end])
		off += 32
	}
	for i := 1; i < in; i++ {
		for j := 0; j < out; j++ {
			t := e.tbl[off : off+32]
			vectMulPlusAVX2_32B(t, inVS[i][start:end], outVS[j][start:end])
			off += 32
		}
	}
}

func (e *encAVX2) matrixMulRemain(start, end int, inVS, outVS [][]byte) {
	undone := end - start
	if undone >= 32 {
		e.matrixMul32B(start, end, inVS, outVS)
	}
	done := (undone >> 5) << 5
	undone = undone - done
	if undone > 0 {
		in := e.data
		out := e.parity
		gen := e.genMatrix
		start = start + done
		for i := 0; i < in; i++ {
			for j := 0; j < out; j++ {
				if i == 0 {
					vectMul(gen[j*out+i], inVS[i][start:end], outVS[j][start:end])
				} else {
					vectMulPlus(gen[j*out+i], inVS[i][start:end], outVS[j][start:end])
				}
			}
		}
	}
}

func (e *encSSSE3) Encode(vects [][]byte) (err error) {
	err = checkEncVects(e.data, e.parity, vects)
	if err != nil {
		return
	}
	inVS := vects[:e.data]
	outVS := vects[e.data:]
	size := len(inVS[0])
	start, end := 0, 0
	do := unitSize
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, inVS, outVS)
			start = end
		} else {
			e.matrixMulRemain(start, size, inVS, outVS)
			start = size
		}
	}
	return
}

//go:noescape
func vectMulSSSE3(tbl, in, out []byte)

//go:noescape
func vectMulPlusSSSE3(tbl, in, out []byte)

func (e *encSSSE3) matrixMul(start, end int, inVS, outVS [][]byte) {
	off := 0
	in := e.data
	out := e.parity
	for i := 0; i < out; i++ {
		t := e.tbl[off : off+32]
		vectMulSSSE3(t, inVS[0][start:end], outVS[i][start:end])
		off += 32
	}
	for i := 1; i < in; i++ {
		for j := 0; j < out; j++ {
			t := e.tbl[off : off+32]
			vectMulPlusSSSE3(t, inVS[i][start:end], outVS[j][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulSSSE3_16B(tbl, inV, outV []byte)

//go:noescape
func vectMulPlusSSSE3_16B(tbl, inV, outV []byte)

func (e *encSSSE3) matrixMul16B(start, end int, inVS, outVS [][]byte) {
	in := e.data
	out := e.parity
	off := 0
	for i := 0; i < out; i++ {
		t := e.tbl[off : off+32]
		vectMulSSSE3_16B(t, inVS[0][start:end], outVS[i][start:end])
		off += 32
	}
	for i := 1; i < in; i++ {
		for j := 0; j < out; j++ {
			t := e.tbl[off : off+32]
			vectMulPlusSSSE3_16B(t, inVS[i][start:end], outVS[j][start:end])
			off += 32
		}
	}
}

func (e *encSSSE3) matrixMulRemain(start, end int, inVS, outVS [][]byte) {
	undone := end - start
	if undone >= 32 {
		e.matrixMul16B(start, end, inVS, outVS)
	}
	done := (undone >> 4) << 4
	undone = undone - done
	if undone > 0 {
		in := e.data
		out := e.parity
		gen := e.genMatrix
		start = start + done
		for i := 0; i < in; i++ {
			for j := 0; j < out; j++ {
				if i == 0 {
					vectMul(gen[j*out+i], inVS[i][start:end], outVS[j][start:end])
				} else {
					vectMulPlus(gen[j*out+i], inVS[i][start:end], outVS[j][start:end])
				}
			}
		}
	}
}

// set shard nil if lost
func (e *encAVX2) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

func (e *encAVX2) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func (e *encAVX2) getInverseCache(has []int) (matrix, error) {
	data := e.data
	parity := e.parity
	em := e.encodeMatrix
	if !e.enableInverseCache {
		return makeInverse(em, has, data)
	}
	cnt := 0
	for i := 0; i < data+parity; i++ {
		if 
	}
}

func makeInverse(em matrix, has []int, data int) (matrix, error) {
	dm := newMatrix(data, data)
	for i, p := range has {
		dm[i] = em[p]
	}
	dgm, err := dm.invert(data)
	if err != nil {
		return dgm, err
	}
	return dgm, nil
}

func (e *encAVX2) reconst(vects [][]byte, dataOnly bool) (err error) {
	data := e.data
	parity := e.parity
	if data+parity != len(vects) {
		return errors.New(fmt.Sprintf("rs.Enc: vects not match, data: %d parity: %d vects: %d", data, parity, len(vects)))
	}
	info, err := makeReconstInfo(data, vects, dataOnly)
	if err != nil {
		return
	}
	if info.dataOK && info.parityOK {
		return
	}
	em := e.encodeMatrix
	if !info.dataOK {
		dm := newMatrix(data, data)
		for i, p := range info.has {
			dm[i] = em[p]
		}
		if e.enableInverseCache {

		}
		dgm, err := dm.invert(data)
		if err != nil {
			return
		}
		dataLost := info.data
		rgData := make([]byte, len(dataLost)*data)
		for i, p := range dataLost {
			copy(rgData[i*data:i*data+data], dgm[p*data:p*data+data])
		}
		e.reconstData(vects, info.vectSize, dataLost, rgData)
	}
	if !info.parityOK {
		parityLost := info.parity
		rgParity := make([]byte, len(parityLost)*data)
		for i, p := range parityLost {
			copy(rgParity[i*data:i*data+data], em[data*data+p*data:data*data+p*data+data])
		}
		e.reconstParity(vects, info.vectSize, parityLost, rgParity)
	}
	return nil
	stat, err := getReconstInfo(e.data, e.parity, vects, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := e.reconstData(vects, stat.vectSize, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		e.reconstParity(vects, stat.vectSize, stat.parityLost)
	}
	return nil
}

func (e *encAVX2) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, e.data, e.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &encAVX2{data: e.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (e *encAVX2) reconstParity(shards matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(e.data, e.parity)
	numPL := len(parityLost)
	gen := newMatrix(numPL, e.data)
	for i, l := range parityLost {
		gen[i] = genTmp[l-e.data]
	}
	dpTmp := newMatrix(e.data+numPL, size)
	for i := 0; i < e.data; i++ {
		dpTmp[i] = shards[i]
	}
	for i, l := range parityLost {
		shards[l] = make([]byte, size)
		dpTmp[i+e.data] = shards[l]
	}
	e := &encAVX2{data: e.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}

func (r *encSSSE3) reconst(shards matrix, dataOnly bool) (err error) {
	stat, err := getReconstInfo(r.data, r.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := r.reconstData(shards, stat.vectSize, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		r.reconstParity(shards, stat.vectSize, stat.parityLost)
	}
	return nil
}

func (r *encSSSE3) Reconstruct(shards matrix) (err error) {
	return r.reconst(shards, false)
}

func (r *encSSSE3) ReconstructData(shards matrix) (err error) {
	return r.reconst(shards, true)
}

////////////// Internal Functions //////////////

func (r *encSSSE3) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, r.data, r.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &encSSSE3{data: r.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (r *encSSSE3) reconstParity(shards matrix, size int, parityLost []int) {
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
	e := &encSSSE3{data: r.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}
