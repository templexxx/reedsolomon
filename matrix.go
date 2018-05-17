package reedsolomon

import "errors"

// matrix row*cols bytes
type matrix []byte

// genEncMatrix generate encoding matrix. upper: Identity_Matrix; lower: Cauchy_Matrix
func genEncMatrix(d, p int) matrix {
	r := d + p
	m := make([]byte, r*d)
	// create identity matrix upper
	for i := 0; i < d; i++ {
		m[i*d+i] = byte(1)
	}
	// create cauchy matrix below
	off := d * d // offset of encMatrix
	for i := d; i < r; i++ {
		for j := 0; j < d; j++ {
			d := i ^ j
			a := inverseTbl[d]
			m[off] = byte(a)
			off++
		}
	}
	return m
}

// [A|B] -> [B]
func (m matrix) subMatrix(n int) (b matrix) {
	b = matrix(make([]byte, n*n))
	for i := 0; i < n; i++ {
		off := i * n
		copy(b[off:off+n], m[2*off+n:2*(off+n)])
	}
	return
}

var ErrNoSquare = errors.New("not a square matrix")

func (m matrix) invert(n int) (im matrix, err error) { // im: inverse_matrix of m
	if n != len(m)/n {
		err = ErrNoSquare
		return
	}
	// step1: (m) -> (m|I)
	mI := matrix(make([]byte, 2*n*n))
	off := 0
	for i := 0; i < n; i++ {
		copy(mI[2*off:2*off+n], m[off:off+n])
		mI[2*off+n+i] = byte(1)
		off += n
	}
	// step2: Gaussian Elimination
	err = mI.gauss(n)
	im = mI.subMatrix(n)
	return
}

// swap row[i] & row[j], col = n
func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}

var ErrSingularMatrix = errors.New("matrix is singular")

// (A|I) -> (I|A')
func (m matrix) gauss(n int) error {
	c := 2 * n // c: cols_num of m
	// main_diagonal(left_part) -> 1 & left_part -> upper_triangular
	for i := 0; i < n; i++ {
		// m[i*c+i]: element of main_diagonal(left_part)
		if m[i*c+i] == 0 { // swap until get a non-zero element
			for j := i + 1; j < n; j++ {
				if m[j*c+i] != 0 {
					m.swap(i, j, c)
					break
				}
			}
		}
		if m[i*c+i] == 0 { // all element in one col are zero
			return ErrSingularMatrix
		}
		// main_diagonal(left_part) -> 1
		if m[i*c+i] != 1 {
			e := m[i*c+i]
			s := inverseTbl[e] // s * e = 1
			for j := 0; j < c; j++ {
				m[i*c+j] = mulTbl[m[i*c+j]][s] // all element * s (in i row)
			}
		}
		// left_part -> upper_triangular
		for j := i + 1; j < n; j++ {
			if m[j*c+i] != 0 {
				s := m[j*c+i] // s ^ (s * m[i*c+i]) = 0, m[i*c+i] = 1
				for k := 0; k < c; k++ {
					m[j*c+k] ^= mulTbl[s][m[i*c+k]] // all element ^ (s * row_i[k]) (in j row)
				}
			}
		}
	}
	// element upper main_diagonal(left_part) -> 0
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			if m[j*c+i] != 0 {
				s := m[j*c+i] // s ^ (s * m[i*c+i]) = 0, m[i*c+i] = 1
				for k := 0; k < c; k++ {
					m[j*c+k] ^= mulTbl[s][m[i*c+k]] // all element ^ (s * row_i[k]) (in j row)
				}
			}
		}
	}
	return nil
}
