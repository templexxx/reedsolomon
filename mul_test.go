// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestMulVect(t *testing.T) {
	max := testSize

	testMulVect(t, max, base, -1)

	switch getCPUFeature() {
	case avx512:
		testMulVect(t, max, avx2, base)
		testMulVect(t, max, avx512, avx2)
	case avx2:
		testMulVect(t, max, avx2, base)
	}
}

func testMulVect(t *testing.T, maxSize, feat, cmpFeat int) {

	rand.Seed(time.Now().UnixNano())

	fs := featToStr(feat)

	start, n := 1, 1
	if feat != base {
		start, n = 16, 16 // The min size for SIMD instructions.
	}

	for size := start; size <= maxSize; size += n {
		for c := 0; c <= 255; c++ {
			d := make([]byte, size)
			act := make([]byte, size)
			fillRandom(d)

			mulVect(byte(c), d, act, feat)

			exp := make([]byte, size)
			if cmpFeat < 0 {
				for i, v := range d {
					exp[i] = gfmul(uint8(c), v)
				}
			} else {
				mulVect(byte(c), d, exp, base)
			}

			if !bytes.Equal(act, exp) {
				t.Fatalf("%s mismatched with %s, size: %d",
					fs, featToStr(cmpFeat), size)
			}
		}
	}

	t.Logf("%s pass, max_size: %d",
		fs, maxSize)
}

func TestMulVectXOR(t *testing.T) {
	max := testSize

	testMulVectXOR(t, max, base, -1)

	switch getCPUFeature() {
	case avx512:
		testMulVectXOR(t, max, avx2, base)
		testMulVectXOR(t, max, avx512, avx2)
	case avx2:
		testMulVectXOR(t, max, avx2, base)
	}
}

func testMulVectXOR(t *testing.T, maxSize, feat, cmpFeat int) {

	rand.Seed(time.Now().UnixNano())

	fs := featToStr(feat)

	start, n := 1, 1
	if feat != base {
		start, n = 16, 16 // The min size for SIMD instructions.
	}

	for size := start; size <= maxSize; size += n {

		for c := 0; c <= 255; c++ {
			d := make([]byte, size)
			act := make([]byte, size)
			fillRandom(d)
			fillRandom(act)
			exp := make([]byte, size)
			copy(exp, act)
			mulVectXOR(byte(c), d, act, feat)

			if cmpFeat < 0 {
				for i, v := range d {
					exp[i] ^= gfmul(uint8(c), v)
				}
			} else {
				mulVectXOR(byte(c), d, exp, cmpFeat)
			}

			if !bytes.Equal(act, exp) {
				t.Fatalf("%s mismatched, size: %d", fs, size)
			}
		}
	}

	t.Logf("%s pass, max_size: %d",
		fs, maxSize)
}
