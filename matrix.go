// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"errors"
)

// matrix stores row*column bytes in a single flat slice.
// A 1D layout is used instead of a 2D slice because encoding/generator matrices
// are usually small (at most a few hundred bytes), and this layout is generally
// more cache-friendly and GC-friendly.
type matrix []byte

// makeEncodeMatrix builds an encoding matrix.
// Upper part: identity matrix.
// Lower part: Cauchy matrix.
// The encoding matrix matches Intel ISA-L's gf_gen_cauchy1_matrix:
// https://github.com/intel/isa-l/blob/master/erasure_code/ec_base.c
//
// Note:
// This is not the most common approach for building encoding matrices.
// Mixing this library with other erasure-code implementations may produce
// incompatible results.
//
// The common approach is to start from a Vandermonde matrix and use
// elementary transformations to make the upper part identity.
// That approach is slightly more complex.
//
// A known incorrect pattern (documented in ISA-L and in this repository)
// is to combine an upper identity matrix with a lower Vandermonde matrix directly;
// that can produce singular sub-matrices.
// See invertible.jpeg for a proof.
func makeEncodeMatrix(d, p int) matrix {
	r := d + p
	m := make([]byte, r*d)
	// Build upper identity matrix.
	for i := 0; i < d; i++ {
		m[i*d+i] = 1
	}

	// Build lower Cauchy matrix: 1/(i+j), where 0 <= j < d and d <= i < 2*d.
	off := d * d // Skip the identity matrix.
	for i := d; i < r; i++ {
		for j := 0; j < d; j++ {
			m[off] = inverseTbl[i^j]
			off++
		}
	}
	return m
}

func (m matrix) makeReconstMatrix(survived, needReconst []int) (rm matrix, err error) {

	d, nn := len(survived), len(needReconst)
	rm = make([]byte, nn*d)
	for i, l := range needReconst {
		copy(rm[i*d:i*d+d], m[l*d:l*d+d])
	}
	return
}

// makeEncMatrixForReconst computes an encoding matrix for reconstruction by
// inverting the survived portion of the original encoding matrix.
func (m matrix) makeEncMatrixForReconst(survived []int) (em matrix, err error) {
	d := len(survived)
	m2 := make([]byte, d*d)
	for i, l := range survived {
		copy(m2[i*d:i*d+d], m[l*d:l*d+d])
	}
	em, err = matrix(m2).invert(len(survived))
	if err != nil {
		return
	}
	return
}

var ErrNotSquare = errors.New("not a square matrix")
var ErrSingularMatrix = errors.New("matrix is singular")

// invert computes and returns m's inverse matrix.
func (m matrix) invert(n int) (inv matrix, err error) {
	if n*n != len(m) {
		err = ErrNotSquare
		return
	}

	mm := make([]byte, 2*n*n)
	left := mm[:n*n]
	copy(left, m) // Copy m, avoiding side effect.

	// Build the identity matrix on the right side.
	inv = mm[n*n:]
	for i := 0; i < n; i++ {
		inv[i*n+i] = 1
	}

	for i := 0; i < n; i++ {
		// Pivot if needed.
		if left[i*n+i] == 0 {
			// Find and swap with a row whose current-column value is non-zero.
			// If none exists, the matrix is singular.
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
			// Scale the row so the pivot becomes 1.
			for j := 0; j < n; j++ {
				left[i*n+j] = gfMul(left[i*n+j], v)
				inv[i*n+j] = gfMul(inv[i*n+j], v)
			}
		}

		// Use elementary row operations to eliminate all non-pivot entries
		// in the current column.
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}

			v := left[j*n+i]
			if v != 0 {
				for k := 0; k < n; k++ {
					left[j*n+k] ^= gfMul(v, left[i*n+k])
					inv[j*n+k] ^= gfMul(v, inv[i*n+k])
				}
			}
		}
	}

	return
}

// swap swaps row i and row j of an n*n matrix.
func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}
