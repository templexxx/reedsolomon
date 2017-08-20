package reedsolomon

import "fmt"

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

func newRS(data, parity int, encodeMatrix matrix) (enc EncodeReconster) {
	if extension == none {
		return &encBase{data: data, parity: parity, em: encodeMatrix}
	}
	c := make(map[uint64]matrix)
	t := genTbls(encodeMatrix[data:])
	if extension == avx2 {
		return &encAVX2{data: data, parity: parity, em: encodeMatrix, mc: matrixCache{cache: c}, tbl: t}
	} else {
		return &encSSSE3{data: data, parity: parity, em: encodeMatrix, mc: matrixCache{cache: c}, tbl: t}
	}
}

// generate generator_matrix's low_high tbls
func genTbls(gen matrix) []byte {
	fmt.Print("sdf")
	rows := len(gen)
	cols := len(gen[0])
	tbls := make([]byte, 32*rows*cols)
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j][i]
			tbl := lowhighTbl[c][:]
			copy32B(tbls[off:off+32], tbl)
		}
		off += 32
	}
	return tbls
}

//go:noescape
func copy32B(dst, src []byte) // it need SSE2, first introduced in 2001. So assume all amd64 has sse2

// size of sub-vector
const unitSize int = 16 * 1024

func (e *encAVX2) Encode(shards matrix) (err error) {
	err = checkEncodeShards(e.data, e.parity, shards)
	if err != nil {
		return
	}
	in := shards[:e.data]
	out := shards[e.data:]
	size := len(in[0])
	start, end := 0, 0
	do := unitSize
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, in, out)
			start = end
		} else {
			e.matrixMulRemain(start, size, in, out)
			start = size
		}
	}
	return
}

func (e *encAVX2) matrixMul(start, end int, in, out matrix) {
	off := 0
	for i := 0; i < e.parity; i++ {
		t := e.tbl[off : off+32]
		vectMulAVX2(t, in[0][start:end], out[i][start:end])
		off += 32
	}
	for i := 1; i < e.data; i++ {
		for oi := 0; oi < e.parity; oi++ {
			t := e.tbl[off : off+32]
			vectMulPlusAVX2(t, in[i][start:end], out[oi][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulAVX2(tbl, in, out []byte) // coefficient multiply vect

//go:noescape
func vectMulPlusAVX2(tbl, in, out []byte) // coefficient multiply vect plus last result

func (e *encAVX2) matrixMulRemain(start, end int, in, out matrix) {
	undone := end - start
	if undone >= 32 {
		e.matrixMul32(start, end, in, out)
	}
	done := (undone >> 5) << 5
	undone = undone - done
	if undone > 0 {
		g := e.em[e.data:]
		start = start + done
		for i := 0; i < e.data; i++ {
			for oi := 0; oi < e.parity; oi++ {
				if i == 0 {
					vectMul(g[oi][i], in[i][start:end], out[oi][start:end])
				} else {
					vectMulPlus(g[oi][i], in[i][start:end], out[oi][start:end])
				}
			}
		}
	}
}

// coefficient multiply vect 32Bytes every loop
func (e *encAVX2) matrixMul32(start, end int, in, out matrix) {
	off := 0
	for i := 0; i < e.parity; i++ {
		t := e.tbl[off : off+32]
		vectMulAVX2Loop32(t, in[0][start:end], out[i][start:end])
		off += 32
	}
	for i := 1; i < e.data; i++ {
		for oi := 0; oi < e.parity; oi++ {
			t := e.tbl[off : off+32]
			vectMulPlusAVX2Loop32(t, in[i][start:end], out[oi][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulAVX2Loop32(tbl, in, out []byte)

//go:noescape
func vectMulPlusAVX2Loop32(tbl, in, out []byte)

func (e *encSSSE3) Encode(shards matrix) (err error) {
	err = checkEncodeShards(e.data, e.parity, shards)
	if err != nil {
		return
	}
	in := shards[:e.data]
	out := shards[e.data:]
	size := len(in[0])
	start, end := 0, 0
	do := unitSize
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, in, out)
			start = end
		} else {
			e.matrixMulRemain(start, size, in, out)
			start = size
		}
	}
	return
}

func (e *encSSSE3) matrixMul(start, end int, in, out matrix) {
	off := 0
	for i := 0; i < e.parity; i++ {
		t := e.tbl[off : off+32]
		vectMulSSSE3(t, in[0][start:end], out[i][start:end])
		off += 32
	}
	for i := 1; i < e.data; i++ {
		for oi := 0; oi < e.parity; oi++ {
			t := e.tbl[off : off+32]
			vectMulPlusSSSE3(t, in[i][start:end], out[oi][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulSSSE3(tbl, in, out []byte) // coefficient multiply vect

//go:noescape
func vectMulPlusSSSE3(tbl, in, out []byte) // coefficient multiply vect plus last result

func (e *encSSSE3) matrixMulRemain(start, end int, in, out matrix) {
	undone := end - start
	if undone >= 16 {
		e.matrixMul16(start, end, in, out)
	}
	done := (undone >> 4) << 4
	undone = undone - done
	if undone > 0 {
		g := e.em[e.data:]
		start = start + done
		for i := 0; i < e.data; i++ {
			for oi := 0; oi < e.parity; oi++ {
				if i == 0 {
					vectMul(g[oi][i], in[i][start:end], out[oi][start:end])
				} else {
					vectMulPlus(g[oi][i], in[i][start:end], out[oi][start:end])
				}
			}
		}
	}
}

// coefficient multiply vect 16Bytes every loop
func (e *encSSSE3) matrixMul16(start, end int, in, out matrix) {
	off := 0
	for i := 0; i < e.parity; i++ {
		t := e.tbl[off : off+32]
		vectMulSSSE3Loop16(t, in[0][start:end], out[i][start:end])
		off += 32
	}
	for i := 1; i < e.data; i++ {
		for oi := 0; oi < e.parity; oi++ {
			t := e.tbl[off : off+32]
			vectMulPlusSSSE3Loop16(t, in[i][start:end], out[oi][start:end])
			off += 32
		}
	}
}

//go:noescape
func vectMulSSSE3Loop16(tbl, in, out []byte)

//go:noescape
func vectMulPlusSSSE3Loop16(tbl, in, out []byte)

// set shard nil if lost
func (e *encAVX2) Reconstruct(shards matrix) (err error) {
	return e.reconst(shards, false)
}

func (e *encAVX2) ReconstructData(shards matrix) (err error) {
	return e.reconst(shards, true)
}

func (e *encAVX2) reconst(shards matrix, dataOnly bool) (err error) {
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
	stat, err := reconstInfo(r.data, r.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := r.reconstData(shards, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		r.reconstParity(shards, stat.size, stat.parityLost)
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
