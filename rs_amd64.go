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

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

//go:noescape
func copy32B(dst, src []byte) // Need SSE2(introduced in 2001)

func initTbl(g matrix, rows, cols int) []byte {
	tbl := make([]byte, 32*len(g))
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := g[j*cols+i]
			t := lowhighTbl[c][:]
			copy32B(tbl[off:off+32], t)
			off += 32
		}
	}
	return tbl
}

// At most 38760 inverse matrix (data: 14, parity: 6, calc by mathtool/cntinverse)
func okCache(data, parity int) bool {
	vects := data + parity
	if vects < 21 && parity < 7 {
		return true
	}
	if vects < 33 && parity < 5 {
		return true
	}
	return false
}

type (
	encAVX2  encSIMD
	encSSSE3 encSIMD
	encSIMD  struct {
		data         int
		parity       int
		total        int
		encode       matrix
		gen          matrix
		tbl          []byte
		enableCache  bool
		inverseCache sync.Map
	}
)

func newRS(d, p int, em matrix) (enc EncodeReconster) {
	g := em[d*d:]
	n := d + p
	ext := getEXT()
	if ext == none {
		return &encBase{data: d, parity: p, total: n, encode: em, gen: g}
	}
	t := initTbl(g, p, d)
	ok := okCache(d, p)
	if ext == avx2 {
		return &encAVX2{data: d, parity: p, total: n, encode: em, gen: g, tbl: t, enableCache: ok}
	}
	return &encSSSE3{data: d, parity: p, total: n, encode: em, gen: g, tbl: t, enableCache: ok}
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

func (e *encAVX2) Info() RSInfo {
	return RSInfo{Data: e.data, Parity: e.parity}
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
func mulVectAVX2(tbl, inV, outV []byte)

//go:noescape
func mulVectAddAVX2(tbl, inV, outV []byte)

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
		end = start + do
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
		start = end
	}
	if undone > do {
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
	}
}

func (e *encSSSE3) Info() RSInfo {
	return RSInfo{Data: e.data, Parity: e.parity}
}

func (e *encSSSE3) Encode(vects [][]byte) (err error) {
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
func mulVectSSSE3(tbl, inV, outV []byte)

//go:noescape
func mulVectAddSSSE3(tbl, inV, outV []byte)

func (e *encSSSE3) matrixMul(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	tbl := e.tbl
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := tbl[off : off+32]
			if i != 0 {
				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encSSSE3) matrixMulRemain(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	if do >= 16 {
		end = start + do
		tbl := e.tbl
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := tbl[off : off+32]
				if i != 0 {
					mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
				} else {
					mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
				}
				off += 32
			}
		}
		start = end
	}
	if undone > do {
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
	}
}

func (e *encAVX2) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

func (e *encAVX2) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func (e *encAVX2) ReconstWithPos(vects [][]byte, pos, dLost, pLost []int) error {
	return e.reconstWithPos(vects, pos, dLost, pLost, false)
}

func (e *encAVX2) ReconsDatatWithPos(vects [][]byte, pos, dLost []int) error {
	return e.reconstWithPos(vects, pos, dLost, nil, true)
}

func (e *encAVX2) makeInverse(pos []int) (im matrix, err error) {
	d := e.data
	em := e.encode
	if !e.enableCache {
		return makeInverse(em, pos, d)
	}
	var key uint32
	for _, h := range pos {
		key += 1 << uint8(h)
	}
	v, ok := e.inverseCache.Load(key)
	if ok {
		return v.(matrix), nil
	}
	m, err := makeInverse(em, pos, d)
	if err != nil {
		return nil, err
	}
	e.inverseCache.Store(key, m)
	return m, nil
}

func (e *encAVX2) reconstWithPos(vects [][]byte, pos, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	size, err := checkReconst(d, p, vects)
	if err != nil {
		return
	}

	em := e.encode
	dLCnt := len(dLost)
	if dLCnt != 0 {
		im, err2 := e.makeInverse(pos)
		if err2 != nil {
			return err2
		}
		g := make([]byte, dLCnt*d)
		for i, p := range dLost {
			copy(g[i*d:i*d+d], im[p*d:p*d+d])
		}
		vtmp := make([][]byte, d+dLCnt)
		j := 0
		for i, v := range vects {
			if v != nil {
				if j < d {
					vtmp[j] = vects[i]
					j++
				}
			}
		}
		for _, i := range dLost {
			vects[i] = make([]byte, size)
			vtmp[j] = vects[i]
			j++
		}
		t := initTbl(g, dLCnt, d)
		etmp := &encAVX2{data: d, parity: dLCnt, gen: g, tbl: t}
		etmp.Encode(vtmp)
	}
	pLCnt := len(pLost)
	if pLCnt != 0 && !dataOnly {
		g := make([]byte, pLCnt*d)
		for i, p := range pLost {
			copy(g[i*d:i*d+d], em[p*d:p*d+d])
		}
		vtmp := make([][]byte, d+pLCnt)
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			vects[p] = make([]byte, size)
			vtmp[d+i] = vects[p]
		}
		t := initTbl(g, pLCnt, d)
		etmp := &encAVX2{data: d, parity: dLCnt, gen: g, tbl: t}
		etmp.Encode(vtmp)
	}
	return nil
}

func (e *encAVX2) reconst(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	total := e.total
	has := 0
	k := 0
	pos := make([]int, d)
	dLost := make([]int, 0)
	pLost := make([]int, 0)
	for i := 0; i < total; i++ {
		if vects[i] != nil {
			has++
			if k < d {
				pos[k] = i
				k++
			}
		} else {
			if i < d {
				dLost = append(dLost, i)
			} else {
				pLost = append(pLost, i)
			}
		}
	}
	if has == total {
		return nil
	}
	if has < d {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconstWithPos(vects, pos, dLost, pLost, dataOnly)
}

func (e *encSSSE3) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

func (e *encSSSE3) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func (e *encSSSE3) ReconstWithPos(vects [][]byte, pos, dLost, pLost []int) error {
	return e.reconstWithPos(vects, pos, dLost, pLost, false)
}

func (e *encSSSE3) ReconsDatatWithPos(vects [][]byte, pos, dLost []int) error {
	return e.reconstWithPos(vects, pos, dLost, nil, true)
}

func (e *encSSSE3) makeInverse(pos []int) (im matrix, err error) {
	d := e.data
	em := e.encode
	if !e.enableCache {
		return makeInverse(em, pos, d)
	}
	var key uint32
	for _, h := range pos {
		key += 1 << uint8(h)
	}
	v, ok := e.inverseCache.Load(key)
	if ok {
		return v.(matrix), nil
	}
	m, err := makeInverse(em, pos, d)
	if err != nil {
		return nil, err
	}
	e.inverseCache.Store(key, m)
	return m, nil
}

func (e *encSSSE3) reconstWithPos(vects [][]byte, pos, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	size, err := checkReconst(d, p, vects)
	if err != nil {
		return
	}

	em := e.encode
	dLCnt := len(dLost)
	if dLCnt != 0 {
		im, err2 := e.makeInverse(pos)
		if err2 != nil {
			return err2
		}
		g := make([]byte, dLCnt*d)
		for i, p := range dLost {
			copy(g[i*d:i*d+d], im[p*d:p*d+d])
		}
		vtmp := make([][]byte, d+dLCnt)
		j := 0
		for i, v := range vects {
			if v != nil {
				if j < d {
					vtmp[j] = vects[i]
					j++
				}
			}
		}
		for _, i := range dLost {
			vects[i] = make([]byte, size)
			vtmp[j] = vects[i]
			j++
		}
		t := initTbl(g, dLCnt, d)
		etmp := &encSSSE3{data: d, parity: dLCnt, gen: g, tbl: t}
		etmp.Encode(vtmp)
	}
	pLCnt := len(pLost)
	if pLCnt != 0 && !dataOnly {
		g := make([]byte, pLCnt*d)
		for i, p := range pLost {
			copy(g[i*d:i*d+d], em[p*d:p*d+d])
		}
		vtmp := make([][]byte, d+pLCnt)
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			vects[p] = make([]byte, size)
			vtmp[d+i] = vects[p]
		}
		t := initTbl(g, pLCnt, d)
		etmp := &encSSSE3{data: d, parity: dLCnt, gen: g, tbl: t}
		etmp.Encode(vtmp)
	}
	return nil
}

func (e *encSSSE3) reconst(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	total := e.total
	has := 0
	k := 0
	pos := make([]int, d)
	dLost := make([]int, 0)
	pLost := make([]int, 0)
	for i := 0; i < total; i++ {
		if vects[i] != nil {
			has++
			if k < d {
				pos[k] = i
				k++
			}
		} else {
			if i < d {
				dLost = append(dLost, i)
			} else {
				pLost = append(pLost, i)
			}
		}
	}
	if has == total {
		return nil
	}
	if has < d {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconstWithPos(vects, pos, dLost, pLost, dataOnly)
}
