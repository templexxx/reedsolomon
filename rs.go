// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// Package reedsolomon implements Erasure Codes (systematic codes),
// it's based on:
// Reed-Solomon Codes over GF(2^8).
// Primitive Polynomial:  x^8+x^4+x^3+x^2+1.
//
// Galois Filed arithmetic using Intel SIMD instructions (AVX512 or AVX2).
package reedsolomon

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/templexxx/cpu"
	xor "github.com/templexxx/xorsimd"
)

// RS Reed-Solomon Codes receiver.
type RS struct {
	DataNum   int // DataNum is the number of data row vectors.
	ParityNum int // ParityNum is the number of parity row vectors.

	// CPU's feature.
	// With SIMD feature, performance will be much better.
	cpuFeat int

	encMatrix matrix // Encoding matrix.
	GenMatrix matrix // Generator matrix.

	inverseCacheEnabled bool
	inverseCache        *sync.Map // Inverse matrix's cache.
	// Limitation of cache, total inverse matrix = C(DataNum+ParityNum, DataNum)
	// = (DataNum+ParityNum)! / ParityNum!DataNum!
	// If there is no limitation, memory will explode. See mathtool/cntinverse for details.
	inverseCacheMax uint64
	inverseCacheN   uint64 // cached inverse matrix.

	*gmu
}

var ErrIllegalVects = errors.New("illegal data/parity number: <= 0 or data+parity > 256")

const (
	maxVects                   = 256
	kib                        = 1024
	mib                        = 1024 * kib
	maxInverseMatrixCapInCache = 16 * mib // Keeping inverse matrix cache small, 16 MiB is enough for most cases.
)

// New create an RS with specific data & parity numbers.
func New(dataNum, parityNum int) (r *RS, err error) {

	return newWithFeature(dataNum, parityNum, featUnknown)
}

func newWithFeature(dataNum, parityNum, feat int) (r *RS, err error) {
	d, p := dataNum, parityNum
	if d <= 0 || p <= 0 || d+p > maxVects {
		return nil, ErrIllegalVects
	}

	e := makeEncodeMatrix(d, p)
	g := e[d*d:]
	r = &RS{DataNum: d, ParityNum: p,
		encMatrix: e, GenMatrix: g}

	if r.DataNum+r.ParityNum <= 64 { // I'm using 64bit bitmap as inverse matrix cache's key.
		r.inverseCacheEnabled = true
		r.inverseCache = new(sync.Map)
		r.inverseCacheMax = maxInverseMatrixCapInCache / uint64(r.DataNum) / uint64(r.DataNum)
	}

	r.cpuFeat = feat
	if r.cpuFeat == featUnknown {
		r.cpuFeat = getCPUFeature()
	}

	r.gmu = new(gmu)
	r.initFunc(r.cpuFeat)

	return
}

// CPU Features.
const (
	featUnknown = iota
	featAVX2
	featNoSIMD
)

func getCPUFeature() int {
	if cpu.X86.HasAVX2 {
		return featAVX2
	}
	return featNoSIMD
}

// Encode encodes data for generating parity.
// It multiplies generator matrix by vects[:r.DataNum] to get parity vectors,
// and write into vects[r.DataNum:].
func (r *RS) Encode(vects [][]byte) (err error) {
	err = r.checkEncode(vects)
	if err != nil {
		return
	}
	r.encode(vects, false)
	return
}

var (
	ErrMismatchVects    = errors.New("too few/many vectors given")
	ErrZeroVectSize     = errors.New("vector size is 0")
	ErrMismatchVectSize = errors.New("vectors size mismatched")
)

func (r *RS) checkEncode(vects [][]byte) (err error) {
	rows := len(vects)
	if r.DataNum+r.ParityNum != rows {
		return ErrMismatchVects
	}
	size := len(vects[0])
	if size == 0 {
		return ErrZeroVectSize
	}
	for i := 1; i < rows; i++ {
		if len(vects[i]) != size {
			return ErrMismatchVectSize
		}
	}
	return
}

// encode data piece by piece.
// Split vectors for cache-friendly (see func getSplitSize(n int) int for details).
//
// updateOnly: means update old results by XOR new results, but not write new results directly.
// You can see Methods Encode and Update to figure out difference.
func (r *RS) encode(vects [][]byte, updateOnly bool) {
	dv, pv := vects[:r.DataNum], vects[r.DataNum:]
	size := len(vects[0])
	splitSize := getSplitSize(size)
	start := 0
	for start < size {
		end := start + splitSize
		if end > size {
			end = size
		}
		r.encodePart(start, end, dv, pv, updateOnly)
		start = end
	}
}

// size must be divisible by 16,
// it's the smallest size for SIMD instructions,
// see code block one16b in *_amd64.s for more details.
func getSplitSize(n int) int {
	l1d := cpu.X86.Cache.L1D
	if l1d <= 0 { // Cannot detect cache size(-1) or CPU is not X86(0).
		l1d = 32 * 1024
	}

	if n < 16 {
		return 16
	}
	// Half of L1 Data Cache Size is an empirical data.
	// Fit L1 Data Cache Size, but won't pollute too much in the next round.
	if n < l1d/2 {
		return (n >> 4) << 4
	}
	return l1d / 2
}

func (r *RS) encodePart(start, end int, dv, pv [][]byte, updateOnly bool) {
	undone := end - start
	do := (undone >> 4) << 4 // do could be 0(when undone < 16)
	d, p, g := r.DataNum, r.ParityNum, r.GenMatrix
	if do >= 16 {
		end2 := start + do
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly {
					r.mulVectXOR(g[j*d+i], dv[i][start:end2], pv[j][start:end2])
				} else {
					r.mulVect(g[j*d+i], dv[0][start:end2], pv[j][start:end2])
				}
			}
		}
	}

	if undone > do { // 0 < undone-do < 16
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly {
					mulVectXORNoSIMD(g[j*d+i], dv[i][start:end], pv[j][start:end])
				} else {
					mulVectNoSIMD(g[j*d], dv[0][start:end], pv[j][start:end])
				}
			}
		}
	}
}

// Reconst reconstructs missing vectors,
// vects: All vectors, len(vects) = dataNum + parityNum.
// survived: Survived data & parity indexes, len(survived) must >= dataNum.
// needReconst: Vectors index which need to be reconstructed.
// needReconst has higher priority than survived:
// e.g., survived: [1,2,3] needReconst [0,1] -> survived: [2,3] needReconst [0,1]
// When len(survived) == 0, assuming all vectors survived, will be refreshed by needReconst later:
// survived vectors must have correct data.
//
// e.g.,:
// in 3+2, the whole index: [0,1,2,3,4],
// if vects[0,4] are lost & they need to be reconstructed
// (Maybe you only need to reconstruct vects[0] when lost vects[0,4], so the needReconst should be [0], but not [0,4]).
// the survived will be [1,2,3] ,and you must be sure that vects[1,2,3] have correct data,
// results will be written into vects[needReconst] directly.
func (r *RS) Reconst(vects [][]byte, survived, needReconst []int) (err error) {

	var dataNeedReconstN int
	survived, needReconst, dataNeedReconstN, err = r.checkReconst(survived, needReconst)
	if err != nil {
		if err == ErrNoNeedReconst {
			return nil
		}
		return
	}

	err = r.reconstData(vects, survived, needReconst[:dataNeedReconstN])
	if err != nil {
		return
	}
	return r.reconstParity(vects, needReconst[dataNeedReconstN:])
}

var (
	ErrNoNeedReconst = errors.New("no need reconst")
	ErrTooManyLost   = errors.New("too many lost")
)

const (
	vectUnknown     = uint8(0)
	vectSurvived    = uint8(1)
	vectNeedReconst = uint8(2)
)

func checkVectIdx(idx []int, d, p int) error {
	n := d + p
	for _, i := range idx {
		if i < 0 || i >= n {
			return ErrIllegalVects
		}
	}
	return nil
}

// check arguments, return:
// 1. survived index
// 2. data & parity indexes which needed to be reconstructed (sorted after return)
// 3. cnt of data vectors needed to be reconstructed.
func (r *RS) checkReconst(survived, needReconst []int) (vs, nr []int, dn int, err error) {
	if len(needReconst) == 0 {
		err = ErrNoNeedReconst
		return
	}

	d, p := r.DataNum, r.ParityNum

	if err = checkVectIdx(survived, d, p); err != nil {
		return
	}
	if err = checkVectIdx(needReconst, d, p); err != nil {
		return
	}

	status := make([]uint8, d+p)

	if len(survived) == 0 { // Set all survived if no given survived index.
		for i := range status {
			status[i] = vectSurvived
		}
	}
	for _, v := range survived {
		status[v] = vectSurvived
	}

	fullDataRequired := false
	for _, v := range needReconst {
		status[v] = vectNeedReconst // Origin survived status will be replaced if they're conflicting.
		if !fullDataRequired && v >= d {
			fullDataRequired = true // Need to reconstruct parity, full data vectors required.
		}
	}
	if fullDataRequired {
		for i, v := range status[:d] {
			if v == vectUnknown {
				status[i] = vectNeedReconst
			}
		}
	}

	ints := make([]int, d+2*p)
	vs = ints[:d+p][:0]
	nr = ints[d+p:][:0]
	for i, s := range status {
		switch s {
		case vectSurvived:
			vs = append(vs, i)
		case vectNeedReconst:
			if i < d {
				dn++
			}
			nr = append(nr, i)
		}
	}

	if len(vs) < d || len(nr) > p {
		err = ErrTooManyLost
		return
	}
	return
}

func (r *RS) reconstData(vects [][]byte, survived, needReconst []int) (err error) {

	nn := len(needReconst)
	if nn == 0 {
		return nil
	}

	d := r.DataNum
	survived = survived[:d] // Only need dataNum vectors for reconstruction.

	gm, err := r.getReconstMatrix(survived, needReconst)
	if err != nil {
		return
	}
	vs := make([][]byte, d+nn)
	for i, row := range survived {
		vs[i] = vects[row]
	}
	for i, row := range needReconst {
		vs[i+d] = vects[row]
	}
	return r.reconst(vs, gm, nn)
}

func (r *RS) reconstParity(vects [][]byte, needReconst []int) (err error) {

	nn := len(needReconst)
	if nn == 0 {
		return nil
	}

	d := r.DataNum
	gm := make([]byte, nn*d)
	for i, l := range needReconst {
		copy(gm[i*d:i*d+d], r.encMatrix[l*d:l*d+d])
	}

	vs := make([][]byte, d+nn)
	for i := 0; i < d; i++ {
		vs[i] = vects[i]
	}
	for i, p := range needReconst {
		vs[i+d] = vects[p]
	}

	return r.reconst(vs, gm, nn)
}

func (r *RS) reconst(vects [][]byte, gm matrix, pn int) error {

	rTmp := &RS{DataNum: r.DataNum, ParityNum: pn, GenMatrix: gm, cpuFeat: r.cpuFeat, gmu: r.gmu}
	return rTmp.Encode(vects)

}

func (r *RS) getReconstMatrix(survived, needReconst []int) (rm []byte, err error) {

	if !r.inverseCacheEnabled {
		em, err2 := r.encMatrix.makeEncMatrixForReconst(survived)
		if err2 != nil {
			return nil, err2
		}
		return em.makeReconstMatrix(survived, needReconst)
	}
	return r.getReconstMatrixFromCache(survived, needReconst)
}

func (r *RS) getReconstMatrixFromCache(survived, needReconst []int) (rm matrix, err error) {

	key := makeInverseCacheKey(survived)

	emRaw, ok := r.inverseCache.Load(key)
	if ok {
		em := emRaw.(matrix)
		return em.makeReconstMatrix(survived, needReconst)
	}

	em, err := r.encMatrix.makeEncMatrixForReconst(survived)
	if err != nil {
		return
	}
	if atomic.AddUint64(&r.inverseCacheN, 1) <= r.inverseCacheMax {
		r.inverseCache.Store(key, em)
	}
	return em.makeReconstMatrix(survived, needReconst)
}

func makeInverseCacheKey(survived []int) uint64 {
	var key uint64
	for _, i := range survived {
		key += 1 << uint8(i) // elements in survived are unique and sorted, okay to use add.
	}
	return key
}

// Update updates parity_data when one data_vect changes.
// row: It's the new data's index in the whole vectors.
func (r *RS) Update(oldData []byte, newData []byte, row int, parity [][]byte) (err error) {

	err = r.checkUpdate(oldData, newData, row, parity)
	if err != nil {
		return
	}

	// Step1: old_data xor new_data.
	buf := make([]byte, len(oldData))
	xor.Encode(buf, [][]byte{oldData, newData})

	// Step2: recalculate parity.
	vects := make([][]byte, 1+r.ParityNum)
	vects[0] = buf
	gm := make([]byte, r.ParityNum)
	for i := 0; i < r.ParityNum; i++ {
		col := row
		off := i*r.DataNum + col
		c := r.GenMatrix[off]
		gm[i] = c
		vects[i+1] = parity[i]
	}
	rs := &RS{DataNum: 1, ParityNum: r.ParityNum, GenMatrix: gm, cpuFeat: r.cpuFeat, gmu: r.gmu}
	rs.encode(vects, true)
	return nil
}

var (
	ErrMismatchParityNum = errors.New("parity number mismatched")
	ErrIllegalVectIndex  = errors.New("illegal vect index")
)

func (r *RS) checkUpdate(oldData []byte, newData []byte, row int, parity [][]byte) (err error) {
	if len(parity) != r.ParityNum {
		return ErrMismatchParityNum
	}
	size := len(newData)
	if size == 0 {
		return ErrZeroVectSize
	}
	if size != len(oldData) {
		return ErrMismatchVectSize
	}

	for i := range parity {
		if len(parity[i]) != size {
			return ErrMismatchVectSize
		}
	}
	if row >= r.DataNum || row < 0 {
		return ErrIllegalVectIndex
	}
	return
}

// Replace replaces oldData vectors with 0 or replaces 0 with newData vectors.
//
// It's used in two situations:
// 1. We didn't have enough data for filling in a stripe, but still did ec encode,
// we need replace several zero vectors with new vectors which have data after we get enough data finally.
// 2. After compact, we may have several useless vectors in a stripe,
// we need replaces these useless vectors with zero vectors for free space.
//
// In practice,
// If len(replaceRows) > dataNum-parityNum, it's better to use Encode,
// because Replace need to read len(replaceRows) + parityNum vectors,
// if replaceRows are too many, the cost maybe larger than Encode
// (Encode only need read dataNum).
//
// Warn:
// data's index & replaceRows must have the same sort.
func (r *RS) Replace(data [][]byte, replaceRows []int, parity [][]byte) (err error) {

	err = r.checkReplace(data, replaceRows, parity)
	if err != nil {
		return
	}

	d, p := r.DataNum, r.ParityNum
	rn := len(replaceRows)

	// Make generator matrix for replacing.
	//
	// Values in replaceRows are row index of data,
	// and also the column index of generator matrix
	gm := make([]byte, p*rn)
	off := 0
	for i := 0; i < p; i++ {
		for j := 0; j < rn; j++ {
			k := i*d + replaceRows[j]
			gm[off] = r.GenMatrix[k]
			off++
		}
	}

	vects := make([][]byte, p+rn)
	for i := range data {
		vects[i] = data[i]
	}

	for i := range parity {
		vects[rn+i] = parity[i]
	}

	updateRS := &RS{DataNum: rn, ParityNum: p,
		GenMatrix: gm, cpuFeat: r.cpuFeat, gmu: r.gmu}
	updateRS.encode(vects, true)
	return nil
}

var (
	ErrTooManyReplace  = errors.New("too many data for replacing")
	ErrMismatchReplace = errors.New("number of replaceRows and data mismatch")
)

func (r *RS) checkReplace(data [][]byte, replaceRows []int, parity [][]byte) (err error) {
	if len(data) > r.DataNum {
		return ErrTooManyReplace
	}

	if len(replaceRows) != len(data) {
		return ErrMismatchReplace
	}

	if len(parity) != r.ParityNum {
		return ErrMismatchParityNum
	}

	size := len(data[0])
	if size == 0 {
		return ErrZeroVectSize
	}
	for i := range data {
		if size != len(data[i]) {
			return ErrMismatchVectSize
		}
	}
	for i := range parity {
		if size != len(parity[i]) {
			return ErrMismatchVectSize
		}
	}

	for _, rr := range replaceRows {
		if rr >= r.DataNum || rr < 0 {
			return ErrIllegalVectIndex
		}
	}
	return
}
