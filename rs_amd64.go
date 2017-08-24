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

// limitVectMC : data+parity must < limitVectMC for having inverse matrix cache
// there is at most 38760 inverse matrix (data: 14, parity: 6, calculated by mathtool/cntinverse)
const (
	limitVectMC        = 33
	limitParityMC      = 5
	limitSmallVectMC   = 21
	limitSmallParityMC = 7
)

func cacheInverse(data, parity int) bool {
	vects := data + parity
	if vects < limitSmallVectMC && parity < limitSmallParityMC {
		return true
	}
	if vects < limitVectMC && parity < limitParityMC {
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
	c := make(map[uint32]matrix)
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

func makeAVX2Do(size int) int {
	if size < unitSize {
		c := size / 128
		if c == 0 {
			return unitSize
		}
		return c * 128
	}
	return unitSize
}

func (e *encAVX2) Encode(vects [][]byte) (err error) {
	err = checkEncVects(e.data, e.parity, vects)
	if err != nil {
		return
	}
	inVS := vects[:e.data]
	outVS := vects[e.data:]
	size := len(inVS[0])
	start, end := 0, 0
	do := makeAVX2Do(size)
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

func makeSSSE3Do(size int) int {
	if size < unitSize {
		c := size / 32
		if c == 0 {
			return unitSize
		}
		return c * 32
	}
	return unitSize
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
	do := makeSSSE3Do(size)
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

func (e *encAVX2) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

func (e *encAVX2) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func (e *encAVX2) getInverseCache(has []int) (matrix, error) {
	data := e.data
	em := e.encodeMatrix
	if !e.enableInverseCache {
		return makeInverse(em, has, data)
	}
	var key uint32
	for _, h := range has {
		key += 1 << uint8(h)
	}
	e.inverseCache.RLock()
	m, ok := e.inverseCache.cache[key]
	if ok {
		e.inverseCache.RUnlock()
		return m, nil
	}
	e.inverseCache.RUnlock()
	m, err := makeInverse(em, has, data)
	if err != nil {
		return nil, err
	}
	e.inverseCache.Lock()
	e.inverseCache.cache[key] = m
	e.inverseCache.Unlock()
	return m, nil
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
		im, err2 := e.getInverseCache(info.has)
		if err2 != nil {
			return err2
		}
		dataLost := info.data
		rgData := make([]byte, len(dataLost)*data)
		for i, p := range dataLost {
			copy(rgData[i*data:i*data+data], im[p*data:p*data+data])
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
}

func (e *encAVX2) reconstData(vects [][]byte, size int, lost []int, gen matrix) {
	data := e.data
	out := len(lost)
	vtmp := make([][]byte, data+out)
	cnt := 0
	for i, v := range vects {
		if v != nil {
			if cnt < e.data {
				vtmp[cnt] = vects[i]
				cnt++
			}
		}
	}
	for _, p := range lost {
		vtmp[cnt] = vects[p]
		cnt++
	}
	t := makeTbl(gen, out, data)
	etmp := &encAVX2{data: data, parity: out, genMatrix: gen, tbl: t}
	etmp.Encode(vtmp)
}

func (e *encAVX2) reconstParity(vects [][]byte, size int, lost []int, gen matrix) {
	data := e.data
	out := len(lost)
	vtmp := make([][]byte, data+out)
	for i := 0; i < data; i++ {
		vtmp[i] = vects[i]
	}
	for i, p := range lost {
		vtmp[data+i] = vects[p]
	}
	t := makeTbl(gen, out, data)
	etmp := &encAVX2{data: e.data, parity: out, genMatrix: gen, tbl: t}
	etmp.Encode(vtmp)
}

func (e *encSSSE3) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

func (e *encSSSE3) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func (e *encSSSE3) getInverseCache(has []int) (matrix, error) {
	data := e.data
	em := e.encodeMatrix
	if !e.enableInverseCache {
		return makeInverse(em, has, data)
	}
	var key uint32
	for _, h := range has {
		key += 1 << uint8(h)
	}
	e.inverseCache.RLock()
	m, ok := e.inverseCache.cache[key]
	if ok {
		e.inverseCache.RUnlock()
		return m, nil
	}
	e.inverseCache.RUnlock()
	m, err := makeInverse(em, has, data)
	if err != nil {
		return nil, err
	}
	e.inverseCache.Lock()
	e.inverseCache.cache[key] = m
	e.inverseCache.Unlock()
	return m, nil
}

func (e *encSSSE3) reconst(vects [][]byte, dataOnly bool) (err error) {
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
		im, err2 := e.getInverseCache(info.has)
		if err2 != nil {
			return err2
		}
		dataLost := info.data
		rgData := make([]byte, len(dataLost)*data)
		for i, p := range dataLost {
			copy(rgData[i*data:i*data+data], im[p*data:p*data+data])
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
}

func (e *encSSSE3) reconstData(vects [][]byte, size int, lost []int, gen matrix) {
	data := e.data
	out := len(lost)
	vtmp := make([][]byte, data+out)
	cnt := 0
	for i, v := range vects {
		if v != nil {
			if cnt < e.data {
				vtmp[cnt] = vects[i]
				cnt++
			}
		}
	}
	for _, p := range lost {
		vtmp[cnt] = vects[p]
		cnt++
	}
	t := makeTbl(gen, out, data)
	etmp := &encSSSE3{data: data, parity: out, genMatrix: gen, tbl: t}
	etmp.Encode(vtmp)
}

func (e *encSSSE3) reconstParity(vects [][]byte, size int, lost []int, gen matrix) {
	data := e.data
	out := len(lost)
	vtmp := make([][]byte, data+out)
	for i := 0; i < data; i++ {
		vtmp[i] = vects[i]
	}
	for i, p := range lost {
		vtmp[data+i] = vects[p]
	}
	t := makeTbl(gen, out, data)
	etmp := &encSSSE3{data: e.data, parity: out, genMatrix: gen, tbl: t}
	etmp.Encode(vtmp)
}
