package reedsolomon

import "errors"

type matrix []byte

func newMatrix(rows, cols int) matrix {
	m := make([]byte, rows*cols)
	return m
}

func genEncMatrixCauchy(data, parity int) matrix {
	rows := data + parity
	cols := data
	m := newMatrix(rows, cols)
	for i := 0; i < cols; i++ {
		m[i*data+i] = byte(1)
	}

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

func genVandMatrix(rows, cols int) matrix {
	m := newMatrix(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m[i*cols+j] = gfExp(byte(i), j)
		}
	}
	return m
}

func gfExp(b byte, n int) byte {
	if n == 0 {
		return 1
	}
	if b == 0 {
		return 0
	}
	a := logTbl[b]
	ret := int(a) * n
	for ret >= 255 {
		ret -= 255
	}
	return byte(expTbl[ret])
}

// TODO vand test
func genEncMatrixVand(data, parity int) (matrix, error) {
	rows := data + parity
	vm := genVandMatrix(rows, data)
	top := newMatrix(data, data)
	copy(top, vm[:data*data])
	topI, err := top.invert(data)
	if err != nil {
		return nil, err
	}
	return vm.mul(topI, data+parity, data), nil
}

func (m matrix) mul(right matrix, rows, cols int) matrix {
	ret := newMatrix(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			var v byte
			for k := 0; k < cols; k++ {
				v ^= gfMul(m[i*cols+k], right[k*cols+j])
			}
			ret[i*cols+j] = v
		}
	}
	return ret
}

func (m matrix) invert(n int) (matrix, error) {
	raw := newMatrix(n, 2*n)
	// [m] -> [m|I]
	for i := 0; i < n; i++ {
		t := i * n
		copy(raw[2*t:2*t+n], m[t:t+n])
		raw[2*t+i+n] = byte(1)
	}
	// [m|I] -> [I|m'] TODO I?
	err := raw.gaussJordan(n, 2*n)
	if err != nil {
		return nil, err
	}
	// [I|m'] -> [m']
	return raw.subMatrix(n), nil
}

var errSingular = errors.New("rs.invert: matrix is singular")

func (m matrix) gaussJordan(rows, columns int) error {
	for r := 0; r < rows; r++ {
		if m[2*r*rows+r] == 0 {
			for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
				if m[2*rowBelow*rows+r] != 0 {
					m.swap(r, rowBelow, 2*rows)
					break
				}
			}
		}
		if m[2*r*rows+r] == 0 {
			return errSingular
		}
		if m[2*r*rows+r] != 1 {
			d := m[2*r*rows+r]
			scale := inverseTbl[d]
			for c := 0; c < columns; c++ {
				m[2*r*rows+c] = gfMul(m[2*r*rows+c], scale)
			}
		}
		for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
			if m[2*rowBelow*rows+r] != 0 {
				scale := m[2*rowBelow*rows+r]
				for c := 0; c < columns; c++ {
					m[2*rowBelow*rows+c] ^= gfMul(scale, m[2*r*rows+c])
				}
			}
		}
	}
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

func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}

func gfMul(a, b byte) byte {
	return mulTbl[a][b]
}

func (m matrix) subMatrix(size int) matrix {
	ret := newMatrix(size, size)
	for i := 0; i < size; i++ {
		copy(ret[i*size:i*size+size], m[2*i*size+size:2*i*size+2*size])
	}
	return ret
}
