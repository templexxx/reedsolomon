/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)
*/

package reedsolomon

import (
	"errors"
	"fmt"
)

// EncodeReconster implements for Reed-Solomon Encoding/Reconstructing
type EncodeReconster interface {
	Encode(vects [][]byte) error
	Reconstruct(vects [][]byte) error
	ReconstructData(vects [][]byte) error
}

func checkCfg(data, parity int) error {
	if (data <= 0) || (parity <= 0) {
		return errors.New("rs.New: data or parity <= 0")
	}
	if data+parity > 256 {
		return errors.New("rs.New: data+parity > 256")
	}
	return nil
}

// New create an EncodeReconster (vandermonde matrix as Encoding matrix)
func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	return newRS(data, parity, e), nil
}

// NewCauchy create an EncodeReconster (cauchy matrix as Generator Matrix)
func NewCauchy(data, parity int) (enc EncodeReconster, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newRS(data, parity, e), nil
}

func checkER(d, p int, vs [][]byte, okNil bool) (size int, err error) {
	if d+p != len(vs) {
		err = errors.New("rs.checkER: vects not match rs args")
		return
	}
	for _, v := range vs {
		if len(v) != 0 {
			size = len(v)
		}
	}
	if size == 0 {
		err = errors.New("rs.checkER: vects size = 0")
		return
	}
	for _, v := range vs {
		if len(v) != size {
			if v == nil && okNil {
				continue
			}
			err = errors.New("rs.checkER: vects size mismatch")
			return
		}
	}
	return
}

func checkEnc(d, p int, vs [][]byte) (size int, err error) {
	return checkER(d, p, vs, false)
}

type encBase struct {
	data         int
	parity       int
	total        int // data+parity
	encodeMatrix matrix
	genMatrix    matrix
}

// Encode : multiply generator-matrix with data
func (e *encBase) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	_, err = checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	g := e.genMatrix
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			if i != 0 {
				mulVectAdd(g[j*d+i], dv[i], pv[j])
			} else {
				mulVect(g[j*d], dv[0], pv[j])
			}
		}
	}
	return
}

func mulVect(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] = t[a[i]]
	}
}

func mulVectAdd(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] ^= t[a[i]]
	}
}

// Reconstruct : reconstruct lost data & parity
// set shard nil if lost
func (e *encBase) Reconstruct(vects [][]byte) (err error) {
	return e.reconst(vects, false)
}

// ReconstructData  : reconstruct lost data
func (e *encBase) ReconstructData(vects [][]byte) (err error) {
	return e.reconst(vects, true)
}

func checkReconst(d, p int, vs [][]byte) (size int, err error) {
	return checkER(d, p, vs, true)
}

func (e *encBase) reconst(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	size, err := checkReconst(d, p, vects)
	if err != nil {
		return
	}
	hasCnt := 0
	total := e.total
	dBuf := make([][]byte, d) // dBuf: reorganize data
	dBufCnt := 0
	dBufPos := make([]int, d)
	dLost := make([]int, 0)
	pLost := make([]int, 0)
	for i := 0; i < total; i++ {
		if vects[i] != nil {
			hasCnt++
			if dBufCnt < d {
				dBufPos[dBufCnt] = i
				dBuf[dBufCnt] = vects[i]
				dBufCnt++
			}
		} else {
			if i < d {
				dLost = append(dLost, i)
			} else {
				pLost = append(pLost, i)
			}
		}
	}
	if hasCnt == total {
		return nil
	}
	if hasCnt < d {
		return errors.New("rs.Reconst: not enough vects")
	}
	em := e.encodeMatrix
	if len(dLost) != 0 {
		im, err2 := makeInverse(em, dBufPos, d)
		if err2 != nil {
			return err2
		}
		rgD := make([]byte, len(dLost)*d)
		for i, p := range dLost {
			copy(rgD[i*d:i*d+d], im[p*d:p*d+d])
		}
		e.reconstData(vects, size, dLost, rgD)
	}
	if len(pLost) != 0 && !dataOnly {
		rgP := make([]byte, len(pLost)*d)
		for i, p := range pLost {
			copy(rgP[i*d:i*d+d], em[p*d:p*d+d])
		}
		e.reconstParity(vects, size, pLost, rgP)
	}
	return nil
}

type reconstInfo struct {
	okData   bool
	okParity bool
	vectSize int
	has      []int
	data     []int
	parity   []int
}

func makeReconstInfo(data, parity int, vects [][]byte, dataOnly bool) (info *reconstInfo, err error) {
	_, err = checkReconst(data, parity, vects)
	if err != nil {
		return
	}
	cnt := 0
	for i, v := range vects {
		if v == nil {
			if i < data {
				info.data = append(info.data, i)
			} else {
				info.parity = append(info.parity, i)
			}
		} else {
			if cnt < data {
				if cnt == 0 {
					s := len(vects[i])
					if s != 0 {
						info.vectSize = len(vects[i])
					} else {
						err = errors.New("rs.Reconst: vects size = 0")
						return
					}
				} else {
					if info.vectSize != len(vects[i]) {
						err = errors.New("rs.Reconst: vects size not match")
						return
					}
				}
				info.has = append(info.has, i)
				cnt++
			}
		}
	}
	if cnt != data {
		err = fmt.Errorf("rs.Reconst: not enough vects, has: %d, data: %d", cnt, data)
		return
	}

	if len(info.data) == 0 {
		info.okData = true
	}
	if len(info.parity) == 0 {
		info.okParity = true
	} else {
		if dataOnly {
			info.okParity = true
		}
	}
	if info.okData && info.okParity {
		return
	}
	return
}

func makeInverse(em matrix, has []int, data int) (matrix, error) {
	m := newMatrix(data, data)
	for i, p := range has {
		copy(m[i*data:i*data+data], em[p*data:p*data+data])
	}
	im, err := m.invert(data)
	if err != nil {
		return nil, err
	}
	return im, nil
}

func (e *encBase) reconstData(vects [][]byte, size int, lost []int, gen matrix) {
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
		vects[p] = make([]byte, size)
		vtmp[cnt] = vects[p]
		cnt++
	}

	etmp := &encBase{data: data, parity: out, genMatrix: gen}
	etmp.Encode(vtmp)
}

func (e *encBase) reconstParity(vects [][]byte, size int, lost []int, gen matrix) {
	data := e.data
	out := len(lost)
	vtmp := make([][]byte, data+out)
	for i := 0; i < data; i++ {
		vtmp[i] = vects[i]
	}
	for i, p := range lost {
		vects[p] = make([]byte, size)
		vtmp[data+i] = vects[p]
	}
	etmp := &encBase{data: e.data, parity: out, genMatrix: gen}
	etmp.Encode(vtmp)
}
