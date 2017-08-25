/*
	Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
*/

package reedsolomon

import (
	"errors"
	"fmt"
)

type encBase struct {
	data         int
	parity       int
	encodeMatrix matrix
	genMatrix    matrix
}

type EncodeReconster interface {
	Encode(vects [][]byte) error
	Reconstruct(vects [][]byte) error
	ReconstructData(vects [][]byte) error
}

func checkNumVects(data, parity int) error {
	if (data <= 0) || (parity <= 0) {
		return errors.New("rs.New: data or parity <= 0")
	}
	if data+parity >= 255 { //usually, data <= 20 & parity <= 6
		return errors.New(fmt.Sprintf("rs.New: data+parity >= 255"))
	}
	return nil
}

// New: vandermonde matrix as Encoding matrix
func New(data, parity int) (enc EncodeReconster, err error) {
	err = checkNumVects(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	return newRS(data, parity, e), nil
}

// NewCauchy: cauchy matrix as Generator Matrix
func NewCauchy(data, parity int) (enc EncodeReconster, err error) {
	err = checkNumVects(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newRS(data, parity, e), nil
}

func checkEncVects(in, out int, vects [][]byte) error {
	v := len(vects)
	if in+out != v {
		return errors.New(fmt.Sprintf("rs.Enc: vects not match, in: %d out: %d vects: %d", in, out, v))
	}

	s := len(vects[0])
	if s == 0 {
		return errors.New("rs.Enc: vects size = 0")
	}
	for i := 1; i < v; i++ {
		if len(vects[i]) != s {
			return errors.New("rs.Enc: vects size not match")
		}
	}
	return nil
}

// Encode : multiply generator-matrix with data
func (e *encBase) Encode(vects [][]byte) (err error) {
	in := e.data
	out := e.parity
	err = checkEncVects(in, out, vects)
	if err != nil {
		return
	}
	gen := e.genMatrix
	for i := 0; i < in; i++ {
		for j := 0; j < out; j++ {
			if i != 0 {
				vectMulPlus(gen[j*in+i], vects[:in][i], vects[in:][j])
			} else {
				vectMul(gen[j*in], vects[:in][0], vects[in:][j])
			}
		}
	}
	return
}

func vectMul(c byte, inV, outV []byte) {
	t := mulTbl[c]
	for i := 0; i < len(inV); i++ {
		outV[i] = t[inV[i]]
	}
}

func vectMulPlus(c byte, inV, outV []byte) {
	t := mulTbl[c]
	for i := 0; i < len(inV); i++ {
		outV[i] ^= t[inV[i]]
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

func makeLostInfo(data int, vects [][]byte) (has, dataLost, parityLost []int) {
	cnt := 0
	for i, v := range vects {
		if v == nil {
			if i < data {
				dataLost = append(dataLost, i)
			} else {
				parityLost = append(parityLost, i)
			}
		} else {
			if cnt < data {
				has = append(has, i)
				cnt++
			}
		}
	}
	return has, dataLost, parityLost
}

func isMatchVectSize(size int, list []int, vects [][]byte) bool {
	for i := 1; i < len(list); i++ {
		if size != len(vects[list[i]]) {
			return false
		}
	}
	return true
}

type reconstInfo struct {
	dataOK   bool
	parityOK bool
	vectSize int
	has      []int
	data     []int
	parity   []int
}

func makeReconstInfo(data int, vects [][]byte, dataOnly bool) (info reconstInfo, err error) {
	has, dataLost, parityLost := makeLostInfo(data, vects)
	if len(dataLost) == 0 {
		info.dataOK = true
	}
	if len(parityLost) == 0 {
		info.parityOK = true
	} else {
		if dataOnly {
			info.parityOK = true
		}
	}
	if info.dataOK && info.parityOK {
		return
	}
	if len(has) != data {
		err = errors.New(fmt.Sprintf("rs.Reconst: not enough vects, have: %d, data: %d", len(has), data))
		return
	}
	size := len(vects[has[0]])
	if size == 0 {
		err = errors.New("rs.Reconst: vects size = 0")
		return
	}
	if !isMatchVectSize(size, has, vects) {
		err = errors.New("rs.Reconst: vects size not match")
		return
	}
	info.has = has
	info.data = dataLost
	info.parity = parityLost
	info.vectSize = size
	return
}

func makeInverse(em matrix, has []int, data int) (matrix, error) {
	m := newMatrix(data, data)
	for i, p := range has {
		m[i] = em[p]
	}
	im, err := m.invert(data)
	if err != nil {
		return nil, err
	}
	return im, nil
}

func (e *encBase) reconst(vects [][]byte, dataOnly bool) (err error) {
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
		im, err2 := makeInverse(em, info.has, data)
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
		vtmp[data+i] = vects[p]
	}
	etmp := &encBase{data: e.data, parity: out, genMatrix: gen}
	etmp.Encode(vtmp)
}
