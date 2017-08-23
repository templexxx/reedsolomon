package reedsolomon

import (
	"errors"
)

type matrix []byte

func newMatrix(rows, cols int) matrix {
	m := make([]byte, rows*cols)
	return m
}

// generate a EncodeMatrix with a vandermonde matrix
func genEncMatrixVand(data, parity int) (matrix, error) {
	rows := data + parity
	vm := genVandMatrix(rows, data)
	top := newMatrix(data, data)
	for i := range top {
		copy(top[i], vm[i])
	}
	topInv, err := top.invert(data)
	if err != nil {
		return nil, err
	}
	return vm.mul(topInv), nil
}

// generate a EncodeMatrix : identity-matrix(upper) cauchy-matrix(lower)
func genEncMatrixCauchy(data, parity int) matrix {
	rows := data + parity
	cols := data
	m := newMatrix(rows, cols)
	// identity matrix
	for j := 0; j < cols; j++ {
		m[j*data+j] = byte(1)
	}
	// cauchy matrix
	p := data * data
	for i := cols; i < rows; i++ {
		for j := 0; j < cols; j++ {
			d := i ^ j
			a := inverseTbl[d]
			m[p] = byte(a)
			p++
		}
	}
	return m
}



func (m matrix) invert(n int) (matrix, error) {
	raw := newMatrix(n, 2*n)
	for i := 0; i < n; i++ {
		t := i * n
		copy(raw[2*t:2*t+n], m[t:t+n])
		raw[2*t+i+n] = byte(1)
	}
	err := raw.gaussJordan(n, 2*n)
	if err != nil {
		return nil, err
	}
	return raw.subMatrix(n), nil
}

func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}

var ErrSingular = errors.New("reedsolomon: matrix is singular")

func (m matrix) gaussJordan(rows, columns int) error {
	for r := 0; r < rows; r++ {
		// If the element on the diagonal is 0, find a row below
		// that has a non-zero and swap them.
		if m[2*r*rows+r] == 0 {
			for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
				if m[2*rowBelow*rows+r] != 0 {
					m.swap(r, rowBelow, 2*rows)
					break
				}
			}
		}
		// After swap, if we find all elements in this column is 0, it means the Matrix's det is 0
		if m[2*r*rows+r] == 0 {
			return ErrSingular
		}
		// Scale to 1.
		if m[2*r*rows+r] != 1 {
			d := m[2*r*rows+r]
			scale := inverseTbl[d]
			// every element(this column) * m[r][r]'s inverse
			for c := 0; c < columns; c++ {
				m[2*r*rows+c] = gfMul(m[2*r*rows+c], scale)
			}
		}
		//Make everything below the 1 be a 0 by subtracting a multiple of it
		for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
			if m[2*rowBelow*rows+r] != 0 {
				// scale * m[r][r] = scale, scale + scale = 0
				// makes m[r][r+1] = 0 , then calc left elements
				scale := m[2*rowBelow*rows+r]
				for c := 0; c < columns; c++ {
					m[2*rowBelow*rows+c] ^= gfMul(scale, m[2*r*rows+c])
				}
			}
		}
	}
	// Now clear the part above the main diagonal.
	// same logic with clean upper
	for d := 0; d < rows; d++ {
		for rowAbove := 0; rowAbove < d; rowAbove++ {
			if m[2*rowAbove*rows+d] != 0 {
				scale := m[2*rowAbove*rows+d]
				for c := 0; c < columns; c++ {
					m[2*rowAbove*rows+c] ^= gfMul(scale, m[2*d*rows+c])
				}
			}
		}
	}
	return nil
}

func (m matrix) subMatrix(size int) matrix {
	ret := newMatrix(size, size)
	for i := 0; i < size; i++ {
		copy(ret[i*size:i*size+size], m[2*i*size+size:2*i*size+2*size])
	}
	return ret
}

func genVandMatrix(rows, cols int) matrix {
	raw := newMatrix(rows, cols)
	for r, row := range raw {
		for c := range row {
			raw[r][c] = gfExp(byte(r), c)
		}
	}
	return raw
}

func gfExp(a byte, n int) byte {
	if n == 0 {
		return 1
	}
	if a == 0 {
		return 0
	}
	logA := logTbl[a]
	logResult := int(logA) * n
	for logResult >= 255 {
		logResult -= 255
	}
	return byte(expTbl[logResult])
}

func gfMul(a, b byte) byte {
	return mulTbl[a][b]
}

// Multiply multiplies this matrix (the one on the left) by another
// matrix (the one on the right) and returns a new matrix with the result.
func (m matrix) mul(right matrix) matrix {
	result := newMatrix(len(m), len(right[0]))
	for r, row := range result {
		for c := range row {
			var value byte
			for i := range m[0] {
				value ^= gfMul(m[r][i], right[i][c])
			}
			result[r][c] = value
		}
	}
	return result
}
