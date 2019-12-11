// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"errors"
)

// matrix: row*column bytes,
// I use one slice but not 2D slice,
// because type matrix is only as encoding/generator matrix's
// container, hundreds bytes at most,
// so it maybe more cache-friendly and GC-friendly.
type matrix []byte

// makeSurvivedMatrix makes an encoding matrix.
// High portion: Identity Matrix;
// Lower portion: Cauchy Matrix.
// The Encoding Matrix is as same as Intel ISA-L's gf_gen_cauchy1_matrix:
// https://github.com/intel/isa-l/blob/master/erasure_code/ec_base.c
//
// Warn:
// It maybe not the common way to make an encoding matrix,
// so it may corrupt when mix this lib with other erasure codes libs.
//
// The common way to make a encoding matrix is using a
// Vandermonde Matrix, then use elementary transformation
// to make an identity matrix in the high portion of the matrix.
// But it's a little complicated.
//
// And there is a wrong way to use Vandermonde Matrix
// (see Intel ISA-L, and this lib's document warn the issue),
// in the wrong way, they use an identity matrix in the high portion,
// and a Vandermonde matrix in the lower directly,
// and this encoding matrix's submatrix maybe singular.
// You can find a proof in invertible.jpeg.
func makeEncodeMatrix(d, p int) matrix {
	r := d + p
	m := make([]byte, r*d)
	// Create identity matrix upper.
	for i := 0; i < d; i++ {
		m[i*d+i] = 1
	}

	// Create cauchy matrix below. (1/(i + j), 0 <= j < d, d <= i < 2*d)
	off := d * d // Skip the identity matrix.
	for i := d; i < r; i++ {
		for j := 0; j < d; j++ {
			m[off] = inverseTbl[i^j]
			off++
		}
	}
	return m
}

// makeReconstMatrix is according to
// m(encoding matrix for reconstruction,
// see "func (m matrix) makeEncMatrixForReconst(dpHas []int) (em matrix, err error)")
// & dpHas & dLost(data lost)
// to make a new matrix for reconstructing.
// Warn:
// len(dpHas) must = dataNum,
// you may need to cut the dpHas before use this method.
func (m matrix) makeReconstMatrix(dpHas, dLost []int) (rm matrix, err error) {

	d, lostN := len(dpHas), len(dLost)
	rm = make([]byte, lostN*d)
	for i, l := range dLost {
		copy(rm[i*d:i*d+d], m[l*d:l*d+d])
	}
	return
}

// makeEncMatrixForReconst makes an encoding matrix by calculating
// the inverse matrix of survived encoding matrix.
func (m matrix) makeEncMatrixForReconst(dpHas []int) (em matrix, err error) {
	d := len(dpHas)
	hm := make([]byte, d*d)
	for i, l := range dpHas {
		copy(hm[i*d:i*d+d], m[l*d:l*d+d])
	}
	em, err = matrix(hm).invert(len(dpHas))
	if err != nil {
		return
	}
	return
}

var ErrNotSquare = errors.New("not a square matrix")
var ErrSingularMatrix = errors.New("matrix is singular")

// invert calculates m's inverse matrix,
// and return it or any error.
func (m matrix) invert(n int) (inv matrix, err error) {
	if n*n != len(m) {
		err = ErrNotSquare
		return
	}

	mm := make([]byte, 2*n*n)
	left := mm[:n*n]
	copy(left, m) // Copy m, avoiding side affect.

	// Make an identity matrix.
	inv = mm[n*n:]
	for i := 0; i < n; i++ {
		inv[i*n+i] = 1
	}

	for i := 0; i < n; i++ {
		// Pivoting.
		if left[i*n+i] == 0 {
			// Find a row with non-zero in current column and swap.
			// If there is no one, means it's a singular matrix.
			var j int
			for j = i + 1; j < n; j++ {
				if left[j*n+i] != 0 {
					break
				}
			}
			if j == n {
				return nil, ErrSingularMatrix
			}

			matrix(left).swap(i, j, n)
			inv.swap(i, j, n)
		}

		if left[i*n+i] != 1 {
			v := inverseTbl[left[i*n+i]] // 1/pivot
			// Scale row.
			for j := 0; j < n; j++ {
				left[i*n+j] = gfmul(left[i*n+j], v)
				inv[i*n+j] = gfmul(inv[i*n+j], v)
			}
		}

		// Use elementary transformation to
		// make all elements(except pivot) in the left matrix
		// become 0.
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}

			v := left[j*n+i]
			if v != 0 {
				for k := 0; k < n; k++ {
					left[j*n+k] ^= gfmul(v, left[i*n+k])
					inv[j*n+k] ^= gfmul(v, inv[i*n+k])
				}
			}
		}
	}

	return
}

// swap square matrix row[i] & row[j], col = n
func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}
