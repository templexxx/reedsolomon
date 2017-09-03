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

func getEXT() int {
	if hasAVX2() {
		return avx2
	} else if hasSSSE3() {
		return ssse3
	} else {
		return none
	}
}

// TODO del after test
func (e *encAVX2) CloseCache() {
	e.enableCache = false
	return
}
func (e *encAVX2) OpenCache() {
	e.enableCache = true
	return
}
func (e *encSSSE3) CloseCache() {
	e.enableCache = false
	return
}
func (e *encSSSE3) OpenCache() {
	e.enableCache = true
	return
}

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

//go:noescape
func copy32B(dst, src []byte) // Need SSE2(introduced in 2001)

func initTbl(g matrix, rows, cols int, tbl []byte) {
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := g[j*cols+i]
			t := lowhighTbl[c][:]
			copy32B(tbl[off:off+32], t)
			off += 32
		}
	}
}

// At most 3060 inverse matrix (when data=18, parity=4, calc by mathtool/cntinverse)
// In practice,  data usually below 12, parity below 5
func okCache(data, parity int) bool {
	if data < 19 && parity < 5 {
		return true
	}
	return false
}

type (
	encSSSE3 encSIMD
	encAVX2  encSIMD
	encSIMD  struct {
		data   int
		parity int
		encode matrix
		gen    matrix
		tbl    []byte
		// all cache&buf here is design for small vect size ( < 4KB )
		// it will same time for calculating inverse matrix, initTbls & GC
		// but it's not so important for big vect
		// TODO add sync.pool maybe
		enableCache  bool
		inverseCache sync.Map
		tblCache     sync.Map
	}
)

func newRS(d, p int, em matrix) (enc EncodeReconster) {
	g := em[d*d:]
	ext := getEXT()
	if ext == none {
		return &encBase{data: d, parity: p, encode: em, gen: g}
	}
	t := make([]byte, d*p*32)
	initTbl(g, p, d, t)
	ok := okCache(d, p)
	//if ext == avx2 {
	e := &encAVX2{data: d, parity: p, encode: em, gen: g, tbl: t, enableCache: ok}
	return e
	//}
	//e := &encSSSE3{data: d, parity: p, encode: em, gen: g, tbl: t, enableCache: ok}
	//return e
}

// Size of sub-vector
const unit int = 16 * 1024

func getDo(n int) int {
	if n < unit {
		c := n >> 4
		if c == 0 {
			return unit
		}
		return c << 4
	}
	return unit
}

func (e *encAVX2) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemain(start, size, dv, pv)
			start = size
		}
	}
	return
}

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectAddAVX2(tbl, d, p []byte)

func (e *encAVX2) matrixMul(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	tbl := e.tbl
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := tbl[off : off+32]
			if i != 0 {
				mulVectAddAVX2(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectAVX2(t, dv[0][start:end], pv[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encAVX2) matrixMulRemain(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	if do >= 16 {
		end2 := start + do
		tbl := e.tbl
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := tbl[off : off+32]
				if i != 0 {
					mulVectAddAVX2(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectAVX2(t, dv[0][start:end2], pv[j][start:end2])
				}
				off += 32
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 < 0 {
			g := e.gen
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVect(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		} else {
			tbl := e.tbl
			off := 0
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := tbl[off : off+32]
					if i != 0 {
						mulVectAddAVX2(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectAVX2(t, dv[0][start2:end], pv[j][start2:end])
					}
					off += 32
				}
			}
		}
	}
}

func (e *encAVX2) Reconstruct(vects [][]byte) (err error) {
	return e.reconstruct(vects, false)
}

func (e *encAVX2) ReconstructData(vects [][]byte) (err error) {
	return e.reconstruct(vects, true)
}

func (e *encAVX2) ReconstWithPos(vects [][]byte, has, dLost, pLost []int) error {
	return e.reconstWithPos(vects, has, dLost, pLost, false)
}

func (e *encAVX2) ReconstDataWithPos(vects [][]byte, has, dLost []int) error {
	return e.reconstWithPos(vects, has, dLost, nil, true)
}

func (e *encAVX2) makeTbl(npos, dLost []int) (t, gen []byte, err error) {
	d := e.data
	em := e.encode
	cnt := len(dLost)
	baseLen := d * cnt
	matrixbuf := make([]byte, 4*d*d+cnt*d+32*cnt*d)
	if !e.enableCache {
		m := matrixbuf[:d*d]
		for i, l := range npos {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := matrixbuf[d*d : 3*d*d]
		im := matrixbuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return nil, nil, err2
		}
		g := matrixbuf[4*d*d : 4*d*d+cnt*d]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		tbl := matrixbuf[4*d*d+cnt*d:]
		initTbl(g, cnt, d, tbl)
		return tbl, g, nil
	}
	var ikey uint64
	for _, p := range npos {
		ikey += 1 << uint8(p)
	}
	tkey := ikey
	for _, p := range dLost {
		tkey += 1 << uint8(p+32)
	}
	gt, ok := e.tblCache.Load(tkey)
	if ok {
		g := gt.([]byte)[:cnt*d]
		tbl := gt.([]byte)[baseLen : baseLen+baseLen*32]
		return tbl, g, nil
	}
	v, ok := e.inverseCache.Load(ikey)
	if ok {
		im := v.(matrix)
		g := matrixbuf[4*d*d : 4*d*d+cnt*d]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		tbl := matrixbuf[4*d*d+cnt*d:]
		initTbl(g, cnt, d, tbl)
		e.tblCache.Store(tkey, matrixbuf[4*d*d:])
		return tbl, g, nil
	}
	m := matrixbuf[:d*d]
	for i, l := range npos {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := matrixbuf[d*d : 3*d*d]
	im := matrixbuf[3*d*d : 4*d*d]
	err2 := matrix(m).invert(raw, d, im)
	if err2 != nil {
		return nil, nil, err2
	}
	e.inverseCache.Store(ikey, im)

	g := matrixbuf[4*d*d : 4*d*d+cnt*d]
	for i, l := range dLost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	tbl := matrixbuf[4*d*d+cnt*d:]
	initTbl(g, cnt, d, tbl)
	e.tblCache.Store(tkey, matrixbuf[4*d*d:])
	return tbl, g, nil
}

// TODO maybe don't need make lost vect, just = vects[i]
func (e *encAVX2) reconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	em := e.encode
	dCnt := len(dLost)
	size := len(vects[has[0]])
	if dCnt != 0 {
		npos := make([]int, d)
		for i := range npos {
			npos[i] = i // init new position
		}
		dpos := has[d-dCnt:] //  lost-data vects will be replaced by these parity
		// lost-data need a new place if their position aren't beginning
		// with vects[d], key:old-place value:new-place
		dnpos := make(map[int]int)
		for i, l := range dLost {
			if vects[l] == nil {
				vects[l] = make([]byte, size)
			}
			vects[l], vects[dpos[i]] = vects[dpos[i]], vects[l]
			npos[l] = dpos[i]
			if dpos[i] != i+d {
				dnpos[dpos[i]] = i + d
				vects[i+d], vects[dpos[i]] = vects[dpos[i]], vects[i+d]
			}
		}
		t, g, err2 := e.makeTbl(npos, dLost)
		if err2 != nil {
			return
		}
		etmp := &encAVX2{data: d, parity: dCnt, gen: g, tbl: t}
		err2 = etmp.Encode(vects[:d+dCnt])
		if err2 != nil {
			return err2
		}

		// swap vects back
		if dCnt == p {
			for i, l := range dLost {
				vects[l], vects[dpos[i]] = vects[dpos[i]], vects[l]
			}
		} else {
			for i := d + p - 1; i >= d; i-- {
				if v, ok := dnpos[i]; ok {
					vects[i], vects[v] = vects[v], vects[i]
				}
			}
			for i, l := range dLost {
				vects[l], vects[dpos[i]] = vects[dpos[i]], vects[l]
			}
		}
	}
	if dataOnly {
		return
	}
	pCnt := len(pLost)
	if pCnt != 0 {
		// TODO add parity lost sync.map
		gt := make([]byte, pCnt*d+32*pCnt*d)
		g := gt[:pCnt*d]
		for i, l := range pLost {
			copy(g[i*d:i*d+d], em[l*d:l*d+d])
		}
		t := gt[pCnt*d:]
		initTbl(g, pCnt, d, t)
		for i, l := range pLost {
			if vects[l] == nil {
				vects[l] = make([]byte, size)
			}
			if l != d+i {
				vects[l], vects[d+i] = vects[d+i], vects[l]
			}
		}
		etmp := &encAVX2{data: d, parity: pCnt, gen: g, tbl: t}
		err2 := etmp.Encode(vects[:d+pCnt])
		if err2 != nil {
			return err2
		}
		for i, l := range pLost {
			if l != d+i {
				vects[l], vects[d+i] = vects[d+i], vects[l]
			}
		}
	}
	return
}

func (e *encAVX2) reconstWithPos(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	// TODO check more, maybe element in has show in lost & deal with len(has) > d
	if len(has) != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dCnt := len(dLost)
	if dCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	pCnt := len(pLost)
	if pCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

func (e *encAVX2) reconstruct(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	t := d + p
	// TODO do I need sync.pool here?
	listBuf := make([]int, t+p)
	has := listBuf[:d]
	dLost := listBuf[d:t]
	pLost := listBuf[t : t+p]
	hasCnt, dCnt, pCnt := 0, 0, 0
	for i := 0; i < t; i++ {
		if vects[i] != nil {
			if hasCnt < d {
				has[hasCnt] = i
				hasCnt++
			}
		} else {
			if i < d {
				if dCnt < p {
					dLost[dCnt] = i
					dCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			} else {
				if pCnt < p {
					pLost[pCnt] = i
					pCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			}
		}
	}
	if hasCnt != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dLost = dLost[:dCnt]
	pLost = pLost[:pCnt]
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

//func (e *encSSSE3) Encode(vects [][]byte) (err error) {
//	d := e.data
//	p := e.parity
//	size, err := checkEnc(d, p, vects)
//	if err != nil {
//		return
//	}
//	dv := vects[:d]
//	pv := vects[d:]
//	start, end := 0, 0
//	do := getDo(size)
//	for start < size {
//		end = start + do
//		if end <= size {
//			e.matrixMul(start, end, dv, pv)
//			start = end
//		} else {
//			e.matrixMulRemain(start, size, dv, pv)
//			start = size
//		}
//	}
//	return
//}
//
////go:noescape
//func mulVectSSSE3(tbl, d, p []byte)
//
////go:noescape
//func mulVectAddSSSE3(tbl, d, p []byte)
//
//func (e *encSSSE3) matrixMul(start, end int, dv, pv [][]byte) {
//	d := e.data
//	p := e.parity
//	tbl := e.tbl
//	off := 0
//	for i := 0; i < d; i++ {
//		for j := 0; j < p; j++ {
//			t := tbl[off : off+32]
//			if i != 0 {
//				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
//			} else {
//				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
//			}
//			off += 32
//		}
//	}
//}
//
//func (e *encSSSE3) matrixMulRemain(start, end int, dv, pv [][]byte) {
//	undone := end - start
//	do := (undone >> 4) << 4
//	d := e.data
//	p := e.parity
//	if do >= 16 {
//		end = start + do
//		tbl := e.tbl
//		off := 0
//		for i := 0; i < d; i++ {
//			for j := 0; j < p; j++ {
//				t := tbl[off : off+32]
//				if i != 0 {
//					mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
//				} else {
//					mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
//				}
//				off += 32
//			}
//		}
//		start = end
//	}
//	if undone > do {
//		g := e.gen
//		for i := 0; i < d; i++ {
//			for j := 0; j < p; j++ {
//				if i != 0 {
//					mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
//				} else {
//					mulVect(g[j*d], dv[0][start:], pv[j][start:])
//				}
//			}
//		}
//	}
//}
