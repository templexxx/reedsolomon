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
	"sort"
	"sync"

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

	cacheEnabled  bool      // Cache inverse matrix or not.
	inverseMatrix *sync.Map // Inverse matrix's cache.
}

// EnableAVX512 may slow down CPU Clock (maybe not).
// TODO need more research:
// https://lemire.me/blog/2018/04/19/by-how-much-does-avx-512-slow-down-your-cpu-a-first-experiment/
//
// You can modify it before new RS.
var EnableAVX512 = true

var ErrIllegalVects = errors.New("illegal data/parity number: <= 0 or data+parity > 256")

// New create an RS with specific data & parity numbers.
func New(dataNum, parityNum int) (r *RS, err error) {

	d, p := dataNum, parityNum
	if d <= 0 || p <= 0 || d+p > 256 {
		return nil, ErrIllegalVects
	}

	e := makeEncodeMatrix(d, p)
	g := e[d*d:]
	r = &RS{DataNum: d, ParityNum: p,
		encMatrix: e, GenMatrix: g}

	// At most 35960 inverse matrices (when data=28, parity=4).
	// There is no need to keep too many matrices in cache,
	// too many parity num will slow down the encode performance,
	// so the cache won't effect much.
	// Warn:
	// You can modify it,
	// but be careful that it may cause memory explode
	// (you can use mathtool/combi.go to calculate how many inverse matrices you will have),
	// and data+parity must < 64 (tips: see the codes about cache inverse matrix).
	if r.DataNum < 29 && r.ParityNum < 5 {
		r.cacheEnabled = true
		r.inverseMatrix = new(sync.Map)
	}

	r.cpuFeat = getCPUFeature()
	return
}

// CPU Features.
const (
	avx512 = iota
	avx2
	base // No supported features, using basic way.
)

func getCPUFeature() int {
	if hasAVX512() && EnableAVX512 {
		return avx512
	} else if cpu.X86.HasAVX2 {
		return avx2
	} else {
		return base
	}
}

func hasAVX512() (ok bool) {

	return cpu.X86.HasAVX512VL &&
		cpu.X86.HasAVX512BW &&
		cpu.X86.HasAVX512F &&
		cpu.X86.HasAVX512DQ
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
	ErrMismatchVects    = errors.New("too few/many vects given")
	ErrZeroVectSize     = errors.New("vect size is 0")
	ErrMismatchVectSize = errors.New("vects size mismatched")
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

// encode encodes data piece by piece.
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
	d, p, g, f := r.DataNum, r.ParityNum, r.GenMatrix, r.cpuFeat
	if do >= 16 {
		end2 := start + do
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly {
					mulVectXOR(g[j*d+i], dv[i][start:end2], pv[j][start:end2], f)
				} else {
					mulVect(g[j*d+i], dv[0][start:end2], pv[j][start:end2], f)
				}
			}
		}
	}

	if undone > do { // 0 < undone-do < 16
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly {
					mulVectXORBase(g[j*d+i], dv[i][start:end], pv[j][start:end])
				} else {
					mulVectBase(g[j*d], dv[0][start:end], pv[j][start:end])
				}
			}
		}
	}
}

// Reconst reconstructs missing vectors,
// vects: All vectors, len(vects) = dataNum + parityNum.
// dpHas: Survived data & parity index, need dataNum indexes at least.
// needReconst: Vectors indexes which need to be reconstructed.
//
// e.g:
// in 3+2, the whole index: [0,1,2,3,4],
// if vects[0,4] are lost & they need to be reconstructed
// (Maybe you only need vects[0], so the needReconst should be [0], but not [0,4]).
// the "dpHas" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct data,
// results will be written into vects[0]&vects[4] directly.
func (r *RS) Reconst(vects [][]byte, dpHas, needReconst []int) (err error) {
	err = r.checkReconst(dpHas, needReconst)
	if err != nil {
		if err == ErrNoNeedReconst {
			return nil
		}
		return
	}

	// Make sure we have right data vectors for reconstructing parity.
	for i := 0; i < r.DataNum; i++ {
		if !isIn(i, dpHas) && !isIn(i, needReconst) {
			needReconst = append(needReconst, i)
		}
	}
	dataNeed, parityNeed := SplitNeedReconst(r.DataNum, needReconst)
	if len(dataNeed) != 0 {
		err = r.reconstData(vects, dpHas, dataNeed)
		if err != nil {
			return
		}
	}
	if len(parityNeed) != 0 {
		err = r.reconstParity(vects, parityNeed)
		if err != nil {
			return
		}
	}
	return
}

var (
	ErrNoNeedReconst   = errors.New("no need reconst")
	ErrTooManyLost     = errors.New("too many lost")
	ErrHasLostConflict = errors.New("dpHas&lost are conflicting")
)

func (r *RS) checkReconst(dpHas, needReconst []int) (err error) {
	d, p := r.DataNum, r.ParityNum
	if len(needReconst) == 0 {
		return ErrNoNeedReconst
	}
	if len(needReconst) > p || len(dpHas) < d {
		return ErrTooManyLost
	}

	for _, i := range needReconst {
		if i < 0 || i >= d+p {
			return ErrIllegalVectIndex
		}
	}
	for _, i := range dpHas {
		if i < 0 || i >= d+p {
			return ErrIllegalVectIndex
		}
	}

	for _, i := range dpHas {
		if isIn(i, needReconst) {
			err = ErrHasLostConflict
			return
		}
	}

	return
}

// SplitNeedReconst splits data lost & parity lost.
func SplitNeedReconst(dataCnt int, needReconst []int) (dataNeed, parityNeed []int) {
	sort.Ints(needReconst)
	for i, l := range needReconst {
		if l >= dataCnt {
			return needReconst[:i], needReconst[i:]
		}
	}
	return needReconst, nil
}

func isIn(e int, s []int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func (r *RS) reconstData(vects [][]byte, dpHas, dNeedReconst []int) (err error) {

	d := r.DataNum
	sort.Ints(dpHas)
	dpHas = dpHas[:d] // Only need dataNum vectors for reconstruction.
	lostCnt := len(dNeedReconst)
	vTmp := make([][]byte, d+lostCnt)

	for i, row := range dpHas {
		vTmp[i] = vects[row]
	}
	for i, row := range dNeedReconst {
		vTmp[i+d] = vects[row]
	}

	rm, err := r.getReconstMatrix(dpHas, dNeedReconst)
	if err != nil {
		return
	}
	rTmp := &RS{DataNum: d, ParityNum: lostCnt, GenMatrix: rm, cpuFeat: r.cpuFeat}
	return rTmp.Encode(vTmp)
}

func (r *RS) getReconstMatrix(dpHas, dLost []int) (rm []byte, err error) {

	if !r.cacheEnabled {
		em, err2 := r.encMatrix.makeEncMatrixForReconst(dpHas)
		if err2 != nil {
			return nil, err2
		}
		return em.makeReconstMatrix(dpHas, dLost)
	}
	return r.getReconstMatrixFromCache(dpHas, dLost)
}

func (r *RS) getReconstMatrixFromCache(dpHas, dLost []int) (rm matrix, err error) {
	var bitmap uint64 // indicate dpHas
	for _, i := range dpHas {
		bitmap += 1 << uint8(i)
	}

	emRaw, ok := r.inverseMatrix.Load(bitmap)
	if ok {
		em := emRaw.(matrix)
		return em.makeReconstMatrix(dpHas, dLost)
	}

	em, err := r.encMatrix.makeEncMatrixForReconst(dpHas)
	if err != nil {
		return
	}
	r.inverseMatrix.Store(bitmap, em)
	return em.makeReconstMatrix(dpHas, dLost)
}

func (r *RS) reconstParity(vects [][]byte, pLost []int) (err error) {
	d := r.DataNum
	lostN := len(pLost)

	g := make([]byte, lostN*d)
	for i, l := range pLost {
		copy(g[i*d:i*d+d], r.encMatrix[l*d:l*d+d])
	}

	vTmp := make([][]byte, d+lostN)
	for i := 0; i < d; i++ {
		vTmp[i] = vects[i]
	}
	for i, p := range pLost {
		vTmp[i+d] = vects[p]
	}

	rTmp := &RS{DataNum: d, ParityNum: lostN, GenMatrix: g, cpuFeat: r.cpuFeat}
	return rTmp.Encode(vTmp)
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
	rs := &RS{DataNum: 1, ParityNum: r.ParityNum, GenMatrix: gm, cpuFeat: r.cpuFeat}
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
// In practice,
// If len(replaceRows) > dataNum-parityNum, it's better to use Encode,
// because Replace need to read len(replaceRows) + parityNum vectors,
// if replaceRows are too many, the cost maybe larger than Encode
// (Encode only need read dataNum).
// Think about an EC compute node, and dataNum+parityNum data nodes model.
//
// It's used in two situations:
// 1. We didn't have enough data for filling in a stripe, but still did ec encode,
// we need replace several zero vectors with new vectors which have data after we get enough data finally.
// 2. After compact, we may have several useless vectors in a stripe,
// we need replaces these useless vectors with zero vectors for free space.
//
// Warn:
// data's index & replaceRows must has the same sort.
func (r *RS) Replace(data [][]byte, replaceRows []int, parity [][]byte) (err error) {

	err = r.checkReplace(data, replaceRows, parity)
	if err != nil {
		return
	}

	d, p := r.DataNum, r.ParityNum
	rn := len(replaceRows)

	// Make generator matrix for replacing.
	//
	// Values in replaceRows are row indexes of data,
	// and also the column indexes of generator matrix
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
		GenMatrix: gm, cpuFeat: r.cpuFeat}
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
