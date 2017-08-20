package reedsolomon

import (
	"errors"
)

type matrix [][]byte

func newMatrix(rows, cols int) matrix {
	m := matrix(make([][]byte, rows))
	for i := range m {
		m[i] = make([]byte, cols)
	}
	return m
}

// generate a EncodeMatrix : identity_matrix(upper) cauchy_matrix(lower)
func genEncMatrixCauchy(d, p int) matrix {
	rows := d + p
	cols := d
	m := newMatrix(rows, cols)
	// identity matrix
	for j := 0; j < cols; j++ {
		m[j][j] = byte(1)
	}
	// cauchy matrix
	for i := cols; i < rows; i++ {
		for j := 0; j < cols; j++ {
			d := i ^ j
			a := inverseTbl[d]
			m[i][j] = byte(a)
		}
	}
	return m
}

func genCauchyMatrix(d, p int) matrix {
	rows := d + p
	cols := d
	m := newMatrix(p, cols)
	start := 0
	for i := cols; i < rows; i++ {
		for j := 0; j < cols; j++ {
			d := i ^ j
			a := inverseTbl[d]
			m[start][j] = byte(a)
		}
		start++
	}
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
	topInv, err := top.invert()
	if err != nil {
		return nil, err
	}
	return vm.mul(topInv), nil
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

func (m matrix) invert() (matrix, error) {
	size := len(m)
	iM := identityMatrix(size)
	mIM, _ := m.augIM(iM)

	err := mIM.gaussJordan()
	if err != nil {
		return nil, err
	}
	return mIM.subMatrix(size), nil
}

func identityMatrix(n int) matrix {
	m := newMatrix(n, n)
	for i := 0; i < n; i++ {
		m[i][i] = byte(1)
	}
	return m
}

// IN -> (IN|I)
func (m matrix) augIM(iM matrix) (matrix, error) {
	result := newMatrix(len(m), len(m[0])+len(iM[0]))
	for r, row := range m {
		for c := range row {
			result[r][c] = m[r][c]
		}
		cols := len(m[0])
		for c := range iM[0] {
			result[r][cols+c] = iM[r][c]
		}
	}
	return result, nil
}

var ErrSingular = errors.New("reedsolomon: matrix is singular")

// (IN|I) -> (I|OUT)
func (m matrix) gaussJordan() error {
	rows := len(m)
	columns := len(m[0])
	// clear out the part below the main diagonal and scale the main
	// diagonal to be 1.
	for r := 0; r < rows; r++ {
		// if the element on the diagonal is 0, find a row below
		// that has a non-zero and swap them.
		if m[r][r] == 0 {
			for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
				if m[rowBelow][r] != 0 {
					m.swapRows(r, rowBelow)
					break
				}
			}
		}
		// after swap, if we find all elements in this column is 0, it means the matrix's det is 0
		if m[r][r] == 0 {
			return ErrSingular
		}
		// scale to 1.
		if m[r][r] != 1 {
			d := m[r][r]
			scale := inverseTbl[d]
			// every element(this column) * m[e][e]'s mc
			for c := 0; c < columns; c++ {
				m[r][c] = gfMul(m[r][c], scale)
			}
		}
		// make everything below the 1 be a 0 by subtracting a multiple of it
		for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
			if m[rowBelow][r] != 0 {
				// scale * m[e][e] = scale, scale + scale = 0
				// makes m[e][e+1] = 0 , then calc left elements
				scale := m[rowBelow][r]
				for c := 0; c < columns; c++ {
					m[rowBelow][c] ^= gfMul(scale, m[r][c])
				}
			}
		}
	}
	// now clear the part above the main diagonal.
	// same logic with clean upper
	for d := 0; d < rows; d++ {
		for rowAbove := 0; rowAbove < d; rowAbove++ {
			if m[rowAbove][d] != 0 {
				scale := m[rowAbove][d]
				for c := 0; c < columns; c++ {
					m[rowAbove][c] ^= gfMul(scale, m[d][c])
				}
			}
		}
	}
	return nil
}

// (I|OUT) -> OUT
func (m matrix) subMatrix(size int) matrix {
	result := newMatrix(size, size)
	for r := 0; r < size; r++ {
		for c := size; c < size*2; c++ {
			result[r][c-size] = m[r][c]
		}
	}
	return result
}

// exchanges two rows in the matrix.
func (m matrix) swapRows(r1, r2 int) {
	m[r2], m[r1] = m[r1], m[r2]
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
