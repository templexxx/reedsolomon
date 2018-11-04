/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2)
	Platform: X86-64 (amd64)
*/

package reedsolomon

import (
	"errors"
	"sort"
	"sync"

	"github.com/templexxx/cpu"
	xor "github.com/templexxx/xorsimd"
)

// RS Reed-Solomon Codes receiver
type RS struct {
	DataCnt       int
	ParityCnt     int
	cpu           int
	encodeMatrix  matrix // encoding_matrix
	genMatrix     matrix // generator_matrix
	cacheEnabled  bool   // cache inverse_matrix
	inverseMatrix *sync.Map
}

// CPU Features
const (
	base = iota
	avx2
	avx512
)

var EnableAVX512 = false

// New create an RS
func New(dataCnt, parityCnt int) (r *RS, err error) {

	err = checkCfg(dataCnt, parityCnt)
	if err != nil {
		return
	}

	e := genEncMatrix(dataCnt, parityCnt)
	g := e[dataCnt*dataCnt:]
	r = &RS{DataCnt: dataCnt, ParityCnt: parityCnt,
		encodeMatrix: e, genMatrix: g, inverseMatrix: new(sync.Map)}
	r.enableCache()

	r.cpu = getCPUFeature()

	return
}

func getCPUFeature() int {
	if useAVX512() {
		return avx512
	} else if cpu.X86.HasAVX2 {
		return avx2
	} else {
		return base
	}
}

func useAVX512() (ok bool) {
	if !hasAVX512() {
		return
	}
	if !useAVX512() {
		return
	}
	return true
}

func hasAVX512() (ok bool) {
	if !cpu.X86.HasAVX512VL {
		return
	}
	if !cpu.X86.HasAVX512BW {
		return
	}
	if !cpu.X86.HasAVX512F {
		return
	}
	if !cpu.X86.HasAVX512DQ {
		return
	}
	return true
}

var ErrMinVects = errors.New("data or parity <= 0")
var ErrMaxVects = errors.New("data+parity >= 256")

func checkCfg(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return ErrMinVects
	}
	if d+p >= 256 {
		return ErrMaxVects
	}
	return nil
}

// At most 35960 inverse_matrix (when data=28, parity=4)
func (r *RS) enableCache() {
	if r.DataCnt < 29 && r.ParityCnt < 5 { // data+parity can't be bigger than 64 (tips: see the codes about make inverse matrix)
		r.cacheEnabled = true
	} else {
		r.cacheEnabled = false
	}
}

// Encode outputs parity into vects
func (r *RS) Encode(vects [][]byte) (err error) {
	err = r.checkEncode(vects)
	if err != nil {
		return
	}
	r.encode(vects, false)
	return
}

var ErrVectCnt = errors.New("vects != data + parity")
var ErrVectSizeZero = errors.New("vect size cannot equal 0")
var ErrVectSizeMismatch = errors.New("vects size mismatch")

func (r *RS) checkEncode(vects [][]byte) (err error) {
	rows := len(vects)
	if r.DataCnt+r.ParityCnt != rows {
		err = ErrVectCnt
		return
	}
	size := len(vects[0])
	if size == 0 {
		err = ErrVectSizeZero
		return
	}
	for i := 1; i < rows; i++ {
		if len(vects[i]) != size {
			err = ErrVectSizeMismatch
			return
		}
	}
	return
}

func (r *RS) encode(vects [][]byte, updateOnly bool) {
	dv, pv := vects[:r.DataCnt], vects[r.DataCnt:]
	size := len(vects[0])
	splitSize := getSplitSize(size)
	start := 0
	for start < size {
		end := start + splitSize
		if end <= size {
			r.encodePart(start, end, dv, pv, updateOnly)
			start = end
		} else {
			r.encodePart(start, size, dv, pv, updateOnly) // calculate left_data (< splitSize)
			start = size
		}
	}
}

const L1DataCacheSize = 32 * 1024

// split vects for cache-friendly (size must be divisible by 16)
func getSplitSize(n int) int {
	if n < 16 {
		return 16
	}
	if n < L1DataCacheSize/2 {
		return (n >> 4) << 4
	}
	return L1DataCacheSize / 2
}

// encode data[i][start:end]
func (r *RS) encodePart(start, end int, dataVects, parityVects [][]byte, updateOnly bool) {
	undoneSize := end - start
	splitSize := (undoneSize >> 4) << 4 // splitSize could be 0(when undoneSize < 16)
	d, p, g, cF := r.DataCnt, r.ParityCnt, r.genMatrix, r.cpu
	if splitSize >= 16 {
		end2 := start + splitSize
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly == true {
					coeffMulVectUpdate(g[j*d+i], dataVects[i][start:end2], parityVects[j][start:end2], cF)
				} else {
					coeffMulVect(g[j*d+i], dataVects[0][start:end2], parityVects[j][start:end2], cF)
				}
			}
		}
	}
	if undoneSize > splitSize { // 0 < undoneSize-splitSize < 16
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				if i != 0 || updateOnly == true {
					coeffMulVectUpdateBase(g[j*d+i], dataVects[i][start:end], parityVects[j][start:end])
				} else {
					coeffMulVectBase(g[j*d], dataVects[0][start:end], parityVects[j][start:end])
				}
			}
		}
	}
}

func coeffMulVectBase(c byte, d, p []byte) {
	t := mulTbl[c]
	for i := 0; i < len(d); i++ {
		p[i] = t[d[i]]
	}
}

func coeffMulVectUpdateBase(c byte, d, p []byte) {
	t := mulTbl[c]
	for i := 0; i < len(d); i++ {
		p[i] ^= t[d[i]]
	}
}

var ErrMismatchParityCnt = errors.New("mismatch parity cnt")
var ErrIllegalUpdateSize = errors.New("illegal update size")
var ErrIllegalUpdateRow = errors.New("illegal update row")

// UpdateParity update parity_data when one data_vect changes
func (r *RS) Update(oldData []byte, newData []byte, updateRow int, parity [][]byte) (err error) {
	// check args
	if len(parity) != r.ParityCnt {
		err = ErrMismatchParityCnt
		return
	}
	size := len(newData)
	if size <= 0 {
		err = ErrIllegalUpdateSize
		return
	}
	if size != len(oldData) {
		err = ErrIllegalUpdateSize
		return
	}
	for i := range parity {
		if len(parity[i]) != size {
			err = ErrIllegalUpdateSize
			return
		}
	}
	if updateRow >= r.DataCnt {
		err = ErrIllegalUpdateRow
		return
	}

	// step1: buf (old_data xor new_data)
	buf := make([]byte, size)
	xor.Encode(buf, [][]byte{oldData, newData})
	// step2: reEnc parity
	updateVects := make([][]byte, 1+r.ParityCnt)
	updateVects[0] = buf
	updateGenMatrix := make([]byte, r.ParityCnt)
	// make update_generator_matrix & update_vects
	for i := 0; i < r.ParityCnt; i++ {
		col := updateRow
		off := i*r.DataCnt + col
		c := r.genMatrix[off]
		updateGenMatrix[i] = c
		updateVects[i+1] = parity[i]
	}
	updateRS := &RS{DataCnt: 1, ParityCnt: r.ParityCnt, genMatrix: updateGenMatrix, cpu: r.cpu}
	updateRS.encode(updateVects, true)
	return nil
}

// Reconst repair missing vects, len(dpHas) == dataCnt
// e.g:
// in 3+2, the whole index: [0,1,2,3,4]
// if vects[0,4] are lost & they need to be reconst
// the "dpHas" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct data
// results will be put into vects[0]&vects[4]
// dataOnly: only reconst data or not
func (r *RS) Reconst(vects [][]byte, dpHas, needReconst []int) (err error) {
	err = r.checkReconst(dpHas, needReconst)
	if err != nil {
		if err == ErrNoNeedReconst {
			return nil
		}
		return
	}
	sort.Ints(dpHas)
	// make sure we have right data vects for reconst parity
	for i := 0; i < r.DataCnt; i++ {
		if !isIn(i, dpHas) && !isIn(i, needReconst) {
			needReconst = append(needReconst, i)
		}
	}
	dNeedReconst, pNeedReconst := SplitNeedReconst(r.DataCnt, needReconst)
	if len(dNeedReconst) != 0 {
		err = r.reconstData(vects, dpHas, dNeedReconst)
		if err != nil {
			return
		}
	}
	if len(pNeedReconst) != 0 {
		err = r.reconstParity(vects, pNeedReconst)
		if err != nil {
			return
		}
	}
	return
}

func (r *RS) reconstData(vects [][]byte, dpHas, dNeedReconst []int) (err error) {
	d := r.DataCnt
	lostCnt := len(dNeedReconst)
	vTmp := make([][]byte, d+lostCnt)
	for i, row := range dpHas {
		vTmp[i] = vects[row]
	}
	for i, row := range dNeedReconst {
		vTmp[i+d] = vects[row]
	}
	g, err := r.getGenMatrix(dpHas, dNeedReconst)
	if err != nil {
		return
	}
	rTmp := &RS{DataCnt: d, ParityCnt: lostCnt, genMatrix: g, cpu: r.cpu}
	err = rTmp.Encode(vTmp)
	if err != nil {
		return
	}
	return
}

func (r *RS) reconstParity(vects [][]byte, lost []int) (err error) {
	d := r.DataCnt
	lostCnt := len(lost)
	vTmp := make([][]byte, d+lostCnt)
	g := make([]byte, lostCnt*d)
	for i, l := range lost {
		copy(g[i*d:i*d+d], r.encodeMatrix[l*d:l*d+d])
	}
	for i := 0; i < d; i++ {
		vTmp[i] = vects[i]
	}
	for i, p := range lost {
		vTmp[i+d] = vects[p]
	}
	rTmp := &RS{DataCnt: d, ParityCnt: lostCnt, genMatrix: g, cpu: r.cpu}
	err = rTmp.Encode(vTmp)
	if err != nil {
		return
	}
	return
}

var ErrNoNeedReconst = errors.New("no need reconst")
var ErrTooManyLost = errors.New("too many lost vects")
var ErrDPHasMismatchDataCnt = errors.New("len(dpHas) must = dataCnt")
var ErrIllegalIndex = errors.New("illegal index")
var ErrHasLostConflict = errors.New("dpHas&lost are conflicting")

func (r *RS) checkReconst(dpHas, needReconst []int) (err error) {
	d, p := r.DataCnt, r.ParityCnt
	if len(needReconst) == 0 {
		err = ErrNoNeedReconst
		return
	}
	if len(needReconst) > p {
		err = ErrTooManyLost
		return
	}
	if len(dpHas) != d {
		err = ErrDPHasMismatchDataCnt
		return
	}
	for _, i := range dpHas {
		if i < 0 || i >= d+p {
			err = ErrIllegalIndex
			return
		}
		if isIn(i, needReconst) {
			err = ErrHasLostConflict
			return
		}
	}
	for _, i := range needReconst {
		if i < 0 || i >= d+p {
			err = ErrIllegalIndex
			return
		}
	}
	return
}

// SplitNeedReconst split data_lost & parity_lost
func SplitNeedReconst(dataCnt int, needReconst []int) (dNeedReconst, pNeedReconst []int) {
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

func (r *RS) getGenMatrix(dpHas, dLost []int) (gm []byte, err error) {
	d := r.DataCnt
	lostCnt := len(dLost)
	if !r.cacheEnabled { // no cache
		eNew, err2 := r.makeEncodeMatrix(dpHas)
		if err2 != nil {
			return nil, err2
		}
		gm = make([]byte, lostCnt*d)
		for i, l := range dLost {
			copy(gm[i*d:i*d+d], eNew[l*d:l*d+d])
		}
		return
	}
	gm, err = r.getGenMatrixFromCache(dpHas, dLost)
	if err != nil {
		return
	}
	return
}

// according to encoding_matrix & dpHas make a new encoding_matrix
func (r *RS) makeEncodeMatrix(dpHas []int) (em []byte, err error) {
	d := r.DataCnt
	m := make([]byte, d*d)
	for i, l := range dpHas {
		copy(m[i*d:i*d+d], r.encodeMatrix[l*d:l*d+d])
	}
	em, err = matrix(m).invert(d)
	if err != nil {
		return
	}
	return
}

func (r *RS) getGenMatrixFromCache(dpHas, dLost []int) (gm []byte, err error) {
	var bitmap uint64 // indicate dpHas
	for _, i := range dpHas {
		bitmap += 1 << uint8(i)
	}
	d, lostCnt := r.DataCnt, len(dLost)
	v, ok := r.inverseMatrix.Load(bitmap)
	if ok {
		im := v.([]byte)
		gm = make([]byte, lostCnt*d)
		for i, l := range dLost {
			copy(gm[i*d:i*d+d], im[l*d:l*d+d])
		}
		return
	}
	em, err := r.makeEncodeMatrix(dpHas)
	if err != nil {
		return
	}
	r.inverseMatrix.Store(bitmap, em)
	gm = make([]byte, lostCnt*d)
	for i, l := range dLost {
		copy(gm[i*d:i*d+d], em[l*d:l*d+d])
	}
	return
}
