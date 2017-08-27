/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)
*/

package reedsolomon

import (
	"errors"
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

func (e *encBase) reconst(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	size, err := checkReconst(d, p, vects)
	if err != nil {
		return
	}
	hasCnt := 0
	total := e.total
	k := 0
	dPos := make([]int, d) // for invert matrix
	dLost := make([]int, 0)
	pLost := make([]int, 0)
	for i := 0; i < total; i++ {
		if vects[i] != nil {
			hasCnt++
			if k < d {
				dPos[k] = i
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
	if hasCnt == total {
		return nil
	}
	if hasCnt < d {
		return errors.New("rs.Reconst: not enough vects")
	}

	em := e.encodeMatrix
	dLCnt := len(dLost)
	if dLCnt != 0 {
		im, err2 := makeInverse(em, dPos, d)
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

		etmp := &encBase{data: d, parity: dLCnt, genMatrix: g}
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
		etmp := &encBase{data: d, parity: pLCnt, genMatrix: g}
		etmp.Encode(vtmp)
	}
	return nil
}
