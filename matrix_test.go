// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"bytes"
	"flag"
	"fmt"
	"math/bits"
	"testing"
)

func TestMakeEncodeMatrix(t *testing.T) {
	act := makeEncodeMatrix(4, 4)
	exp := []byte{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
		71, 167, 122, 186,
		167, 71, 186, 122,
		122, 186, 71, 167,
		186, 122, 167, 71}
	if !bytes.Equal(act, exp) {
		t.Fatal("mismatch")
	}
}

func TestMatrixSwap(t *testing.T) {
	n := 7
	m := make([]byte, n*n)
	fillRandom(m)
	exp := make([]byte, n*n)
	copy(exp, m)
	matrix(exp).swap(0, 1, n)
	matrix(exp).swap(0, 1, n)
	if !bytes.Equal(exp, m) {
		t.Fatalf("swap mismatch")
	}
}

func TestMatrixInvert(t *testing.T) {
	testCases := []struct {
		matrixData  []byte
		n           int
		expect      []byte
		ok          bool
		expectedErr error
	}{
		{
			[]byte{
				56, 23, 98,
				3, 100, 200,
				45, 201, 123},
			3,
			[]byte{
				175, 133, 33,
				130, 13, 245,
				112, 35, 126},
			true,
			nil,
		},
		{
			[]byte{
				0, 23, 98,
				3, 100, 200,
				45, 201, 123},
			3,
			[]byte{
				245, 128, 152,
				188, 64, 135,
				231, 81, 239},
			true,
			nil,
		},
		{
			[]byte{
				1, 0, 0, 0, 0,
				0, 1, 0, 0, 0,
				0, 0, 0, 1, 0,
				0, 0, 0, 0, 1,
				7, 7, 6, 6, 1},
			5,
			[]byte{
				1, 0, 0, 0, 0,
				0, 1, 0, 0, 0,
				123, 123, 1, 122, 122,
				0, 0, 1, 0, 0,
				0, 0, 0, 1, 0},
			true,
			nil,
		},
		{
			[]byte{
				4, 2,
				12, 6},
			2,
			nil,
			false,
			ErrSingularMatrix,
		},
		{
			[]byte{7, 8, 9},
			2,
			nil,
			false,
			ErrNotSquare,
		},
	}

	for i, c := range testCases {
		m := matrix(c.matrixData)
		actual, actualErr := m.invert(c.n)
		if actualErr != nil && c.ok {
			t.Errorf("case.%d, expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !c.ok {
			t.Errorf("case.%d, expected to fail with <ERROR> \"%s\", but passed", i+1, c.expectedErr)
		}
		if actualErr != nil && !c.ok {
			if c.expectedErr != actualErr {
				t.Errorf("case.%d, expected to fail with error \"%s\", but instead failed with error \"%s\"", i+1, c.expectedErr, actualErr)
			}
		}
		if actualErr == nil && c.ok {
			if !bytes.Equal(c.expect, actual) {
				t.Errorf("case.%d, mismatch", i+1)
			}
		}
	}
}

func TestMakeEncMatrixForReconst(t *testing.T) {
	d, p := 4, 4
	em := makeEncodeMatrix(d, p)
	survivied, _ := genIdxForTest(d, p, d, p)
	emr, err := em.makeEncMatrixForReconst(survivied)
	if err != nil {
		t.Fatal(err)
	}
	hasM := make([]byte, d*d)
	for i, h := range survivied {
		copy(hasM[i*d:i*d+d], em[h*d:h*d+d])
	}
	if !mul(emr, hasM, d).isIdentity(d) {
		t.Fatal("make wrong encoding matrix for reconstruction")
	}
}

// Check all sub-matrices when one or more vectors are missing.
// Warning:
// Do not use very large numbers here.
// The number of combinations can explode and make the test impractical.
func TestEncMatrixInvertibleAll(t *testing.T) {
	testEncMatrixInvertible(t, 10, 4)
	testEncMatrixInvertible(t, 15, 4)
}

func testEncMatrixInvertible(t *testing.T, d, p int) {
	encMatrix := makeEncodeMatrix(d, p)
	var bitmap uint64
	cnt := 0
	// More missing vectors means a larger bitmap range.
	var min uint64 = (1 << (d + 1)) - 1 ^ (1 << (d - 1)) // Min value when lost one data row vector.
	var max uint64 = ((1 << d) - 1) << p                 // Max value when lost parity-num data row vectors.
	for bitmap = min; bitmap <= max; bitmap++ {
		if bits.OnesCount64(bitmap) != d {
			continue
		}
		cnt++
		v := bitmap
		dpHas := make([]int, d)
		c := 0
		for i := 0; i < d+p; i++ {
			var j uint64 = 1 << i
			if j&v == j {
				dpHas[c] = i
				c++
			}
		}

		m := make([]byte, d*d)
		for i := 0; i < d; i++ {
			copy(m[i*d:i*d+d], encMatrix[dpHas[i]*d:dpHas[i]*d+d])
		}
		im, err := matrix(m).invert(d)
		if err != nil {
			t.Fatalf("encode matrix is singular, d:%d, p:%d, dpHas:%#v", d, p, dpHas)
		}

		// Verify A * A' = I to validate inversion.
		if !mul(im, m, d).isIdentity(d) {
			t.Fatalf("matrix invert wrong, d:%d, p:%d, dpHas:%#v", d, p, dpHas)
		}
	}
	t.Logf("%d+%d pass invertible test, total submatrix(with lost): %d", d, p, cnt)
}

var Invertible = flag.Bool("invert-test", false,
	"checking encoding matrices' sub-matrices are invertible or not by pick up sub-matrix randomly")

// Check whether encoding matrix sub-matrices are invertible.
// Randomly sample one sub-matrix for each data+parity pair.
//
// This test may take about 100 seconds.
// Unless you modified Galois-field or matrix logic, running it every time is unnecessary,
// so it is skipped by default to save development time.
func TestEncMatrixInvertibleRandom(t *testing.T) {

	if !*Invertible {
		t.Skip("skip the test, because it may cost too much time")
	}

	for d := 1; d < 256; d++ {
		for p := 1; p < 256; p++ {
			if d+p > 256 {
				continue
			}

			encMatrix := makeEncodeMatrix(d, p)
			survived, _ := genIdxForTest(d, p, d, p)
			m := make([]byte, d*d)
			for i := 0; i < d; i++ {
				copy(m[i*d:i*d+d], encMatrix[survived[i]*d:survived[i]*d+d])
			}

			im, err := matrix(m).invert(d)
			if err != nil {
				t.Fatalf("encode matrix is singular, d:%d, p:%d, dpHas:%#v", d, p, survived)
			}

			// Verify A * A' = I to validate inversion.
			if !mul(im, m, d).isIdentity(d) {
				t.Fatalf("matrix invert wrong, d:%d, p:%d, dpHas:%#v", d, p, survived)
			}
		}
	}
}

// mul multiplies two n*n matrices.
func mul(a, b matrix, n int) (out matrix) {

	out = make([]byte, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			d := byte(0)
			for k := 0; k < n; k++ {
				d ^= gfMul(a[n*i+k], b[n*k+j])

			}
			out[i*n+j] = d
		}
	}
	return
}

func (m matrix) isIdentity(n int) bool {
	im := make([]byte, n*n)
	for i := 0; i < n; i++ {
		im[i*n+i] = 1
	}
	return bytes.Equal(m, im)
}

func BenchmarkMakeEncMatrixForReconst(b *testing.B) {
	dps := [][2]int{ // data, parity
		{4, 4},
		{10, 4},
		{16, 16},
		{64, 64},
		{128, 128},
		{255, 1},
		{256, 0},
	}
	benchMatrixInvertRun(b, dps)
}

func benchMatrixInvertRun(b *testing.B, dps [][2]int) {
	for _, dp := range dps {
		d, p := dp[0], dp[1]
		b.Run(fmt.Sprintf("(%d+%d)", d, p), func(b *testing.B) {
			m := makeEncodeMatrix(d, p)
			survived, _ := genIdxForTest(d, p, d, p)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := m.makeEncMatrixForReconst(survived)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
