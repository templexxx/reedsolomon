package reedsolomon

import "sync"

// SIMD Instruction Extensions
const (
	none = iota
	avx2
	ssse3
)

func getEXT() int {
	if hasAVX2() {
		return avx2
	} else if hasSSSE3() {
		return ssse3
	} else {
		return none
	}
}

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

// at most 38760 inverse matrix (data: 14, parity: 6, calc by mathtool/cntinverse)
func cacheInverse(data, parity int) bool {
	vects := data + parity
	if vects < 21 && parity < 7 {
		return true
	}
	if vects < 33 && parity < 5 {
		return true
	}
	return false
}

//go:noescape
func copy32B(dst, src []byte) // need SSE2(introduced in 2001)

func initTbl(gen matrix, rows, cols int) []byte {
	tbl := make([]byte, 32*len(gen))
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j*cols+i]
			t := lowhighTbl[c][:]
			copy32B(tbl[off:off+32], t)
			off += 32
		}
	}
	return tbl
}

type (
	encAVX2  encSIMD
	encSSSE3 encSIMD
	encSIMD  struct {
		data               int
		parity             int
		encodeMatrix       matrix
		genMatrix          matrix
		tbl                []byte //  multiply-tables of element in generator-matrix
		enableInverseCache bool
		// TODO *sync.map
		inverseCache matrixCache // inverse matrix's cache
	}
	matrixCache struct {
		sync.RWMutex
		cache map[uint32]matrix
	}
)

func newRS(data, parity int, encodeMatrix matrix) (enc EncodeReconster) {
	gen := encodeMatrix[data*data:]
	ext := getEXT()
	if ext == none {
		return &encBase{data: data, parity: parity, total: data + parity, encodeMatrix: encodeMatrix, genMatrix: gen}
	}
	t := initTbl(gen, parity, data)
	enable := cacheInverse(data, parity)

	if ext == avx2 {
		if enable {
			c := make(map[uint32]matrix)
			return &encAVX2{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: true, inverseCache: matrixCache{cache: c}}
		}
		return &encAVX2{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: false}
	}
	if enable {
		c := make(map[uint32]matrix)
		return &encSSSE3{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: true, inverseCache: matrixCache{cache: c}}
	}
	return &encSSSE3{data: data, parity: parity, encodeMatrix: encodeMatrix, genMatrix: gen, tbl: t, enableInverseCache: false}
}

// size of sub-vector
const unitSize int = 16 * 1024

func makeDo(size int) int {
	if size < unitSize {
		c := size >> 4
		if c == 0 {
			return unitSize
		}
		return c << 4
	}
	return unitSize
}

func (e *encAVX2) Encode(vects [][]byte) (err error) {
	size, err := checkEnc(e.data, e.parity, vects)
	if err != nil {
		return
	}
	inVS := vects[:e.data]
	outVS := vects[e.data:]
	start, end := 0, 0
	do := makeDo(size)
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
	for i := 0; i < e.data; i++ {
		for j := 0; j < e.parity; j++ {
			t := e.tbl[off : off+32]
			if i != 0 {
				vectMulPlusAVX2(t, inVS[i][start:end], outVS[j][start:end])
			} else {
				vectMulAVX2(t, inVS[0][start:end], outVS[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encAVX2) matrixMulRemain(start, end int, inVS, outVS [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	if do >= 16 {
		e.matrixMul(start, start+do, inVS, outVS)
	}
	if undone > do {
		in := e.data
		out := e.parity
		start += do
		// TODO sync.pool or sync.mutex
		// it will alloc on stack i think
		// TODO need escape analyse here
		buf := make([][]byte, in+out)
		for i := range buf {
			buf[i] = make([]byte, 16)
		}
		inTmp := buf[:in]
		for i := range inTmp {
			copy(inTmp[i], inVS[i][start:])
		}
		outTmp := buf[in:]
		for i := range outTmp {
			copy(outTmp[i], outVS[i][start:])
		}
		off := 0
		for i := 0; i < in; i++ {
			for j := 0; j < out; j++ {
				t := e.tbl[off : off+32]
				if i != 0 {
					vectMulPlusAVX2(t, inTmp[i], outTmp[j])
				} else {
					vectMulAVX2(t, inTmp[0], outTmp[j])
				}
				off += 32
			}
		}
		for i := range outTmp {
			copy(outVS[i][start:], outTmp[i])
		}
	}
}

func (e *encSSSE3) Encode(vects [][]byte) (err error) {
	size, err := checkEnc(e.data, e.parity, vects)
	if err != nil {
		return
	}
	inVS := vects[:e.data]
	outVS := vects[e.data:]
	start, end := 0, 0
	do := makeDo(size)
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
	for i := 0; i < e.data; i++ {
		for j := 0; j < e.parity; j++ {
			t := e.tbl[off : off+32]
			if i != 0 {
				vectMulPlusSSSE3(t, inVS[i][start:end], outVS[j][start:end])
			} else {
				vectMulSSSE3(t, inVS[0][start:end], outVS[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encSSSE3) matrixMulRemain(start, end int, inVS, outVS [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	if do >= 16 {
		e.matrixMul(start, start+do, inVS, outVS)
	}
	if undone > do {
		start += do
		g := e.genMatrix
		for i := 0; i < e.data; i++ {
			for j := 0; j < e.parity; j++ {
				if i != 0 {
					mulVectAdd(g[j*e.data+i], inVS[i][start:], outVS[j][start:])
				} else {
					mulVect(g[j*e.data], inVS[0][start:], outVS[j][start:])
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
	//data := e.data
	//parity := e.parity
	//info, err := makeReconstInfo(data, parity, vects, dataOnly)
	//if err != nil {
	//	return
	//}
	//if info.okData && info.okParity {
	//	return
	//}
	//em := e.encodeMatrix
	//if !info.okData {
	//	im, err2 := e.getInverseCache(info.has)
	//	if err2 != nil {
	//		return err2
	//	}
	//	dataLost := info.data
	//	rgData := make([]byte, len(dataLost)*data)
	//	for i, p := range dataLost {
	//		copy(rgData[i*data:i*data+data], im[p*data:p*data+data])
	//	}
	//	e.reconstData(vects, info.vectSize, dataLost, rgData)
	//}
	//if !info.okParity {
	//	parityLost := info.parity
	//	rgParity := make([]byte, len(parityLost)*data)
	//	for i, p := range parityLost {
	//		copy(rgParity[i*data:i*data+data], em[data*data+p*data:data*data+p*data+data])
	//	}
	//	e.reconstParity(vects, info.vectSize, parityLost, rgParity)
	//}
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
	t := initTbl(gen, out, data)
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
	t := initTbl(gen, out, data)
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
	//data := e.data
	//parity := e.parity
	//info, err := makeReconstInfo(data, parity, vects, dataOnly)
	//if err != nil {
	//	return
	//}
	//if info.okData && info.okParity {
	//	return
	//}
	//em := e.encodeMatrix
	//if !info.okData {
	//	im, err2 := e.getInverseCache(info.has)
	//	if err2 != nil {
	//		return err2
	//	}
	//	dataLost := info.data
	//	rgData := make([]byte, len(dataLost)*data)
	//	for i, p := range dataLost {
	//		copy(rgData[i*data:i*data+data], im[p*data:p*data+data])
	//	}
	//	e.reconstData(vects, info.vectSize, dataLost, rgData)
	//}
	//if !info.okParity {
	//	parityLost := info.parity
	//	rgParity := make([]byte, len(parityLost)*data)
	//	for i, p := range parityLost {
	//		copy(rgParity[i*data:i*data+data], em[data*data+p*data:data*data+p*data+data])
	//	}
	//	e.reconstParity(vects, info.vectSize, parityLost, rgParity)
	//}
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
	t := initTbl(gen, out, data)
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
	t := initTbl(gen, out, data)
	etmp := &encSSSE3{data: e.data, parity: out, genMatrix: gen, tbl: t}
	etmp.Encode(vtmp)
}
