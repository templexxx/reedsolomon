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

func genEncMatrixVand(data, parity int) (matrix, error) {
	total := data + parity
	vm := genVandMatrix(total, data)
	top := newMatrix(data, data)
	copy(top, vm[:data*data])
	topI, err := top.invert(data)
	if err != nil {
		return nil, err
	}
	return vm.mul(topI, total, data), nil
}

func (m matrix) invert(n int) (matrix, error) {
	raw := newMatrix(n, 2*n)
	// [m] -> [m|I]
	for i := 0; i < n; i++ {
		t := i * n
		copy(raw[2*t:2*t+n], m[t:t+n])
		raw[2*t+i+n] = byte(1)
	}
	// [m|I] -> [I|m']
	err := raw.gauss(n, 2*n)
	if err != nil {
		return nil, err
	}
	// [I|m'] -> [m']
	return raw.subMatrix(n), nil
}

func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}

func gfMul(a, b byte) byte {
	return mulTbl[a][b]
}

var errSingular = errors.New("rs.invert: matrix is singular")

func (m matrix) gauss(rows, cols int) error {
	for i := 0; i < rows; i++ {
		if m[i*cols+i] == 0 {
			for j := i + 1; j < rows; j++ {
				if m[j*cols+i] != 0 {
					m.swap(i, j, cols)
					break
				}
			}
		}
		if m[i*cols+i] == 0 {
			return errSingular
		}
		if m[i*cols+i] != 1 {
			d := m[i*cols+i]
			scale := inverseTbl[d]
			for c := 0; c < cols; c++ {
				m[i*cols+c] = gfMul(m[i*cols+c], scale)
			}
		}
		for j := i + 1; j < rows; j++ {
			if m[j*cols+i] != 0 {
				scale := m[j*cols+i]
				for c := 0; c < cols; c++ {
					m[j*cols+c] ^= gfMul(scale, m[i*cols+c])
				}
			}
		}
	}
	for k := 0; k < rows; k++ {
		for j := 0; j < k; j++ {
			if m[j*cols+k] != 0 {
				scale := m[j*cols+k]
				for c := 0; c < cols; c++ {
					m[j*cols+c] ^= gfMul(scale, m[k*cols+c])
				}
			}
		}
	}
	return nil
}

func (m matrix) mul(right matrix, rows, cols int) matrix {
	r := newMatrix(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			var v byte
			for k := 0; k < cols; k++ {
				v ^= gfMul(m[i*cols+k], right[k*cols+j])
			}
			r[i*cols+j] = v
		}
	}
	return r
}

func (m matrix) subMatrix(n int) matrix {
	r := newMatrix(n, n)
	for i := 0; i < n; i++ {
		off := i * n
		copy(r[off:off+n], m[2*off+n:2*(off+n)])
	}
	return r
}
